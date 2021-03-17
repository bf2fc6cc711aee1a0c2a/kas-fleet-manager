package main

import (
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/spf13/pflag"
)

func main() {
	env := environments.Environment()

	err := env.AddFlags(pflag.CommandLine)
	if err != nil {
		panic(err)
	}

	err = env.Initialize()
	if err != nil {
		panic(err)
	}

	kcClient := keycloak.NewClient(env.Config.Keycloak, env.Config.Keycloak.KafkaRealm)
	accessToken, _ := kcClient.GetToken()

	last := 0
	for {
		clients, err := kcClient.GetClients(accessToken, last, 1000)
		if err != nil {
			panic(err)
		}
		last += 1000
		if len(clients) == 0 {
			break
		}
		for _, client := range clients {
			attributes := client.Attributes
			att := *attributes
			if len(att) > 0 {
				if att["connector-fleetshard-operator-cluster-id"] != "" {
					fmt.Println("deleting", *client.ID, "=", att)
					derr := kcClient.DeleteClient(*client.ID, accessToken)
					if derr != nil {
						panic(derr)
					}
				}
			}
		}
	}

}
