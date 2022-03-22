package supportedkafkainstancetypes

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

func NewSupportedKafkaInstanceTypesCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "supported_kafka_instance_types",
		Short: "Perform supported Kafka instance types actions directly",
		Long:  "Perform supported Kafka instance types actions directly",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			err := env.CreateServices()
			if err != nil {
				glog.Fatalf("Unable to initialize environment: %s", err.Error())
			}
		},
	}
	cmd.AddCommand(
		NewGetCommand(env),
	)

	return cmd
}
