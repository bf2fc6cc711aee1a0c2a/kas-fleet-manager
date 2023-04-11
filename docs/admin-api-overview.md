# Admin API overview

Fleet Manager provides a set of administrative endpoints which are a set of
privileged endpoints only intended to be used by administrator of the Fleet
Manager. The reason for that is that these endpoints allow
performing potentially destructive actions, actions that should only be
performed by Fleet Manager administrators, accessing sensitive information
and accessing cross-tenant information. This set of endpoints are also often
referred as Admin API endpoints.

Authentication and Authorization is performed for the requests made against
Admin API endpoints. OIDC is the protocol used to perform Authentication
and Authorization. Therefore, when making a request to the Admin API endpoints,
an OIDC token has to be provided with such request.

The available Fleet Manager API endpoints are documented in:
* [KAS Fleet Manager Admin API OpenAPI specification file](../openapi/kas-fleet-manager-private-admin.yaml)
* [Connector Fleet Manager Admin API OpenAPI specifications file](../openapi/connector_mgmt-private-admin.yaml)

## Authentication

To be able to perform Authentication in the Admin API endpoints, the location of
the OIDC authorization server that issues the OIDC tokens used to call
the Fleet Manager Admin API endpoints has to be provided. This is configured
through the following Fleet Manager CLI flags:
* `--admin-api-sso-base-url`: The Base URL of the OIDC authorization server
  endpoint. By default set to `https://auth.redhat.com`
* `--admin-api-sso-endpoint-uri`: The URL path of the OIDC realm in the
  authorization server specified in `--admin-api-sso-base-url`. By default
  set to `/auth/realms/EmployeeIDP`
* `--admin-api-sso-realm`: The name of the OIDC realm to be used in the
  OIDC authorization server endpoint specified in `admin-api-sso-base-url`. By
  default set to `EmployeeIDP`. The value specified here has to match the realm
  name specified as part of the URL path specified in
  `--admin-api-sso-endpoint-uri`

Specifically, Fleet Manager performs the following as part of the
Authentication process of the Admin API endpoint calls:
* The JWT token signature is verified by calling the JWKs endpoint of
  the provided OIDC authorization server. It expects the location
  of the JWKs endpoint in the following URL:
  `<admin-api-sso-base-url>/auth/realms/<admin-api-sso-realm>/protocol/openid-connect/certs`
* The JWT token claim `issuer` is checked to ensure it has been issued by
  the provided OIDC authorization server. It expects the value of
  the JWT token claim `issuer` to be:
  `<admin-api-sso-base-url>/auth/realms/<admin-api-sso-realm>`

The OIDC server used to issue OIDC tokens for the Fleet Manager Admin API
endpoints can be any OIDC server implementation.

## Authorization

The different Admin API endpoints authorization is performed through
role based access control (RBAC).

RBAC roles are mapped to different Admin API endpoint HTTP methods.

The role names that the Fleet Manager will consider and their mappings to
specific Admin API endpoint HTTP methods are defined and can be configured
through the
[admin api authorization configuration file](../config/admin-authz-configuration.yaml).
See its contents for details on it.

When running KAS Fleet Manager, the `--admin-authz-config-file` CLI flag can be
provided to point to an admin api authorization configuration file. By default if
the flag is not provided Fleet Manager will look for it in
`config/admin-authz-configuration.yaml`.

The defined RBAC roles in the Fleet Manager are matched against the JWT
claims of the received OIDC token as part of the request.
Specifically, when a request is performed to a Fleet Manager admin API
endpoint, Fleet Manager checks if any of the role names defined in the
admin api authorization configuration file for the HTTP method being used
matches with any element in the `.realm_access.roles` array, part
of the JWT claims in the JWT token. If there is no match
the request is unauthorized.

The configured OIDC server (see [authentication](#authentication))
has to be configured in a way that the issued OIDC tokens
contain the set of desired RBAC roles as part of its JWT claims.

Specifically, the RBAC roles have to be specified
as a JWT token claim located in a `.realm_access.roles` array. If that JWT
claim does not exist Fleet Manager assumes no RBAC roles are provided and the
request will be unauthorized.

## Example configuration scenario

An example that shows a configuration scenario where Keycloak is used as
the OIDC server that issues the OIDC tokens used to perform requests to the
Fleet Manager Admin API endpoints can be found
[here](https://github.com/bf2fc6cc711aee1a0c2a/architecture/blob/main/_adr/88/index.adoc).
The example also shows a set of concrete RBAC roles defined for the Fleet
Manager Admin API endpoints.
