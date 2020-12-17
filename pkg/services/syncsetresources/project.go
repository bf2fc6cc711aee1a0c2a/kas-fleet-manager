package syncsetresources

import (
	projectv1 "github.com/openshift/api/project/v1"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/constants"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Builds an OpenShift project
func BuildProject(name string) *projectv1.Project {
	return &projectv1.Project{
		TypeMeta: metav1.TypeMeta{
			APIVersion: projectv1.SchemeGroupVersion.String(),
			Kind:       "Project",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: constants.NamespaceLabels,
		},
	}
}
