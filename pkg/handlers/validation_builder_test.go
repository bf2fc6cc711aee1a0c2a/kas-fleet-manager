package handlers

import (
	"regexp"
	"testing"

	"github.com/onsi/gomega"
)

var (
	emptyString = ""
)

func Test_Validation(t *testing.T) {
	shortString := "short"
	with := []string{shortString}
	without := []string{"other string"}
	lettersRegexp := regexp.MustCompile(`[a-zA-Z]+`)
	digitsRegexp := regexp.MustCompile(`[0-9]+`)
	type args struct {
		field  string
		value  *string
		option ValidateOption
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Should throw an error if string value is too short",
			args: args{
				field:  "test-field",
				value:  &shortString,
				option: MinLen(10),
			},
			wantErr: true,
		},
		{
			name: "Should not throw an error if string value is greater than min length",
			args: args{
				field:  "test-field",
				value:  &shortString,
				option: MinLen(1),
			},
			wantErr: false,
		},
		{
			name: "Should throw an error if string value is too long",
			args: args{
				field:  "test-field",
				value:  &shortString,
				option: MaxLen(1),
			},
			wantErr: true,
		},
		{
			name: "Should not throw an error if string value is not longer than max length",
			args: args{
				field:  "test-field",
				value:  &shortString,
				option: MaxLen(10),
			},
			wantErr: false,
		},
		{
			name: "Should throw an error if string is not one of the array elements",
			args: args{
				field:  "test-field",
				value:  &shortString,
				option: IsOneOf(without...),
			},
			wantErr: true,
		},
		{
			name: "Should not throw an error if string is one of the array elements",
			args: args{
				field:  "test-field",
				value:  &shortString,
				option: IsOneOf(with...),
			},
			wantErr: false,
		},
		{
			name: "Should throw an error if string does not match regex expression",
			args: args{
				field:  "test-field",
				value:  &shortString,
				option: Matches(digitsRegexp),
			},
			wantErr: true,
		},
		{
			name: "Should throw an error if string to match regex expression is empty",
			args: args{
				field:  "test-field",
				value:  &emptyString,
				option: Matches(digitsRegexp),
			},
			wantErr: true,
		},
		{
			name: "Should not throw an error if string matches regex expression",
			args: args{
				field:  "test-field",
				value:  &shortString,
				option: Matches(lettersRegexp),
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(Validation(tt.args.field, tt.args.value, tt.args.option)() != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_WithDefault(t *testing.T) {
	d := "default"
	type args struct {
		field  string
		value  *string
		option ValidateOption
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Should set default value if passed value is empty",
			args: args{
				field:  "test-field",
				value:  &emptyString,
				option: WithDefault(d),
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(WithDefault(*tt.args.value)(tt.args.field, tt.args.value) != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}
