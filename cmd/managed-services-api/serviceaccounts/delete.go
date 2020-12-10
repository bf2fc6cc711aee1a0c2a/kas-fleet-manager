package serviceaccounts

import (
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"gitlab.cee.redhat.com/service/managed-services-api/cmd/managed-services-api/environments"
	"gitlab.cee.redhat.com/service/managed-services-api/cmd/managed-services-api/flags"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/auth"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
)

// NewDeleteCommand command for deleting kafkas.
func NewDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a managed-services-api serviceaccount",
		Long:  "Delete a managed-services-api serviceaccount.",
		Run:   runDelete,
	}
	err := environments.Environment().AddFlags(cmd.PersistentFlags())
	if err != nil {
		glog.Fatalf("Unable to add environment flags to serve command: %s", err.Error())
	}

	cmd.Flags().String(FlagSaID, "", "Service Account id")
	cmd.Flags().String(FlagOrgID, "", "OCM org id")
	return cmd
}

func runDelete(cmd *cobra.Command, args []string) {
	id := flags.MustGetDefinedString(FlagSaID, cmd.Flags())
	orgId := flags.MustGetDefinedString(FlagOrgID, cmd.Flags())
	if err := environments.Environment().Initialize(); err != nil {
		glog.Fatalf("Unable to initialize environment: %s", err.Error())
	}
	env := environments.Environment()
	// setup required services
	keycloakService := services.NewKeycloakService(env.Config.Keycloak)

	ctx := cmd.Context()
	ctx = auth.SetOrgIdContext(ctx, orgId)
	err := keycloakService.DeleteServiceAccount(ctx, id)
	if err != nil {
		glog.Fatalf("Unable to delete service account: %s", err.Error())
	}

	glog.V(10).Infof("Deleted service account with id %s", id)
}
