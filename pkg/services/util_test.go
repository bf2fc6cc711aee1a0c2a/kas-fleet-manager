package services

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/onsi/gomega"
	"gorm.io/gorm"
)

func TestHandleGetError(t *testing.T) {
	type args struct {
		resourceType string
		field        string
		value        interface{}
		err          error
	}
	tests := []struct {
		name string
		args args
		want *errors.ServiceError
	}{
		{
			name: "should return a Get error",
			args: args{
				resourceType: "resourceType",
				field:        "field",
				value:        "value",
				err:          nil,
			},
			want: errors.New(errors.ErrorGeneral, "Unable to find resourceType with field='value'"),
		},
		{
			name: "should return a not found error if the err is record not found",
			args: args{
				resourceType: "resourceType",
				field:        "field",
				value:        "value",
				err:          gorm.ErrRecordNotFound,
			},
			want: errors.NotFound("resourceType with field='value' not found"),
		},
		{
			name: "should return a Get error but with redacted information due to the field being personally identifiable",
			args: args{
				resourceType: "resourceType",
				field:        "username",
				value:        "value",
				err:          nil,
			},
			want: errors.New(errors.ErrorGeneral, "Unable to find resourceType with username='<redacted>'"),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(HandleGetError(tt.args.resourceType, tt.args.field, tt.args.value, tt.args.err)).To(gomega.Equal(tt.want))
		})
	}
}

func TestHandleGoneError(t *testing.T) {
	type args struct {
		resourceType string
		field        string
		value        interface{}
	}
	tests := []struct {
		name string
		args args
		want *errors.ServiceError
	}{
		{
			name: "should return a gone error",
			args: args{
				resourceType: "resourceType",
				field:        "field",
				value:        "value",
			},
			want: errors.New(errors.ErrorGone, "resourceType with field='value' has been deleted"),
		},
		{
			name: "should return a gone error but with redacted information due to the field being personally identifiable",
			args: args{
				resourceType: "resourceType",
				field:        "username",
				value:        "value",
			},
			want: errors.New(errors.ErrorGone, "resourceType with username='<redacted>' has been deleted"),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(HandleGoneError(tt.args.resourceType, tt.args.field, tt.args.value)).To(gomega.Equal(tt.want))
		})
	}
}

func TestHandleDeleteError(t *testing.T) {
	type args struct {
		resourceType string
		field        string
		value        interface{}
		err          error
	}
	tests := []struct {
		name string
		args args
		want *errors.ServiceError
	}{
		{
			name: "should return delete error",
			args: args{
				resourceType: "resourceType",
				field:        "field",
				value:        "value",
			},
			want: errors.New(errors.ErrorGeneral, "Unable to delete resourceType with field='value'"),
		},
		{
			name: "should return delete error but with redacted information due to the field being personally identifiable",
			args: args{
				resourceType: "resourceType",
				field:        "username",
				value:        "value",
			},
			want: errors.New(errors.ErrorGeneral, "Unable to delete resourceType with username='<redacted>'"),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(HandleDeleteError(tt.args.resourceType, tt.args.field, tt.args.value, tt.args.err)).To(gomega.Equal(tt.want))
		})
	}
}

func TestHandleCreateError(t *testing.T) {
	type args struct {
		resourceType string
		err          error
	}
	tests := []struct {
		name string
		args args
		want *errors.ServiceError
	}{
		{
			name: "should return conflict error",
			args: args{
				resourceType: "resourceType",
				err:          errors.GeneralError("violates unique constraint"),
			},
			want: errors.Conflict("This resourceType already exists"),
		},
		{
			name: "should return general error",
			args: args{
				resourceType: "resourceType",
				err:          errors.GeneralError("error"),
			},
			want: errors.GeneralError("Unable to create resourceType: KAFKAS-MGMT-9: error"),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(HandleCreateError(tt.args.resourceType, tt.args.err)).To(gomega.Equal(tt.want))
		})
	}
}

func TestHandleUpdateError(t *testing.T) {
	type args struct {
		resourceType string
		err          error
	}
	tests := []struct {
		name string
		args args
		want *errors.ServiceError
	}{
		{
			name: "should return conflict error",
			args: args{
				resourceType: "resourceType",
				err:          errors.GeneralError("violates unique constraint"),
			},
			want: errors.Conflict("Changes to resourceType conflict with existing records"),
		},
		{
			name: "should return general error",
			args: args{
				resourceType: "resourceType",
				err:          errors.GeneralError("error"),
			},
			want: errors.GeneralError("Unable to update resourceType: KAFKAS-MGMT-9: error"),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(HandleUpdateError(tt.args.resourceType, tt.args.err)).To(gomega.Equal(tt.want))
		})
	}
}
