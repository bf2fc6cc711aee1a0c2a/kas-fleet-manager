package keycloak

import "github.com/Nerzal/gocloak/v11"

// We create a type alias to gocloak.GoCloak to be able to autogenerate
// mocks for it
//
//go:generate moq -out gocloak_client_interface_moq.go . goCloakClientInterface
type goCloakClientInterface = gocloak.GoCloak
