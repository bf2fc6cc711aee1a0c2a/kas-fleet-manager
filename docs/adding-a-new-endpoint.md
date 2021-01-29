# Adding a New Endpoint
This document covers the steps on how to add a new endpoint to the service.

## Modify the OpenAPI spec
The OpenAPI spec needs to be modified and generated to make new schemas available to be used by the service.

Follow the [Modifying the API definition](../CONTRIBUTING.md#modifying-the-api-definition) guide on how to do this.

## Add required types for your new endpoint
> **Note**: This step is only required if your new endpoint requires new internal types to be defined.

Types used by the service are located in the `pkg/api` directory. Please add new types or modify existing types here.

## Modify or add new converters/presenters
> **Note**: This step is only required if your new endpoint requires addition or modification to the existing type converters/presenters

Converters are functions responsible for converting the incoming request using an openapi model to an internal type used by the service.

Presenters are functions responsible for converting internal types to the endpoint's response model as defined in the openapi specification.

Converters/presenters are defined in the `pkg/api/presenters` directory. Please add/modify existing converters and presenters here.

## Add a new handler
Handlers are defined in the `pkg/handlers` directory. Add a handler for your new endpoint here.

### Format
All handlers should follow a specific format as defined in this [framework](https://github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/blob/main/pkg/handlers/framework.go). See existing handlers as an example.

### Request Validation
Any request validation should be specified in the handler config's `Validate` field as seen below.

```go
cfg := &handlerConfig{
    Validate: []validate{
        ...
        validateLength(&someVar, "sampleVal", &minRequiredFieldLength, nil),
        ...
    },
    ...
}
```

Validation functions are available in [validation.go](https://github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/blob/master/pkg/handlers/validation.go). Please add any new validations in this file if required.

### Services
Any backend functionality called from your handler should be specified in `pkg/services`.

## Add your new endpoint to the API server
The [api_server.go](https://github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/blob/master/cmd/kas-fleet-manager/server/api_server.go) contains the definition of the service's endpoints. Add your new endpoint to the router and attach your handler using `HandleFunc()` here.

For example

```go
...
router := apiV1Router.PathPrefix("/your_new_endpoint").Subrouter()
router.HandleFunc("/{id}", handler.GetResource).Methods(http.MethodGet)
...
```

If your endpoint requires authentication, please add the following line as the last line of the router definition:

```go
router.Use(authMiddleware.AuthenticateAccountJWT)
```



## Add a new command to the CLI
The CLI will only be used for local development and testing. If a new endpoint was added, a new command should be added to the CLI to reflect that new endpoint.

The CLI is built using [Cobra](https://github.com/spf13/cobra) and all of the commands are located in `cmd/kas-fleet-manager`.

```
/cloudprovider - command definition for the /cloudprovider endpoint
/cluster - command definition for the /cluster endpoint
/flags - util functions for flags validation
/kafka - command definition for the /kafka endpoint
```

If your endpoint is using a new resource, a new folder should be created here with the following files:
- _cmd.go_: The main entry point for your command
- _flags.go_: Definition for common flags used across your command. 

Any subcommands (i.e. `get`, `list`, `create`) should be added as separate files.

## Unit and Integration Tests
Please see the [Contributing](../CONTRIBUTING.md) guide for more information.