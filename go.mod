module github.com/ramonbutter/velero-plugin-status-saver

go 1.14

require (
	github.com/NautiluX/managed-velero-plugin v0.0.0-20210630104730-31f9e165143e
	github.com/lithammer/shortuuid/v3 v3.0.7
	github.com/openshift/api v0.0.0-20200917102736-0a191b5b9bb0
	github.com/openshift/aws-account-operator/pkg/apis v0.0.0-20210625194031-783ea73f0d13
	github.com/openshift/route-monitor-operator v0.0.0-20210623125755-21d41321079c
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.48.1
	github.com/sirupsen/logrus v1.8.1
	github.com/speps/go-hashids/v2 v2.0.1
	github.com/vmware-tanzu/velero v1.6.1
	k8s.io/api v0.21.2
	k8s.io/apimachinery v0.21.2
	k8s.io/cli-runtime v0.21.2
	k8s.io/client-go v0.21.2
	sigs.k8s.io/controller-runtime v0.9.2
)
