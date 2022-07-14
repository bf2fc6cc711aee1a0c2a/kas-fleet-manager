package handlers

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/compat"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/onsi/gomega"
)

func Test_errorHandler(t *testing.T) {
	req, rw := GetHandlerParams("GET", "/", nil, t)
	type args struct {
		w   http.ResponseWriter
		r   *http.Request
		cfg *HandlerConfig
		err *errors.ServiceError
	}
	tests := []struct {
		name           string
		args           args
		wantStatusCode int
	}{
		{
			name:           "Should call error handler with empty HandleConfig",
			wantStatusCode: http.StatusInternalServerError,
			args: args{
				w:   rw,
				r:   req,
				cfg: &HandlerConfig{},
				err: errors.GeneralError("test"),
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			errorHandler(tt.args.r, tt.args.w, tt.args.cfg, tt.args.err)
			g.Expect(rw.Code).To(gomega.Equal(tt.wantStatusCode))
		})
	}
}

func Test_Handle(t *testing.T) {
	var regionCapacityListItem api.RegionCapacityListItem
	req, rw := GetHandlerParams("GET", "/", nil, t)

	pReq, _ := http.NewRequest("POST", "/", bytes.NewBuffer([]byte(`{"instance_type":"test"}`)))

	type args struct {
		w          http.ResponseWriter
		r          *http.Request
		cfg        *HandlerConfig
		httpStatus int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Should call Handle and return no error when no error is returned in the action",
			args: args{
				w: rw,
				r: req,
				cfg: &HandlerConfig{
					Action: func() (interface{}, *errors.ServiceError) {
						return nil, nil
					},
				},
				httpStatus: http.StatusOK,
			},
		},
		{
			name: "Should call Handle and return an error when an error is returned in the action",
			args: args{
				w: rw,
				r: req,
				cfg: &HandlerConfig{
					Action: func() (interface{}, *errors.ServiceError) {
						return nil, errors.NotFound("some action error")
					},
				},
				httpStatus: http.StatusOK,
			},
		},
		{
			name: "Should MarshallInto without any error",
			args: args{
				w: rw,
				r: pReq,
				cfg: &HandlerConfig{
					MarshalInto: &regionCapacityListItem,
					Action: func() (interface{}, *errors.ServiceError) {
						return nil, nil
					},
				},
			},
		},
		{
			name: "Should validate without error",
			args: args{
				w: rw,
				r: pReq,
				cfg: &HandlerConfig{
					Validate: []Validate{
						func() *errors.ServiceError {
							return nil
						},
					},
					Action: func() (interface{}, *errors.ServiceError) {
						return nil, nil
					},
				},
			},
		},
		{
			name: "Should throw an error if validation fails",
			args: args{
				w: rw,
				r: pReq,
				cfg: &HandlerConfig{
					Validate: []Validate{
						func() *errors.ServiceError {
							return errors.GeneralError("validation failed")
						},
					},
					Action: func() (interface{}, *errors.ServiceError) {
						return nil, nil
					},
				},
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			Handle(tt.args.w, tt.args.r, tt.args.cfg, tt.args.httpStatus)
			g.Expect(rw.Code).ToNot(gomega.Equal(0))
		})
	}
}

func Test_HandleDelete(t *testing.T) {
	req, rw := GetHandlerParams("DELETE", "/", nil, t)
	type args struct {
		w          http.ResponseWriter
		r          *http.Request
		cfg        *HandlerConfig
		httpStatus int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Should call HandleDelete and return no error when no error is returned in the action",
			args: args{
				w: rw,
				r: req,
				cfg: &HandlerConfig{
					Action: func() (interface{}, *errors.ServiceError) {
						return nil, nil
					},
				},
				httpStatus: http.StatusOK,
			},
		},
		{
			name: "Should call HandleDelete and return an error when an error is returned in the action",
			args: args{
				w: rw,
				r: req,
				cfg: &HandlerConfig{
					Action: func() (interface{}, *errors.ServiceError) {
						return nil, errors.NotFound("some action error")
					},
				},
				httpStatus: http.StatusOK,
			},
		},
		{
			name: "Should validate without error",
			args: args{
				w: rw,
				r: req,
				cfg: &HandlerConfig{
					Validate: []Validate{
						func() *errors.ServiceError {
							return nil
						},
					},
					Action: func() (interface{}, *errors.ServiceError) {
						return nil, nil
					},
				},
			},
		},
		{
			name: "Should throw an error if validation fails",
			args: args{
				w: rw,
				r: req,
				cfg: &HandlerConfig{
					Validate: []Validate{
						func() *errors.ServiceError {
							return errors.GeneralError("validation failed")
						},
					},
					Action: func() (interface{}, *errors.ServiceError) {
						return nil, nil
					},
				},
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			HandleDelete(tt.args.w, tt.args.r, tt.args.cfg, tt.args.httpStatus)
			g.Expect(rw.Code).ToNot(gomega.Equal(0))
		})
	}
}

