package ocm

import (
	"testing"

	. "github.com/onsi/gomega"
)

func Test_GetProduct(t *testing.T) {
	type fields struct {
		t KafkaQuotaType
	}

	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "should return RHOSAK for standard quota",
			fields: fields{
				t: StandardQuota,
			},
			want: string(RHOSAKProduct),
		},
		{
			name: "should return RHOSAKTrial for developer quota",
			fields: fields{
				t: DeveloperQuota,
			},
			want: string(RHOSAKTrialProduct),
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(tt.fields.t.GetProduct()).To(Equal(tt.want))
		})
	}
}

func Test_GetResourceName(t *testing.T) {
	type fields struct {
		t KafkaQuotaType
	}

	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "should return 'rhosak' for standard quota",
			fields: fields{
				t: StandardQuota,
			},
			want: ResourceName,
		},
		{
			name: "should return 'rhosak' for developer quota",
			fields: fields{
				t: DeveloperQuota,
			},
			want: ResourceName,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(tt.fields.t.GetResourceName()).To(Equal(tt.want))
		})
	}
}

func Test_Equals(t *testing.T) {
	type fields struct {
		t KafkaQuotaType
	}

	type args struct {
		t1 KafkaQuotaType
	}

	tests := []struct {
		name   string
		args   args
		fields fields
		want   bool
	}{
		{
			name: "should return true when comparing two StandardQuota types",
			args: args{
				t1: StandardQuota,
			},
			fields: fields{
				t: StandardQuota,
			},
			want: true,
		},
		{
			name: "should return true when comparing two DeveloperQuota types",
			args: args{
				t1: DeveloperQuota,
			},
			fields: fields{
				t: DeveloperQuota,
			},
			want: true,
		},
		{
			name: "should return false when comparing DeveloperQuota and StandardQuota",
			args: args{
				t1: StandardQuota,
			},
			fields: fields{
				t: DeveloperQuota,
			},
			want: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(tt.fields.t.Equals(tt.args.t1)).To(Equal(tt.want))
		})
	}
}
