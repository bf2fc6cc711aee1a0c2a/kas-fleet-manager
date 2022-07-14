package handlers_test

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"github.com/onsi/gomega"
)

var (
	testUrlHost  = "http://test-url.com"
	invalidField = "#invalid!"
)

func Test_ValidateAsyncEnabled(t *testing.T) {
	asyncRequest, err := http.NewRequest("GET", fmt.Sprintf("%s?async=true", testUrlHost), nil)
	if err != nil {
		t.Fatal(err)
	}
	nonAsyncRequest, err := http.NewRequest("GET", testUrlHost, nil)
	if err != nil {
		t.Fatal(err)
	}
	type args struct {
		r      *http.Request
		action string
	}

	tests := []struct {
		name        string
		args        args
		wantErr     bool
		expectedErr string
	}{
		{
			name: "No error thrown if async param is provided in the URL",
			args: args{
				r: asyncRequest,
			},
			wantErr: false,
		},
		{
			name: "Should throw an error if async param is not provided in the URL",
			args: args{
				r: nonAsyncRequest,
			},
			wantErr:     true,
			expectedErr: errors.ErrorSyncActionNotSupportedReason,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			err := handlers.ValidateAsyncEnabled(tt.args.r, tt.args.action)()
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			if err != nil {
				g.Expect(err.Reason).To(gomega.Equal(tt.expectedErr))
			}
		})
	}
}

func Test_ValidateServiceAccountName(t *testing.T) {
	field := "name"
	validName := "valid-name"
	type args struct {
		value *string
		field string
	}
	tests := []struct {
		name            string
		args            args
		wantErr         bool
		expectedErrCode errors.ServiceErrorCode
	}{
		{
			name: "No error thrown if service account name is valid",
			args: args{
				field: field,
				value: &validName,
			},
			wantErr: false,
		},
		{
			name: "Should throw an error if service account name is invalid",
			args: args{
				field: field,
				value: &invalidField,
			},
			wantErr:         true,
			expectedErrCode: errors.ErrorMalformedServiceAccountName,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			err := handlers.ValidateServiceAccountName(tt.args.value, tt.args.field)()
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			if err != nil {
				g.Expect(err.Code).To(gomega.Equal(tt.expectedErrCode))
			}
		})
	}
}

func Test_ValidateServiceAccountDesc(t *testing.T) {
	field := "description"
	validDesc := "valid description"
	type args struct {
		value *string
		field string
	}
	tests := []struct {
		name            string
		args            args
		wantErr         bool
		expectedErrCode errors.ServiceErrorCode
	}{
		{
			name: "No error thrown if service account desc is valid",
			args: args{
				field: field,
				value: &validDesc,
			},
			wantErr: false,
		},
		{
			name: "Should throw an error if service account desc is invalid",
			args: args{
				field: field,
				value: &invalidField,
			},
			wantErr:         true,
			expectedErrCode: errors.ErrorMalformedServiceAccountDesc,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			err := handlers.ValidateServiceAccountDesc(tt.args.value, tt.args.field)()
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			if err != nil {
				g.Expect(err.Code).To(gomega.Equal(tt.expectedErrCode))
			}
		})
	}
}

func Test_ValidateServiceAccountId(t *testing.T) {
	field := "account-id"
	validId := "b92ba7eb-2636-dee7-93cf-8f3fc14a3ccc"
	type args struct {
		value *string
		field string
	}
	tests := []struct {
		name            string
		args            args
		wantErr         bool
		expectedErrCode errors.ServiceErrorCode
	}{
		{
			name: "No error thrown if service account id is valid",
			args: args{
				field: field,
				value: &validId,
			},
			wantErr: false,
		},
		{
			name: "Should throw an error if service account id is invalid",
			args: args{
				field: field,
				value: &invalidField,
			},
			wantErr:         true,
			expectedErrCode: errors.ErrorMalformedServiceAccountId,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			err := handlers.ValidateServiceAccountId(tt.args.value, tt.args.field)()
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			if err != nil {
				g.Expect(err.Code).To(gomega.Equal(tt.expectedErrCode))
			}
		})
	}
}

func Test_ValidateServiceAccountClientId(t *testing.T) {
	field := "account-id"
	validIdMasSSO := "srvc-acct-b92ba7eb-2636-dee7-93cf-8f3fc14a3ccc"
	validIdRedhatSSO := "b92ba7eb-2636-dee7-93cf-8f3fc14a3ccc"

	type args struct {
		value       *string
		field       string
		ssoProvider string
	}
	tests := []struct {
		name            string
		args            args
		wantErr         bool
		expectedErrCode errors.ServiceErrorCode
	}{
		{
			name: "No error thrown if mas sso service account client id is valid",
			args: args{
				field:       field,
				value:       &validIdMasSSO,
				ssoProvider: keycloak.MAS_SSO,
			},
			wantErr: false,
		},
		{
			name: "Should throw an error if mas sso service account client id is invalid",
			args: args{
				field:       field,
				value:       &invalidField,
				ssoProvider: keycloak.MAS_SSO,
			},
			wantErr:         true,
			expectedErrCode: errors.ErrorMalformedServiceAccountId,
		},
		{
			name: "No error thrown for redhat service account client id",
			args: args{
				field:       field,
				value:       &validIdRedhatSSO,
				ssoProvider: keycloak.REDHAT_SSO,
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			err := handlers.ValidateServiceAccountClientId(tt.args.value, tt.args.field, tt.args.ssoProvider)()
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			if err != nil {
				g.Expect(err.Code).To(gomega.Equal(tt.expectedErrCode))
			}
		})
	}
}

