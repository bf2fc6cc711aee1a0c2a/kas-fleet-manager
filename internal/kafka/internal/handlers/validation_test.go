package handlers

import (
	"context"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/onsi/gomega"
	"net/http"
	"testing"
)

func Test_Validation_validateKafkaClusterNameIsUnique(t *testing.T) {
	type args struct {
		kafkaService services.KafkaService
		name         string
		context      context.Context
	}

	tests := []struct {
		name string
		arg  args
		want *errors.ServiceError
	}{
		{
			name: "throw an error when the KafkaService call throws an error",
			arg: args{
				kafkaService: &services.KafkaServiceMock{
					ListFunc: func(ctx context.Context, listArgs *coreServices.ListArguments) (api.KafkaList, *api.PagingMeta, *errors.ServiceError) {
						return nil, &api.PagingMeta{Total: 4}, errors.GeneralError("count failed from database")
					},
				},
				name:    "some-name",
				context: context.TODO(),
			},
			want: errors.GeneralError("count failed from database"),
		},
		{
			name: "throw an error when name is already used",
			arg: args{
				kafkaService: &services.KafkaServiceMock{
					ListFunc: func(ctx context.Context, listArgs *coreServices.ListArguments) (api.KafkaList, *api.PagingMeta, *errors.ServiceError) {
						return nil, &api.PagingMeta{Total: 1}, nil
					},
				},
				name:    "duplicate-name",
				context: context.TODO(),
			},
			want: &errors.ServiceError{
				HttpCode: http.StatusConflict,
				Reason:   "Kafka cluster name is already used",
				Code:     36,
			},
		},
		{
			name: "does not throw an error when name is unique",
			arg: args{
				kafkaService: &services.KafkaServiceMock{
					ListFunc: func(ctx context.Context, listArgs *coreServices.ListArguments) (api.KafkaList, *api.PagingMeta, *errors.ServiceError) {
						return nil, &api.PagingMeta{Total: 0}, nil
					},
				},
				name:    "unique-name",
				context: context.TODO(),
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			validateFn := ValidateKafkaClusterNameIsUnique(&tt.arg.name, tt.arg.kafkaService, tt.arg.context)
			err := validateFn()
			gomega.Expect(tt.want).To(gomega.Equal(err))
		})
	}
}

func Test_Validations_validateKafkaClusterNames(t *testing.T) {
	tests := []struct {
		description string
		name        string
		expectError bool
	}{
		{
			description: "valid kafka cluster name",
			name:        "test-kafka1",
			expectError: false,
		},
		{
			description: "valid kafka cluster name with multiple '-'",
			name:        "test-my-cluster",
			expectError: false,
		},
		{
			description: "invalid kafka cluster name begins with number",
			name:        "1test-cluster",
			expectError: true,
		},
		{
			description: "invalid kafka cluster name with invalid characters",
			name:        "test-c%*_2",
			expectError: true,
		},
		{
			description: "invalid kafka cluster name with upper-case letters",
			name:        "Test-cluster",
			expectError: true,
		},
		{
			description: "invalid kafka cluster name with spaces",
			name:        "test cluster",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			validateFn := ValidKafkaClusterName(&tt.name, "name")
			err := validateFn()
			if tt.expectError {
				gomega.Expect(err).Should(gomega.HaveOccurred())
			} else {
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			}
		})
	}
}
