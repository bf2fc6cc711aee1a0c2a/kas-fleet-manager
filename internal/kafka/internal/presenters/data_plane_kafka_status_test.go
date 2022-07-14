package presenters

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/private"

	mock "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/data_plane"

	"github.com/onsi/gomega"
)

func TestConvertDataPlaneKafkaStatus(t *testing.T) {
	type args struct {
		status map[string]private.DataPlaneKafkaStatus
	}

	tests := []struct {
		name string
		args args
		want []*dbapi.DataPlaneKafkaStatus
	}{
		{
			name: "should return converted dataplane kafka status with passing non-empty RegionCapacityListItem",
			args: args{
				status: mock.BuildPrivateDataPlaneKafkaStatus(nil),
			},
			want: []*dbapi.DataPlaneKafkaStatus{mock.BuildDbDataPlaneKafkaStatus(nil)},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(ConvertDataPlaneKafkaStatus(tt.args.status)).To(gomega.Equal(tt.want))
		})
	}
}
