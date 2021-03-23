package syncsetresources

import (
	"fmt"
	"strings"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildKeycloakResources builds the Keycloak client and ca secret
func BuildKeycloakResources(kafkaRequest *api.KafkaRequest, keycloakConfig *config.KeycloakConfig, namespace string) []interface{} {
	keycloakResources := []interface{}{
		buildKeycloakClientSecret(keycloakConfig, kafkaRequest.Name, kafkaRequest.SsoClientSecret, namespace),
		buildKeycloakCASecret(keycloakConfig, kafkaRequest.Name, namespace),
	}

	return keycloakResources
}

func buildKeycloakClientSecret(keycloakConfig *config.KeycloakConfig, kafkaName, clientSecretValue, namespace string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      kafkaName + "-sso-secret",
			Namespace: namespace,
		},
		Type: corev1.SecretType("Opaque"),
		Data: map[string][]byte{
			constants.MASClientSecretKey: []byte(clientSecretValue),
		},
	}
}

func buildKeycloakCASecret(keycloakConfig *config.KeycloakConfig, kafkaName, namespace string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      kafkaName + "-sso-cert",
			Namespace: namespace,
		},
		Type: corev1.SecretType("Opaque"),
		Data: map[string][]byte{
			keycloakConfig.TLSTrustedCertificatesKey: []byte(keycloakConfig.TLSTrustedCertificatesValue),
		},
	}
}

// BuildKeycloakClientNameIdentifier builds an identifier based on the kafka request id
func BuildKeycloakClientNameIdentifier(kafkaRequestID string) string {
	return fmt.Sprintf("%s-%s", "kafka", strings.ToLower(kafkaRequestID))
}
