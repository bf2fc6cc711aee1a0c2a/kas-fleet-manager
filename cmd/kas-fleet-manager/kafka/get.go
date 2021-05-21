package kafka

import (
	"encoding/json"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/flags"
	customOcm "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

// NewGetCommand gets a new command for getting kafkas.
func NewGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a kafka request",
		Long:  "Get a kafka request.",
		Run:   runGet,
	}
	err := environments.Environment().AddFlags(cmd.PersistentFlags())
	if err != nil {
		glog.Fatalf("Unable to add environment flags to serve command: %s", err.Error())
	}

	cmd.Flags().String(FlagID, "", "Kafka id")

	return cmd
}

func runGet(cmd *cobra.Command, _ []string) {
	id := flags.MustGetDefinedString(FlagID, cmd.Flags())

	if err := environments.Environment().Initialize(); err != nil {
		glog.Fatalf("Unable to initialize environment: %s", err.Error())
	}

	env := environments.Environment()

	// setup required services
	ocmClient := customOcm.NewClient(env.Clients.OCM.Connection)

	clusterService := services.NewClusterService(env.DBFactory, ocmClient, env.Config.AWS, env.Config.OSDClusterConfig)
	keycloakService := services.NewKeycloakService(env.Config.Keycloak, env.Config.Keycloak.KafkaRealm)
	QuotaService := services.NewQuotaService(ocmClient)
	kafkaService := services.NewKafkaService(env.DBFactory, clusterService, keycloakService, env.Config.Kafka, env.Config.AWS, QuotaService)

	kafkaRequest, err := kafkaService.GetById(id)
	if err != nil {
		glog.Fatalf("Unable to get kafka request: %s", err.Error())
	}
	indentedKafkaRequest, marshalErr := json.MarshalIndent(kafkaRequest, "", "    ")
	if marshalErr != nil {
		glog.Fatalf("Failed to format kafka request: %s", marshalErr.Error())
	}
	glog.V(10).Infof("%s", indentedKafkaRequest)
}
