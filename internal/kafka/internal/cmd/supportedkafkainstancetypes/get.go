package supportedkafkainstancetypes

import (
	"encoding/json"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/flags"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

func NewGetCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a list of supported Kafka instance types by cloud region",
		Long:  "Get a list of supported Kafka instance types by cloud region",
		Run: func(cmd *cobra.Command, args []string) {
			runGet(env, cmd, args)
		},
	}
	cmd.Flags().String(FlagCloudProvider, "", "cloud provider id")
	cmd.Flags().String(FlagCloudRegion, "", "cloud region name")

	return cmd
}

func runGet(env *environments.Env, cmd *cobra.Command, _ []string) {
	id := flags.MustGetDefinedString(FlagCloudProvider, cmd.Flags())
	region := flags.MustGetDefinedString(FlagCloudProvider, cmd.Flags())
	var supportedKafkaInstanceTypeService services.SupportedKafkaInstanceTypesService
	var kafkaConfig *config.KafkaConfig
	env.MustResolveAll(&supportedKafkaInstanceTypeService, &kafkaConfig)

	regionInstanceTypeList, err := supportedKafkaInstanceTypeService.GetSupportedKafkaInstanceTypesByRegion(id, region)
	if err != nil {
		glog.Fatalf("Unable to get supported Kafka instance type list: %s", err.Error())
	}

	supportedKafkaInstanceTypeList := public.SupportedKafkaInstanceTypesList{
		InstanceTypes: []public.SupportedKafkaInstanceType{},
	}

	for _, regionInstanceType := range regionInstanceTypeList {
		converted := presenters.PresentSupportedKafkaInstanceType(&regionInstanceType)
		supportedKafkaInstanceTypeList.InstanceTypes = append(supportedKafkaInstanceTypeList.InstanceTypes, converted)
	}

	output, marshalErr := json.MarshalIndent(supportedKafkaInstanceTypeList, "", "    ")
	if marshalErr != nil {
		glog.Fatalf("Failed to format supported Kafka instance type list: %s", marshalErr.Error())
	}
	glog.V(10).Infof("%s", output)
}
