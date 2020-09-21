package kafka

import (
	"encoding/json"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"gitlab.cee.redhat.com/service/managed-services-api/cmd/managed-services-api/environments"
	"gitlab.cee.redhat.com/service/managed-services-api/cmd/managed-services-api/flags"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
)

// NewGetCommand gets a new command for getting kafkas.
func NewGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a managed-services-api kafka request",
		Long:  "Get a managed-services-api kafka request.",
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
	syncsetService := services.NewSyncsetService(env.Clients.OCM.Connection)
	kafkaService := services.NewKafkaService(env.DBFactory, syncsetService)

	kafkaRequest, err := kafkaService.Get(id)
	if err != nil {
		glog.Fatalf("Unable to get kafka request: %s", err.Error())
	}
	indentedKafkaRequest, marshalErr := json.MarshalIndent(kafkaRequest, "", "    ")
	if marshalErr != nil {
		glog.Fatalf("Failed to format kafka request: %s", marshalErr.Error())
	}
	glog.Infof("%s", indentedKafkaRequest)
}
