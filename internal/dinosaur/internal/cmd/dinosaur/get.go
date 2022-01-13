package dinosaur

import (
	"encoding/json"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/flags"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

// NewGetCommand gets a new command for getting dinosaurs.
func NewGetCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a dinosaur request",
		Long:  "Get a dinosaur request.",
		Run: func(cmd *cobra.Command, args []string) {
			runGet(env, cmd, args)
		},
	}
	cmd.Flags().String(FlagID, "", "Dinosaur id")

	return cmd
}

func runGet(env *environments.Env, cmd *cobra.Command, _ []string) {
	id := flags.MustGetDefinedString(FlagID, cmd.Flags())
	var dinosaurService services.DinosaurService
	env.MustResolveAll(&dinosaurService)

	dinosaurRequest, err := dinosaurService.GetById(id)
	if err != nil {
		glog.Fatalf("Unable to get dinosaur request: %s", err.Error())
	}
	indentedDinosaurRequest, marshalErr := json.MarshalIndent(dinosaurRequest, "", "    ")
	if marshalErr != nil {
		glog.Fatalf("Failed to format dinosaur request: %s", marshalErr.Error())
	}
	glog.V(10).Infof("%s", indentedDinosaurRequest)
}
