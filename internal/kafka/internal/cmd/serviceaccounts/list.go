package serviceaccounts

import (
	"context"
	"encoding/json"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	presenters2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/flags"
	"strconv"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"

	"github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

func NewListCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "lists all service accounts",
		Long:  "lists all service accounts",
		Run: func(cmd *cobra.Command, args []string) {
			runList(env, cmd)
		},
	}

	cmd.Flags().String(FlagFirst, "", "Service Account first result")
	cmd.Flags().String(FlagMax, "", "Service Account max result")
	cmd.Flags().String(FlagOrgID, "", "OCM organisation id")
	return cmd
}

func runList(env *environments.Env, cmd *cobra.Command) {
	first, err := strconv.Atoi(flags.MustGetDefinedString(FlagFirst, cmd.Flags()))
	if err != nil {
		glog.Fatalf("Unable to read flag first: %s", err.Error())
	}
	max, err := strconv.Atoi(flags.MustGetDefinedString(FlagMax, cmd.Flags()))
	if err != nil {
		glog.Fatalf("Unable to read flag max: %s", err.Error())
	}

	orgId := flags.MustGetDefinedString(FlagOrgID, cmd.Flags())

	// setup required services
	keycloakService := services.NewKeycloakService(env.Config.Keycloak, env.Config.Keycloak.KafkaRealm)

	// create jwt with claims and set it in the context
	jwt := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"org_id": orgId,
	})
	ctx := auth.SetTokenInContext(context.TODO(), jwt)

	sa, svcErr := keycloakService.ListServiceAcc(ctx, first, max)
	if svcErr != nil {
		glog.Fatalf("Unable to list service account list: %s", svcErr.Error())
	}
	serviceAccountList := public.ServiceAccountList{
		Kind:  "ServiceAccountList",
		Items: []public.ServiceAccountListItem{},
	}

	for _, account := range sa {
		converted := presenters2.PresentServiceAccountListItem(&account)
		serviceAccountList.Items = append(serviceAccountList.Items, converted)
	}

	output, marshalErr := json.MarshalIndent(serviceAccountList, "", "    ")
	if marshalErr != nil {
		glog.Fatalf("Failed to format service account list: %s", err.Error())
	}

	glog.V(10).Infof("%s", output)

}
