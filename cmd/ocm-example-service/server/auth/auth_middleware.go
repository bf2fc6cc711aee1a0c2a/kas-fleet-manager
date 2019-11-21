package auth

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"time"

	jwtmiddleware "github.com/auth0/go-jwt-middleware"
	"github.com/dgrijalva/jwt-go"
	"github.com/getsentry/sentry-go"

	"gitlab.cee.redhat.com/service/ocm-example-service/pkg/errors"
)

type JWTMiddleware interface {
	AuthenticateAccountJWT(next http.Handler) http.Handler
}

type AuthMiddleware struct {
	CertURL string
	CertCA  string
	keyMap  map[string]*rsa.PublicKey
	jwtmw   *jwtmiddleware.JWTMiddleware
}

var _ JWTMiddleware = &AuthMiddleware{}

func NewAuthMiddleware(certURL string, certCA string) (*AuthMiddleware, error) {
	if certURL == "" {
		return nil, fmt.Errorf("JWKS certificate URL must be provided")
	}

	middleware := AuthMiddleware{
		CertURL: certURL,
		CertCA:  certCA,
	}

	// Add the JWT Middleware
	middleware.jwtmw = jwtmiddleware.New(jwtmiddleware.Options{
		ValidationKeyGetter: middleware.getValidationToken,
		SigningMethod:       jwt.SigningMethodRS256,
		UserProperty:        util.ContextAuthKey,
		// TODO once we have better debugging tools, we should optionally enable this
		Debug: false,
	})

	err := middleware.populateKeyMap()
	return &middleware, err
}

// Middleware handler to validate JWT tokens and authenticate users
func (a *AuthMiddleware) AuthenticateAccountJWT(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		err := a.jwtmw.CheckJWT(w, r)
		if err != nil {
			handleError(ctx, w, errors.ErrorUnauthorized, fmt.Sprintf("Unable to verify JWT token: %s", err))
			return
		}
		// Update the context, as the jwt middleware will update it
		ctx = r.Context()
		payload, err := util.GetAuthPayload(r)
		if err != nil {
			handleError(ctx, w, errors.ErrorUnauthorized, fmt.Sprintf("Unable to get payload details from JWT token: %s", err))
			return
		}

		// Append the username to the request context
		ctx = SetUsernameContext(ctx, payload.Username)
		*r = *r.WithContext(ctx)

		// Add username to sentry context
		if hub := sentry.GetHubFromContext(ctx); hub != nil {
			hub.ConfigureScope(func(scope *sentry.Scope) {
				scope.SetUser(sentry.User{ID: username})
			})
		}

		next.ServeHTTP(w, r)
	})
}

func (a *AuthMiddleware) populateKeyMap() error {
	// Load the trusted CA certificates:
	trustedCAs, err := x509.SystemCertPool()
	if err != nil {
		return fmt.Errorf("can't load system trusted CAs: %v", err)
	}

	if a.CertCA != "" {
		trustedCAs.AppendCertsFromPEM([]byte(a.CertCA))
	}

	// Try to read the JWT public key object file.
	a.keyMap, err = downloadPublicKeys(a.CertURL, trustedCAs)
	return err
}
