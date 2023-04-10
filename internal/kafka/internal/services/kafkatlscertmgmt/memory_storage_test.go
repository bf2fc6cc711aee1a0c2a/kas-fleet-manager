package kafkatlscertmgmt

import (
	"context"
	"testing"

	"github.com/onsi/gomega"
)

func Test_memoryStorage_Load(t *testing.T) {
	type args struct {
		key   string
		value []byte
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "successfully loads the same value stored",
			args: args{
				key:   "some-key",
				value: []byte("some byte"),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		testcase := tt
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			storage := newInMemoryStorage()

			storeErr := storage.Store(context.Background(), testcase.args.key, testcase.args.value)
			g.Expect(storeErr != nil).To(gomega.Equal(testcase.wantErr))

			outputValue, loadErr := storage.Load(context.Background(), testcase.args.key)
			g.Expect(loadErr == nil)
			g.Expect(outputValue).To(gomega.Equal(testcase.args.value))
		})
	}
}
