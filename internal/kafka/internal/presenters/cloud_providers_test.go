package presenters

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	mock "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/cloud_providers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"

	"github.com/onsi/gomega"
)

func TestGetRegionCapacityItems(t *testing.T) {
	type args struct {
		capacityItems []api.RegionCapacityListItem
	}

	tests := []struct {
		name string
		args args
		want []public.RegionCapacityListItem
	}{
		{
			name: "should return region capacity items with passing non-empty api.RegionCapacityListItem",
			args: args{
				capacityItems: mock.BuildApiRegionCapacityListItemSlice(nil),
			},
			want: mock.BuildRegionCapacityListItemSlice(nil),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(GetRegionCapacityItems(tt.args.capacityItems)).To(gomega.Equal(tt.want))
		})
	}
}

func TestPresentCloudProvider(t *testing.T) {
	type args struct {
		cloudProvider *api.CloudProvider
	}

	tests := []struct {
		name string
		args args
		want public.CloudProvider
	}{
		{
			name: "should return CloudProvider as presented to the end user with passing non-empty api.CloudProvider",
			args: args{
				cloudProvider: mock.BuildApiCloudProvider(nil),
			},
			want: mock.BuildCloudProvider(nil),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(PresentCloudProvider(tt.args.cloudProvider)).To(gomega.Equal(tt.want))
		})
	}
}

func TestPresentCloudRegion(t *testing.T) {
	type args struct {
		cloudRegion *api.CloudRegion
	}

	tests := []struct {
		name string
		args args
		want *public.CloudRegion
	}{
		{
			name: "should return CloudRegion as presented to the end user with passing non-empty api.CloudRegion",
			args: args{
				cloudRegion: mock.BuildApiCloudRegion(func(cloudRegion *api.CloudRegion) {
					cloudRegion.Capacity = mock.BuildApiRegionCapacityListItemSlice(nil)
				}),
			},
			want: mock.BuildCloudRegion(func(cloudRegion *public.CloudRegion) {
				cloudRegion.Capacity = mock.BuildRegionCapacityListItemSlice(nil)
			}),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			cloudRegion := PresentCloudRegion(tt.args.cloudRegion)
			g.Expect(&cloudRegion).To(gomega.Equal(tt.want))
		})
	}
}
