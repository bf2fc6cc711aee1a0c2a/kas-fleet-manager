package cloudprovider

import (
	"encoding/json"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"gitlab.cee.redhat.com/service/managed-services-api/cmd/managed-services-api/environments"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/openapi"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/presenters"
	customOcm "gitlab.cee.redhat.com/service/managed-services-api/pkg/ocm"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
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

	// setup required services
	ocmClient := customOcm.NewClient(env.Clients.OCM.Connection)

	clusterService := services.NewCloudProvidersService(ocmClient)

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
