# Interacting with Fleet Manager

A user can interact with the Fleet Manager in two main ways:
* [Directly interacting with the API server](#interacting-directly-with-the-api)
* [Using the RHOAS CLI](#interacting-using-the-rhoas-cli)

## Interacting directly with the API

The Fleet Manager API requires OIDC Bearer tokens, which use the JWT format,
for authentication. The token is passed as an `Authorization` HTTP header value.

To be able to perform requests to the Fleet Manager API, you first need
to get a JWT token from the configured OIDC authentication server, which is
Red Hat SSO (sso.redhat.com) by default.
Assuming the default OIDC authentication server is being used, this can be
performed by interacting with the OCM API.
This can be easily done interacting with it through the
[ocm cli](https://github.com/openshift-online/ocm-cli/releases)
and retrieve OCM tokens using it. To do so:
1. Login to your desired OCM environment via web console using your Red Hat
   account credentials. For example, for the OCM production environment, go to
   https://console.redhat.com and login.
1. Get your OCM offline token by going to https://console.redhat.com/openshift/token
1. Login to your desired OCM environment through the OCM CLI by providing the
   OCM offline token and environment information:
   ```
   ocm login --token <ocm-offline-token> --url <ocm-api-url>
   ```
   `<ocm-api-url>` is the URL of the OCM API. Some shorthands can also
   be provided like `production` or `staging`
1. Generate an OCM token by running: `ocm token`. The generated token is the token
   that should be used to perform a request to the Fleet Manager API. For example:
  ```
  curl -H "Authorization: Bearer <result-of-ocm-token-command>" http://127.0.0.1:/8000/api/kafkas_mgmt
  ```
  OCM tokens other than the OCM offline token have an expiration time so a
  new one will need to be generated when that happens

There are some additional steps needed if you want to be able to perform
certain actions that have additional requirements. See the
[_User Account & Organization Setup_](getting-credentials-and-accounts.md#user-account--organization-setup) for more information

### Creating a Kafka Request

The following example shows how to create a Kafka Request in AWS:
```
curl -v -XPOST -H "Authorization: Bearer $(ocm token)" http://localhost:8000/api/kafkas_mgmt/v1/kafkas?async=true -d '{ "region": "us-east-1", "cloud_provider": "aws",  "name": "test-kafka", "multi_az":true}'
```

#### Listing a Kafka Request

The following example shows how to List a Kafka Request:
```
curl -v -XGET -H "Authorization: Bearer $(ocm token)" http://localhost:8000/api/kafkas_mgmt/v1/kafkas/<kafka_request_id> | jq
```

### Listing all Kafka Requests
The following example shows how to List all Kafka Requests:
```
curl -v -XGET -H "Authorization: Bearer $(ocm token)" http://localhost:8000/api/kafkas_mgmt/v1/kafkas | jq
```

### Deleting a Kafka Request
The following example shows how to delete a Kafka Request
```
curl -v -X DELETE -H "Authorization: Bearer $(ocm token)" http://localhost:8000/api/kafkas_mgmt/v1/kafkas/<kafka_request_id>?async=true
```

### Using the Kafka Admin Server API

The Kafka Admin Server API is used for managing topics, acls, and consumer groups
on a given Kafka Request.

The Kafka Admin Server OpenAPI specification can be found
[here](https://github.com/bf2fc6cc711aee1a0c2a/kafka-admin-api/blob/main/kafka-admin/.openapi/kafka-admin-rest.yaml).

To get the Kakfa Admin Server API endpoint for a given Kafka instance: after the
Kafka instance is in ready state call the GET kafka Instance endpoint against
the Fleet Manager API.

Assuming the Fleet Manager is running in a local process on port 8000 that
can be done by executing:
```
curl -H "Authorization: Bearer $(ocm token)" http://localhost:8000/api/kafkas_mgmt/v1/kafkas/<kafka_id> | jq .admin_api_server_url
```
## Interacting using the RHOAS CLI

An alternative to interact with the Fleet Manager API is to make use of the
[RHOAS CLI](https://github.com/redhat-developer/app-services-cli).

First [run the Fleet Manager locally](../README.md#running-fleet-manager-for-the-first-time-in-your-local-environment)

Then, run the RHOAS CLI pointing to the locally running Fleet Manager:
```
./rhoas login --auth-url=https://sso.stage.redhat.com/auth/realms/redhat-external --api-gateway=http://localhost:8000
```

Various kafka specific operations can be performed through the RHOAS CLI
as described [here](https://github.com/redhat-developer/app-services-cli/blob/main/docs/commands/rhoas_kafka.md)
