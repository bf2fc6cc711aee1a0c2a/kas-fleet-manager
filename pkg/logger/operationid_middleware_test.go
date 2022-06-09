package logger

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/gomega"
)

func getTestCtxWithOpId() context.Context {
	return context.WithValue(context.Background(), OpIDKey, testOpId)
}

func Test_WithOpID(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name     string
		args     args
		opIdSet  bool
		wantOpId string
	}{
		{
			name: "should return context with OpID previously set in the context",
			args: args{
				ctx: context.Background(),
			},
			opIdSet: false,
		},
		{
			name: "should return context with new OpID if now set in the context",
			args: args{
				ctx: getTestCtxWithOpId(),
			},
			opIdSet:  true,
			wantOpId: testOpId,
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			context := WithOpID(tt.args.ctx)
			Expect(context.Value(OpIDKey)).ToNot(Equal(""))
			Expect(context.Value(OpIDKey) == tt.wantOpId).To(Equal(tt.opIdSet))
		})
	}
}

func Test_GetOperationID(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "should return an empty string if OpID value is not set in the context",
			args: args{
				ctx: context.Background(),
			},
			want: "",
		},
		{
			name: "should return OpID value when its set in the context",
			args: args{
				ctx: getTestCtxWithOpId(),
			},
			want: testOpId,
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			Expect(GetOperationID(tt.args.ctx)).To(Equal(tt.want))
		})
	}
}

func Test_OperationIDMiddleware(t *testing.T) {
	RegisterTestingT(t)
	h := http.NewServeMux()
	opIdHandler := OperationIDMiddleware(h)
	req, err := http.NewRequest("GET", "/", nil)
	Expect(err).NotTo(HaveOccurred())
	opIdHandler.ServeHTTP(httptest.NewRecorder(), req)

	// after creating middleware handler, value of OpIDKey key in the context should be set
	Expect(req.Context().Value(OpIDKey)).ToNot(Equal(""))
}
