package presenters

import (
	"reflect"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
)

func TestConvertDataPlaneClusterStatus_AvailableStrimziVersions(t *testing.T) {
	tests := []struct {
		name  string
		input private.DataPlaneClusterUpdateStatusRequest
		want  []api.StrimziVersion
	}{
		{
			name:  "When setting a non empty ordered list of strimzi versions that list is stored as is",
			input: *sampleValidDataPlaneClusterUpdateStatusRequestWithAvailableStrimziVersions([]string{"v1", "v2", "v3"}),
			want: []api.StrimziVersion{
				api.StrimziVersion("v1"),
				api.StrimziVersion("v2"),
				api.StrimziVersion("v3"),
			},
		},
		{
			name:  "When setting a non empty unordered list of strimzi versions that list is stored in a lexicographically ascending order",
			input: *sampleValidDataPlaneClusterUpdateStatusRequestWithAvailableStrimziVersions([]string{"v5", "v3", "v2"}),
			want: []api.StrimziVersion{
				api.StrimziVersion("v2"),
				api.StrimziVersion("v3"),
				api.StrimziVersion("v5"),
			},
		},
		{
			name:  "When setting an empty list of strimzi versions that list is stored as the empty list",
			input: *sampleValidDataPlaneClusterUpdateStatusRequestWithAvailableStrimziVersions([]string{}),
			want:  []api.StrimziVersion{},
		},
		{
			name:  "When setting a nil list of strimzi versions that list is stored as the empty list",
			input: *sampleValidDataPlaneClusterUpdateStatusRequestWithAvailableStrimziVersions(nil),
			want:  []api.StrimziVersion{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := ConvertDataPlaneClusterStatus(tt.input)
			if !reflect.DeepEqual(res.AvailableStrimziVersions, tt.want) {
				t.Errorf("want: %v got: %v", tt.want, res)
			}
		})
	}
}

func sampleValidDataPlaneClusterUpdateStatusRequestWithAvailableStrimziVersions(versions []string) *private.DataPlaneClusterUpdateStatusRequest {
	return &private.DataPlaneClusterUpdateStatusRequest{
		Conditions: []private.DataPlaneClusterUpdateStatusRequestConditions{
			{
				Type:   "Ready",
				Status: "True",
			},
		},
		Total: private.DataPlaneClusterUpdateStatusRequestTotal{
			IngressEgressThroughputPerSec: &[]string{"test"}[0],
			Connections:                   &[]int32{1000000}[0],
			DataRetentionSize:             &[]string{"test"}[0],
			Partitions:                    &[]int32{1000000}[0],
		},
		NodeInfo: &private.DataPlaneClusterUpdateStatusRequestNodeInfo{
			Ceiling:                &[]int32{20}[0],
			Floor:                  &[]int32{3}[0],
			Current:                &[]int32{5}[0],
			CurrentWorkLoadMinimum: &[]int32{3}[0],
		},
		Remaining: private.DataPlaneClusterUpdateStatusRequestTotal{
			Connections:                   &[]int32{1000000}[0],
			Partitions:                    &[]int32{1000000}[0],
			IngressEgressThroughputPerSec: &[]string{"test"}[0],
			DataRetentionSize:             &[]string{"test"}[0],
		},
		ResizeInfo: &private.DataPlaneClusterUpdateStatusRequestResizeInfo{
			NodeDelta: &[]int32{3}[0],
			Delta: &private.DataPlaneClusterUpdateStatusRequestResizeInfoDelta{
				Connections:                   &[]int32{10000}[0],
				Partitions:                    &[]int32{10000}[0],
				IngressEgressThroughputPerSec: &[]string{"test"}[0],
				DataRetentionSize:             &[]string{"test"}[0],
			},
		},
		StrimziVersions: versions,
	}
}
