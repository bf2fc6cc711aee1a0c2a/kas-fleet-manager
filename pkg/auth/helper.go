package auth

import (
	"crypto/rsa"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/shared"

	"github.com/golang-jwt/jwt/v4"
	amv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
)

const (
	defaultOcmTokenIssuer = "https://sso.redhat.com/auth/realms/redhat-external"
	tokenClaimType        = "Bearer"
	TokenExpMin           = 30
	JwkKID                = "kastestkey"
)

type AuthHelper struct {
	JWTPrivateKey  *rsa.PrivateKey
	JWTCA          *rsa.PublicKey
	ocmTokenIssuer string
}

// Creates an auth helper to be used for creating new accounts and jwt.
func NewAuthHelper(jwtKeyFilePath, jwtCAFilePath, ocmTokenIssuer string) (*AuthHelper, error) {
	jwtKey, jwtCA, err := ParseJWTKeys(jwtKeyFilePath, jwtCAFilePath)
	if err != nil {
		return nil, err
	}

	ocmTokenIss := ocmTokenIssuer
	if ocmTokenIssuer == "" {
		ocmTokenIss = defaultOcmTokenIssuer
	}

	return &AuthHelper{
		JWTPrivateKey:  jwtKey,
		JWTCA:          jwtCA,
		ocmTokenIssuer: ocmTokenIss,
	}, nil
}

// Creates a new account with the specified values
func (authHelper *AuthHelper) NewAccount(username, name, email string, orgId string) (*amv1.Account, error) {
	var firstName string
	var lastName string
	names := strings.SplitN(name, " ", 2)
	if len(names) < 2 {
		firstName = name
		lastName = ""
	} else {
		firstName = names[0]
		lastName = names[1]
	}

	builder := amv1.NewAccount().
		ID(uuid.New().String()).
		Username(username).
		FirstName(firstName).
		LastName(lastName).
		Email(email).
		Organization(amv1.NewOrganization().ExternalID(orgId))

	acct, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("Unable to build account: %s", err.Error())
	}
	return acct, nil
}

// Creates a signed token. By default, this will create a signed ocm token if the issuer was not specified in the given claims.
func (authHelper *AuthHelper) CreateSignedJWT(account *amv1.Account, jwtClaims jwt.MapClaims) (string, error) {
	token, err := authHelper.CreateJWTWithClaims(account, jwtClaims)
	if err != nil {
		return "", err
	}

	// private key and public key taken from http://kjur.github.io/jsjws/tool_jwt.html
	// the go-jwt-middleware pkg we use does the same for their tests
	return token.SignedString(authHelper.JWTPrivateKey)
}

// Creates a JSON web token with the claims specified. By default, this will create an ocm JWT if the issuer was not specified in the given claims.
// Any given claim with nil value will be removed from the claims
func (authHelper *AuthHelper) CreateJWTWithClaims(account *amv1.Account, jwtClaims jwt.MapClaims) (*jwt.Token, error) {
	claims := jwt.MapClaims{
		"typ": tokenClaimType,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Minute * time.Duration(TokenExpMin)).Unix(),
	}

	if jwtClaims == nil || jwtClaims["iss"] == nil || jwtClaims["iss"] == "" || jwtClaims["iss"] == authHelper.ocmTokenIssuer {
		// Set default claim values for ocm tokens
		claims["iss"] = authHelper.ocmTokenIssuer
		claims[ocmUsernameKey] = account.Username()
		claims["first_name"] = account.FirstName()
		claims["last_name"] = account.LastName()
		claims["account_id"] = account.ID()
		claims["rh-user-id"] = account.ID()
		org, ok := account.GetOrganization()
		if ok {
			claims[ocmOrgIdKey] = org.ExternalID()
		}

		if account.Email() != "" {
			claims["email"] = account.Email()
		}
	}

	// TODO: Set default claim for sso token here.

	// Override default and add properties from the specified claims. Remove any key with nil value
	for k, v := range jwtClaims {
		if v == nil {
			delete(claims, k)
		} else {
			claims[k] = v
		}
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	// Set the token header kid to the same value we expect when validating the token
	// The kid is an arbitrary identifier for the key
	// See https://tools.ietf.org/html/rfc7517#section-4.5
	token.Header["kid"] = JwkKID
	token.Header["alg"] = jwt.SigningMethodRS256.Alg()

	return token, nil
}

func (authHelper *AuthHelper) GetJWTFromSignedToken(signedToken string) (*jwt.Token, error) {
	return jwt.Parse(signedToken, func(token *jwt.Token) (interface{}, error) {
		return authHelper.JWTCA, nil
	})
}

// Parses JWT Private and Public Keys from the given path
func ParseJWTKeys(jwtKeyFilePath, jwtCAFilePath string) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	projectRootDir := shared.GetProjectRootDir()
	privateBytes, err := ioutil.ReadFile(filepath.Join(projectRootDir, jwtKeyFilePath))
	if err != nil {
		return nil, nil, fmt.Errorf("Unable to read JWT key file %s: %s", jwtKeyFilePath, err.Error())
	}
	pubBytes, err := ioutil.ReadFile(filepath.Join(projectRootDir, jwtCAFilePath))
	if err != nil {
		return nil, nil, fmt.Errorf("Unable to read JWT ca file %s: %s", jwtCAFilePath, err.Error())
	}

	// Parse keys
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEMWithPassword(privateBytes, "passwd") //nolint
	if err != nil {
		return nil, nil, fmt.Errorf("Unable to parse JWT private key: %s", err.Error())
	}
	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(pubBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("Unable to parse JWT ca: %s", err.Error())
	}

	return privateKey, pubKey, nil
}
