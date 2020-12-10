package serviceaccounts

import (
	"context"
	"encoding/json"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"gitlab.cee.redhat.com/service/managed-services-api/cmd/managed-services-api/environments"
	"gitlab.cee.redhat.com/service/managed-services-api/cmd/managed-services-api/flags"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
)

// NewDeleteCommand command for deleting kafkas.
func NewResetCredsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset-credentials",
		Short: "Reset Credentials of a managed-services-api serviceaccount",
		Long:  "Reset Credentials of a managed-services-api serviceaccount.",
		Run:   runResetCreds,
	}
	err := environments.Environment().AddFlags(cmd.PersistentFlags())
	if err != nil {
		glog.Fatalf("Unable to add environment flags to serve command: %s", err.Error())
	}

	cmd.Flags().String(FlagSaID, "", "Service Account id")
	return cmd
}

func runResetCreds(cmd *cobra.Command, args []string) {
	id := flags.MustGetDefinedString(FlagSaID, cmd.Flags())
	if err := environments.Environment().Initialize(); err != nil {
		glog.Fatalf("Unable to initialize environment: %s", err.Error())
	}
	env := environments.Environment()
	// setup required services
	keycloakService := services.NewKeycloakService(env.Config.Keycloak)
	ctx := context.TODO()
	sa, err := keycloakService.ResetServiceAccountCredentials(ctx, id)
	if err != nil {
		glog.Fatalf("Unable to reset credentials of a service account: %s", err.Error())
	}
	output, marshalErr := json.MarshalIndent(sa, "", "    ")
	if marshalErr != nil {
		glog.Fatalf("Failed to format service account: %s", err.Error())
	}
	glog.V(10).Infof("%s", output)
}
