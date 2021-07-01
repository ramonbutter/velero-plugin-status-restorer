package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/ramonbutter/velero-plugin-status-saver/internal/patch/pkg/job"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"strings"

	//pkgrestore "github.com/vmware-tanzu/velero/pkg/restore"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	MaxCrWaitTime time.Duration = 5 * time.Minute
)

func main() {
	cr := []byte(os.Args[1])
	log := logrus.New()

	var obj unstructured.Unstructured
	json.Unmarshal(cr, &obj)

	start := time.Now()
	fmt.Printf("Input CR:\n---------\n%s\n---------\n\n", cr)
	fmt.Printf("Start watching for CR kind %s at %v for %v.\n", obj.GetKind(), start, MaxCrWaitTime)
	if !job.RestoreStateRequired(obj.GetKind()) {
		fmt.Printf("Skipping type %s\n", obj.GetKind())
		return
	}
	for time.Since(start) < MaxCrWaitTime {
		if NewStatusRestorer(log).addStatus(obj) {
			return
		}
		time.Sleep(10 * time.Second)
	}
	panic("Failed to update status")
}

/*
 *	Restores Status from input CR
 *
 */
type StatusRestorer struct {
	log           logrus.FieldLogger
	dynamicClient dynamic.Interface
}

func NewStatusRestorer(log logrus.FieldLogger) *StatusRestorer {
	config := userConfig()
	c, err := dynamic.NewForConfig(config)
	errExit("Failed to create client", err)
	return &StatusRestorer{
		log:           log,
		dynamicClient: c,
	}
}

// os.argv[1]
func (p *StatusRestorer) addStatus(content unstructured.Unstructured) bool {

	namespace := content.GetNamespace()
	name := content.GetName()
	p.log.Info("--------------------------------------------")
	p.log.Info(content)
	p.log.Info("--------------------------------------------")

	gv, _ := schema.ParseGroupVersion(content.GetAPIVersion())
	resource := strings.ToLower(fmt.Sprint(content.GetKind(), "s"))
	gvr := gv.WithResource(resource)

	p.log.Infof("Getting client for name %s namespace %s, resource %q", name, namespace, &gvr)

	c := p.dynamicClient.Resource(gvr).Namespace(namespace)

	rv, err := c.Get(context.TODO(), name, metav1.GetOptions{})
	errLog("Failed to get Resource", err)

	if rv == nil {
		p.log.Info("Waiting for CR to be created: ", name)
		time.Sleep(1 * time.Second)
		return false
	}

	_content := rv.UnstructuredContent()
	_content["status"] = content.UnstructuredContent()["status"]
	rv.SetUnstructuredContent(_content)

	_, err = c.UpdateStatus(context.TODO(), rv, metav1.UpdateOptions{})
	errLog("Failed to update Status: ", err)
	if err == nil {
		p.log.Info("Updated status successfully with ", _content["status"])
		return true
	}
	return false
}

func userConfig() *rest.Config {

	// In Cluster Condition
	log.Printf("Fetching In-Cluster Kube API config")
	cfg, err := rest.InClusterConfig()
	errLog("Failed to in-Cluster conifg", err)
	if err == nil {
		return cfg
	}

	// User kubefile config
	log.Printf("Fetching user kubefile config")
	usr, err := user.Current()
	errLog("Failed to get current user", err)
	path := filepath.Join(usr.HomeDir, ".kube", "config")
	cfg, err = clientcmd.BuildConfigFromFlags("", path)
	errLog("Failed to get user config", err)
	if err == nil {
		return cfg
	}

	log.Print("Loading default set")
	c, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	errLog("Failed to load", err)
	clientConfig := clientcmd.NewDefaultClientConfig(*c, nil)
	cfg, err = clientConfig.ClientConfig()
	log.Print("KubeAPI is at ", cfg.Host)
	if err == nil {
		return cfg
	}
	errExit("Failed to fetch kubeconfig", err)
	return nil
}

func errExit(msg string, err error) {
	if err != nil {
		log.Fatalf("%s: %#v", msg, err)
	}
}

func errLog(msg string, err error) {
	if err != nil {
		log.Printf("%s: %#v", msg, err)
	}
}