func Test_HandleGet(t *testing.T) {
	req, rw := GetHandlerParams("GET", "/{id}", nil, t)
	type args struct {
		w          http.ResponseWriter
		r          *http.Request
		cfg        *HandlerConfig
		httpStatus int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Should call HandleGet and return no error when no error is returned in the action",
			args: args{
				w: rw,
				r: req,
				cfg: &HandlerConfig{
					Action: func() (interface{}, *errors.ServiceError) {
						return nil, nil
					},
				},
				httpStatus: http.StatusOK,
			},
		},
		{
			name: "Should call HandleGet and return an error when an error is returned in the action",
			args: args{
				w: rw,
				r: req,
				cfg: &HandlerConfig{
					Action: func() (interface{}, *errors.ServiceError) {
						return nil, errors.NotFound("some action error")
					},
				},
				httpStatus: http.StatusOK,
			},
		},
		{
			name: "Should validate without error",
			args: args{
				w: rw,
				r: req,
				cfg: &HandlerConfig{
					Validate: []Validate{
						func() *errors.ServiceError {
							return nil
						},
					},
					Action: func() (interface{}, *errors.ServiceError) {
						return nil, nil
					},
				},
			},
		},
		{
			name: "Should throw an error if validation fails",
			args: args{
				w: rw,
				r: req,
				cfg: &HandlerConfig{
					Validate: []Validate{
						func() *errors.ServiceError {
							return errors.GeneralError("validation failed")
						},
					},
					Action: func() (interface{}, *errors.ServiceError) {
						return nil, nil
					},
				},
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			HandleGet(tt.args.w, tt.args.r, tt.args.cfg)
			g.Expect(rw.Code).ToNot(gomega.Equal(0))
		})
	}
}

func Test_HandleList(t *testing.T) {
	req, rw := GetHandlerParams("GET", "/", nil, t)
	type args struct {
		w          http.ResponseWriter
		r          *http.Request
		cfg        *HandlerConfig
		httpStatus int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Should call HandleList and return no error when no error is returned in the action",
			args: args{
				w: rw,
				r: req,
				cfg: &HandlerConfig{
					Action: func() (interface{}, *errors.ServiceError) {
						return api.RegionCapacityListItem{
							InstanceType: "test",
						}, nil
					},
				},
				httpStatus: http.StatusOK,
			},
		},
		{
			name: "Should call HandleList and return an error when an error is returned in the action",
			args: args{
				w: rw,
				r: req,
				cfg: &HandlerConfig{
					Action: func() (interface{}, *errors.ServiceError) {
						return nil, errors.NotFound("some action error")
					},
				},
				httpStatus: http.StatusOK,
			},
		},
		{
			name: "Should validate without error",
			args: args{
				w: rw,
				r: req,
				cfg: &HandlerConfig{
					Validate: []Validate{
						func() *errors.ServiceError {
							return nil
						},
					},
					Action: func() (interface{}, *errors.ServiceError) {
						return nil, nil
					},
				},
			},
		},
		{
			name: "Should throw an error if validation fails",
			args: args{
				w: rw,
				r: req,
				cfg: &HandlerConfig{
					Validate: []Validate{
						func() *errors.ServiceError {
							return errors.GeneralError("validation failed")
						},
					},
					Action: func() (interface{}, *errors.ServiceError) {
						return nil, nil
					},
				},
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			HandleList(tt.args.w, tt.args.r, tt.args.cfg)
			g.Expect(rw.Code).ToNot(gomega.Equal(0))
		})
	}
}

func Test_ConvertToPrivateError(t *testing.T) {
	type args struct {
		e compat.Error
	}

	tests := []struct {
		name string
		args args
		want compat.PrivateError
	}{
		{
			name: "should return converted PrivateError",
			args: args{
				e: compat.Error{},
			},
			want: compat.PrivateError{},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(ConvertToPrivateError(tt.args.e)).To(gomega.Equal(tt.want))
		})
	}
}
