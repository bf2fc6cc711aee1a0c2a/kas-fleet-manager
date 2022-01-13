package vault

import (
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/services/vault"
	"github.com/golang/glog"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"os"
)

func NewListCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List the vault secrets",
		Long:  "List the vault secrets",

		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			err := env.CreateServices()
			if err != nil {
				glog.Fatalf("Unable to initialize environment: %s", err.Error())
			}
		},

		Run: func(cmd *cobra.Command, args []string) {
			env.MustInvoke(runList)
		},
	}
	return cmd
}

func runList(vaultService vault.VaultService) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Secret Key", "Owning Resource"})
	err := vaultService.ForEachSecret(func(key string, owner string) bool {
		table.Append([]string{key, owner})
		return true
	})
	if err != nil {
		fmt.Println("error:", err)
	}
	table.Render()
}
