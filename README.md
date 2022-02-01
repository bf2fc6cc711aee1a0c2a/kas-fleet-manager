# Fleet Manager Golang Template

This project is an example fleet management service. Fleet managers govern service 
instances across a range of cloud provider infrastructure and regions. They are 
responsible for service placement, service lifecycle including blast radius aware 
upgrades,control of the operators handling each service instance, DNS management, 
infrastructure scaling and pre-flight checks such as quota entitlement, export control, 
terms acceptance and authorization. They also provide the public APIs of our platform 
for provisioning and managing service instances.


To help you while reading the code the example service implements a simple collection
of _dinosaurs_ and their provisioning, so you can immediately know when something is 
infrastructure or business logic. Anything that talks about dinosaurs is business logic, 
which you will want to replace with your our concepts. The rest is infrastructure, and you
will probably want to preserve without change.

For a real service written using the same fleet management pattern see the
[kas-fleet-manager](https://github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager).

To contact the people that created this template go to [zulip](https://bf2.zulipchat.com/).

## Prerequisites
* [Golang 1.16+](https://golang.org/dl/)
* [Docker](https://docs.docker.com/get-docker/) - to create database
* [ocm cli](https://github.com/openshift-online/ocm-cli/releases) - ocm command line tool
* [Node.js v12.20+](https://nodejs.org/en/download/) and [npm](https://docs.npmjs.com/downloading-and-installing-node-js-and-npm)

## Using the template for the first time
The [implementation](./docs/implementation.md) talks about the main components of this template. 
To bootstrap your application, after cloning the repository. 

1. Replace _dinosaurs_ placeholder with your own business entity / objects
2. Implement code that have TODO comments
   ```go
   // TODO
   ```

## Run for the first time
Please make sure you have followed all of the prerequisites above first and the [populating configuration guide](docs/populating-configuration.md).

1. Compile the binary
```
make binary
```
2. Clean up and Creating the database
    - Create database tables
    ```
    make db/setup && make db/migrate
    ```
    - Optional - Verify tables and records are created
    ```
    make db/login
    ```
    ```
    # List all the tables
    serviceapitests# \dt
    
    ```

3. Start the service
    ```
    ./fleet-manager serve
    ```
    >**NOTE**: The service has numerous feature flags which can be used to enable/disable certain features 
    of the service. Please see the [feature flag](./docs/feature-flags.md) documentation for more information.
4. Verify the local service is working
    ```
    curl -H "Authorization: Bearer $(ocm token)" http://localhost:8000/api/dinosaurs_mgmt/v1/dinosaurs
   {"kind":"DinosaurRequestList","page":1,"size":0,"total":0,"items":[]}
    ```
   >NOTE: Change _dinosaur_ to your own rest resource

## Using the Service

### View the API docs

```
# Start Swagger UI container
make run/docs

# Launch Swagger UI and Verify from a browser: http://localhost:8082

# Remove Swagger UI conainer
make run/docs/teardown
```
## Additional CLI commands

In addition to the REST API exposed via `make run`, there are additional commands to interact directly
with the service (i.e. cluster creation/scaling, Dinosaur creation, Errors list, etc.) without having to use a 
REST API client.

To use these commands, run `make binary` to create the `./fleet-manager` CLI.

Run `./fleet-manager -h` for information on the additional commands.

## Environments

The service can be run in a number of different environments. Environments are essentially bespoke
sets of configuration that the service uses to make it function differently. Environments can be
set using the `OCM_ENV` environment variable. Below are the list of known environments and their
details.

- `development` - The `staging` OCM environment is used. Sentry is disabled. Debugging utilities
   are enabled. This should be used in local development.
- `testing` - The OCM API is mocked/stubbed out, meaning network calls to OCM will fail. The auth
   service is mocked. This should be used for unit testing.
- `integration` - Identical to `testing` but using an emulated OCM API server to respond to OCM API
   calls, instead of a basic mock. This can be used for integration testing to mock OCM behaviour.
- `production` - Debugging utilities are disabled, Sentry is enabled. environment can be ignored in
   most development and is only used when the service is deployed.

## Additional docs
- [Adding new endpoint](docs/adding-a-new-endpoint.md)
- [Adding new CLI flag](docs/adding-new-flags.md)
- [Automated testing](docs/automated-testing.md)
- [Deploying fleet manager via Service Delivery](docs/onboarding-with-service-delivery.md)
- [Requesting credentials and accounts](docs/getting-credentials-and-accounts.md)
- [Data Plane Setup](docs/data-plane-osd-cluster-options.md)
- [Access Control](docs/access-control.md)
- [Quota Management](docs/quota-management-list-configuration.md)
- [Running the Service on an OpenShift cluster](./docs/deploying-fleet-manager-to-openshift.md)
- [Explanation of JWT token claims used across the fleet-manager](docs/jwt-claims.md)

## Contributing
See the [contributing guide](CONTRIBUTING.md) for general guidelines on how to contribute back to the template.