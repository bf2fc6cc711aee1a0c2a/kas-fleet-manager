package services

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	serviceError "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	pkgErr "github.com/pkg/errors"
	"gorm.io/gorm"
)

const (
	resourceType           = "sampleResource"
	mockKafkaRequestID     = "9bsv0s3fd06g02i2be9g" // sample kafka request ID
	mockIDWithInvalidChars = "vp&xG^nl9MStC@SI*#c$6V^TKq0"
)

func Test_HandleGetError(t *testing.T) {
	cause := pkgErr.WithStack(gorm.ErrInvalidData)
	type args struct {
		resourceType string
		field        string
		value        interface{}
		err          error
	}
	tests := []struct {
		name string
		args args
		want *serviceError.ServiceError
	}{
		{
			name: "Handler should return a general error for any errors other than record not found",
			args: args{
				resourceType: resourceType,
				field:        "id",
				value:        "sample-id",
				err:          cause,
			},
			want: serviceError.NewWithCause(serviceError.ErrorGeneral, cause, "Unable to find %s with id='sample-id'", resourceType),
		},
		{
			name: "Handler should return a not found error if record was not found in the database",
			args: args{
				resourceType: resourceType,
				field:        "id",
				value:        "sample-id",
				err:          gorm.ErrRecordNotFound,
			},
			want: serviceError.NotFound("%s with id='sample-id' not found", resourceType),
		},
		{
			name: "Handler should redact sensitive fields from the error message",
			args: args{
				resourceType: resourceType,
				field:        "email",
				value:        "sample@example.com",
				err:          gorm.ErrRecordNotFound,
			},
			want: serviceError.NotFound("%s with email='<redacted>' not found", resourceType),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := services.HandleGetError(tt.args.resourceType, tt.args.field, tt.args.value, tt.args.err); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleGetError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_handleCreateError(t *testing.T) {
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
			name: "Handler should return a general error for any other errors than violating unique constraints",
			args: args{
				resourceType: resourceType,
				err:          gorm.ErrInvalidField,
			},
			want: errors.GeneralError("Unable to create %s: %s", resourceType, gorm.ErrInvalidField.Error()),
		},
		{
			name: "Handler should return a conflict error if creation error is due to violating unique constraints",
			args: args{
				resourceType: resourceType,
				err:          fmt.Errorf("transaction violates unique constraints"),
			},
			want: errors.Conflict("This %s already exists", resourceType),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := services.HandleCreateError(tt.args.resourceType, tt.args.err); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("handleCreateError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_handleUpdateError(t *testing.T) {
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
			name: "Handler should return a general error for any other errors than violating unique constraints",
			args: args{
				resourceType: resourceType,
				err:          gorm.ErrInvalidData,
			},
			want: errors.GeneralError("Unable to update %s: %s", resourceType, gorm.ErrInvalidData.Error()),
		},
		{
			name: "Handler should return a conflict error if update error is due to violating unique constraints",
			args: args{
				resourceType: resourceType,
				err:          fmt.Errorf("transaction violates unique constraints"),
			},
			want: errors.Conflict("Changes to %s conflict with existing records", resourceType),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := services.HandleUpdateError(tt.args.resourceType, tt.args.err); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("handleUpdateError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_TruncateString(t *testing.T) {
	exampleString := "example-string"
	type args struct {
		str string
		num int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "should truncate string successfully",
			args: args{
				str: exampleString,
				num: 10,
			},
			want: exampleString[0:10],
		},
		{
			name: "should not truncate string if wanted length is less than given string length",
			args: args{
				str: exampleString,
				num: 15,
			},
			want: exampleString,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TruncateString(tt.args.str, tt.args.num); got != tt.want {
				t.Errorf("TruncateString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_buildTruncateKafkaIdentifier(t *testing.T) {
	mockShortKafkaName := "kafka"
	mockLongKafkaName := "sample-kafka-name-long"

	type args struct {
		kafkaRequest *dbapi.KafkaRequest
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "build kafka identifier with a short name successfully",
			args: args{
				kafkaRequest: &dbapi.KafkaRequest{
					Name: mockShortKafkaName,
				},
			},
			want: fmt.Sprintf("%s-%s", mockShortKafkaName, strings.ToLower(mockKafkaRequestID)),
		},
		{
			name: "build kafka identifier with a long name successfully",
			args: args{
				kafkaRequest: &dbapi.KafkaRequest{
					Name: mockLongKafkaName,
				},
			},
			want: fmt.Sprintf("%s-%s", mockLongKafkaName[0:truncatedNameLen], strings.ToLower(mockKafkaRequestID)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.args.kafkaRequest.ID = mockKafkaRequestID
			if got := buildTruncateKafkaIdentifier(tt.args.kafkaRequest); got != tt.want {
				t.Errorf("buildTruncateKafkaIdentifier() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_MaskProceedingandTrailingDash(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "should replace '-' prefix and suffix with a subdomain safe value",
			args: args{
				name: "-example-name-",
			},
			want: fmt.Sprintf("%[1]sexample-name%[1]s", appendChar),
		},
		{
			name: "should not replace '-' if its not a prefix or suffix of the given string",
			args: args{
				name: "example-name",
			},
			want: "example-name",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MaskProceedingandTrailingDash(tt.args.name); got != tt.want {
				t.Errorf("MaskProceedingandTrailingDash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_replaceHostSpecialChar(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "replace all invalid characters in an invalid host name",
			args: args{
				name: fmt.Sprintf("-host-%s", mockIDWithInvalidChars),
			},
			want: "ahost-vp-xg-nl-mstc-si-c--v-tkqa",
		},
		{
			name: "valid hostname should not be modified",
			args: args{
				name: "sample-host-name",
			},
			want: "sample-host-name",
		},
		{
			name: "should return an error if given host name is an empty string",
			args: args{
				name: "",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := replaceHostSpecialChar(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("replaceHostSpecialChar() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("replaceHostSpecialChar() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_contains(t *testing.T) {
	searchedString := "findMe"
	someSlice := []string{"some", "string", "values"}
	sliceWithFindMe := []string{"some", "string", "values", "findMe"}
	var emptySlice []string
	type args struct {
		slice []string
		s     string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Check for a string in an empty slice",
			args: args{
				s:     searchedString,
				slice: emptySlice,
			},
			want: false,
		},
		{
			name: "Check for a string in a non-empty slice that doesn't contain the string",
			args: args{
				s:     searchedString,
				slice: someSlice,
			},
			want: false,
		},
		{
			name: "Check for a string in a non-empty slice that contains that string",
			args: args{
				s:     searchedString,
				slice: sliceWithFindMe,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shared.Contains(tt.args.slice, tt.args.s)
			if got != tt.want {
				t.Errorf("contains() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_ParseString(t *testing.T) {
	type args struct {
		size string
	}
	tests := []struct {
		name    string
		args    args
		want    int64
		wantErr bool
	}{
		{
			name: "should parse valid size ID and return an int value with no error",
			args: args{
				size: "x1",
			},
			want: 1,
		},
		{
			name: "should throw an error when attempting to parse invalid size ID",
			args: args{
				size: "xx",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSize(tt.args.size)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseSize() = %v, want %v", got, tt.want)
			}
		})
	}
}
