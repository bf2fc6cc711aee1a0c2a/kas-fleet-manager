package kafka

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/flags"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	customOcm "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"

	"github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

const (
	FlagPage = "page"
	FlagSize = "size"
)

// NewListCommand creates a new command for listing kafkas.
func NewListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "lists all managed kafka requests",
		Long:  "lists all managed kafka requests",
		Run:   runList,
	}
	err := environments.Environment().AddFlags(cmd.PersistentFlags())
	if err != nil {
		glog.Fatalf("Unable to add environment flags to serve command: %s", err.Error())
	}

	cmd.Flags().String(FlagOwner, "test-user", "Username")
	cmd.Flags().String(FlagPage, "1", "Page index")
	cmd.Flags().String(FlagSize, "100", "Number of kafka requests per page")

	return cmd
}

func runList(cmd *cobra.Command, _ []string) {
	owner := flags.MustGetDefinedString(FlagOwner, cmd.Flags())
	page := flags.MustGetString(FlagPage, cmd.Flags())
	size := flags.MustGetString(FlagSize, cmd.Flags())

	if err := environments.Environment().Initialize(); err != nil {
		glog.Fatalf("Unable to initialize environment: %s", err.Error())
	}

	env := environments.Environment()

	// setup required services
	ocmClient := customOcm.NewClient(env.Clients.OCM.Connection)

	clusterService := services.NewClusterService(env.DBFactory, ocmClient, env.Config.AWS, env.Config.OSDClusterConfig)
	syncsetService := services.NewSyncsetService(ocmClient)
	keycloakService := services.NewKeycloakService(env.Config.Keycloak, env.Config.Keycloak.KafkaRealm)
	QuotaService := services.NewQuotaService(ocmClient)
	kafkaService := services.NewKafkaService(env.DBFactory, syncsetService, clusterService, keycloakService, env.Config.Kafka, env.Config.AWS, QuotaService)

	// create jwt with claims and set it in the context
	jwt := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"username": owner,
	})
	ctx := auth.SetTokenInContext(context.TODO(), jwt)

	// build list arguments
	url := url.URL{}
	query := url.Query()
	query.Add(FlagPage, page)
	query.Add(FlagSize, size)
	listArgs := services.NewListArguments(query)

	kafkaList, paging, err := kafkaService.List(ctx, listArgs)
	if err != nil {
		glog.Fatalf("Unable to list kafka request: %s", err.Error())
	}

	// format output
	kafkaRequestList := openapi.KafkaRequestList{
		Kind:  "KafkaRequestList",
		Page:  int32(paging.Page),
		Size:  int32(paging.Size),
		Total: int32(paging.Total),
		Items: []openapi.KafkaRequest{},
	}

	for _, kafkaRequest := range kafkaList {
		converted := presenters.PresentKafkaRequest(kafkaRequest)
		kafkaRequestList.Items = append(kafkaRequestList.Items, converted)
	}

	output, marshalErr := json.MarshalIndent(kafkaRequestList, "", "    ")
	if marshalErr != nil {
		glog.Fatalf("Failed to format kafka request list: %s", err.Error())
	}

	glog.V(10).Infof("%s", output)
}
