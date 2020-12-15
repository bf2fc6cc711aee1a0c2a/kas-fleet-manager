package syncsetresources

import (
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/config"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/constants"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildCanary builds a canary deployment
func BuildCanary(kafkaRequest *api.KafkaRequest, kafkaConfig *config.KafkaConfig, kafkaName, namespace string) *appsv1.Deployment {
	canaryName := kafkaName + "-canary"

	canary := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      canaryName,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: getCanaryLabels(canaryName),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: getCanaryLabels(canaryName),
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  canaryName,
							Image: kafkaConfig.KafkaCanaryImage,
							Env: []corev1.EnvVar{
								{
									Name:  "KAFKA_BOOTSTRAP_SERVERS",
									Value: kafkaRequest.Name + "-kafka-bootstrap:9092",
								},
								{
									Name:  "TOPIC_RECONCILE_MS",
									Value: "5000",
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "metrics",
									ContainerPort: 8080,
								},
							},
						},
					},
				},
			},
		},
	}

	return canary
}

func getCanaryLabels(canaryName string) map[string]string {
	return map[string]string{
		"app":                                    canaryName,
		constants.ObservabilityCanaryPodLabelKey: constants.ObservabilityCanaryPodLabelValue,
	}
}
