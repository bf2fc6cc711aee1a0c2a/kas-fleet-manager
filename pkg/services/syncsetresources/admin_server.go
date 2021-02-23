package syncsetresources

import (
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/config"

	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// BuildAdminServerResources builds the admin server deployment, service and route
func BuildAdminServerResources(kafkaRequest *api.KafkaRequest, kafkaConfig *config.KafkaConfig, kafkaName, namespace string) []interface{} {
	adminServerName := kafkaName + "-admin-server"

	adminServerResources := []interface{}{
		buildAdminServerDeployment(kafkaRequest, kafkaConfig, adminServerName, namespace),
		buildAdminServerService(adminServerName, namespace),
		buildAdminServerRoute(kafkaRequest, kafkaConfig, adminServerName, namespace),
	}
	return adminServerResources
}

func buildAdminServerDeployment(kafkaRequest *api.KafkaRequest, kafkaConfig *config.KafkaConfig, adminServerName, namespace string) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      adminServerName,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": adminServerName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": adminServerName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  adminServerName,
							Image: kafkaConfig.KafkaAdminServerImage,
							Env: []corev1.EnvVar{
								{
									Name:  "KAFKA_ADMIN_BOOTSTRAP_SERVERS",
									Value: kafkaRequest.Name + "-kafka-bootstrap:9095",
								},
								{
									Name:      "CORS_ALLOW_LIST_REGEX",
									Value:     "(https?:\\/\\/localhost(:\\d*)?)|(https:\\/\\/(qaprodauth\\.)?cloud\\.redhat\\.com)|(https:\\/\\/(prod|qa|ci|stage)\\.foo\\.redhat\\.com:1337)",
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 8080,
								},
							},
						},
					},
				},
			},
		},
	}
}

func buildAdminServerService(adminServerName, namespace string) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      adminServerName,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": adminServerName,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       8080,
					TargetPort: intstr.FromString("http"),
				},
			},
		},
	}
}

func buildAdminServerRoute(kafkaRequest *api.KafkaRequest, kafkaConfig *config.KafkaConfig, adminServerName, namespace string) *routev1.Route {
	adminServerRoute := &routev1.Route{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "route.openshift.io/v1",
			Kind:       "Route",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      adminServerName,
			Namespace: namespace,
			Labels:    getAdminServerRouteLabels(kafkaConfig),
		},
		Spec: routev1.RouteSpec{
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: adminServerName,
			},
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString("http"),
			},
			Host: "admin-server-" + kafkaRequest.BootstrapServerHost,
			TLS: &routev1.TLSConfig{
				Termination: routev1.TLSTerminationEdge,
			},
		},
	}

	if kafkaConfig.EnableKafkaExternalCertificate {
		adminServerRoute.Spec.TLS.Certificate = kafkaConfig.KafkaTLSCert
		adminServerRoute.Spec.TLS.Key = kafkaConfig.KafkaTLSKey
	}

	return adminServerRoute
}

func getAdminServerRouteLabels(_ *config.KafkaConfig) map[string]string {
	labels := make(map[string]string)
	labels[IngressLabelName] = IngressLabelValue // allows admin server route to use the sharded IngressController

	return labels
}
