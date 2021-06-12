package observatorium

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/flags"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

func NewRunGetStateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-state",
		Short: "Fetch kafka state metric from Prometheus",
		Run:   runGethResourceStateMetrics,
	}

	cmd.Flags().String(FlagName, "", "Kafka name")
	cmd.Flags().String(FlagNameSpace, "", "Kafka namepace")

	return cmd
}
func runGethResourceStateMetrics(cmd *cobra.Command, _args []string) {

	name := flags.MustGetDefinedString(FlagName, cmd.Flags())
	namespace := flags.MustGetDefinedString(FlagNameSpace, cmd.Flags())

	if err := environments.Environment().Initialize(); err != nil {
		glog.Fatalf("Unable to initialize environment: %s", err.Error())
	}

	env := environments.Environment()
	kafkaState, err := env.Services.Observatorium.GetKafkaState(name, namespace)
	if err != nil {
		glog.Error("An error occurred while attempting to fetch Observatorium data from Prometheus", err.Error())
		return
	}
	if len(kafkaState.State) > 0 {
		glog.Infof("kafka state is %s ", kafkaState.State)
	} else {
		glog.Infof("kafka state not found for paramerters %s %s ", name, namespace)
	}

}
