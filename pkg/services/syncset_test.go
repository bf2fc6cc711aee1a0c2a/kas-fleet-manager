package services

import (
	"errors"
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/syncsetresources"
	"reflect"
	"strings"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	strimzi "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/kafka.strimzi.io/v1beta1"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/ocm"
	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	projectv1 "github.com/openshift/api/project/v1"
	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	testCanaryName      string
	testAdminServerName string
)

func init() {
	// applying same rule for truncate and K8S sanitizing Kafka request name
	sanitizedtestKafkaRequestName, _ := replaceNamespaceSpecialChar(fmt.Sprintf("%s-%s", truncateString(testKafkaRequestName, truncatedNameLen), strings.ToLower(testID)))
	testCanaryName = sanitizedtestKafkaRequestName + "-canary"
	testAdminServerName = sanitizedtestKafkaRequestName + "-admin-server"
}

// build a test project object
func buildProject(modifyFn func(project *projectv1.Project)) *projectv1.Project {
	project := &projectv1.Project{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "project.openshift.io/v1",
			Kind:       "Project",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   fmt.Sprintf("%s-%s", testUser, testID),
			Labels: constants.NamespaceLabels,
		},
	}
	if modifyFn != nil {
		modifyFn(project)
	}
	return project
}

// build a test kafka object
func buildKafka(modifyFn func(kafka *strimzi.Kafka)) *strimzi.Kafka {
	kafka := &strimzi.Kafka{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testKafkaRequestName,
			Namespace: fmt.Sprintf("%s-%s", testUser, testID),
			Labels: map[string]string{
				syncsetresources.IngressLabelValue: syncsetresources.IngressLabelName,
			},
		},
	}
	if modifyFn != nil {
		modifyFn(kafka)
	}
	return kafka
}

// build a test canary object
func buildCanary(modifyFn func(canary *appsv1.Deployment)) *appsv1.Deployment {
	canary := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      testCanaryName,
			Namespace: fmt.Sprintf("%s-%s", testUser, testID),
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":                                    testCanaryName,
					constants.ObservabilityCanaryPodLabelKey: constants.ObservabilityCanaryPodLabelValue,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":                                    testCanaryName,
						constants.ObservabilityCanaryPodLabelKey: constants.ObservabilityCanaryPodLabelValue,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  testCanaryName,
							Image: "quay.io/ppatierno/strimzi-canary:0.0.1-1",
							Env: []corev1.EnvVar{
								{
									Name:  "KAFKA_BOOTSTRAP_SERVERS",
									Value: testKafkaRequestName + "-kafka-bootstrap:9092",
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
	if modifyFn != nil {
		modifyFn(canary)
	}
	return canary
}

// build a test admin server object
func buildAdminServer(modifyFn func(adminServer *appsv1.Deployment)) *appsv1.Deployment {
	adminServer := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      testAdminServerName,
			Namespace: fmt.Sprintf("%s-%s", testUser, testID),
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": testAdminServerName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": testAdminServerName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  testAdminServerName,
							Image: "quay.io/sknot/kafka-admin-api:0.0.1",
							Env: []corev1.EnvVar{
								{
									Name:  "KAFKA_ADMIN_BOOTSTRAP_SERVERS",
									Value: testKafkaRequestName + "-kafka-bootstrap:9095",
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
	if modifyFn != nil {
		modifyFn(adminServer)
	}
	return adminServer
}

// build a test admin server service object
func buildAdminServerService(modifyFn func(adminServerService *corev1.Service)) *corev1.Service {
	adminServerService := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      testAdminServerName,
			Namespace: fmt.Sprintf("%s-%s", testUser, testID),
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": testAdminServerName,
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
	if modifyFn != nil {
		modifyFn(adminServerService)
	}
	return adminServerService
}

// build a test admin server route object
func buildAdminServerRoute(modifyFn func(adminServerService *routev1.Route)) *routev1.Route {
	adminServerRoute := &routev1.Route{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "route.openshift.io/v1",
			Kind:       "Route",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      testAdminServerName,
			Namespace: fmt.Sprintf("%s-%s", testUser, testID),
			Labels: map[string]string{
				syncsetresources.IngressLabelName: syncsetresources.IngressLabelValue,
			},
		},
		Spec: routev1.RouteSpec{
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: testAdminServerName,
			},
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString("http"),
			},
			Host: "admin-server-" + fmt.Sprintf("%s-%s.clusterDNS", testKafkaRequestName, testID),
			TLS: &routev1.TLSConfig{
				Termination: routev1.TLSTerminationEdge,
			},
		},
	}
	if modifyFn != nil {
		modifyFn(adminServerRoute)
	}
	return adminServerRoute
}

func TestSyncsetService_Create(t *testing.T) {
	type fields struct {
		ocmClient ocm.Client
	}
	keycloakConfig := config.NewKeycloakConfig()
	kafkaConfig := config.NewKafkaConfig()

	kafkaSyncBuilder, _, _ := newKafkaSyncsetBuilder(&api.KafkaRequest{
		Name:      testKafkaRequestName,
		ClusterID: testClusterID,
	}, kafkaConfig, keycloakConfig, "")

	type args struct {
		syncsetBuilder *v1.SyncsetBuilder
	}

	wantedSynceset, _ := kafkaSyncBuilder.ID(testSyncsetID).Build()

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		want    *v1.Syncset
	}{
		{
			name: "successful syncset creation",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					CreateSyncSetFunc: func(clusterId string, syncset *v1.Syncset) (*v1.Syncset, error) {
						return syncset, nil
					},
				},
			},
			args: args{
				syncsetBuilder: kafkaSyncBuilder,
			},
			want: wantedSynceset,
		},
		{
			name: "failed syncset creation",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					CreateSyncSetFunc: func(clusterId string, syncset *v1.Syncset) (*v1.Syncset, error) {
						return syncset, errors.New("MGD-SERV-API-9: failed to create syncset: test-syncset-id for cluster id: test-cluster-id")
					},
				},
			},
			args: args{
				syncsetBuilder: kafkaSyncBuilder,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &syncsetService{
				ocmClient: tt.fields.ocmClient,
			}
			got, err := k.Create(tt.args.syncsetBuilder, testSyncsetID, testClusterID)

			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() got = %v, want %v", got, tt.want)
			}

		})
	}
}

