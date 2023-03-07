package arrays

import (
	"testing"
	"unicode"

	"github.com/onsi/gomega"
)

func Test_FindFirst(t *testing.T) {
	type args struct {
		predicate func(x interface{}) bool
		values    []interface{}
	}
	tests := []struct {
		name          string
		args          args
		expectedIndex int
		expectedValue interface{}
	}{
		{
			name: "Return -1 index and nil if no match",
			args: args{
				predicate: func(x interface{}) bool { return x.(int) > 5 },
				values:    []interface{}{1, 2, 3, 4},
			},
			expectedIndex: -1,
			expectedValue: nil,
		},
		{
			name: "Return with index and matching integer",
			args: args{
				predicate: func(x interface{}) bool { return x.(int) > 5 },
				values:    []interface{}{1, 2, 3, 4, 5, 6, 7, 8},
			},
			expectedIndex: 5,
			expectedValue: 6,
		},
		{
			name: "Return with index and matching string",
			args: args{
				predicate: func(x interface{}) bool { return x.(string) == "Red" },
				values:    []interface{}{"This", "Is", "Red", "Hat"},
			},
			expectedIndex: 2,
			expectedValue: "Red",
		},
		{
			name: "Return with index and matching incompatible types",
			args: args{
				predicate: func(x interface{}) bool {
					if val, ok := x.(int); ok {
						return val > 5
					} else {
						return false
					}
				},
				values: []interface{}{1, 2, "Hello", 4, 5, 6, 7, 8},
			},
			expectedIndex: 5,
			expectedValue: 6,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			idx, _ := FindFirst(tt.args.values, tt.args.predicate)
			g.Expect(idx).To(gomega.Equal(tt.expectedIndex))
		})
	}
}

func Test_AnyMatch(t *testing.T) {
	type args struct {
		ary       []string
		val       string
		predicate PredicateFunc[string]
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test_AnyMatch success",
			args: args{
				ary:       []string{"This", "Is", "Red", "Hat"},
				val:       "Red",
				predicate: func(x string) bool { return x == "Red" },
			},
			want: true,
		},
		{
			name: "Test_AnyMatch Not Found",
			args: args{
				ary:       []string{"This", "Is", "Red", "Hat"},
				val:       "Red",
				predicate: func(x string) bool { return x == "Blue" },
			},
			want: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(AnyMatch(tt.args.ary, tt.args.predicate)).To(gomega.Equal(tt.want))
		})
	}
}

func Test_NoneMatch(t *testing.T) {
	type args struct {
		ary       []string
		val       string
		predicate PredicateFunc[string]
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "FindFirst failed",
			args: args{
				ary:       []string{"This", "Is", "Red", "Hat"},
				val:       "Red",
				predicate: func(x string) bool { return x == "Red" },
			},
			want: false,
		},
		{
			name: "NoneMatch success",
			args: args{
				ary:       []string{"This", "Is", "Red", "Hat"},
				val:       "Red",
				predicate: func(x string) bool { return x == "Blue" },
			},
			want: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(NoneMatch(tt.args.ary, tt.args.predicate)).To(gomega.Equal(tt.want))
		})
	}
}

func Test_AllMatch(t *testing.T) {
	type args struct {
		ary       []int
		predicate PredicateFunc[int]
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "AllMatch success",
			args: args{
				ary:       []int{3, 5, 9, 11, 13},
				predicate: func(x int) bool { return x%2 != 0 },
			},
			want: true,
		},
		{
			name: "AllMatch failed",
			args: args{
				ary:       []int{3, 5, 9, 12, 13},
				predicate: func(x int) bool { return x%2 != 0 },
			},
			want: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(AllMatch(tt.args.ary, tt.args.predicate)).To(gomega.Equal(tt.want))
		})
	}
}

func Test_FindFirst_String(t *testing.T) {
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
			idx, _ := FindFirst(tt.args.ary, tt.args.predicate)
			g.Expect(idx).To(gomega.Equal(tt.want))
		})
	}
}

func Test_Filter(t *testing.T) {
	g := gomega.NewWithT(t)
	// gets only the capitalized string
	res := Filter([]string{"this", "is", "Red", "Hat"}, func(x string) bool { return x != "" && unicode.IsUpper([]rune(x)[0]) })
	g.Expect(res).To(gomega.HaveLen(2))
	g.Expect(res).To(gomega.Equal([]string{"Red", "Hat"}))
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

func Test_Map(t *testing.T) {

	numbers := []string{"zero", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine"}

	type args[K interface{}, V interface{}] struct {
		ary    []K
		mapper func(x K) V
	}

	testsIntToString := []struct {
		name string
		args args[int, string]
		want []string
	}{
		{
			name: "Transforms ints to strings",
			args: args[int, string]{
				ary: []int{1, 5, 8, 3, 2},
				mapper: func(x int) string {
					return numbers[x]
				},
			},
			want: []string{"one", "five", "eight", "three", "two"},
		},
	}

	for _, testcase := range testsIntToString {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			values := Map(tt.args.ary, tt.args.mapper)
			g.Expect(values).To(gomega.Equal(tt.want))
			//g.Expect(Contains(tt.args.ary, tt.args.value)).To(gomega.Equal(tt.want))
		})
	}

	testsIntToInt := []struct {
		name string
		args args[int, int]
		want []int
	}{
		{
			name: "Transforms int to int",
			args: args[int, int]{
				ary: []int{1, 5, 8, 3, 2},
				mapper: func(x int) int {
					return x * 2
				},
			},
			want: []int{2, 10, 16, 6, 4},
		},
	}

	for _, testcase := range testsIntToInt {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			values := Map(tt.args.ary, tt.args.mapper)
			g.Expect(values).To(gomega.Equal(tt.want))
		})
	}
}

func Test_Reduce(t *testing.T) {
	type args[K any, V any] struct {
		ary     []K
		reducer ReducerFunc[K, V]
	}

	tests := []struct {
		name string
		args args[int, int]
		want int
	}{
		{
			name: "Reduce to sum",
			args: args[int, int]{
				ary: []int{0, 1, 2, 3, 4},
				reducer: func(previous int, current int) int {
					return previous + current
				},
			},
			want: 10,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(Reduce(tt.args.ary, tt.args.reducer, 0)).To(gomega.Equal(tt.want))
		})
	}
}

func Test_ForEach(t *testing.T) {
	var ary = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}
	sum := 0
	ForEach(ary, func(x int) {
		sum = sum + x
	})

	g := gomega.NewWithT(t)
	g.Expect(sum).To(gomega.Equal(45))
}

func TestIsEmpty(t *testing.T) {
	type args struct {
		values []int
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "return true if array is nil",
			args: args{
				values: nil,
			},
			want: true,
		},
		{
			name: "return true if array has zero elements",
			args: args{
				values: []int{},
			},
			want: true,
		},
		{
			name: "return false if array has elements",
			args: args{
				values: []int{1, 2},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		testcase := tt
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			got := IsEmpty(testcase.args.values)
			g.Expect(got).To(gomega.Equal(testcase.want))
		})
	}
}
