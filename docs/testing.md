# Testing

## Running unit tests

To run unit tests run the following command:
```
make test
```

## Running integration tests

Integration tests can be executed against a real or "emulated" OCM environment.
Executing against an emulated environment can be useful to get fast feedback
as OpenShift clusters will not actually be provisioned, reducing testing
time greatly.

Both scenarios require a database and OCM token to be setup before running
integration tests, run:
```
make db/setup
make ocm/setup OCM_OFFLINE_TOKEN=<ocm-offline-token> OCM_ENV=development
```

To run a local keycloak container and setup realm configuration:
```
make sso/setup
make sso/config
make keycloak/setup MAS_SSO_CLIENT_ID=kas-fleet-manager MAS_SSO_CLIENT_SECRET=kas-fleet-manager OSD_IDP_MAS_SSO_CLIENT_ID=kas-fleet-manager OSD_IDP_MAS_SSO_CLIENT_SECRET=kas-fleet-manager
```

To run integration tests with an "emulated" OCM environment, run:
```
OCM_ENV=integration make test/integration
```

To run integration tests with a real OCM environment, run:

```
make test/integration
```

>NOTE: Make sure that the keycloak service that's running locally is exposed via
       the internet using a reverse proxy service like [ngrok](https://ngrok.com/).
```
ngrok http 8180
....wait for ngrok to run, then copy the generated URL and use them as mas-sso base url in "internal/kafka/internal/environments/development.go" file.
```

To stop and remove the database container when finished, run:
```
make db/teardown
```

To stop and remove the keycloak container when finished, run:
```
make sso/teardown
```

## Best practices

Testing best practices for KAS Fleet Manager can be found in the [following document](./best-practices/testing.md)