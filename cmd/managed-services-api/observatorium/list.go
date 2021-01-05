package observatorium

import (
	"context"
	"encoding/json"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"gitlab.cee.redhat.com/service/managed-services-api/cmd/managed-services-api/environments"
	"gitlab.cee.redhat.com/service/managed-services-api/cmd/managed-services-api/flags"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/openapi"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/presenters"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/auth"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/client/observatorium"
)

func NewRunListMetricsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list kafka users metrics from Prometheus",
		Run:   runListMetrics,
	}
	cmd.Flags().String(FlagID, "", "Kafka id")
	cmd.Flags().String(FlagOwner, "", "Username")

	return cmd
}
func runListMetrics(cmd *cobra.Command, _args []string) {
	id := flags.MustGetDefinedString(FlagID, cmd.Flags())
	owner := flags.MustGetDefinedString(FlagOwner, cmd.Flags())

	if err := environments.Environment().Initialize(); err != nil {
		glog.Fatalf("Unable to initialize environment: %s", err.Error())
	}

	env := environments.Environment()
	kafkaMetrics := &observatorium.KafkaMetrics{}
	ctx := auth.SetUsernameContext(context.TODO(), owner)
	q := observatorium.RangeQuery{}
	q.FillDefaults()

	kafkaId, err := env.Services.Observatorium.GetMetricsByKafkaId(ctx, kafkaMetrics, id, q)
	if err != nil {
		glog.Error("An error occurred while attempting to get metrics data ", err.Error())
		return
	}
	metricsList := openapi.MetricsList{
		Kind: "Metrics",
		Id:   kafkaId,
	}
	metrics, err := presenters.PresentMetrics(kafkaMetrics)
	if err != nil {
		glog.Error("An error occurred while attempting to present metrics data ", err.Error())
		return
	}
	metricsList.Items = metrics
	output, marshalErr := json.MarshalIndent(metricsList, "", "    ")
	if marshalErr != nil {
		glog.Fatalf("Failed to format metrics list: %s", err.Error())
	}

	glog.V(10).Infof("%s %s", kafkaId, output)

}
