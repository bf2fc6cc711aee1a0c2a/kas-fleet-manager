package serviceaccounts

import (
	"context"
	"encoding/json"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"gitlab.cee.redhat.com/service/managed-services-api/cmd/managed-services-api/environments"
	"gitlab.cee.redhat.com/service/managed-services-api/cmd/managed-services-api/flags"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/openapi"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/presenters"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
	"strconv"
)

func NewListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "lists all service accounts",
		Long:  "lists all service accounts",
		Run:   runList,
	}
	err := environments.Environment().AddFlags(cmd.PersistentFlags())
	if err != nil {
		glog.Fatalf("Unable to add environment flags to serve command: %s", err.Error())
	}
	cmd.Flags().String(FlagFirst, "", "Service Account first result")
	cmd.Flags().String(FlagMax, "", "Service Account max result")
	return cmd
}

func runList(cmd *cobra.Command, args []string) {
	first, err := strconv.Atoi(flags.MustGetDefinedString(FlagFirst, cmd.Flags()))
	if err != nil {
		glog.Fatalf("Unable to read flag first: %s", err.Error())
	}
	max, err := strconv.Atoi(flags.MustGetDefinedString(FlagMax, cmd.Flags()))
	if err != nil {
		glog.Fatalf("Unable to read flag first: %s", err.Error())
	}
	if err := environments.Environment().Initialize(); err != nil {
		glog.Fatalf("Unable to initialize environment: %s", err.Error())
	}

	env := environments.Environment()

	// setup required services
	keycloakService := services.NewKeycloakService(env.Config.Keycloak)
	ctx := context.TODO()
	sa, err := keycloakService.ListServiceAcc(ctx, first, max)
	if err != nil {
		glog.Fatalf("Unable to list service account list: %s", err.Error())
	}
	serviceAccountList := openapi.ServiceAccountList{
		Kind:  "ServiceAccountList",
		Items: []openapi.ServiceAccountListItem{},
	}

	for _, account := range sa {
		converted := presenters.PresentServiceAccountListItem(&account)
		serviceAccountList.Items = append(serviceAccountList.Items, converted)
	}

	output, marshalErr := json.MarshalIndent(serviceAccountList, "", "    ")
	if marshalErr != nil {
		glog.Fatalf("Failed to format service account list: %s", err.Error())
	}

	glog.V(10).Infof("%s", output)

}
