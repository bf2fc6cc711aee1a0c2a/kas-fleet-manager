package cloudprovider

import (
	"encoding/json"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"gitlab.cee.redhat.com/service/managed-services-api/cmd/managed-services-api/environments"
	"gitlab.cee.redhat.com/service/managed-services-api/cmd/managed-services-api/flags"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/openapi"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/presenters"
)

// NewRegionsListCommand creates a new command for listing regions.
func NewRegionsListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "regions",
		Short: "lists all supported cloud providers",
		Long:  "lists all supported cloud providers",
		Run:   runRegionsList,
	}
	err := environments.Environment().AddFlags(cmd.PersistentFlags())
	if err != nil {
		glog.Fatalf("Unable to add environment flags to serve command: %s", err.Error())
	}
	cmd.Flags().String(FlagID, "aws", "Cloud provider id")

	return cmd
}

func runRegionsList(cmd *cobra.Command, _ []string) {

	id := flags.MustGetDefinedString(FlagID, cmd.Flags())

	if err := environments.Environment().Initialize(); err != nil {
		glog.Fatalf("Unable to initialize environment: %s", err.Error())
	}
	env := environments.Environment()
	config := env.Services.Config

	cloudProviderService := env.Services.CloudProviders

	cloudRegions, err := cloudProviderService.ListCloudProviderRegions(id)
	if err != nil {
		glog.Fatalf("Unable to list cloud provider regions: %s", err.Error())
	}

	regionList := openapi.CloudRegionList{
		Kind:  "CloudRegionList",
		Total: int32(len(cloudRegions)),
		Size:  int32(len(cloudRegions)),
		Page:  int32(1),
	}
	for _, cloudRegion := range cloudRegions {
		cloudRegion.Enabled = config.IsRegionSupportedForProvider(cloudRegion.CloudProvider, cloudRegion.Id)
		converted := presenters.PresentCloudRegion(&cloudRegion)
		regionList.Items = append(regionList.Items, converted)
	}

	output, marshalErr := json.MarshalIndent(regionList, "", "    ")
	if marshalErr != nil {
		glog.Fatalf("Failed to format  cloud provider region list: %s", err.Error())
	}

	glog.V(10).Infof("%s", output)

}
