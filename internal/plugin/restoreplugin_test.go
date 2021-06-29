package plugin_test

import (
	"testing"

	"github.com/openshift/route-monitor-operator/api/v1alpha1"
	plugin "github.com/rbutter/velero-plugin-example/internal/plugin"
	logrus "github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)


func TestAbc(t *testing.T) {
	routeMonitor := v1alpha1.RouteMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "scott-pilgrim",
			Namespace:         "the-world",
			DeletionTimestamp: nil,
			Finalizers:        []string{},
		},
		Status: v1alpha1.RouteMonitorStatus{
			RouteURL: "fake-route-url",
		},
		Spec: v1alpha1.RouteMonitorSpec{
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

	var obj unstructured.Unstructured
	var itemFromBackup unstructured.Unstructured
	obj.SetUnstructuredContent(objMap)
	itemFromBackup.SetUnstructuredContent(itemFromBackupMap)

	executeOutput, _ := action.Execute(&velero.RestoreItemActionExecuteInput{
		Item:           obj, 
		ItemFromBackup: itemFromBackup,
		Restore:        nil,
	})

	
	
}