func TestValidateMaxLength(t *testing.T) {
	type args struct {
		value  string
		field  string
		maxLen func() *int
	}

	tests := []struct {
		name        string
		args        args
		wantErr     bool
		expectedErr string
	}{
		{
			name: "Value fits length",
			args: args{
				value:  "This is a very long string",
				field:  "test-field",
				maxLen: func() *int { x := 50; return &x },
			},
			wantErr: false,
		},
		{
			name: "Value too long",
			args: args{
				value:  "This is a very long string",
				field:  "test-field",
				maxLen: func() *int { x := 5; return &x },
			},
			wantErr:     true,
			expectedErr: errors.MaximumFieldLengthExceeded("%s is not valid. Maximum length %d is required", "test-field", 5).Error(),
		},
		{
			name: "Value nil maxLength",
			args: args{
				value:  "This is a very long string",
				field:  "test-field",
				maxLen: func() *int { return nil },
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			err := handlers.ValidateMaxLength(&tt.args.value, tt.args.field, tt.args.maxLen())()
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			if err != nil {
				g.Expect(err.Error()).To(gomega.Equal(tt.expectedErr))
			}
		})
	}

}

func TestValidateMinLength(t *testing.T) {
	type args struct {
		value  string
		field  string
		minLen int
	}

	tests := []struct {
		name        string
		args        args
		wantErr     bool
		expectedErr string
	}{
		{
			name: "Value fits min length",
			args: args{
				value:  "This is a very long string",
				field:  "test-field",
				minLen: 5,
			},
			wantErr: false,
		},
		{
			name: "Value too short",
			args: args{
				value:  "Hi!",
				field:  "test-field",
				minLen: 5,
			},
			wantErr:     true,
			expectedErr: errors.MinimumFieldLengthNotReached("%s is not valid. Minimum length %d is required.", "test-field", 5).Error(),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			err := handlers.ValidateMinLength(&tt.args.value, tt.args.field, tt.args.minLen)()
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			if err != nil {
				g.Expect(err.Error()).To(gomega.Equal(tt.expectedErr))
			}
		})
	}
}

func TestValidateLength(t *testing.T) {
	type args struct {
		value  string
		field  string
		maxLen func() *int
		minLen int
	}

	tests := []struct {
		name        string
		args        args
		wantErr     bool
		expectedErr string
	}{
		{
			name: "Value fits length",
			args: args{
				value:  "This is a very long string",
				field:  "test-field",
				maxLen: func() *int { x := 50; return &x },
			},
			wantErr: false,
		},
		{
			name: "Value too long",
			args: args{
				value:  "This is a very long string",
				field:  "test-field",
				maxLen: func() *int { x := 5; return &x },
			},
			wantErr:     true,
			expectedErr: errors.MaximumFieldLengthExceeded("%s is not valid. Maximum length %d is required", "test-field", 5).Error(),
		},
		{
			name: "Value too short",
			args: args{
				value:  "This is a not so long string",
				field:  "test-field",
				maxLen: func() *int { x := 50; return &x },
				minLen: 40,
			},
			wantErr:     true,
			expectedErr: errors.MinimumFieldLengthNotReached("%s is not valid. Minimum length %d is required.", "test-field", 40).Error(),
		},
		{
			name: "Value nil maxLength",
			args: args{
				value:  "This is a very long string",
				field:  "test-field",
				maxLen: func() *int { return nil },
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			err := handlers.ValidateLength(&tt.args.value, tt.args.field, tt.args.minLen, tt.args.maxLen())()
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			if err != nil {
				g.Expect(err.Error()).To(gomega.Equal(tt.expectedErr))
			}
		})
	}
}

func Test_ValidatQueryParam(t *testing.T) {
	intParseableField := "int-test"
	notParseableField := "test"
	validQueryParam := url.Values{}
	validQueryParam.Add(intParseableField, "1")
	validQueryParam.Add(notParseableField, "abc")
	type args struct {
		queryParams url.Values
		field       string
	}

	tests := []struct {
		name            string
		args            args
		wantErr         bool
		expectedErrCode errors.ServiceErrorCode
	}{
		{
			name: "No error thrown if query param is valid",
			args: args{
				field:       intParseableField,
				queryParams: validQueryParam,
			},
			wantErr: false,
		},
		{
			name: "Should throw an error if query param is empty",
			args: args{
				field:       intParseableField,
				queryParams: url.Values{},
			},
			wantErr:         true,
			expectedErrCode: errors.ErrorBadRequest,
		},
		{
			name: "Should throw an error if query param is not parseable",
			args: args{
				field:       notParseableField,
				queryParams: validQueryParam,
			},
			wantErr:         true,
			expectedErrCode: errors.ErrorBadRequest,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			err := handlers.ValidatQueryParam(tt.args.queryParams, tt.args.field)()
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			if err != nil {
				g.Expect(err.Code).To(gomega.Equal(tt.expectedErrCode))
			}
		})
	}
}
