package handlers

import (
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/server"
	"github.com/golang/glog"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/openshift-online/ocm-sdk-go/authentication"
	pkgErrors "github.com/pkg/errors"
)

func NewAuthenticationBuilder(ServerConfig *server.ServerConfig, KeycloakConfig *keycloak.KeycloakConfig) (*authentication.HandlerBuilder, error) {

	authnLogger, err := sdk.NewGlogLoggerBuilder().
		InfoV(glog.Level(1)).
		DebugV(glog.Level(5)).
		Build()

	if err != nil {
		return nil, pkgErrors.Wrap(err, "unable to create authentication logger")
	}

	return authentication.NewHandler().
			Logger(authnLogger).
			KeysURL(ServerConfig.JwksURL).                      //ocm JWK JSON web token signing certificates URL
			KeysFile(ServerConfig.JwksFile).                    //ocm JWK backup JSON web token signing certificates
			KeysURL(KeycloakConfig.KafkaRealm.JwksEndpointURI). // mas-sso JWK Cert URL
			Error(fmt.Sprint(errors.ErrorUnauthenticated)).
			Service(errors.CONNECTOR_MGMT_ERROR_CODE_PREFIX).
			Public("^/api/connector_mgmt/?$").
			Public("^/api/connector_mgmt/v1/?$").
			Public("^/api/connector_mgmt/v1/openapi/?$"),
		nil
}
