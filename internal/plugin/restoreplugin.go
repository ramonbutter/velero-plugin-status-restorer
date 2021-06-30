/*
Copyright 2018, 2019 the Velero contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package plugin

import (
	"context"
	"fmt"
	"log"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	uuid "github.com/nu7hatch/gouuid"
	"github.com/sirupsen/logrus"
	client "github.com/vmware-tanzu/velero/pkg/client"

	//pkgrestore "github.com/vmware-tanzu/velero/pkg/restore"

	"github.com/vmware-tanzu/velero/pkg/plugin/velero"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// RestorePlugin is a restore item action plugin for Velero
type RestorePlugin struct {
	log                    logrus.FieldLogger
}

// NewRestorePlugin instantiates a RestorePlugin.
func NewRestorePlugin(log logrus.FieldLogger) *RestorePlugin {
	return &RestorePlugin{log: log}
}

// AppliesTo returns information about which resources this action should be invoked for.
// The IncludedResources and ExcludedResources slices can include both resources
// and resources with group names. These work: "ingresses", "ingresses.extensions".
// A RestoreItemAction's Execute function will only be invoked on items that match the returned
// selector. A zero-valued ResourceSelector matches all resources.g
func (p *RestorePlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{}, nil
}


// Execute allows the RestorePlugin to perform arbitrary logic with the item being restored,
// in this case, setting a custom annotation on the item being restored.
func (p *RestorePlugin) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {
	p.log.Info("Hello from my RestorePlugin!")
	p.log.Info("0.6!")

	//executeOutput, err := action.Execute(&velero.RestoreItemActionExecuteInput{
	//	Item:           obj,					// modified object (status clepanic: assignment to entry in nil map

	//})

	metadata, err := meta.Accessor(input.Item)
	if err != nil {
		return &velero.RestoreItemActionExecuteOutput{}, err
	}

	annotations := metadata.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	annotations["velero.io/my-restore-plugin"] = "1"
	metadata.SetAnnotations(annotations)



	content := input.ItemFromBackup.UnstructuredContent()
	if content["status"] != ""{
		var obj unstructured.Unstructured
		obj.SetUnstructuredContent(input.ItemFromBackup.UnstructuredContent())
		go NewStatusRestorer(p.log).addStatus(obj)
	}

	return velero.NewRestoreItemActionExecuteOutput(input.Item), nil
}



/*
 *	Restores Status from input CR
 *
 */
type StatusRestorer struct {
	log                    logrus.FieldLogger
	dynamicClient		   dynamic.Interface
	resourceClients        map[resourceClientKey]client.Dynamic
}
type resourceClientKey struct {
	resource  schema.GroupVersionResource
	namespace string
}

func NewStatusRestorer(log logrus.FieldLogger) *StatusRestorer{
    config := userConfig()
	c, err := dynamic.NewForConfig(config)
	errExit("Failed to create client", err)
	return &StatusRestorer{	log: log, 
							dynamicClient  : c,
						  }
}


func (p *StatusRestorer)addStatus(content unstructured.Unstructured) {
	u, err := uuid.NewV4()
	p.log = p.log.WithFields(logrus.Fields{
		"thread":          u.String(),
	})

	namespace := content.GetNamespace()
	name := content.GetName()
	p.log.Info("Starting GoRoutine for %s ", name)
	p.log.Info("--------------------------------------------")
	fmt.Println(content)
	p.log.Info("--------------------------------------------")
	
	gv, _ := schema.ParseGroupVersion(content.GetAPIVersion())
	resource := strings.ToLower(fmt.Sprint(content.GetKind(),"s"))
	gvr := gv.WithResource(resource)

	p.log.Infof("Processing namespace %q, resource %q", namespace, &gvr)

	c := p.dynamicClient.Resource(gvr).Namespace(namespace)
	errExit(fmt.Sprintf("error getting resource client for namespace %q, resource %q", namespace, &gvr), err)
	
	for i := 0; i < 1000; i++{

		rv, err := c.Get(context.TODO(), name, metav1.GetOptions{})
		errLog("Failed to get Resource", err)

		if rv == nil {
			p.log.Info("Waiting for CR to be created: ", name)
			time.Sleep(1*time.Second)
			continue
		}

		_content := rv.UnstructuredContent()
		_content["status"] = content.UnstructuredContent()["status"]
		rv.SetUnstructuredContent(_content)
		
		rv, err = c.UpdateStatus(context.TODO(), rv, metav1.UpdateOptions{})
		errLog("Failed to update Status: ", err)
	}
	p.log.Info("Finished GoRoutine")
}


func userConfig() *rest.Config {

	// In Cluster Condition
	log.Printf("Fetching In-Cluster Kube API config")
	cfg, err := rest.InClusterConfig()
	errLog("Failed to in-Cluster conifg", err)

	// User kubefile config
	log.Printf("Fetching user kubefile config")
	usr, err := user.Current()
	errLog("Failed to get current user", err)
	path := filepath.Join(usr.HomeDir, ".kube", "config")
	cfg, err = clientcmd.BuildConfigFromFlags("", path)
	errExit("Failed to get user config", err)

	log.Print("Loading default set")
	c, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	errExit("Failed to load", err)
	clientConfig := clientcmd.NewDefaultClientConfig(*c, nil)
	cfg, err = clientConfig.ClientConfig()
	
	log.Print("KubeAPI is at ", cfg.Host)
	return cfg
}



func errExit(msg string, err error) {
	if err != nil {
		log.Fatalf("%s: %#v", msg, err)
	}
}

func errLog(msg string, err error){
	if err != nil {
		log.Printf("%s: %#v", msg, err)
	}
}