package kafka

import (
	"encoding/json"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/flags"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

// NewGetCommand gets a new command for getting kafkas.
func NewGetCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a kafka request",
		Long:  "Get a kafka request.",
		Run: func(cmd *cobra.Command, args []string) {
			runGet(env, cmd, args)
		},
	}
	cmd.Flags().String(FlagID, "", "Kafka id")

	return cmd
}

func runGet(env *environments.Env, cmd *cobra.Command, _ []string) {
	id := flags.MustGetDefinedString(FlagID, cmd.Flags())
	var kafkaService services.KafkaService
	env.MustResolveAll(&kafkaService)

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
