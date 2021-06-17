package main

import (
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/golang/glog"
	"github.com/spf13/pflag"
)

func main() {
	env, err := environments.NewEnv(environments.GetEnvironmentStrFromEnv(),
		kafka.ConfigProviders().AsOption(),
	)
	if err != nil {
		glog.Fatalf("error initializing: %v", err)
	}

	err = env.AddFlags(pflag.CommandLine)
	if err != nil {
		panic(err)
	}

	err = env.CreateServices()
	if err != nil {
		panic(err)
	}

	kcClient := keycloak.NewClient(env.Config.Keycloak, env.Config.Keycloak.KafkaRealm)
	accessToken, _ := kcClient.GetToken()

	last := 0
	for {
		clients, err := kcClient.GetClients(accessToken, last, 100, "")
		if err != nil {
			panic(err)
		}
		last += 100
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
