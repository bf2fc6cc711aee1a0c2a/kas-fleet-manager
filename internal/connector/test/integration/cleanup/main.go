package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/providers/connector"
	"github.com/golang/glog"
	"github.com/spf13/pflag"
)

func main() {
	env, err := environments.New(environments.GetEnvironmentStrFromEnv(),
		connector.ConfigProviders(false),
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

	var keycloakConfig *keycloak.KeycloakConfig
	env.MustResolve(&keycloakConfig)
	kcClient := keycloak.NewClient(keycloakConfig, keycloakConfig.KafkaRealm)
	accessToken, _ := kcClient.GetToken()

	kcClientKafkaSre := keycloak.NewClient(keycloakConfig, keycloakConfig.OSDClusterIDPRealm)
	accessTokenKafkaSre, _ := kcClientKafkaSre.GetToken()

	if len(os.Args) > 1 {
		argsWithoutProg := os.Args[1:]
		for _, clientIdToDelete := range argsWithoutProg {
			confirmed := askForConfirmation("Do you really want to delete keycloak client with id: " + clientIdToDelete)
			if confirmed {
				fmt.Println("    Deleting client with id:", clientIdToDelete)
				deleteErr := kcClient.DeleteClient(clientIdToDelete, accessToken)
				if deleteErr != nil {
					fmt.Println("    Something went wrong:", deleteErr.Error())
				}
			} else {
				fmt.Println("    Aborting deletion of keycloak client with id:", clientIdToDelete, "since it was not confirmed.")
			}
			fmt.Println("")
		}
	}

	fmt.Println("Starting to search for expired keycloak test clients to be removed,")
	fmt.Println("it might take a while ...")
	last := 0
	thereAreMoreClients := true
	for thereAreMoreClients {
		clients, err := kcClient.GetClients(accessToken, last, 100, "")
		if err != nil {
			panic(err)
		}
		clientsKafkaSre, err := kcClientKafkaSre.GetClients(accessTokenKafkaSre, last, 100, "")
		if err != nil {
			panic(err)
		}
		last += 100
		if len(clients) == 0 {
			thereAreMoreClients = false
		}
		allClients := append(clients, clientsKafkaSre...)
		for _, client := range allClients {
			attributes := *client.Attributes
			if len(attributes) > 0 && strings.HasPrefix(*client.ClientID, "kas-fleetshard-agent-") || strings.HasPrefix(*client.ClientID, "srvc-acct-") {
				fmt.Println("Found client with id:", *client.ID, "and clientID:", *client.ClientID)
				fmt.Println("    Deleting client with id:", *client.ID, "and clientID:", *client.ClientID)
				deleteErr := kcClient.DeleteClient(*client.ID, accessToken)
				if deleteErr != nil {
					panic(deleteErr)
				}
			} else if strings.HasPrefix(*client.ClientID, "c8") {
				deleteKafkaSreErr := kcClientKafkaSre.DeleteClient(*client.ID, accessTokenKafkaSre)
				if deleteKafkaSreErr != nil {
					panic(deleteKafkaSreErr)
				}
			}
		}
	}
}

func askForConfirmation(s string) bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s [y/n]: ", s)

		response, err := reader.ReadString('\n')
		if err != nil {
			panic(err)
		}

		response = strings.ToLower(strings.TrimSpace(response))

		if response == "y" || response == "yes" {
			return true
		} else if response == "n" || response == "no" {
			return false
		}
	}
}
