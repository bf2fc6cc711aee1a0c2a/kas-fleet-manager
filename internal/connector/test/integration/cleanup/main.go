package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/providers/connector"
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
	kcClient := keycloak.NewClient(keycloakConfig, keycloakConfig.DinosaurRealm)
	accessToken, _ := kcClient.GetToken()

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
		last += 100
		if len(clients) == 0 {
			thereAreMoreClients = false
		}
		for _, client := range clients {
			attributes := *client.Attributes
			if len(attributes) > 0 && (attributes["expire_date"] != "") {
				fmt.Println("Found client with id:", *client.ID, "and clientID:", *client.ClientID, "and expire_date:", attributes["expire_date"])
				expirationTime, parseErr := time.Parse(time.RFC3339, attributes["expire_date"])
				if parseErr != nil {
					fmt.Println("    Skipping client with id:", *client.ID, "and clientID:", *client.ClientID, "since its expiration time", attributes["expire_date"], "did not time.Parse correctly in time.RFC3339 format:", parseErr.Error())
				}
				if time.Now().Local().After(expirationTime) {
					fmt.Println("    Deleting client with id:", *client.ID, "and clientID:", *client.ClientID, "since it expired at", attributes["expire_date"])
					deleteErr := kcClient.DeleteClient(*client.ID, accessToken)
					if deleteErr != nil {
						panic(deleteErr)
					}
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
