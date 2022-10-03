# Admin API endpoints authorization
Admin API endpoints authorization is granted by membership to rover groups that represent the -read/write/full roles per environment. More information can be found [here](https://github.com/bf2fc6cc711aee1a0c2a/architecture/blob/main/_adr/88/index.adoc).

Configuration of the roles for development purposes can be found [here](../config/admin-authz-configuration.yaml). When deploying kas-fleet-manager to an OSD cluster, `ADMIN_AUTHZ_CONFIG` can be provided to control the authorization for the admin API endpoints. Then the config can be fine tuned by the following flags:

- `ADMIN_API_SSO_BASE_URL` - base url of the admin API SSO endpoint
- `ADMIN_API_SSO_ENDPOINT_URI` - admin API SSO Endpoint URI
- `ADMIN_API_SSO_REALM` - admin API SSO Realm
