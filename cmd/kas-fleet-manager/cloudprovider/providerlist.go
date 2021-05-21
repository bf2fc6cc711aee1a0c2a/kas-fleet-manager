package cloudprovider

import (
	"encoding/json"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/presenters"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

// NewProviderListCommand a new command for listing providers.
func NewProviderListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "providers",
		Short: "lists all supported cloud providers",
		Long:  "lists all supported cloud providers",
		Run:   runProviderList,
	}
	err := environments.Environment().AddFlags(cmd.PersistentFlags())
	if err != nil {
		glog.Fatalf("Unable to add environment flags to serve command: %s", err.Error())
	}

	return cmd
}

func runProviderList(cmd *cobra.Command, _ []string) {

	if err := environments.Environment().Initialize(); err != nil {
		glog.Fatalf("Unable to initialize environment: %s", err.Error())
	}

	env := environments.Environment()
	config := env.Services.Config

	clusterService := env.Services.CloudProviders

	cloudProviders, err := clusterService.ListCloudProviders()
	if err != nil {
		glog.Fatalf("Unable to list cloud providers: %s", err.Error())
	}
	cloudProviderList := openapi.CloudProviderList{
		Kind:  "CloudProviderList",
		Total: int32(len(cloudProviders)),
		Size:  int32(len(cloudProviders)),
		Page:  int32(1),
	}

	for _, cloudProvider := range cloudProviders {
		cloudProvider.Enabled = config.IsProviderSupported(cloudProvider.Id)
		converted := presenters.PresentCloudProvider(&cloudProvider)
		cloudProviderList.Items = append(cloudProviderList.Items, converted)
	}

	output, marshalErr := json.MarshalIndent(cloudProviderList, "", "    ")
	if marshalErr != nil {
		glog.Fatalf("Failed to format cloud provider list: %s", err.Error())
	}

	glog.V(10).Infof("%s", output)
}
