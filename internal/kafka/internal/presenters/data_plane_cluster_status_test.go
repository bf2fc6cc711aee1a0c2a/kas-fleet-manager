package presenters

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/private"
	mock "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/data_plane"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"

	. "github.com/onsi/gomega"
)

func TestConvertDataPlaneClusterStatus_AvailableStrimziVersions(t *testing.T) {
	tests := []struct {
		name                            string
		inputClusterUpdateStatusRequest *private.DataPlaneClusterUpdateStatusRequest
		want                            []api.StrimziVersion
		wantErr                         bool
	}{
		{
			name: "When setting a non empty ordered list of strimzi it is stored as is",
			inputClusterUpdateStatusRequest: mock.BuildValidDataPlaneClusterUpdateStatusRequest(func(statusRequest *private.DataPlaneClusterUpdateStatusRequest) {
				statusRequest.Strimzi = *mock.BuildDataPlaneClusterUpdateStatusRequestStrimzi(nil)
			}),
			want:    *mock.BuildApiStrimziVersions(nil),
			wantErr: false,
		},
		{
			name: "When setting a non empty unordered list of strimzi that list is stored in semver ascending order from the version attribute",
			inputClusterUpdateStatusRequest: mock.BuildValidDataPlaneClusterUpdateStatusRequest(func(statusRequest *private.DataPlaneClusterUpdateStatusRequest) {
				statusRequest.Strimzi = []private.DataPlaneClusterUpdateStatusRequestStrimzi{
					{Version: "v5.0.0-0", Ready: true},
					{Version: "v2.0.0-0", Ready: false},
					{Version: "v3.0.0-0", Ready: true},
				}
			}),
			want: []api.StrimziVersion{
				{Version: "v2.0.0-0", Ready: false},
				{Version: "v3.0.0-0", Ready: true},
				{Version: "v5.0.0-0", Ready: true},
			},
			wantErr: false,
		},
		{
			name: "When setting an empty list of strimzi that list is stored as the empty list",
			inputClusterUpdateStatusRequest: mock.BuildValidDataPlaneClusterUpdateStatusRequest(func(statusRequest *private.DataPlaneClusterUpdateStatusRequest) {
				statusRequest.Strimzi = []private.DataPlaneClusterUpdateStatusRequestStrimzi{}
			}),
			want:    []api.StrimziVersion{},
			wantErr: false,
		},
		{
			name: "When setting a nil list of strimzi that list is stored as the empty list",
			inputClusterUpdateStatusRequest: mock.BuildValidDataPlaneClusterUpdateStatusRequest(func(statusRequest *private.DataPlaneClusterUpdateStatusRequest) {
				statusRequest.Strimzi = nil
			}),
			want:    []api.StrimziVersion{},
			wantErr: false,
		},
		{
			name:                            "When strimzi nor strimziVersions are defined an empty list is returned",
			inputClusterUpdateStatusRequest: mock.BuildValidDataPlaneClusterUpdateStatusRequest(nil),
			want:                            []api.StrimziVersion{},
			wantErr:                         false,
		},
		{
			name: "When one of the versions in strimzi does not follow the expected format an error is returned",
			inputClusterUpdateStatusRequest: mock.BuildValidDataPlaneClusterUpdateStatusRequest(func(statusRequest *private.DataPlaneClusterUpdateStatusRequest) {
				statusRequest.Strimzi = []private.DataPlaneClusterUpdateStatusRequestStrimzi{
					{Version: "v1.0.0-0", Ready: true},
					{Version: "v2.0.0", Ready: false},
					{Version: "v3.0.0-0", Ready: true},
				}
			}),
			want:    nil,
			wantErr: true,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := ConvertDataPlaneClusterStatus(*tt.inputClusterUpdateStatusRequest)
			gotErr := err != nil
			Expect(gotErr).To(Equal(tt.wantErr))
			if !tt.wantErr {
				Expect(res.AvailableStrimziVersions).To(Equal(tt.want))
			}
		})
	}
}

func TestPresentDataPlaneClusterConfig(t *testing.T) {
	type args struct {
		config *dbapi.DataPlaneClusterConfig
	}

	tests := []struct {
		name string
		args args
		want private.DataplaneClusterAgentConfig
	}{
		{
			name: "should return converted DataplaneClusterAgentConfig",
			args: args{
				config: mock.BuildDataPlaneClusterConfig(nil),
			},
			want: mock.BuildDataplaneClusterAgentConfig(nil),
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(PresentDataPlaneClusterConfig(tt.args.config)).To(Equal(tt.want))
		})
	}
}
func sampleValidDataPlaneClusterUpdateStatusRequest() *private.DataPlaneClusterUpdateStatusRequest {
	return &private.DataPlaneClusterUpdateStatusRequest{
		Conditions: []private.DataPlaneClusterUpdateStatusRequestConditions{
			{
				Type:   "Ready",
				Status: "True",
			},
		},
		Total: private.DataPlaneClusterUpdateStatusRequestTotal{
			IngressThroughputPerSec: &[]string{"test"}[0],
			EgressThroughputPerSec:  &[]string{"test"}[0],
			Connections:             &[]int32{1000000}[0],
			DataRetentionSize:       &[]string{"test"}[0],
			Partitions:              &[]int32{1000000}[0],
		},
		NodeInfo: &private.DatePlaneClusterUpdateStatusRequestNodeInfo{
			Ceiling:                &[]int32{20}[0],
			Floor:                  &[]int32{3}[0],
			Current:                &[]int32{5}[0],
			CurrentWorkLoadMinimum: &[]int32{3}[0],
		},
		Remaining: private.DataPlaneClusterUpdateStatusRequestTotal{
			Connections:             &[]int32{1000000}[0],
			Partitions:              &[]int32{1000000}[0],
			IngressThroughputPerSec: &[]string{"test"}[0],
			EgressThroughputPerSec:  &[]string{"test"}[0],
			DataRetentionSize:       &[]string{"test"}[0],
		},
		ResizeInfo: &private.DatePlaneClusterUpdateStatusRequestResizeInfo{
			NodeDelta: &[]int32{3}[0],
			Delta: &private.DatePlaneClusterUpdateStatusRequestResizeInfoDelta{
				Connections:             &[]int32{10000}[0],
				Partitions:              &[]int32{10000}[0],
				IngressThroughputPerSec: &[]string{"test"}[0],
				EgressThroughputPerSec:  &[]string{"test"}[0],
				DataRetentionSize:       &[]string{"test"}[0],
			},
		},
	}
}

func TestGetAvailableStrimziVersions(t *testing.T) {
	type args struct {
		status private.DataPlaneClusterUpdateStatusRequest
	}

	tests := []struct {
		name string
		args args
		want []api.StrimziVersion
	}{
		{
			name: "should return available strimzi versions from DataPlaneClusterUpdateStatusRequest",
			args: args{
				status: mock.BuildDataPlaneClusterUpdateStatusRequest(nil),
			},
			want: *mock.BuildApiStrimziVersions(nil),
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			convertedConfig, err := getAvailableStrimziVersions(tt.args.status)
			if err == nil {
				Expect(convertedConfig).To(Equal(tt.want))
			}
		})
	}
}
