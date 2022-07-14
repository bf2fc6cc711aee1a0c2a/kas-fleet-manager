package flags

import (
	"fmt"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/onsi/gomega"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	testFlag = "test-flag"
)

func getTestStringFlagSet() *pflag.FlagSet {
	testFlagSet := cobra.Command{
		Use: "test",
	}

	testFlagSet.Flags().String(testFlag, testFlag, "testing flags")

	return testFlagSet.Flags()
}

func TestFlags_MustGetDefinedString(t *testing.T) {
	type args struct {
		flagName string
		flags    *pflag.FlagSet
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "should get defined string from FlagSet",
			args: args{
				flagName: testFlag,
				flags:    getTestStringFlagSet(),
			},
			want: testFlag,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)

			g.Expect(MustGetDefinedString(tt.args.flagName, tt.args.flags)).To(gomega.Equal(tt.want))
		})
	}
}

func TestFlags_MustGetString(t *testing.T) {
	type args struct {
		flagName string
		flags    *pflag.FlagSet
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "should get string from FlagSet",
			args: args{
				flagName: testFlag,
				flags:    getTestStringFlagSet(),
			},
			want: testFlag,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(MustGetString(tt.args.flagName, tt.args.flags)).To(gomega.Equal(tt.want))
		})
	}
}

func TestFlags_MustGetBool(t *testing.T) {
	boolFlagValue := false
	testFlagSet := cobra.Command{
		Use: "test",
	}

	testFlagSet.Flags().Bool(testFlag, boolFlagValue, "testing flags")
	type args struct {
		flagName string
		flags    *pflag.FlagSet
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "should get bool value from FlagSet",
			args: args{
				flagName: testFlag,
				flags:    testFlagSet.Flags(),
			},
			want: boolFlagValue,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(MustGetBool(tt.args.flagName, tt.args.flags)).To(gomega.Equal(tt.want))
		})
	}
}

func TestFlags_undefinedValueMessage(t *testing.T) {
	type args struct {
		flagName string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "should get undefined value message",
			args: args{
				flagName: testFlag,
			},
			want: fmt.Sprintf("flag %s has undefined value", testFlag),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(undefinedValueMessage(tt.args.flagName)).To(gomega.Equal(tt.want))
		})
	}
}

func TestFlags_notFoundMessage(t *testing.T) {
	testErrorMessgae := "test error"
	type args struct {
		flagName string
		e        error
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "should get not found message",
			args: args{
				flagName: testFlag,
				e:        errors.GeneralError(testErrorMessgae),
			},
			want: fmt.Sprintf("could not get flag %s from flag set: %s-%d: %s", testFlag, errors.ERROR_CODE_PREFIX, int32(errors.GeneralError(testErrorMessgae).Code), testErrorMessgae),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			errorMessage := notFoundMessage(tt.args.flagName, tt.args.e)
			g.Expect(errorMessage).To(gomega.Equal(tt.want))
		})
	}
}
