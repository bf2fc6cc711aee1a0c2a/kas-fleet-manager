# Adding a new Config Module
This document covers the steps on how to add a new config module to the service.

1. Add an example configuration file in the /config folder. Ensure that the parameters are documented, see [provider-configuration.yaml](../config/provider-configuration.yaml) as example.

2. Create a new config file with a filename format of `<config>.go` (see [kafka.go](../internal/kafka/internal/config/kafka.go) as an example).
    > **NOTE**: If this configuration is an extension of an already existing configuration, please prefix the filename with the name of the config module its extending i.e. [kafka_supported_instance_types.go](../internal/kafka/internal/config/kafka_supported_instance_types.go) is an extension of the [kafka.go](../internal/kafka/internal/config/kafka.go) config. 

    These files should be created in the following places based on usage :
    -   `/pkg/...` - If it's a common configuration that can be shared across all services
    -   `/internal/kafka/internal/config` - for any Kafka specific configuration
    -   `/internal/connector/internal/config` - for any Connector specific configuration

3. The new config file has to implement the `ConfigModule` [interface](../pkg/environments/interfaces.go). This should be added into one of the following providers file:
    - `CoreConfigProviders()` inside the [core providers file](../pkg/providers/core.go): For any global configuration that is not specific to a particular service (i.e. kafka or connector).
    - `ConfigProviders()` inside [kafka providers](../internal/kafka/providers.go): For any kafka specific configuration.
    - `ConfigProviders()` inside [connector providers](../internal/connector/providers.go): For any connector specific configuration.
    > **NOTE**: If your ConfigModule also implements the ServiceValidator [interface](../pkg/environments/interfaces.go), please ensure to also specify `di.As(new(environments2.ServiceValidator))` when providing the dependency in one of the ConfigProviders listed above. Otherwise, the validation for your configuration will not be called.

4. Create/edit tests for the configuration file if needed with a filename format of `<config_test>.go` in the same directory the config file was created. 

5. Ensure the [service-template](../templates/service-template.yml) is updated. See this [pr](https://github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pull/817) as an example.
