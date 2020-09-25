package kafka

import (
	"encoding/json"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"gitlab.cee.redhat.com/service/managed-services-api/cmd/managed-services-api/environments"
	"gitlab.cee.redhat.com/service/managed-services-api/cmd/managed-services-api/flags"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	customOcm "gitlab.cee.redhat.com/service/managed-services-api/pkg/ocm"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
)

// NewCreateCommand creates a new command for creating kafkas.
func NewCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new managed-services-api kafka request",
		Long:  "Create a new managed-services-api kafka request.",
		Run:   runCreate,
	}
	err := environments.Environment().AddFlags(cmd.PersistentFlags())
	if err != nil {
		glog.Fatalf("Unable to add environment flags to serve command: %s", err.Error())
	}

	cmd.Flags().String(FlagName, "", "Kafka request name")
	cmd.Flags().String(FlagRegion, "eu-west-1", "OCM region ID")
	cmd.Flags().String(FlagProvider, "aws", "OCM provider ID")
	cmd.Flags().String(FlagOwner, "test-user", "User name")
	cmd.Flags().String(FlagClusterID, "000", "Kafka  request cluster ID")
	cmd.Flags().Bool(FlagMultiAZ, false, "Whether Kafka request should be Multi AZ or not")

	return cmd
}

func runCreate(cmd *cobra.Command, _ []string) {
	name := flags.MustGetDefinedString(FlagName, cmd.Flags())
	region := flags.MustGetDefinedString(FlagRegion, cmd.Flags())
	provider := flags.MustGetDefinedString(FlagProvider, cmd.Flags())
	owner := flags.MustGetDefinedString(FlagOwner, cmd.Flags())
	multiAZ := flags.MustGetBool(FlagMultiAZ, cmd.Flags())
	clusterID := flags.MustGetDefinedString(FlagClusterID, cmd.Flags())

	if err := environments.Environment().Initialize(); err != nil {
		glog.Fatalf("Unable to initialize environment: %s", err.Error())
	}

	env := environments.Environment()

	// setup required services
	ocmClient := customOcm.NewClient(env.Clients.OCM.Connection)

	clusterService := services.NewClusterService(env.DBFactory, ocmClient, env.Config.AWS)
	syncsetService := services.NewSyncsetService(env.Clients.OCM.Connection)
	kafkaService := services.NewKafkaService(env.DBFactory, syncsetService, clusterService)

	kafkaRequest := &api.KafkaRequest{
		Region:        region,
		ClusterID:     clusterID,
		CloudProvider: provider,
		MultiAZ:       multiAZ,
		Name:          name,
		Owner:         owner,
	}

	if err := kafkaService.RegisterKafkaJob(kafkaRequest); err != nil {
		glog.Fatalf("Unable to create kafka request: %s", err.Error())
	}
	indentedKafkaRequest, err := json.MarshalIndent(kafkaRequest, "", "    ")
	if err != nil {
		glog.Fatalf("Failed to format kafka request: %s", err.Error())
	}
	glog.Infof("%s", indentedKafkaRequest)
}