func TestSyncsetService_Delete(t *testing.T) {
	type fields struct {
		ocmClient ocm.Client
	}
	keycloakConfig := config.NewKeycloakConfig()
	kafkaConfig := config.NewKafkaConfig()

	kafkaSyncBuilder, _, _ := newKafkaSyncsetBuilder(&api.KafkaRequest{
		Name:      testKafkaRequestName,
		ClusterID: testClusterID,
	}, kafkaConfig, keycloakConfig, "")

	type args struct {
		syncsetBuilder *v1.SyncsetBuilder
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "successful syncset deletion",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					DeleteSyncSetFunc: func(clusterID string, syncsetID string) (int, error) {
						return 204, nil
					},
				},
			},
			args: args{
				syncsetBuilder: kafkaSyncBuilder,
			},
		},
		{
			name: "failed to syncset deletion",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					DeleteSyncSetFunc: func(clusterID string, syncsetId string) (int, error) {
						return 0, errors.New("MGD-SERV-API-9: failed to delete syncset: test-syncset-id for cluster id")
					},
				},
			},
			args: args{
				syncsetBuilder: kafkaSyncBuilder,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &syncsetService{
				ocmClient: tt.fields.ocmClient,
			}
			_, err := k.Delete(testSyncsetID, testClusterID)

			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_newKafkaSyncsetBuilder(t *testing.T) {
	type args struct {
		kafkaRequest *api.KafkaRequest
	}
	tests := []struct {
		name string
		args args
		want []interface{}
	}{
		{
			name: "build syncset with singleAZ Kafka successfully",
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *api.KafkaRequest) {
					kafkaRequest.BootstrapServerHost = fmt.Sprintf("%s-%s.clusterDNS", testKafkaRequestName, testID)
				}),
			},
			want: []interface{}{
				buildProject(nil),
				buildKafka(func(kafka *strimzi.Kafka) {
					kafka.Spec.Kafka.Template = &strimzi.KafkaTemplate{
						Pod: &strimzi.PodTemplate{
							Affinity: &corev1.Affinity{
								PodAntiAffinity: &corev1.PodAntiAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
										{
											TopologyKey: "kubernetes.io/hostname",
										},
									},
								},
							},
						},
					}
					kafka.Spec.Zookeeper.Template = &strimzi.ZookeeperTemplate{
						Pod: &strimzi.PodTemplate{
							Affinity: &corev1.Affinity{
								PodAntiAffinity: &corev1.PodAntiAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
										{
											TopologyKey: "kubernetes.io/hostname",
										},
									},
								},
							},
						},
					}
				}),
				buildCanary(nil),
				buildAdminServer(nil),
				buildAdminServerService(nil),
				buildAdminServerRoute(nil),
				&corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Secret",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      testKafkaRequestName + "-sso-secret",
						Namespace: fmt.Sprintf("%s-%s", testUser, testID),
					},
					Type: corev1.SecretType("Opaque"),
					Data: map[string][]byte{
						"ssoClientSecret": []byte(nil),
					},
				},
				&corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Secret",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      testKafkaRequestName + "-sso-cert",
						Namespace: fmt.Sprintf("%s-%s", testUser, testID),
					},
					Type: corev1.SecretType("Opaque"),
					Data: map[string][]byte{
						"keycloak.crt": []byte(nil),
					},
				},
			},
		},
		{
			name: "build syncset with multiAZ Kafka successfully",
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *api.KafkaRequest) {
					kafkaRequest.BootstrapServerHost = fmt.Sprintf("%s-%s.clusterDNS", testKafkaRequestName, testID)
					kafkaRequest.MultiAZ = true
				}),
			},
			want: []interface{}{
				buildProject(nil),
				buildKafka(func(kafka *strimzi.Kafka) {
					kafka.Spec.Kafka.Rack = &strimzi.Rack{
						TopologyKey: "topology.kubernetes.io/zone",
					}
					kafka.Spec.Kafka.Template = &strimzi.KafkaTemplate{
						Pod: &strimzi.PodTemplate{
							Affinity: &corev1.Affinity{
								PodAntiAffinity: &corev1.PodAntiAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
										{
											TopologyKey: "kubernetes.io/hostname",
										},
									},
								},
							},
						},
					}
					kafka.Spec.Zookeeper.Template = &strimzi.ZookeeperTemplate{
						Pod: &strimzi.PodTemplate{
							Affinity: &corev1.Affinity{
								PodAntiAffinity: &corev1.PodAntiAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
										{
											TopologyKey: "kubernetes.io/hostname",
										},
										{
											TopologyKey: "topology.kubernetes.io/zone",
										},
									},
								},
							},
						},
					}
				}),
				buildCanary(nil),
				buildAdminServer(nil),
				buildAdminServerService(nil),
				buildAdminServerRoute(nil),
				&corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Secret",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      testKafkaRequestName + "-sso-secret",
						Namespace: fmt.Sprintf("%s-%s", testUser, testID),
					},
					Type: corev1.SecretType("Opaque"),
					Data: map[string][]byte{
						"ssoClientSecret": []byte(nil),
					},
				},
				&corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Secret",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      testKafkaRequestName + "-sso-cert",
						Namespace: fmt.Sprintf("%s-%s", testUser, testID),
					},
					Type: corev1.SecretType("Opaque"),
					Data: map[string][]byte{
						"keycloak.crt": []byte(nil),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keycloakConfig := config.KeycloakConfig{}
			keycloakConfig.EnableAuthenticationOnKafka = true

			kafkaConfig := config.NewKafkaConfig()

			got, _, _ := newKafkaSyncsetBuilder(tt.args.kafkaRequest, kafkaConfig, &keycloakConfig, "")
			syncset, err := got.Build()
			if err != nil {
				t.Errorf("newKafkaSyncsetBuilder() failed to build syncset")
			}

			if len(syncset.Resources()) != len(tt.want) {
				t.Errorf("newKafkaSyncsetBuilder() number of resources doesn't match. got = %v want = %v", len(syncset.Resources()), len(tt.want))
			}

			for i, resource := range syncset.Resources() {
				switch resource.(type) {
				case *strimzi.Kafka:
					gotKafka, ok := resource.(*strimzi.Kafka)
					if !ok {
						t.Errorf("newKafkaSyncsetBuilder() failed to convert Kafka resource")
					}
					wantKafka, _ := tt.want[i].(*strimzi.Kafka)

					// compare AZ config
					if !reflect.DeepEqual(gotKafka.Spec.Kafka.Rack, wantKafka.Spec.Kafka.Rack) {
						t.Errorf("newKafkaSyncsetBuilder() kafka rack config doesn't match. got = %v want = %v", gotKafka.Spec.Kafka.Rack, wantKafka.Spec.Kafka.Rack)
					}
					if !reflect.DeepEqual(gotKafka.Spec.Zookeeper.Template, wantKafka.Spec.Zookeeper.Template) {
						t.Errorf("newKafkaSyncsetBuilder() zookeeper template doesn't match. got = %v want = %v", gotKafka.Spec.Zookeeper.Template, wantKafka.Spec.Zookeeper.Template)
					}
				case *corev1.Secret:
					gotSecret, ok := resource.(*corev1.Secret)
					if !ok {
						t.Errorf("newKafkaSyncsetBuilder() failed to convert secret resource")
					}
					wantKafka, _ := tt.want[i].(*corev1.Secret)
					if gotSecret.ObjectMeta.Name == wantKafka.ObjectMeta.Name {
						if !reflect.DeepEqual(gotSecret.ObjectMeta, wantKafka.ObjectMeta) {
							t.Errorf("newKafkaSyncsetBuilder() secret meta doesn't match. got = %v want = %v", gotSecret.ObjectMeta, wantKafka.ObjectMeta)
						}
					}

				default:
					if !reflect.DeepEqual(resource, tt.want[i]) {
						t.Errorf("newKafkaSyncsetBuilder() got = %v want = %v", resource, tt.want[i])
					}
				}
			}
		})
	}
}
