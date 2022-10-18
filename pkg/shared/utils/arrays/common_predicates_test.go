package arrays

import (
	"github.com/onsi/gomega"
	"testing"
)

func Test_IsNilPredicate(t *testing.T) {
	buildStringPointer := func(s string) *string {
		return &s
	}

	type args struct {
		val *string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "IsNil true",
			args: args{
				val: nil,
			},
			want: true,
		},
		{
			name: "IsNil false",
			args: args{
				val: buildStringPointer("Hello"),
			},
			want: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(IsNilPredicate(tt.args.val)).To(gomega.Equal(tt.want))
		})
	}
}

func Test_IsNotNilPredicate(t *testing.T) {
	buildStringPointer := func(s string) *string {
		return &s
	}

	type args struct {
		val *string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "IsNotNil false",
			args: args{
				val: nil,
			},
			want: false,
		},
		{
			name: "IsNotNil true",
			args: args{
				val: buildStringPointer("Hello"),
			},
			want: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(IsNotNilPredicate(tt.args.val)).To(gomega.Equal(tt.want))
		})
	}
}

func Test_StringEmptyPredicate(t *testing.T) {
	type args struct {
		val string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Check empty string is detected",
			args: args{
				val: "",
			},
			want: true,
		},
		{
			name: "Check non empty string is detected",
			args: args{
				val: "non empty",
			},
			want: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(StringEmptyPredicate(tt.args.val)).To(gomega.Equal(tt.want))
		})
	}

	buildStringPointer := func(s string) *string {
		return &s
	}

	type args1 struct {
		val *string
	}
	tests1 := []struct {
		name string
		args args1
		want bool
	}{
		{
			name: "Check empty string is detected",
			args: args1{
				val: buildStringPointer(""),
			},
			want: true,
		},
		{
			name: "Check non empty string is detected",
			args: args1{
				val: buildStringPointer("non empty"),
			},
			want: false,
		},
		{
			name: "Check nil string is detected as empty",
			args: args1{
				val: nil,
			},
			want: true,
		},
	}

	for _, testcase := range tests1 {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(StringEmptyPredicate(tt.args.val)).To(gomega.Equal(tt.want))
		})
	}
}

func Test_StringNotEmptyPredicate(t *testing.T) {
	type args struct {
		val string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Check empty string is detected",
			args: args{
				val: "",
			},
			want: false,
		},
		{
			name: "Check non empty string is detected",
			args: args{
				val: "non empty",
			},
			want: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(StringNotEmptyPredicate(tt.args.val)).To(gomega.Equal(tt.want))
		})
	}

	buildStringPointer := func(s string) *string {
		return &s
	}

	type args1 struct {
		val *string
	}
	tests1 := []struct {
		name string
		args args1
		want bool
	}{
		{
			name: "Check empty string is detected",
			args: args1{
				val: buildStringPointer(""),
			},
			want: false,
		},
		{
			name: "Check non empty string is detected",
			args: args1{
				val: buildStringPointer("non empty"),
			},
			want: true,
		},
		{
			name: "Check nil string is detected as empty",
			args: args1{
				val: nil,
			},
			want: false,
		},
	}

	for _, testcase := range tests1 {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(StringNotEmptyPredicate(tt.args.val)).To(gomega.Equal(tt.want))
		})
	}
}

func Test_EqualsPredicate(t *testing.T) {
	type args struct {
		val1 string
		val2 string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Check different empty string",
			args: args{
				val1: "",
				val2: "  ",
			},
			want: false,
		},
		{
			name: "Check different case",
			args: args{
				val1: "red hat",
				val2: "Red Hat",
			},
			want: false,
		},
		{
			name: "Check identical strings",
			args: args{
				val1: "Red Hat",
				val2: "Red Hat",
			},
			want: true,
		},
		{
			name: "Check same string, different trailing spaces",
			args: args{
				val1: "red hat ",
				val2: "Red Hat   ",
			},
			want: false,
		},
		{
			name: "Check same string, different beginning spaces",
			args: args{
				val1: "  red hat",
				val2: "     Red Hat  ",
			},
			want: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			predicate := EqualsPredicate(tt.args.val1)
			g.Expect(predicate(tt.args.val2)).To(gomega.Equal(tt.want))
		})
	}
}

func Test_StringEqualsIgnoreCasePredicate(t *testing.T) {
	type args struct {
		val1 string
		val2 string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Check different empty string",
			args: args{
				val1: "",
				val2: "  ",
			},
			want: false,
		},
		{
			name: "Check different case",
			args: args{
				val1: "red hat",
				val2: "Red Hat",
			},
			want: true,
		},
		{
			name: "Check identical strings",
			args: args{
				val1: "Red Hat",
				val2: "Red Hat",
			},
			want: true,
		},
		{
			name: "Check same string, different trailing spaces",
			args: args{
				val1: "red hat ",
				val2: "Red Hat   ",
			},
			want: false,
		},
		{
			name: "Check same string, different beginning spaces",
			args: args{
				val1: "  red hat",
				val2: "     Red Hat  ",
			},
			want: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			predicate := StringEqualsIgnoreCasePredicate(tt.args.val1)
			g.Expect(predicate(tt.args.val2)).To(gomega.Equal(tt.want))
		})
	}
}

func Test_ComposedPredicate(t *testing.T) {
	buildStringPointer := func(s string) *string {
		return &s
	}

	type args1 struct {
		val        *string
		predicates PredicateFunc[any]
	}
	tests1 := []struct {
		name string
		args args1
		want bool
	}{
		{
			name: "Check string is empty but not nil - success",
			args: args1{
				val: buildStringPointer(""),
				predicates: CompositePredicateAll[any](
					IsNotNilPredicate[any],
					StringEmptyPredicate,
				),
			},
			want: true,
		},
		{
			name: "Check string is empty but not nil - fail",
			args: args1{
				val: nil,
				predicates: CompositePredicateAll[any](
					IsNotNilPredicate[any],
					StringEmptyPredicate,
				),
			},
			want: false,
		},
		{
			name: "Check string is not empty or nil - success",
			args: args1{
				val: nil,
				predicates: CompositePredicateAny[any](
					IsNilPredicate[any],
					StringNotEmptyPredicate,
				),
			},
			want: true,
		},
		{
			name: "Check string is not empty or nil - fail",
			args: args1{
				val: buildStringPointer(""),
				predicates: CompositePredicateAny[any](
					IsNilPredicate[any],
					StringNotEmptyPredicate,
				),
			},
			want: false,
		},
	}

	for _, testcase := range tests1 {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(tt.args.predicates(tt.args.val)).To(gomega.Equal(tt.want))
		})
	}
}
