package migrations

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"

	"github.com/onsi/gomega"
)

const mockKafkaRequestID = "1ilzo99dVkVAoQNJeovhP8pFIzS"
const mockIDWithInvalidChars = "vp&xG^nl9MStC@SI*#c$6V^TKq0"

func Test_buildOldKafkaNamespace(t *testing.T) {
	mockKafkaName := "example-kafka"
	mockKafkaNamespace := "a1ilzo99dvkvaoqnjeovhp8pfizs"

	type args struct {
		kafkaRequest *dbapi.KafkaRequest
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Build namespace name successfully",
			args: args{
				kafkaRequest: &dbapi.KafkaRequest{
					Name: mockKafkaName,
					Meta: api.Meta{
						ID: mockKafkaRequestID,
					},
				},
			},
			wantErr: false,
			want:    mockKafkaNamespace,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			got, err := buildOldKafkaNamespace(tt.args.kafkaRequest)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildOldKafkaNamespace() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_buildKafkaNamespaceIdentifier(t *testing.T) {
	mockShortOwnerUsername := "sample_owner_username_short"
	mockLongOwnerUsername := fmt.Sprintf("sample_owner_username_long_%s", mockKafkaRequestID)
	namespaceLimit := 63 // Maximum namespace name length as validated by OpenShift

	type args struct {
		kafkaRequest *dbapi.KafkaRequest
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "build kafka namespace id successfully with a short owner username",
			args: args{
				kafkaRequest: &dbapi.KafkaRequest{
					Owner: mockShortOwnerUsername,
				},
			},
			want: fmt.Sprintf("%s-%s", mockShortOwnerUsername, strings.ToLower(mockKafkaRequestID)),
		},
		{
			name: "build kafka namespace id successfully with a long owner username",
			args: args{
				kafkaRequest: &dbapi.KafkaRequest{
					Owner: mockLongOwnerUsername,
				},
			},
			want: fmt.Sprintf("%s-%s", mockLongOwnerUsername[0:truncatedNamespaceLen], strings.ToLower(mockKafkaRequestID)),
		},
	}
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			tt.args.kafkaRequest.ID = mockKafkaRequestID
			got := buildKafkaNamespaceIdentifier(tt.args.kafkaRequest)
			if got != tt.want {
				t.Errorf("BuildKafkaNamespaceIdentifier() = %v, want %v", got, tt.want)
			}
			if len(got) > namespaceLimit {
				t.Errorf("BuildKafkaNamespaceIdentifier() namespace identifier length is %v, this is over the %v maximum namespace name limit", len(got), namespaceLimit)
			}
		})
	}
}

func Test_replaceNamespaceSpecialChar(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "replace all invalid characters in an invalid namespace name",
			args: args{
				name: fmt.Sprintf("project-%s-", mockIDWithInvalidChars),
			},
			want: "project-vp-xg-nl9mstc-si-c-6v-tkq0a",
		},
		{
			name: "valid namespace should not be modified",
			args: args{
				name: fmt.Sprintf("project-%s", mockKafkaRequestID),
			},
			want: fmt.Sprintf("project-%s", strings.ToLower(mockKafkaRequestID)),
		},
		{
			name: "should return an error if given namespace name is an empty string",
			args: args{
				name: "",
			},
			wantErr: true,
		},
	}
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			got, err := replaceNamespaceSpecialChar(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReplaceNamespaceSpecialChar() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ReplaceNamespaceSpecialChar() = %v, want %v", got, tt.want)
			}
		})
	}
}
