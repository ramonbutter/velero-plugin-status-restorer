package plugin_test

import (
	"context"
	"testing"
	"time"

	"reflect"

	configv1 "github.com/openshift/api/config/v1"
	routev1 "github.com/openshift/api/route/v1"
	hivev1 "github.com/openshift/hive/apis/hive/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	plugin "github.com/ramonbutter/velero-plugin-status-saver/internal/plugin"
	logrus "github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewClient(flags *genericclioptions.ConfigFlags) (client.Client, error) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(hivev1.AddToScheme(scheme))
	utilruntime.Must(monitoringv1.AddToScheme(scheme))
	utilruntime.Must(routev1.AddToScheme(scheme))
	utilruntime.Must(configv1.AddToScheme(scheme))

	utilruntime.Must(hivev1.AddToScheme(scheme))

	// +kubebuilder:scaffold:scheme

	if flags == nil {
		flags = genericclioptions.NewConfigFlags(false)
	}

	configLoader := flags.ToRawKubeConfigLoader()
	cfg, err := configLoader.ClientConfig()

	if err != nil {
		return nil, err
	}

	cli, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}
	return cli, nil
}

func TestAbc(t *testing.T) {
	testClusterDeployment := hivev1.ClusterDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "cluster-123",
			Namespace:         "default",
			DeletionTimestamp: nil,
			Finalizers:        []string{"test-finalizer"},
		},
		Spec: hivev1.ClusterDeploymentSpec{
			ClusterName: "test-cluster",
			BaseDomain:  "my-domain",
		},
		Status: hivev1.ClusterDeploymentStatus{
			WebConsoleURL:   "console.my-domain.com",
			InstallRestarts: 3,
		},
	}
	namespacedName := types.NamespacedName{Name: testClusterDeployment.Name, Namespace: testClusterDeployment.Namespace}

	log := logrus.New()
	action := plugin.NewRestorePlugin(log)
	itemFromBackupMap, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(&testClusterDeployment)
	cleanedTestClusterDeployment := testClusterDeployment
	cleanedTestClusterDeployment.Status = hivev1.ClusterDeploymentStatus{}
	cleanedTestClusterDeployment.SetFinalizers([]string{})
	objMap, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(&cleanedTestClusterDeployment)

	// routemonitors.monitoring.openshift.io/v1alpha1
	g := schema.GroupVersionKind{
		Group:   "hive.openshift.io",
		Version: "v1",
		Kind:    "ClusterDeployment",
	}
	var obj unstructured.Unstructured
	var itemFromBackup unstructured.Unstructured
	obj.SetUnstructuredContent(objMap)
	obj.SetGroupVersionKind(g)
	itemFromBackup.SetUnstructuredContent(itemFromBackupMap)
	itemFromBackup.SetGroupVersionKind(g)

	testClient, err := NewClient(nil)
	if err != nil {
		log.Errorf("%v", err)
		t.FailNow()
	}
	log.Info("Deleting previous CR")
	deployedClusterDeployment := hivev1.ClusterDeployment{}
	err = testClient.Get(context.Background(), namespacedName, &deployedClusterDeployment)
	if err != nil {
		log.Errorf("%v", err)
	}
	deployedClusterDeployment.SetFinalizers([]string{})
	err = testClient.Update(context.Background(), &deployedClusterDeployment)
	if err != nil {
		log.Errorf("%v", err)
	}
	err = testClient.Delete(context.Background(), &deployedClusterDeployment)
	if err != nil {
		log.Errorf("%v", err)
	}

	log.Info("Executing Restore Plugin")
	executeOutput, _ := action.Execute(&velero.RestoreItemActionExecuteInput{
		Item:           &obj,
		ItemFromBackup: &itemFromBackup,
		Restore:        nil,
	})

	if !reflect.DeepEqual(obj.GetFinalizers(), itemFromBackup.GetFinalizers()) {
		log.Error("Failed to restore Finalizer")
		t.FailNow()
	}

	log.Info("Creating CR")
	// simulating velero behavior
	err = testClient.Create(context.Background(), &testClusterDeployment)
	if err != nil {
		log.Errorf("%v", err)
		t.FailNow()
	}

	if executeOutput == nil {
		t.Errorf("no output given")
	}

	log.Info("Waiting for status update")
	// wait for the status update by the plugin
	deployedClusterDeployment = hivev1.ClusterDeployment{}
	for i := 0; i < 10; i++ {
		time.Sleep(time.Second)
		err := testClient.Get(context.Background(), namespacedName, &deployedClusterDeployment)
		if err == nil {
			//log.Info("RouteMonitor came up: ", deployedRouteMonitor)
			if reflect.DeepEqual(deployedClusterDeployment.Status, testClusterDeployment.Status) &&
				reflect.DeepEqual(deployedClusterDeployment.ObjectMeta.Finalizers, testClusterDeployment.ObjectMeta.Finalizers) {
				t.Log("CR was created correctly:", deployedClusterDeployment)
				break
			}
		}
	}
}
