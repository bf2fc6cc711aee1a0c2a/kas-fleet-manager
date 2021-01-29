package serviceaccounts

import (
	"encoding/json"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/flags"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

func NewCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a service account",
		Long:  "Create a service account.",
		Run:   runCreate,
	}
	err := environments.Environment().AddFlags(cmd.PersistentFlags())
	if err != nil {
		glog.Fatalf("Unable to add environment flags to serve command: %s", err.Error())
	}

	cmd.Flags().String(FlagName, "", "Service Account request name")
	cmd.Flags().String(FlagDesc, "", "Service Account request description")
	cmd.Flags().String(FlagOrgID, "", "OCM org id")

	return cmd
}

func runCreate(cmd *cobra.Command, args []string) {
	name := flags.MustGetDefinedString(FlagName, cmd.Flags())
	description := flags.MustGetDefinedString(FlagDesc, cmd.Flags())
	orgId := flags.MustGetDefinedString(FlagOrgID, cmd.Flags())

	if err := environments.Environment().Initialize(); err != nil {
		glog.Fatalf("Unable to initialize environment: %s", err.Error())
	}

	env := environments.Environment()

	// setup required services
	keycloakService := services.NewKeycloakService(env.Config.Keycloak)

	sa := &api.ServiceAccountRequest{
		Name:        name,
		Description: description,
	}
	ctx := cmd.Context()
	ctx = auth.SetOrgIdContext(ctx, orgId)
	serviceAccount, err := keycloakService.CreateServiceAccount(sa, ctx)
	if err != nil {
		glog.Fatalf("Unable to create service account request: %s", err.Error())
	}
	output, marshalErr := json.MarshalIndent(serviceAccount, "", "    ")
	if marshalErr != nil {
		glog.Fatalf("Failed to format service account request: %s", err.Error())
	}
	glog.V(10).Infof("%s", output)
}
