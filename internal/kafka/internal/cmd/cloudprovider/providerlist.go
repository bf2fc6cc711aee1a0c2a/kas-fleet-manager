package cloudprovider

import (
	"encoding/json"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

// NewProviderListCommand a new command for listing providers.
func NewProviderListCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "providers",
		Short: "lists all supported cloud providers",
		Long:  "lists all supported cloud providers",
		Run: func(cmd *cobra.Command, args []string) {
			runProviderList(env, cmd, args)
		},
	}
	return cmd
}

func runProviderList(env *environments.Env, cmd *cobra.Command, _ []string) {

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
