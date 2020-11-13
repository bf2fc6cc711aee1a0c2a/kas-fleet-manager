package services

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	projectv1 "github.com/openshift/api/project/v1"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	strimzi "gitlab.cee.redhat.com/service/managed-services-api/pkg/api/kafka.strimzi.io/v1beta1"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/ocm"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// build a test project object
func buildProject(modifyFn func(project *projectv1.Project)) *projectv1.Project {
	project := &projectv1.Project{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "project.openshift.io/v1",
			Kind:       "Project",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", testUser, testID),
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
		},
	}
	if modifyFn != nil {
		modifyFn(kafka)
	}
	return kafka
}

func TestSyncsetService_Create(t *testing.T) {
	type fields struct {
		ocmClient ocm.Client
	}

	kafkaSyncBuilder, _, _ := newKafkaSyncsetBuilder(&api.KafkaRequest{
		Name:      testKafkaRequestName,
		ClusterID: testClusterID,
	})

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

	kafkaSyncBuilder, _, _ := newKafkaSyncsetBuilder(&api.KafkaRequest{
		Name:      testKafkaRequestName,
		ClusterID: testClusterID,
	})

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
			err := k.Delete(testSyncsetID, testClusterID)

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
	type want struct {
		project *projectv1.Project
		kafka   *strimzi.Kafka
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    []interface{}
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
				buildKafka(nil),
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
					kafka.Spec.Zookeeper.Template = &strimzi.ZookeeperTemplate{
						Pod: &strimzi.PodTemplate{
							Affinity: corev1.Affinity{
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
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, _ := newKafkaSyncsetBuilder(tt.args.kafkaRequest)
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
				default:
					if !reflect.DeepEqual(resource, tt.want[i]) {
						t.Errorf("newKafkaSyncsetBuilder() got = %v want = %v", resource, tt.want[i])
					}
				}
			}
		})
	}
}
