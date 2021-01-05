package errors

import (
	"encoding/json"
	"os"

	"github.com/golang/glog"
	"github.com/spf13/cobra"

	"gitlab.cee.redhat.com/service/managed-services-api/cmd/managed-services-api/environments"
	"gitlab.cee.redhat.com/service/managed-services-api/cmd/managed-services-api/flags"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	svcErr "gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
)

const (
	FlagsSaveToFile = "save-to-file"
)

// NewListCommand creates a new command for listing the errors which can be returned by the service.
func NewListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List the errors which can be returned by the service",
		Long:  "List the errors which can be returned by the service",
		Run:   runList,
	}
	err := environments.Environment().AddFlags(cmd.PersistentFlags())
	if err != nil {
		glog.Fatalf("Unable to add environment flags to errors command: %s", err.Error())
	}

	cmd.Flags().String(FlagsSaveToFile, "", "File path to save the list of errors in JSON format to (i.e. 'errors.json')")

	return cmd
}

func runList(cmd *cobra.Command, _ []string) {
	filePath := flags.MustGetString(FlagsSaveToFile, cmd.Flags())

	var svcErrors []api.Error

	// add code prefix to service error code
	for _, err := range svcErr.Errors() {
		svcErrors = append(svcErrors, api.Error{
			Code:   svcErr.CodeStr(err.Code),
			Reason: err.Reason,
		})
	}

	svcErrorsJson, err := json.MarshalIndent(svcErrors, "", "\t")
	if err != nil {
		glog.Fatalf("failed to unmarshal struct")
	}

	// Write to stdout if filepath is not defined, otherwise save to the specified file
	if filePath == "" {
		glog.Infoln(string(svcErrorsJson))
	} else {
		file, err := os.Create(filePath)
		if err != nil {
			glog.Fatalf("failed to create file: %v", err)
		}
		defer file.Close()

		if _, err = file.WriteString(string(svcErrorsJson)); err != nil {
			glog.Fatalf("failed to write to file: %v", err)
		}
		glog.Infof("Service errors saved to %s", file.Name())
	}
}
