package job

import (
	"context"
	"strings"

	"github.com/NautiluX/managed-velero-plugin/pkg/k8s"

	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/lithammer/shortuuid/v3"
	"github.com/speps/go-hashids/v2"
)

const (
	BaseName           string = "managed-velero-plugin-status-patch"
	ServiceAccountName        = BaseName + "-sa"
	RoleName                  = BaseName + "-role"
	RoleBindingName           = BaseName + "-rolebinding"
	MigrationNamespace        = "cluster-migration"
)

func RestoreStateRequired(kind string) bool {
	switch kind {
	case
		"Account",
		"AccountClaim",
		"AwsFederatedAccountAccess",
		"CertificateRequest",
		"ClusterDeployment",
		"ClusterSync",
		"ProjectClaim",
		"DNSZone",
		"ProjectReference":
		return true
	}
	return false
}

func CreateJob(cr string) error {
	genericObject, err := k8s.GetGenericObject(cr)
	if err != nil {
		return err
	}
	c, err := k8s.GetClient()
	if err != nil {
		return err
	}

	namespace := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: MigrationNamespace,
		},
	}
	_ = c.Create(context.TODO(), &namespace)

	serviceAccount := v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ServiceAccountName,
			Namespace: MigrationNamespace,
		},
	}
	_ = c.Create(context.TODO(), &serviceAccount)

	clusterRole := rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:      RoleName,
			Namespace: MigrationNamespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"certman.managed.openshift.io"},
				Resources: []string{
					"*",
				},
				Verbs: []string{"*"},
			},
			{
				APIGroups: []string{"gcp.managed.openshift.io"},
				Resources: []string{
					"*",
				},
				Verbs: []string{"*"},
			},
			{
				APIGroups: []string{"hive.openshift.io"},
				Resources: []string{
					"*",
				},
				Verbs: []string{"*"},
			},
			{
				APIGroups: []string{"monitoring.openshift.io"},
				Resources: []string{
					"*",
				},
				Verbs: []string{"*"},
			},
			{
				APIGroups: []string{"aws.managed.openshift.io"},
				Resources: []string{
					"*",
					"accountclaims",
					"accounts",
					"accountpools",
					"awsfederatedaccountaccesses",
					"awsfederatedroles",
				},
				Verbs: []string{"*"},
			},
		},
	}
	_ = c.Create(context.TODO(), &clusterRole)

	clusterRoleBinding := rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: RoleBindingName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      ServiceAccountName,
				Namespace: MigrationNamespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     RoleName,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
	_ = c.Create(context.TODO(), &clusterRoleBinding)

	hd := hashids.NewData()
	hd.Salt = shortuuid.New()
	hd.Alphabet = "1234567890abcdefghijklmnopqrstuvwxyz"
	h, _ := hashids.NewWithData(hd)
	id, _ := h.Encode([]int{1, 2, 3})
	job := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "apply-status-" + strings.ToLower(genericObject.Kind) + "-" + strings.ToLower(id),
			Namespace: MigrationNamespace,
			Labels: map[string]string{
				"restore-kind":      genericObject.Kind,
				"restore-name":      genericObject.Metadata.Name,
				"restore-namespace": genericObject.Metadata.Namespace,
			},
		},
		Spec: batchv1.JobSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					RestartPolicy:      v1.RestartPolicyNever,
					ServiceAccountName: ServiceAccountName,
					Containers: []v1.Container{
						{
							Name:  "apply-status",
							Image: "quay.io/mdewald/managed-velero-plugin-status-patch",
							Args:  []string{cr},
						},
					},
				},
			},
		},
	}
	err = c.Create(context.TODO(), &job)
	return err
}
