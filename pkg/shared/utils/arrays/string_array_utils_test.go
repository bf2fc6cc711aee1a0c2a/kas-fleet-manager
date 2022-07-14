package arrays

import (
	"testing"
	"unicode"

	"github.com/onsi/gomega"
)

func Test_StringFindFirst(t *testing.T) {
	type args struct {
		ary       []string
		val       string
		predicate func(x string) bool
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "FindFirst success",
			args: args{
				ary:       []string{"This", "Is", "Red", "Hat"},
				val:       "Red",
				predicate: func(x string) bool { return x == "Red" },
			},
			want: 2,
		},
		{
			name: "FindFirst Not Found",
			args: args{
				ary:       []string{"This", "Is", "Red", "Hat"},
				val:       "Red",
				predicate: func(x string) bool { return x == "Blue" },
			},
			want: -1,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(FindFirstString(tt.args.ary, tt.args.predicate)).To(gomega.Equal(tt.want))
		})
	}
}

func Test_FilterStringSlice(t *testing.T) {
	g := gomega.NewWithT(t)
	res := FilterStringSlice([]string{"this", "is", "Red", "Hat"}, func(x string) bool { return x != "" && unicode.IsUpper([]rune(x)[0]) })
	g.Expect(res).To(gomega.HaveLen(2))
	g.Expect(res).To(gomega.Equal([]string{"Red", "Hat"}))
}

func Test_StringFirstNonEmpty(t *testing.T) {
	type args struct {
		ary []string
	}
	tests := []struct {
		name    string
		args    args
		wants   string
		wantErr bool
	}{
		{
			name: "Success",
			args: args{
				ary: []string{"", "", "", "Red", "Hat"},
			},
			wants:   "Red",
			wantErr: false,
		},
		{
			name: "Not Found",
			args: args{
				ary: []string{"", "", "", "", ""},
			},
			wants:   "",
			wantErr: true,
		},
		{
			name: "No Values",
			args: args{
				ary: []string{},
			},
			wants:   "",
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			val, err := FirstNonEmpty(tt.args.ary...)
			g.Expect(val).To(gomega.Equal(tt.wants))
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_StringFirstNonEmptyOrDefault(t *testing.T) {
	type args struct {
		ary          []string
		defaultValue string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Value found: Red",
			args: args{
				ary:          []string{"", "", "Red", "Hat"},
				defaultValue: "Red Hat",
			},
			want: "Red",
		},
		{
			name: "Value not found: default: 'Red Hat'",
			args: args{
				ary:          []string{"", "", "", ""},
				defaultValue: "Red Hat",
			},
			want: "Red Hat",
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(FirstNonEmptyOrDefault(tt.args.defaultValue, tt.args.ary...)).To(gomega.Equal(tt.want))
		})
	}
}

func Test_Contains(t *testing.T) {
	type args struct {
		ary   []string
		value string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Value found",
			args: args{
				ary:   []string{"yellow", "green", "blue"},
				value: "green",
			},
			want: true,
		},
		{
			name: "Value not found",
			args: args{
				ary:   []string{"yellow", "green", "blue"},
				value: "red",
			},
			want: false,
		},
		{
			name: "Value not found - Empty Array",
			args: args{
				ary:   []string{},
				value: "red",
			},
			want: false,
		},
		{
			name: "Value not found - Nil Array",
			args: args{
				ary:   nil,
				value: "red",
			},
			want: false,
		},
	}
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(Contains(tt.args.ary, tt.args.value)).To(gomega.Equal(tt.want))
		})
	}
}
