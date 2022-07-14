package handlers

import (
	"testing"

	"github.com/onsi/gomega"
)

func Test_DetermineListRange(t *testing.T) {
	list := []string{"item"}
	type args struct {
		obj  interface{}
		page int
		size int
	}

	tests := []struct {
		name     string
		args     args
		wantSize int
	}{
		{
			name: "Should return the list with page param set to 0 and size set to 1",
			args: args{
				obj:  list,
				page: 0,
				size: 1,
			},
			wantSize: 1,
		},
		{
			name: "Should return the list with page param set to 1 and size set to 0",
			args: args{
				obj:  list,
				page: 1,
				size: 0,
			},
			wantSize: 1,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			_, count := DetermineListRange(tt.args.obj, tt.args.page, tt.args.size)
			g.Expect(count).To(gomega.Equal(tt.wantSize))
		})
	}
}
