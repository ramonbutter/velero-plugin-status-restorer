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
	//"github.com/openshift/client-go/route/clientset/versioned/scheme"
	"context"
	"sync"
	"time"

	uuid "github.com/nu7hatch/gouuid"
	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"

	monitoringv1alpha1 "github.com/openshift/route-monitor-operator/api/v1alpha1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	"k8s.io/apimachinery/pkg/runtime"
)

// RestorePlugin is a restore item action plugin for Velero
type RestorePlugin struct {
	log                    logrus.FieldLogger
	waitGroup              sync.WaitGroup
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
	//	Item:           obj,					// modified object (status cleared)
	//	ItemFromBackup: itemFromBackup,			// original from backup
	//	Restore:        ctx.restore,			/ ????
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

	// status restore
	content := input.Item.UnstructuredContent()
	content["status"] = input.ItemFromBackup.UnstructuredContent()["status"]
	input.Item.SetUnstructuredContent(content)

	if content["status"] != ""{
		p.waitGroup.Add(1)
		go p.addStatus(content)
	}

	p.log.Info("--------------------------------------------")
	p.log.Info(input.Item.UnstructuredContent())
	p.log.Info("--------------------------------------------")

	return velero.NewRestoreItemActionExecuteOutput(input.Item), nil
}
func (p *RestorePlugin)addStatus(content map[string]interface{}) {
	u, err := uuid.NewV4()
	p.log = p.log.WithFields(logrus.Fields{
		"thread":          u.String(),
	})

	namespace := content["metadata"].(map[string]interface{})["namespace"].(string)
	name := content["metadata"].(map[string]interface{})["name"].(string)

	p.log.Info("Starting GoRoutine for %s ", name)
	config, err := rest.InClusterConfig()
	crdConfig := *config
	crdConfig.ContentConfig.GroupVersion = &schema.GroupVersion{Group: "monitoring.openshift.io", Version: "v1alpha1"}
	crdConfig.APIPath = "/apis"

	scheme := runtime.NewScheme()

	utilruntime.Must(monitoringv1alpha1.AddToScheme(scheme))

	crdConfig.NegotiatedSerializer = serializer.NewCodecFactory(scheme)
	crdConfig.UserAgent = rest.DefaultKubernetesUserAgent()

	if err != nil {
		panic(err)
	}


	cr := monitoringv1alpha1.RouteMonitor{}

	client, _ := rest.UnversionedRESTClientFor(&crdConfig)

	for i := 0; i < 1000; i++{
		
		client.Get().
			Namespace(namespace).
			Name(name).
			Resource("routemonitors").
			Do(context.Background()).
			Into(&cr)
        
		p.log.Info("CR: %v", cr)
		p.log.Info("CR: %v", cr.Name)

		//client.``

		time.Sleep(10*time.Second)

	}
	p.log.Info("Finished GoRoutine")

}