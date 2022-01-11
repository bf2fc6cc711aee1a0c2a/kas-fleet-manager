package serviceaccounts

import (
	"encoding/json"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/flags"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"

	"github.com/golang-jwt/jwt/v4"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

func NewCreateCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a service account",
		Long:  "Create a service account.",
		Run: func(cmd *cobra.Command, args []string) {
			runCreate(env, cmd, args)
		},
	}

	cmd.Flags().String(FlagName, "", "Service Account request name")
	cmd.Flags().String(FlagDesc, "", "Service Account request description")
	cmd.Flags().String(FlagOrgID, "", "OCM org id")

	return cmd
}

func runCreate(env *environments.Env, cmd *cobra.Command, args []string) {
	name := flags.MustGetDefinedString(FlagName, cmd.Flags())
	description := flags.MustGetDefinedString(FlagDesc, cmd.Flags())
	orgId := flags.MustGetDefinedString(FlagOrgID, cmd.Flags())

	// setup required services
	var keycloak = KeycloakConfig(env)
	keycloakService := services.NewKeycloakService(keycloak, keycloak.KafkaRealm)

	sa := &api.ServiceAccountRequest{
		Name:        name,
		Description: description,
	}

	ctx := cmd.Context()
	// create jwt with claims and set it in the context
	jwt := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"org_id": orgId,
	})
	ctx = auth.SetTokenInContext(ctx, jwt)

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

func KeycloakConfig(env *environments.Env) (c *keycloak.KeycloakConfig) {
	env.MustResolve(&c)
	return
}
