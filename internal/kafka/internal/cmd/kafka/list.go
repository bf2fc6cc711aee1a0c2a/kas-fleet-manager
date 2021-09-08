package kafka

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/flags"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"

	"github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

const (
	FlagPage = "page"
	FlagSize = "size"
)

// NewListCommand creates a new command for listing kafkas.
func NewListCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "lists all managed kafka requests",
		Long:  "lists all managed kafka requests",
		Run: func(cmd *cobra.Command, args []string) {
			runList(env, cmd, args)
		},
	}
	cmd.Flags().String(FlagOwner, "test-user", "Username")
	cmd.Flags().String(FlagPage, "1", "Page index")
	cmd.Flags().String(FlagSize, "100", "Number of kafka requests per page")

	return cmd
}

func runList(env *environments.Env, cmd *cobra.Command, _ []string) {
	owner := flags.MustGetDefinedString(FlagOwner, cmd.Flags())
	page := flags.MustGetString(FlagPage, cmd.Flags())
	size := flags.MustGetString(FlagSize, cmd.Flags())
	var kafkaService services.KafkaService
	env.MustResolveAll(&kafkaService)

	// create jwt with claims and set it in the context
	jwt := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"username": owner,
	})
	ctx := auth.SetTokenInContext(context.TODO(), jwt)

	// build list arguments
	url := url.URL{}
	query := url.Query()
	query.Add(FlagPage, page)
	query.Add(FlagSize, size)
	listArgs := coreServices.NewListArguments(query)

	kafkaList, paging, err := kafkaService.List(ctx, listArgs)
	if err != nil {
		glog.Fatalf("Unable to list kafka request: %s", err.Error())
	}

	// format output
	kafkaRequestList := public.DinosaurRequestList{
		Kind:  "KafkaRequestList",
		Page:  int32(paging.Page),
		Size:  int32(paging.Size),
		Total: int32(paging.Total),
		Items: []public.DinosaurRequest{},
	}

	for _, kafkaRequest := range kafkaList {
		converted := presenters.PresentKafkaRequest(kafkaRequest)
		kafkaRequestList.Items = append(kafkaRequestList.Items, converted)
	}

	output, marshalErr := json.MarshalIndent(kafkaRequestList, "", "    ")
	if marshalErr != nil {
		glog.Fatalf("Failed to format kafka request list: %s", err.Error())
	}

	glog.V(10).Infof("%s", output)
}
