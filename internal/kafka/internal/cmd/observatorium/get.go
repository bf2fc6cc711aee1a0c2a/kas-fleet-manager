package observatorium

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/flags"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

func NewRunGetStateCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-state",
		Short: "Fetch kafka state metric from Prometheus",
		Run: func(cmd *cobra.Command, args []string) {
			runGethResourceStateMetrics(env, cmd, args)
		},
	}

	cmd.Flags().String(FlagName, "", "Kafka name")
	cmd.Flags().String(FlagNameSpace, "", "Kafka namepace")

	return cmd
}
func runGethResourceStateMetrics(env *environments.Env, cmd *cobra.Command, _args []string) {

	name := flags.MustGetDefinedString(FlagName, cmd.Flags())
	namespace := flags.MustGetDefinedString(FlagNameSpace, cmd.Flags())

	var observatoriumService services.ObservatoriumService
	env.MustResolveAll(&observatoriumService)

	kafkaState, err := observatoriumService.GetKafkaState(name, namespace)
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
