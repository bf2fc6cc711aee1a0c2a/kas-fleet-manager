package handlers_test

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	. "github.com/onsi/gomega"
	"testing"
)

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
			expectedErr: errors.MaximumFieldLengthMissing("%s is not valid. Maximum length %d is required", "test-field", 5).Error(),
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

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handlers.ValidateMaxLength(&tt.args.value, tt.args.field, tt.args.maxLen())()
			Expect(err != nil).To(Equal(tt.wantErr))
			if err != nil {
				Expect(err.Error()).To(Equal(tt.expectedErr))
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

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handlers.ValidateMinLength(&tt.args.value, tt.args.field, tt.args.minLen)()
			Expect(err != nil).To(Equal(tt.wantErr))
			if err != nil {
				Expect(err.Error()).To(Equal(tt.expectedErr))
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
			expectedErr: errors.MaximumFieldLengthMissing("%s is not valid. Maximum length %d is required", "test-field", 5).Error(),
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

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handlers.ValidateLength(&tt.args.value, tt.args.field, tt.args.minLen, tt.args.maxLen())()
			Expect(err != nil).To(Equal(tt.wantErr))
			if err != nil {
				Expect(err.Error()).To(Equal(tt.expectedErr))
			}
		})
	}

}
