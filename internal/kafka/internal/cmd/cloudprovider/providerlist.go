package cloudprovider

import (
	"encoding/json"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	presenters2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
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
			env.MustInvoke(runProviderList)
		},
	}
	return cmd
}

func runProviderList(
	config coreServices.ConfigService,
	cloudProviderService services.CloudProvidersService,
) {

	cloudProviders, err := cloudProviderService.ListCloudProviders()
	if err != nil {
		glog.Fatalf("Unable to list cloud providers: %s", err.Error())
	}
	cloudProviderList := public.CloudProviderList{
		Kind:  "CloudProviderList",
		Total: int32(len(cloudProviders)),
		Size:  int32(len(cloudProviders)),
		Page:  int32(1),
	}

	for _, cloudProvider := range cloudProviders {
		cloudProvider.Enabled = config.IsProviderSupported(cloudProvider.Id)
		converted := presenters2.PresentCloudProvider(&cloudProvider)
		cloudProviderList.Items = append(cloudProviderList.Items, converted)
	}

	output, marshalErr := json.MarshalIndent(cloudProviderList, "", "    ")
	if marshalErr != nil {
		glog.Fatalf("Failed to format cloud provider list: %s", err.Error())
	}

	glog.V(10).Infof("%s", output)
}
