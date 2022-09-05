package shared

import (
	"testing"

	"github.com/onsi/gomega"
)

func Test_LoadOpenAPISpec(t *testing.T) {
	type args struct {
		yamlBytes []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "return OpenApi specification if it can be converted",
			args: args{
				yamlBytes: []byte("{\"a\":\"1\"}"),
			},
			wantErr: false,
			want:    []byte("{\"a\":\"1\"}"),
		},
		{
			name: "return error if OpenApi specification cannot be converted",
			args: args{
				yamlBytes: []byte("{"),
			},
			wantErr: true,
			want:    nil,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			got, err := LoadOpenAPISpecFromYAML(tt.args.yamlBytes)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}
