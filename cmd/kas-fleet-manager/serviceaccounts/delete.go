package serviceaccounts

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/flags"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"

	"github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

// NewDeleteCommand command for deleting kafkas.
func NewDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a serviceaccount",
		Long:  "Delete a serviceaccount.",
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
	keycloakService := services.NewKeycloakService(env.Config.Keycloak, env.Config.Keycloak.KafkaRealm)

	ctx := cmd.Context()
	// create jwt with claims and set it in the context
	jwt := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"org_id": orgId,
	})
	ctx = auth.SetTokenInContext(ctx, jwt)

	err := keycloakService.DeleteServiceAccount(ctx, id)
	if err != nil {
		glog.Fatalf("Unable to delete service account: %s", err.Error())
	}

	glog.V(10).Infof("Deleted service account with id %s", id)
}
