package kafka

import (
	"encoding/json"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/flags"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

// NewCreateCommand creates a new command for creating kafkas.
func NewCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new kafka request",
		Long:  "Create a new kafka request.",
		Run:   runCreate,
	}
	err := environments.Environment().AddFlags(cmd.PersistentFlags())
	if err != nil {
		glog.Fatalf("Unable to add environment flags to serve command: %s", err.Error())
	}

	cmd.Flags().String(FlagName, "", "Kafka request name")
	cmd.Flags().String(FlagRegion, "us-east-1", "OCM region ID")
	cmd.Flags().String(FlagProvider, "aws", "OCM provider ID")
	cmd.Flags().String(FlagOwner, "test-user", "User name")
	cmd.Flags().String(FlagClusterID, "000", "Kafka  request cluster ID")
	cmd.Flags().Bool(FlagMultiAZ, true, "Whether Kafka request should be Multi AZ or not")
	cmd.Flags().String(FlagOrgID, "", "OCM org id")

	return cmd
}

func runCreate(cmd *cobra.Command, _ []string) {
	name := flags.MustGetDefinedString(FlagName, cmd.Flags())
	region := flags.MustGetDefinedString(FlagRegion, cmd.Flags())
	provider := flags.MustGetDefinedString(FlagProvider, cmd.Flags())
	owner := flags.MustGetDefinedString(FlagOwner, cmd.Flags())
	multiAZ := flags.MustGetBool(FlagMultiAZ, cmd.Flags())
	clusterID := flags.MustGetDefinedString(FlagClusterID, cmd.Flags())
	orgId := flags.MustGetDefinedString(FlagOrgID, cmd.Flags())

	if err := environments.Environment().Initialize(); err != nil {
		glog.Fatalf("Unable to initialize environment: %s", err.Error())
	}

	env := environments.Environment()
	kafkaService := env.Services.Kafka

	kafkaRequest := &api.KafkaRequest{
		Region:         region,
		ClusterID:      clusterID,
		CloudProvider:  provider,
		MultiAZ:        multiAZ,
		Name:           name,
		Owner:          owner,
		OrganisationId: orgId,
	}

	if err := kafkaService.RegisterKafkaJob(kafkaRequest); err != nil {
		glog.Fatalf("Unable to create kafka request: %s", err.Error())
	}
	indentedKafkaRequest, err := json.MarshalIndent(kafkaRequest, "", "    ")
	if err != nil {
		glog.Fatalf("Failed to format kafka request: %s", err.Error())
	}
	glog.V(10).Infof("%s", indentedKafkaRequest)
}
