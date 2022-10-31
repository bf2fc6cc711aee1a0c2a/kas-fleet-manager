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

var _ error = testErr{}

type testErr struct{}

func (testErr) Error() string {
	return ""
}

func Test_IsNil(t *testing.T) {
	type args struct {
		value any
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Check naked nil pointer value",
			args: args{
				value: nil,
			},
			want: true,
		},
		{
			name: "Check nil slice value",
			args: args{
				value: func() []int { return nil }(),
			},
			want: true,
		},
		{
			name: "Check non nil pointer value",
			args: args{
				value: func() *string { s := "Hello"; return &s },
			},
			want: false,
		},
		{
			name: "Check nil map value",
			args: args{
				value: func() map[string]int { return nil }(),
			},
			want: true,
		},
		{
			name: "Check non nil map value",
			args: args{
				value: map[string]int{},
			},
			want: false,
		},
		{
			name: "Check non naked nil value",
			args: args{
				value: func() error { var e *testErr; return e }(),
			},
			want: true,
		},
		{
			name: "Check non naked not-nil value",
			args: args{
				value: func() error { e := testErr{}; return &e }(),
			},
			want: false,
		},
		{
			name: "Check non nullable value",
			args: args{
				value: "strings can't be null",
			},
			want: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)

			g.Expect(IsNil(tt.args.value)).To(gomega.Equal(tt.want))
		})
	}
}

func Test_StringEmpty(t *testing.T) {
	type args struct {
		value   string
		pointer *string
		trim    bool
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Check nil pointer value (no trim)",
			args: args{
				value:   "",
				pointer: nil,
			},
			want: true,
		},
		{
			name: "Check nil pointer value (trim)",
			args: args{
				value:   "",
				pointer: nil,
				trim:    true,
			},
			want: true,
		},
		{
			name: "Check empty string",
			args: args{
				value:   "",
				pointer: func() *string { v := ""; return &v }(),
			},
			want: true,
		},
		{
			name: "Check only space string (no trim)",
			args: args{
				value:   "   ",
				pointer: func() *string { v := "   "; return &v }(),
			},
			want: false,
		},
		{
			name: "Check only space string (trim)",
			args: args{
				value:   "   ",
				pointer: func() *string { v := "   "; return &v }(),
				trim:    true,
			},
			want: true,
		},
		{
			name: "Check non empty string (no trim)",
			args: args{
				value:   "This is a string  ",
				pointer: func() *string { v := "This is a string  "; return &v }(),
			},
			want: false,
		},
		{
			name: "Check non empty string (trim)",
			args: args{
				value:   "This is a string  ",
				pointer: func() *string { v := "This is a string   "; return &v }(),
				trim:    true,
			},
			want: false,
		},
	}
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)

			g.Expect(StringEmpty(tt.args.pointer, tt.args.trim)).To(gomega.Equal(tt.want))
			g.Expect(StringEmpty(tt.args.value, tt.args.trim)).To(gomega.Equal(tt.want))
			if !tt.args.trim {
				g.Expect(StringEmpty(tt.args.pointer)).To(gomega.Equal(tt.want), "Error when `trim` is not passed")
				g.Expect(StringEmpty(tt.args.value)).To(gomega.Equal(tt.want), "Error when `trim` is not passed")
			}
		})
	}
}

func Test_StringEqualIgnoreCase(t *testing.T) {
	type values struct {
		value1 string
		value2 string
	}

	type pointers struct {
		value1 *string
		value2 *string
	}

	type args struct {
		values   values
		pointers pointers
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Nil values",
			args: args{
				values: values{
					value1: "",
					value2: "",
				},
				pointers: pointers{
					value1: nil,
					value2: nil,
				},
			},
			want: true,
		},
		{
			name: "Nil and not nil values",
			args: args{
				values: values{
					value1: "",
					value2: "This is a string",
				},
				pointers: pointers{
					value1: nil,
					value2: func() *string { v := "This is a string"; return &v }(),
				},
			},
			want: false,
		},
		{
			name: "Not nil and Nil values",
			args: args{
				values: values{
					value1: "This is a string",
					value2: "",
				},
				pointers: pointers{
					value1: func() *string { v := "This is a string"; return &v }(),
					value2: nil,
				},
			},
			want: false,
		},
		{
			name: "Non equal string",
			args: args{
				values: values{
					value1: "This is a string",
					value2: "This is a different string",
				},
				pointers: pointers{
					value1: func() *string { v := "This is a string"; return &v }(),
					value2: func() *string { v := "This is a different string"; return &v }(),
				},
			},
			want: false,
		},
		{
			name: "Equal same case",
			args: args{
				values: values{
					value1: "This is a string",
					value2: "This is a string",
				},
				pointers: pointers{
					value1: func() *string { v := "This is a string"; return &v }(),
					value2: func() *string { v := "This is a string"; return &v }(),
				},
			},
			want: true,
		},
		{
			name: "Equal different case",
			args: args{
				values: values{
					value1: "This Is a String",
					value2: "This is A strIng",
				},
				pointers: pointers{
					value1: func() *string { v := "This Is a String"; return &v }(),
					value2: func() *string { v := "This is A strIng"; return &v }(),
				},
			},
			want: true,
		},
	}
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(StringEqualsIgnoreCase(tt.args.values.value1, tt.args.values.value2)).To(gomega.Equal(tt.want))
			g.Expect(StringEqualsIgnoreCase(tt.args.pointers.value1, tt.args.pointers.value2)).To(gomega.Equal(tt.want))
		})
	}
}
