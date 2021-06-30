module github.com/rbutter/velero-plugin-example

go 1.14

require (
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d
	github.com/openshift/api v0.0.0-20200917102736-0a191b5b9bb0
	github.com/openshift/route-monitor-operator v0.0.0-20210623125755-21d41321079c
	github.com/pkg/errors v0.9.1
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.48.1
	github.com/sirupsen/logrus v1.8.1
	github.com/vmware-tanzu/velero v1.6.1
	k8s.io/api v0.21.2
	k8s.io/apiextensions-apiserver v0.21.2
	k8s.io/apimachinery v0.21.2
	k8s.io/cli-runtime v0.21.2
	k8s.io/client-go v0.21.2
	sigs.k8s.io/controller-runtime v0.9.2
)
