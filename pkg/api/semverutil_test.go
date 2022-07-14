package api

import (
	"testing"

	"github.com/onsi/gomega"
)

func Test_buildAwareSemanticVersioningCompare(t *testing.T) {
	type args struct {
		v1 string
		v2 string
	}

	tests := []struct {
		name    string
		args    args
		want    int
		wantErr bool
	}{
		{
			name: "When v1 is empty an error is returned",
			args: args{
				v1: "",
				v2: "1.0.0",
			},
			wantErr: true,
		},
		{
			name: "When v2 is empty an error is returned",
			args: args{
				v1: "1.0.1",
				v2: "",
			},
			wantErr: true,
		},
		{
			name: "when v1 is greater than v2 1 is returned",
			args: args{
				v1: "1.0.1",
				v2: "1.0.0",
			},
			want: 1,
		},
		{
			name: "when v2 is greater than v1 -1 is returned",
			args: args{
				v1: "1.0.0",
				v2: "1.0.1",
			},
			want: -1,
		},
		{
			name: "when v1 is equal to v2 0 is returned",
			args: args{
				v1: "1.0.0",
				v2: "1.0.0",
			},
			want: 0,
		},
		{
			name: "comparison works with the 'v' prefix is supported",
			args: args{
				v1: "v1.2.0",
				v2: "v2.1.0",
			},
			want:    -1,
			wantErr: false,
		},
		{
			name: "comparing versions without patch level version number works",
			args: args{
				v1: "1.1",
				v2: "1.3",
			},
			want:    -1,
			wantErr: false,
		},
		{
			name: "comparison with pre-release suffix is supported",
			args: args{
				v1: "1.0.0-rc.1+build2",
				v2: "1.0.0-rc.2+build1",
			},
			want:    -1,
			wantErr: false,
		},
		{
			name: "When two versions have same x.y.z but different build metadata a lexicographical sort of the metadata is performed",
			args: args{
				v1: "1.0.0+buildexample2.2.4",
				v2: "1.0.0+buildexample2.2.1",
			},
			want:    1,
			wantErr: false,
		},
		{
			name: "When two versions have same x.y.z and same pre-release version but a different build metadata a lexicographical sort of the metadata is performed",
			args: args{
				v1: "1.0.0-rc.1+buildexample2",
				v2: "1.0.0-rc.1+buildexample1",
			},
			want:    1,
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			res, err := buildAwareSemanticVersioningCompare(tt.args.v1, tt.args.v2)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			g.Expect(res).To(gomega.Equal(tt.want))
		})
	}

}

func Test_checkIfMinorDowngrade(t *testing.T) {
	type args struct {
		current string
		desired string
	}
	tests := []struct {
		name    string
		args    args
		want    int
		wantErr bool
	}{
		{
			name: "When desired major is smaller than current major, 1 is returned",
			args: args{
				current: "3.6.0",
				desired: "2.6.0",
			},
			want:    1,
			wantErr: false,
		},
		{
			name: "When desired major is greater than current major -1, is returned",
			args: args{
				current: "2.7.0",
				desired: "3.7.0",
			},
			want:    -1,
			wantErr: false,
		},
		{
			name: "When major versions are equal and desired minor is greater than current minor, -1 is returned",
			args: args{
				current: "2.7.0",
				desired: "2.8.0",
			},
			want:    -1,
			wantErr: false,
		},
		{
			name: "When major versions are equal and desired minor is smaller than current minor, 1 is returned",
			args: args{
				current: "2.8.0",
				desired: "2.7.0",
			},
			want:    1,
			wantErr: false,
		},
		{
			name: "When major versions are equal and desired minor is equal to current minor, 0 is returned",
			args: args{
				current: "2.7.0",
				desired: "2.7.0",
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "When major and minor versions are equal and desired patch is equal to current patch, 0 is returned",
			args: args{
				current: "2.7.0",
				desired: "2.7.0",
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "When major and minor versions are equal and desired patch is greater than current patch, 0 is returned",
			args: args{
				current: "2.7.0",
				desired: "2.7.1",
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "When major and minor versions are equal and desired patch is smaller than current patch, 0 is returned",
			args: args{
				current: "2.7.2",
				desired: "2.7.1",
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "When current is empty an error is returned",
			args: args{
				current: "",
				desired: "2.7.1",
			},
			wantErr: true,
		},
		{
			name: "When desired is empty an error is returned",
			args: args{
				current: "2.7.1",
				desired: "",
			},
			wantErr: true,
		},
		{
			name: "When current has an invalid semver version format an error is returned",
			args: args{
				current: "2invalid.6.0",
				desired: "2.7.1",
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			res, err := checkIfMinorDowngrade(tt.args.current, tt.args.desired)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			g.Expect(res).To(gomega.Equal(tt.want))
		})
	}
}
