package plugin_test

import (
	"context"
	"testing"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/openshift/route-monitor-operator/api/v1alpha1"
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

	monitoringopenshiftiov1alpha1 "github.com/openshift/route-monitor-operator/api/v1alpha1"
	monitoringv1alpha1 "github.com/openshift/route-monitor-operator/api/v1alpha1"
)

func NewClient(flags *genericclioptions.ConfigFlags) (client.Client, error) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(monitoringv1alpha1.AddToScheme(scheme))
	utilruntime.Must(monitoringv1.AddToScheme(scheme))
	utilruntime.Must(routev1.AddToScheme(scheme))
	utilruntime.Must(configv1.AddToScheme(scheme))

	utilruntime.Must(monitoringopenshiftiov1alpha1.AddToScheme(scheme))

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
	routeMonitor := v1alpha1.RouteMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "scott-pilgrim",
			Namespace:         "openshift-route-monitor-operator",
			DeletionTimestamp: nil,
			Finalizers:        []string{},
		},
		Status: v1alpha1.RouteMonitorStatus{
			RouteURL: "fake-route-url",
		},
		Spec: v1alpha1.RouteMonitorSpec{
			Route: v1alpha1.RouteMonitorRouteSpec{
				Name:      "test",
				Namespace: "test-space",
			},
			Slo: v1alpha1.SloSpec{
				TargetAvailabilityPercent: "99.5",
			},
		},
	}

	log := logrus.New()
	action := plugin.NewRestorePlugin(log)
	itemFromBackupMap, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(&routeMonitor)
	routeMonitor.Status = v1alpha1.RouteMonitorStatus{}
	objMap, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(&routeMonitor)

	// routemonitors.monitoring.openshift.io/v1alpha1
	g := schema.GroupVersionKind{
		Group:   "monitoring.openshift.io",
		Version: "v1alpha1",
		Kind:    "RouteMonitor",
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
	}
	log.Info("Deleting previous RouteMonitor")
	err = testClient.Delete(context.Background(), &routeMonitor)
	if err != nil {
		log.Errorf("%v", err)
	}

	log.Info("Executing Restore Plugin")
	executeOutput, _ := action.Execute(&velero.RestoreItemActionExecuteInput{
		Item:           &obj,
		ItemFromBackup: &itemFromBackup,
		Restore:        nil,
	})

	log.Info("Creating RouteMonitor")
	// simulating velero behavior
	err = testClient.Create(context.Background(), &routeMonitor)
	if err != nil {
		log.Errorf("%v", err)
	}

	if executeOutput == nil {
		t.Errorf("bla")
	}

	log.Info("Waiting for status update")
	// wait for the status update by the plugin
	deployedRouteMonitor := v1alpha1.RouteMonitor{}
	namespacedName := types.NamespacedName{Name: routeMonitor.Name, Namespace: routeMonitor.Namespace}
	for i := 0; i < 10; i++ {
		testClient.Get(context.Background(), namespacedName, &deployedRouteMonitor)
		time.Sleep(time.Second)
		if deployedRouteMonitor.Name != "" {
			//log.Info("RouteMonitor came up: ", deployedRouteMonitor)
			if deployedRouteMonitor.Status != (v1alpha1.RouteMonitorStatus{}) {
				t.Log("Status is Set", deployedRouteMonitor)
				break
			}
		}
	}
}
