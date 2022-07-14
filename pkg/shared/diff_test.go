package shared

import (
	"testing"

	"github.com/onsi/gomega"
)

func Test_DiffAsJson(t *testing.T) {
	type args struct {
		a     interface{}
		b     interface{}
		aName string
		bName string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "return linebreak if 'a' and 'b' are nil",
			args: args{
				a:     nil,
				b:     nil,
				aName: "testNameA",
				bName: "testNameB",
			},
			want: "\n",
		},
		{
			name: "return linebreak if 'a' and 'b' are aqual",
			args: args{
				a:     "test same string",
				b:     "test same string",
				aName: "testNameA",
				bName: "testNameB",
			},
			want: "\n",
		},
		{
			name: "return linebreak if 'a' and 'b' are empty",
			args: args{
				a:     "",
				b:     "",
				aName: "testNameA",
				bName: "testNameB",
			},
			want: "\n",
		},
		{
			name: "return string of diff from arguments 'a' and 'b'",
			args: args{
				a:     "test1",
				b:     "test2",
				aName: "testNameA",
				bName: "testNameB",
			},
			want: "\n--- testNameA\n+++ testNameB\n@@ -1 +1 @@\n-\"test1\"\n+\"test2\"\n",
		},
		{
			name: "return string of diff from arguments if either 'a' or 'b' are nil",
			args: args{
				a:     nil,
				b:     "test",
				aName: "testNameA",
				bName: "testNameB",
			},
			want: "\n--- testNameA\n+++ testNameB\n@@ -1 +1 @@\n-null\n+\"test\"\n",
		},
		{
			name: "return string of diff from arguments if either 'a' or 'b' are empty",
			args: args{
				a:     "test",
				b:     "",
				aName: "testNameA",
				bName: "testNameB",
			},
			want: "\n--- testNameA\n+++ testNameB\n@@ -1 +1 @@\n-\"test\"\n+\"\"\n",
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			got := DiffAsJson(tt.args.a, tt.args.b, tt.args.aName, tt.args.bName)
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}
