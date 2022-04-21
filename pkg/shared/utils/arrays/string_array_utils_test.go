package arrays

import (
	. "github.com/onsi/gomega"
	"testing"
	"unicode"
)

func TestStringFindFirst(t *testing.T) {
	RegisterTestingT(t)
	type args struct {
		ary       []string
		val       string
		predicate func(x string) bool
	}
	tests := []struct {
		name  string
		args  args
		wants int
	}{
		{
			name: "FindFirst success",
			args: args{
				ary:       []string{"This", "Is", "Red", "Hat"},
				val:       "Red",
				predicate: func(x string) bool { return x == "Red" },
			},
			wants: 2,
		},
		{
			name: "FindFirst Not Found",
			args: args{
				ary:       []string{"This", "Is", "Red", "Hat"},
				val:       "Red",
				predicate: func(x string) bool { return x == "Blue" },
			},
			wants: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx := FindFirstString(tt.args.ary, tt.args.predicate)
			Expect(idx).To(Equal(tt.wants))
		})
	}
}

func TestFilterStringSlice(t *testing.T) {
	RegisterTestingT(t)
	res := FilterStringSlice([]string{"this", "is", "Red", "Hat"}, func(x string) bool { return x != "" && unicode.IsUpper([]rune(x)[0]) })
	Expect(res).To(HaveLen(2))
	Expect(res).To(Equal([]string{"Red", "Hat"}))
}

func TestStringFirstNonEmpty(t *testing.T) {
	RegisterTestingT(t)
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := FirstNonEmpty(tt.args.ary...)
			Expect(val).To(Equal(tt.wants))
			Expect(err != nil).To(Equal(tt.wantErr))
		})
	}
}

func TestStringFirstNonEmptyOrDefault(t *testing.T) {
	RegisterTestingT(t)
	type args struct {
		ary          []string
		defaultValue string
	}
	tests := []struct {
		name  string
		args  args
		wants string
	}{
		{
			name: "Value found: Red",
			args: args{
				ary:          []string{"", "", "Red", "Hat"},
				defaultValue: "Red Hat",
			},
			wants: "Red",
		},
		{
			name: "Value not found: default: 'Red Hat'",
			args: args{
				ary:          []string{"", "", "", ""},
				defaultValue: "Red Hat",
			},
			wants: "Red Hat",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := FirstNonEmptyOrDefault(tt.args.defaultValue, tt.args.ary...)
			Expect(val).To(Equal(tt.wants))
		})
	}
}

func TestContains(t *testing.T) {
	RegisterTestingT(t)
	type args struct {
		ary   []string
		value string
	}
	tests := []struct {
		name  string
		args  args
		wants bool
	}{
		{
			name: "Value found",
			args: args{
				ary:   []string{"yellow", "green", "blue"},
				value: "green",
			},
			wants: true,
		},
		{
			name: "Value not found",
			args: args{
				ary:   []string{"yellow", "green", "blue"},
				value: "red",
			},
			wants: false,
		},
		{
			name: "Value not found - Empty Array",
			args: args{
				ary:   []string{},
				value: "red",
			},
			wants: false,
		},
		{
			name: "Value not found - Nil Array",
			args: args{
				ary:   nil,
				value: "red",
			},
			wants: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ret := Contains(tt.args.ary, tt.args.value)
			Expect(ret).To(Equal(tt.wants))
		})
	}
}
