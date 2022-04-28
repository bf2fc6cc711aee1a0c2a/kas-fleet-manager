package shared

import (
	"errors"
	"testing"

	. "github.com/onsi/gomega"
)

func Test_LoadOpenAPISpec(t *testing.T) {
	type args struct {
		assetFunc func(name string) ([]byte, error)
		asset     string
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "return OpenApi specification if it can be loaded and converted",
			args: args{
				assetFunc: func(name string) ([]byte, error) {
					return []byte("{\"a\":\"1\"}"), nil
				},
			},
			wantErr: false,
			want:    []byte("{\"a\":\"1\"}"),
		},
		{
			name: "return error if OpenApi specification cannot be loaded.",
			args: args{
				assetFunc: func(name string) ([]byte, error) {
					return nil, errors.New("can't load OpenAPI specification from asset '%s'")
				},
			},
			wantErr: true,
			want:    nil,
		},
		{
			name: "return error if OpenApi specification loaded cannot be converted",
			args: args{
				assetFunc: func(name string) ([]byte, error) {
					return []byte("{"), nil
				},
			},
			wantErr: true,
			want:    nil,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadOpenAPISpec(tt.args.assetFunc, tt.args.asset)
			Expect(err != nil).To(Equal(tt.wantErr))
			Expect(got).To(Equal(tt.want))
		})
	}
}
