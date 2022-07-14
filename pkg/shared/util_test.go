package shared

import (
	"testing"

	"github.com/onsi/gomega"
)

func Test_SafeString(t *testing.T) {
	testString := "test string"
	type args struct {
		ptr *string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "return empty if ptr is nil",
			args: args{
				ptr: nil,
			},
			want: "",
		},
		{
			name: "return string if ptr is not nil",
			args: args{
				ptr: &testString,
			},
			want: "test string",
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)

			g.Expect(SafeString(tt.args.ptr)).To(gomega.Equal(tt.want))
		})
	}
}

func Test_SafeInt64(t *testing.T) {
	testInt := int64(10)

	type args struct {
		ptr *int64
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		{
			name: "return 0 if ptr is nil",
			args: args{
				ptr: nil,
			},
			want: 0,
		},
		{
			name: "return int64 if ptr is not nil",
			args: args{
				ptr: &testInt,
			},
			want: int64(10),
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)

			g.Expect(SafeInt64(tt.args.ptr)).To(gomega.Equal(tt.want))
		})
	}
}
