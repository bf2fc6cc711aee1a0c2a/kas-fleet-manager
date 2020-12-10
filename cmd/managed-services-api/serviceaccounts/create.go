package serviceaccounts

import (
	"encoding/json"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"gitlab.cee.redhat.com/service/managed-services-api/cmd/managed-services-api/environments"
	"gitlab.cee.redhat.com/service/managed-services-api/cmd/managed-services-api/flags"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/auth"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
)

func NewCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a managed-services-api service account",
		Long:  "Create a managed-services-api service account.",
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
