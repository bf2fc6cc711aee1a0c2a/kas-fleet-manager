### Environment-sourced variables with defaults
# Can be overriden by setting environment var before running
# Example:
#   OCM_ENV=testing make run
#   export OCM_ENV=testing; make run
MKFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
PROJECT_PATH := $(patsubst %/,%,$(dir $(MKFILE_PATH)))
DOCS_DIR := $(PROJECT_PATH)/docs
SECRETS_DIR := $(PROJECT_PATH)/secrets
OCP_TEMPLATES_DIR :=$(PROJECT_PATH)/templates

.DEFAULT_GOAL := help
SHELL = bash

# The details of the application:
binary:=kas-fleet-manager

# The version needs to be different for each deployment because otherwise the
# cluster will not pull the new image from the internal registry:
version:=$(shell date +%s)

# Default namespace for local deployments
NAMESPACE ?= kas-fleet-manager-${USER}

# The name of the image repository needs to start with the name of an existing
# namespace because when the image is pushed to the internal registry of a
# cluster it will assume that that namespace exists and will try to create a
# corresponding image stream inside that namespace. If the namespace doesn't
# exist the push fails. This doesn't apply when the image is pushed to a public
# repository, like `docker.io` or `quay.io`.
image_repository:=$(NAMESPACE)/kas-fleet-manager

# Tag for the image:
image_tag:=$(version)

# In the development environment we are pushing the image directly to the image
# registry inside the development cluster. That registry has a different name
# when it is accessed from outside the cluster and when it is acessed from
# inside the cluster. We need the external name to push the image, and the
# internal name to pull it.
external_image_registry:=default-route-openshift-image-registry.apps-crc.testing
internal_image_registry:=image-registry.openshift-image-registry.svc:5000

# Test image name that will be used for PR checks
test_image:=test/kas-fleet-manager

DOCKER ?= docker
DOCKER_CONFIG="${PWD}/.docker"

# Default Variables
ENABLE_OCM_MOCK ?= false
OCM_MOCK_MODE ?= emulate-server
JWKS_URL ?= "https://sso.redhat.com/auth/realms/redhat-external/protocol/openid-connect/certs"
MAS_SSO_BASE_URL ?="https://identity.api.stage.openshift.com"
MAS_SSO_REALM ?="rhoas"

GO := go
GOFMT := gofmt
# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell $(GO) env GOBIN))
GOBIN=$(shell $(GO) env GOPATH)/bin
else
GOBIN=$(shell $(GO) env GOBIN)
endif

export GOPROXY=https://proxy.golang.org
export GOPRIVATE=gitlab.cee.redhat.com

LOCAL_BIN_PATH := ${PROJECT_PATH}/bin
# Add the project-level bin directory into PATH. Needed in order
# for `go generate` to use project-level bin directory binaries first
export PATH := ${LOCAL_BIN_PATH}:$(PATH)

GIT ?= git

XMLLINT ?= xmllint
CURL ?= curl

OC ?= oc
OCM ?= ocm

# Set the environment to integration by default
OCM_ENV ?= integration

GOLANGCI_LINT ?= $(LOCAL_BIN_PATH)/golangci-lint
golangci-lint:
ifeq (, $(shell which $(LOCAL_BIN_PATH)/golangci-lint 2> /dev/null))
	@{ \
	set -e ;\
	VERSION="v1.52.2" ;\
	$(CURL) -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/$${VERSION}/install.sh | sh -s -- -b ${LOCAL_BIN_PATH} $${VERSION} ;\
	}
endif

GOTESTSUM ?=$(LOCAL_BIN_PATH)/gotestsum
gotestsum:
ifeq (, $(shell which $(LOCAL_BIN_PATH)/gotestsum 2> /dev/null))
	@{ \
	set -e ;\
	GOTESTSUM_TMP_DIR=$$(mktemp -d) ;\
	cd $$GOTESTSUM_TMP_DIR ;\
	$(GO) mod init tmp ;\
	$(GO) get gotest.tools/gotestsum@v1.9.0 ;\
	mkdir -p ${LOCAL_BIN_PATH} ;\
	$(GO) build -o ${LOCAL_BIN_PATH}/gotestsum gotest.tools/gotestsum ;\
	rm -rf $$GOTESTSUM_TMP_DIR ;\
	}
endif

MOQ ?= ${LOCAL_BIN_PATH}/moq
moq:
ifeq (, $(shell which ${LOCAL_BIN_PATH}/moq 2> /dev/null))
	@{ \
	set -e ;\
	MOQ_TMP_DIR=$$(mktemp -d) ;\
	cd $$MOQ_TMP_DIR ;\
	$(GO) mod init tmp ;\
	$(GO) get github.com/matryer/moq@v0.3.0 ;\
	mkdir -p ${LOCAL_BIN_PATH} ;\
	$(GO) build -o ${LOCAL_BIN_PATH}/moq github.com/matryer/moq ;\
	rm -rf $$MOQ_TMP_DIR ;\
	}
endif

OPENAPI_GENERATOR ?= ${LOCAL_BIN_PATH}/openapi-generator
NPM ?= "$(shell which npm)"
openapi-generator:
ifeq (, $(shell which ${NPM} 2> /dev/null))
	@echo "npm is not available please install it to be able to install openapi-generator"
	exit 1
endif
ifeq (, $(shell which ${LOCAL_BIN_PATH}/openapi-generator 2> /dev/null))
	@{ \
	set -e ;\
	mkdir -p ${LOCAL_BIN_PATH} ;\
	mkdir -p ${LOCAL_BIN_PATH}/openapi-generator-installation ;\
	cd ${LOCAL_BIN_PATH} ;\
	${NPM} install --prefix ${LOCAL_BIN_PATH}/openapi-generator-installation @openapitools/openapi-generator-cli@cli-4.3.1 ;\
	ln -s openapi-generator-installation/node_modules/.bin/openapi-generator openapi-generator ;\
	}
endif

SPECTRAL ?= ${LOCAL_BIN_PATH}/rhoasapi-installation/spectral
NPM ?= "$(shell which npm)"
specinstall:
ifeq (, $(shell which ${NPM} 2> /dev/null))
	@echo "npm is not available please install it to be able to install spectral"
	exit 1
endif
ifeq (, $(shell which ${LOCAL_BIN_PATH}/rhoasapi-installation/spectral 2> /dev/null))
	@{ \
    set -e ;\
    mkdir -p ${LOCAL_BIN_PATH} ;\
    mkdir -p ${LOCAL_BIN_PATH}/rhoasapi-installation ;\
	if [ -d ${LOCAL_BIN_PATH}/spectral-installation ]; then rm -r ${LOCAL_BIN_PATH}/spectral-installation; fi ;\
	if [ -L ${LOCAL_BIN_PATH}/spectral ]; then rm ${LOCAL_BIN_PATH}/spectral; fi ;\
	${NPM} i --prefix ${LOCAL_BIN_PATH}/rhoasapi-installation @stoplight/spectral-cli@6.5.0  ;\
    ${NPM} i --prefix ${LOCAL_BIN_PATH}/rhoasapi-installation @rhoas/spectral-ruleset@0.3.1 ;\
    ln -Fs ${LOCAL_BIN_PATH}/rhoasapi-installation/node_modules/.bin/spectral ${LOCAL_BIN_PATH}/rhoasapi-installation/spectral ;\
	cp .spectral.yaml ${LOCAL_BIN_PATH}/rhoasapi-installation/ ;\
	}
endif


.PHONY: openapi/spec/validate
openapi/spec/validate: specinstall
	@{ \
    mkdir -p ${LOCAL_BIN_PATH} ;\
    mkdir -p ${LOCAL_BIN_PATH}/rhoasapi-installation ;\
	cd ${LOCAL_BIN_PATH}/rhoasapi-installation ;\
	${SPECTRAL} lint ../../openapi/kas-fleet-manager.yaml ../../openapi/kas-fleet-manager-private-admin.yaml ;\
	}

# Prints a list of useful targets.
help:
	@echo "Kafka Service Fleet Manager make targets"
	@echo ""
	@echo "make verify                                             verify source code"
	@echo "make lint                                               lint go files and .yaml templates"
	@echo "make binary                                             compile binaries"
	@echo "make install                                            compile binaries and install in GOPATH bin"
	@echo "make run                                                run the application"
	@echo "make run/docs                                           run swagger and host the api spec"
	@echo "make run/docs/teardown                                  remove the swagger container"
	@echo "make test                                               run unit tests"
	@echo "make test/output/coverage/report                        generate test coverage report in the terminal"
	@echo "make test/html/coverage/report                          generate test coverage html report and see it in browser"
	@echo "make test/integration                                   run integration tests"
	@echo "make test/cluster/cleanup                               remove OSD cluster after running tests against real OCM"
	@echo "make test/run                                           Run the test container"
	@echo "make code/fix                                           format files"
	@echo "make moq/generate"                                      generate go moq files"
	@echo "make generate                                           generate go and openapi modules"
	@echo "make openapi/generate                                   generate openapi modules"
	@echo "make openapi/validate                                   validate openapi schema"
	@echo "make image/build                                        build docker image"
	@echo "make image/build/internal                               build the binary and image for Openshift deployment"
	@echo "make image/build/test                                   build the binary and test the image"
	@echo "make image/push                                         push docker image to the external image registry"
	@echo "make image/push/internal                                push the image to the Openshift internal registry"
	@echo "make setup/git/hooks                                    setup git hooks"
	@echo "make secrets/setup/empty                                setup needed secrets files for the fleet manager to be able to start"
	@echo "make keycloak/setup                                     setup mas sso clientId, clientSecret & crt"
	@echo "make gcp/setup/credentials                              setup GCP credentials"
	@echo "make kafkacert/setup                                    setup the kafka certificate used for Kafka Brokers"
	@echo "make observatorium/token-refresher/setup                setup a local observatorium token refresher"
	@echo "make docker/login/internal                              login to an openshift cluster image registry"
	@echo "make docker/login                                       login to the quay.io container registry"
	@echo "make image/build/push/internal                          build and push image to an openshift cluster image registry."
	@echo "make deploy/db                                          deploy the postgres db via templates to an openshift cluster"
	@echo "make deploy/secrets                                     deploy the secrets via templates to an openshift cluster"
	@echo "make deploy/envoy                                       deploy the envoy config via templates to an openshift cluster"
	@echo "make deploy/service                                     deploy the service via templates to an openshift cluster"
	@echo "make deploy/route                                       deploy the envoy route via templates to an openshift cluster"
	@echo "make deploy/project                                     create a project where the services will be deployed in an openshift cluster"
	@echo "make deploy/token-refresher                             deploys an Observatorium token refresher on an OpenShift cluster"
	@echo "make deploy/observability-remote-write-proxy            deploys an Observability Remote Write Proxy on an OpenShift cluster"
	@echo "make deploy/observability-remote-write-proxy/secrets    setup and deploy the needed secrets for Observability Remote Write Proxy"
	@echo "make deploy/observability-remote-write-proxy/route      deploys the OCP Route used for Observability Remote Write Proxy access"
	@echo "make undeploy                                           remove the service deployments from an openshift cluster"
	@echo "make openapi/spec/validate                              validate OpenAPI spec using spectral"
	@echo "make db/setup                                           setup and run a postgresql container"
	@echo "make db/migrate                                         run kas-fleet-manager data migrations"
	@echo "make db/login                                           log into the psql shell"
	@echo "make make db/generate/insert/cluster                    generate an example insert command for the clusters table"
	@echo "make db/teardown                                        remove and cleanup the postgresql container"
	@echo "make docs/generate/mermaid                              generate mermaid diagrams"
	@echo "make ocm/setup                                          generate secrets specific to ocm authentication"
	@echo "make ocm/login                                          ocm login"
	@echo "make sso/setup                                          local keycloak instance"
	@echo "make sso/config                                         local configure realm"
	@echo "make sso/teardown                                       teardown keycloak instance"
	@echo "make redhatsso/setup                                    setup mas sso clientId & clientSecret"
	@echo "make dataplane/imagepull/secret/setup                   setup dockerconfig image pull secret"
	@echo "$(fake)"
.PHONY: help

# Set git hook path to .githooks/
.PHONY: setup/git/hooks
setup/git/hooks:
	$(GIT) config core.hooksPath .githooks

# Checks if a GOPATH is set, or emits an error message
check-gopath:
ifndef GOPATH
	$(error GOPATH is not set)
endif
.PHONY: check-gopath

# Verifies that source passes standard checks.
# Also verifies that the OpenAPI spec is correct.
verify: check-gopath openapi/validate
	$(GO) vet \
		./cmd/... \
		./pkg/... \
		./internal/... \
		./test/...
.PHONY: verify

# Lint OpenShift templates

lint/templates: specinstall
	$(SPECTRAL) lint templates/*.yml templates/*.yaml --ignore-unknown-format --ruleset .validate-templates.yaml
.PHONY: lint/templates


# Runs linter against go files and .y(a)ml files in the templates directory
# Requires golangci-lint to be installed @ $(go env GOPATH)/bin/golangci-lint
# and spectral installed via npm
lint: golangci-lint lint/templates
	$(GOLANGCI_LINT) run \
		./cmd/... \
		./pkg/... \
		./internal/... \
		./test/...
.PHONY: lint

# Build binaries
# NOTE it may be necessary to use CGO_ENABLED=0 for backwards compatibility with centos7 if not using centos7
binary:
	$(GO) build ./cmd/kas-fleet-manager
.PHONY: binary

# Install
install: verify lint
	$(GO) install ./cmd/kas-fleet-manager
.PHONY: install

# --------------------- Test Targets --------------------- #
# Make targets for running tests related commands

# Set var defaults for test targets
TEST_SUMMARY_FORMAT ?= short-verbose
ifndef TEST_TIMEOUT
	ifeq ($(OCM_ENV), integration)
		TEST_TIMEOUT=15m
	else
		TEST_TIMEOUT=5h
	endif
endif

# Runs the unit tests.
#
# Args:
#   TESTFLAGS: Flags to pass to `go test`. The `-v` argument is always passed.
#
# Examples:
#   make test TESTFLAGS="-run TestSomething"
test: gotestsum
	OCM_ENV=testing $(GOTESTSUM) --junitfile data/results/unit-tests.xml --format $(TEST_SUMMARY_FORMAT) -- -p 1 -v -count=1 -coverprofile cover.out \
		$(shell $(GO) list ./... | grep -v /test) \
		$(TESTFLAGS)

# filter out mocked, generated, and other files which do not need to be tested from the unit test coverage results
	grep -v -e "_moq.go" \
    -e "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/main.go" \
    -e "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/" \
    -e "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/" \
    -e "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/" \
    -e "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/migrations/" \
    -e "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/"  \
    -e "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/converters/"  \
    -e "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/environments/"  \
    -e "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/migrations/" \
    -e "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments/"  \
    -e "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/metrics/providers.go" \
    -e "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/logger/logger.go" \
    -e "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers/json_schema_validation.go" \
    -e "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/error.go" \
    -e "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/metadata.go" \
    -e "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/write_json_response.go" \
    -e "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/handle_error.go" \
    -e "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth/helper.go" \
    -e "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth/context_config.go" \
    -e "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak/config.go" \
    -e "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm/config.go" \
    -e "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm/client.go" \
    -e "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/observatorium/observability_config.go" \
    -e "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/observatorium/api_mock.go" \
    -e "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/account/accountservice_interface.go" \
    -e "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/account/organization_type.go" \
    -e "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/account/account_mock.go" \
    -e "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/sso/keycloak.go" \
    -e "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/sso/keycloak_service_proxy.go" \
    -e "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/handlers/authentication.go" \
    -e "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/sentry/providers.go" \
    -e "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/sentry/config.go" \
    -e "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/signalbus/providers.go" \
    -e "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/signalbus/pg_signalbus.go" \
    -e "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db/connection.go" \
    -e "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db/migrations.go" \
    -e "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db/transaction_middleware.go" \
    -e "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db/logger.go" \
	cover.out > coverage.out
.PHONY: test

test/output/coverage/report:
	@if [ -f coverage.out ]; then $(GO) tool cover -func=coverage.out; else echo "coverage.out file not found"; fi;
.PHONY: test/output/coverage/report

test/html/coverage/report:
	@if [ -f coverage.out ]; then $(GO) tool cover -html=coverage.out; else echo "coverage.out file not found"; fi;
.PHONY: test/html/coverage/report

# Runs the integration tests.
#
# Args:
#   TESTFLAGS: Flags to pass to `go test`. The `-v` argument is always passed.
#
# Example:
#   make test/integration
#   make test/integration TESTFLAGS="-run TestAccounts"     acts as TestAccounts* and run TestAccountsGet, TestAccountsPost, etc.
#   make test/integration TESTFLAGS="-run TestAccountsGet"  runs TestAccountsGet
#   make test/integration TESTFLAGS="-short"                skips long-run tests
test/integration/kafka: gotestsum
	$(GOTESTSUM) --junitfile data/results/kas-fleet-manager-integration-tests.xml --format $(TEST_SUMMARY_FORMAT) -- -p 1 -ldflags -s -v -timeout $(TEST_TIMEOUT) -count=1 \
				./internal/kafka/test/integration/... \
				$(TESTFLAGS)
.PHONY: test/integration/kafka

test/integration/connector: gotestsum
	$(GOTESTSUM) --junitfile data/results/integraton-tests-connector.xml --format $(TEST_SUMMARY_FORMAT) -- -p 1 -ldflags -s -v -timeout $(TEST_TIMEOUT) -count=1 \
				./internal/connector/test/integration/... \
				$(TESTFLAGS)
.PHONY: test/integration/connector

test/integration: test/integration/kafka test/integration/connector
.PHONY: test/integration

test/report-portal-format-results:
ifeq ($(OCM_ENV), development)
	@$(XMLLINT) --xpath '//testsuite[position()<=1]' data/results/kas-fleet-manager-integration-tests.xml > data/results/kas-fleet-manager-integration-tests-stage.xml
endif
.PHONY: test/report-portal-format-results

# push results to report-portal
test/report-portal/push: test/report-portal-format-results
ifeq ($(OCM_ENV), development)
	@$(CURL) -k -X POST "$(REPORTPORTAL_ENDPOINT)/api/v1/$(REPORTPORTAL_PROJECT)/launch/import" \
		-H "accept: */*" -H "Content-Type: multipart/form-data" -H "Authorization: bearer $(REPORTPORTAL_ACCESS_TOKEN)" \
			-F "file=@data/results/kas-fleet-manager-integration-tests-stage.xml;type=text/xml"
else
	@echo "pushing results to report portal skipped"
endif
.PHONY: test/report-portal/push

# remove OSD cluster after running tests against real OCM
# requires OCM_OFFLINE_TOKEN env var exported
test/cluster/cleanup:
	./scripts/cleanup_test_cluster.sh
.PHONY: test/cluster/cleanup

# Target to autogenerate moq files
moq/generate: PRE_DELETE_AUTOGENERATED_MOQ_FILES ?= true
moq/generate: moq
	@if [ "$(PRE_DELETE_AUTOGENERATED_MOQ_FILES)" = "true" ]; then \
		echo "Pre-Deleting autogenerated moq files ..." ; \
		find $(PROJECT_PATH) -type f -name '*_moq.go' -delete -print ; \
		echo "Deletion of autogenerated moq files finished." ; \
	fi
	$(GO) generate -run moq ./...
.PHONY: moq/generate

# generate files
generate: moq/generate openapi/generate
.PHONY: generate

# validate the openapi schema
openapi/validate: openapi-generator
	$(OPENAPI_GENERATOR) validate -i openapi/kas-fleet-manager.yaml
	$(OPENAPI_GENERATOR) validate -i openapi/kas-fleet-manager-private.yaml
	$(OPENAPI_GENERATOR) validate -i openapi/kas-fleet-manager-private-admin.yaml
	$(OPENAPI_GENERATOR) validate -i openapi/connector_mgmt.yaml
	$(OPENAPI_GENERATOR) validate -i openapi/connector_mgmt-private.yaml
	$(OPENAPI_GENERATOR) validate -i openapi/connector_mgmt-private-admin.yaml
.PHONY: openapi/validate

# generate the openapi schema and generated package
openapi/generate: openapi/generate/kas-public openapi/generate/kas-private openapi/generate/kas-admin openapi/generate/connector-public openapi/generate/connector-private openapi/generate/connector-private-admin
.PHONY: openapi/generate

openapi/generate/kas-public: openapi-generator
	rm -rf internal/kafka/internal/api/public
	$(OPENAPI_GENERATOR) validate -i openapi/kas-fleet-manager.yaml
	$(OPENAPI_GENERATOR) generate -i openapi/kas-fleet-manager.yaml -g go -o internal/kafka/internal/api/public --package-name public -t openapi/templates --ignore-file-override ./.openapi-generator-ignore
	$(GOFMT) -w internal/kafka/internal/api/public
.PHONY: openapi/generate/kas-public

openapi/generate/kas-private: openapi-generator
	rm -rf internal/kafka/internal/api/private
	$(OPENAPI_GENERATOR) validate -i openapi/kas-fleet-manager-private.yaml
	$(OPENAPI_GENERATOR) generate -i openapi/kas-fleet-manager-private.yaml -g go -o internal/kafka/internal/api/private --package-name private -t openapi/templates --ignore-file-override ./.openapi-generator-ignore
	$(GOFMT) -w internal/kafka/internal/api/private
.PHONY: openapi/generate/kas-private

openapi/generate/kas-admin: openapi-generator
	rm -rf internal/kafka/internal/api/admin/private
	$(OPENAPI_GENERATOR) validate -i openapi/kas-fleet-manager-private-admin.yaml
	$(OPENAPI_GENERATOR) generate -i openapi/kas-fleet-manager-private-admin.yaml -g go -o internal/kafka/internal/api/admin/private --package-name private -t openapi/templates --ignore-file-override ./.openapi-generator-ignore
	$(GOFMT) -w internal/kafka/internal/api/admin/private
.PHONY: openapi/generate/kas-admin

openapi/generate/connector-public: openapi-generator
	rm -rf internal/connector/internal/api/public
	$(OPENAPI_GENERATOR) validate -i openapi/connector_mgmt.yaml
	$(OPENAPI_GENERATOR) generate -i openapi/connector_mgmt.yaml -g go -o internal/connector/internal/api/public --package-name public --additional-properties=enumClassPrefix=true -t openapi/templates --ignore-file-override ./.openapi-generator-ignore
	$(GOFMT) -w internal/connector/internal/api/public
.PHONY: openapi/generate/connector-public

openapi/generate/connector-private: openapi-generator
	rm -rf internal/connector/internal/api/private
	$(OPENAPI_GENERATOR) validate -i openapi/connector_mgmt-private.yaml
	$(OPENAPI_GENERATOR) generate -i openapi/connector_mgmt-private.yaml -g go -o internal/connector/internal/api/private --package-name private --additional-properties=enumClassPrefix=true -t openapi/templates --ignore-file-override ./.openapi-generator-ignore
	$(GOFMT) -w internal/connector/internal/api/private
.PHONY: openapi/generate/connector-private

openapi/generate/connector-private-admin: openapi-generator
	rm -rf internal/connector/internal/api/admin/private
	$(OPENAPI_GENERATOR) validate -i openapi/connector_mgmt-private-admin.yaml
	$(OPENAPI_GENERATOR) generate -i openapi/connector_mgmt-private-admin.yaml -g go -o internal/connector/internal/api/admin/private --package-name private  --additional-properties=enumClassPrefix=true -t openapi/templates --ignore-file-override ./.openapi-generator-ignore
	$(GOFMT) -w internal/connector/internal/api/admin/private
.PHONY: openapi/generate/connector-private-admin

# clean up code and dependencies
code/fix:
	@$(GO) mod tidy
	@$(GOFMT) -w `find . -type f -name '*.go' -not -path "./vendor/*"`
.PHONY: code/fix

run: install
	kas-fleet-manager migrate
	kas-fleet-manager serve --public-host-url=${PUBLIC_HOST_URL}
.PHONY: run

# Run Swagger and host the api docs
run/docs:
	$(DOCKER) run -u $(shell id -u) --rm --name swagger_ui_docs -d -p 8082:8080 -e URLS="[ \
		{ url: \"./openapi/kas-fleet-manager.yaml\", name: \"Public API\" },\
		{ url: \"./openapi/connector_mgmt.yaml\", name: \"Connector Management API\"},\
		{ url: \"./openapi/connector_mgmt-private.yaml\", name: \"Connector Management Private API\"},\
		{ url: \"./openapi/connector_mgmt-private-admin.yaml\", name: \"Private Connector Management Admin API\"},\
		{ url: \"./openapi/kas-fleet-manager-private.yaml\", name: \"Private API\"},\
		{ url: \"./openapi/kas-fleet-manager-private-admin.yaml\", name: \"Private Admin API\"}]"\
		  -v $(PWD)/openapi/:/usr/share/nginx/html/openapi:Z swaggerapi/swagger-ui
	@echo "Please open http://localhost:8082/"
.PHONY: run/docs

# Remove Swagger container
run/docs/teardown:
	$(DOCKER) container stop swagger_ui_docs
.PHONY: run/docs/teardown

cos-fleet-catalog-camel/setup:
	$(DOCKER) run --name cos-fleet-catalog-camel --rm -d -p 9091:8080 quay.io/lburgazzoli/ccs:latest
	mkdir -p config/connector-types
	echo -n "http://localhost:9091" > config/connector-types/cos-fleet-catalog-camel
.PHONY: cos-fleet-catalog-camel/setup

cos-fleet-catalog-camel/teardown:
	$(DOCKER) stop cos-fleet-catalog-camel
	rm config/connector-types/cos-fleet-catalog-camel
.PHONY: cos-fleet-catalog-camel/teardown

# Touch all the necessary files for fleet manager to start
# See docs/populating-configuration.md for more information
secrets/setup/empty:
	touch $(SECRETS_DIR)/ocm-service.clientId
	touch $(SECRETS_DIR)/ocm-service.clientSecret
	touch $(SECRETS_DIR)/ocm-service.token
	touch $(SECRETS_DIR)/aws.accountid
	touch $(SECRETS_DIR)/aws.accesskey
	touch $(SECRETS_DIR)/aws.secretaccesskey
	touch $(SECRETS_DIR)/aws.route53accesskey
	touch $(SECRETS_DIR)/aws.route53secretaccesskey
	touch $(SECRETS_DIR)/keycloak-service.clientId
	touch $(SECRETS_DIR)/keycloak-service.clientSecret
	touch $(SECRETS_DIR)/redhatsso-service.clientId
	touch $(SECRETS_DIR)/redhatsso-service.clientSecret
	touch $(SECRETS_DIR)/osd-idp-keycloak-service.clientId
	touch $(SECRETS_DIR)/osd-idp-keycloak-service.clientSecret
	touch $(SECRETS_DIR)/sentry.key
	touch $(SECRETS_DIR)/kafka-tls.crt
	touch $(SECRETS_DIR)/kafka-tls.key
	touch $(SECRETS_DIR)/image-pull.dockerconfigjson
	touch $(SECRETS_DIR)/dataplane-observability-config.yaml
.PHONY: secrets/setup/empty

db/setup:
	./scripts/local_db_setup.sh
.PHONY: db/setup

db/migrate:
	$(GO) run ./cmd/kas-fleet-manager migrate
.PHONY: db/migrate

db/teardown:
	./scripts/local_db_teardown.sh
.PHONY: db/teardown

KEYCLOAK_URL ?= http://localhost:8180
KEYCLOAK_PORT_NO ?= 8180
KEYCLOAK_USER ?= admin
KEYCLOAK_PASSWORD ?= admin

sso/setup:
	KEYCLOAK_PORT_NO=$(KEYCLOAK_PORT_NO) KEYCLOAK_USER=$(KEYCLOAK_USER) KEYCLOAK_PASSWORD=$(KEYCLOAK_PASSWORD) ./scripts/mas_sso_setup.sh
.PHONY: sso/setup

sso/config:
	KEYCLOAK_URL=$(KEYCLOAK_URL) KEYCLOAK_USER=$(KEYCLOAK_USER) \
	KEYCLOAK_PASSWORD=$(KEYCLOAK_PASSWORD) ./scripts/mas_sso_config.sh
.PHONY: sso/config

sso/teardown:
	./scripts/mas_sso_teardown.sh
.PHONY: sso/teardown

db/login:
	$(DOCKER) exec -u $(shell id -u) -it kas-fleet-manager-db /bin/bash -c "PGPASSWORD=$(shell cat secrets/db.password) psql -d $(shell cat secrets/db.name) -U $(shell cat secrets/db.user)"
.PHONY: db/login

db/generate/insert/cluster:
	read -r id external_id provider region multi_az<<<"$(shell $(OCM) get /api/clusters_mgmt/v1/clusters/${CLUSTER_ID} | jq '.id, .external_id, .cloud_provider.id, .region.id, .multi_az' | tr -d \" | xargs -n2 echo)";\
	echo -e "Run this command in your database:\n\nINSERT INTO clusters (id, created_at, updated_at, cloud_provider, cluster_id, external_id, multi_az, region, status, provider_type) VALUES ('"$$id"', current_timestamp, current_timestamp, '"$$provider"', '"$$id"', '"$$external_id"', "$$multi_az", '"$$region"', 'cluster_provisioned', 'ocm');";
.PHONY: db/generate/insert/cluster

# --------------------- Image Targets --------------------- #
# make targets for building and pushing images to a registry

# Set var defaults for image targets based on os
ifeq ($(shell uname -s | tr A-Z a-z), darwin)
CONTAINER_IMAGE_BUILD_PLATFORM ?= --platform linux/amd64
endif

# Login to docker
docker/login:
	$(DOCKER) --config="${DOCKER_CONFIG}" login -u "${QUAY_USER}" -p "${QUAY_TOKEN}" quay.io
.PHONY: docker/login

# Login to the OpenShift internal registry
docker/login/internal:
	$(DOCKER) login -u kubeadmin -p $(shell $(OC) whoami -t) $(shell $(OC) get route default-route -n openshift-image-registry -o jsonpath="{.spec.host}")
.PHONY: docker/login/internal

# Build the binary and image
image/build:
	$(DOCKER) --config="${DOCKER_CONFIG}" build $(CONTAINER_IMAGE_BUILD_PLATFORM) --pull -t "$(external_image_registry)/$(image_repository):$(image_tag)" .
.PHONY: image/build

# Build and push the image
image/push: image/build
	$(DOCKER) --config="${DOCKER_CONFIG}" push "$(external_image_registry)/$(image_repository):$(image_tag)"
.PHONY: image/push

# build binary and image for OpenShift deployment
image/build/internal: IMAGE_TAG ?= $(image_tag)
image/build/internal:
	$(DOCKER) build $(CONTAINER_IMAGE_BUILD_PLATFORM) -t "$(shell $(OC) get route default-route -n openshift-image-registry -o jsonpath="{.spec.host}")/$(image_repository):$(IMAGE_TAG)" .
.PHONY: image/build/internal

# push the image to the OpenShift internal registry
image/push/internal: IMAGE_TAG ?= $(image_tag)
image/push/internal: docker/login/internal
	$(DOCKER) push "$(shell $(OC) get route default-route -n openshift-image-registry -o jsonpath="{.spec.host}")/$(image_repository):$(IMAGE_TAG)"
.PHONY: image/push/internal

# build and push the image to an OpenShift cluster's internal registry
# namespace used in the image repository must exist on the cluster before running this command. Run `make deploy/project` to create the namespace if not available.
image/build/push/internal: image/build/internal image/push/internal
.PHONY: image/build/push/internal

# Build the binary and test image
image/build/test: binary
	$(DOCKER) build -t "$(test_image)" -f Dockerfile.integration.test .
.PHONY: image/build/test

# Run the test container
test/run: image/build/test
	$(DOCKER) run -u $(shell id -u) --net=host -p 9876:9876 -i "$(test_image)"
.PHONY: test/run

# Setup for AWS credentials
aws/setup:
	@echo -n "$(AWS_ACCOUNT_ID)" > secrets/aws.accountid
	@echo -n "$(AWS_ACCESS_KEY)" > secrets/aws.accesskey
	@echo -n "$(AWS_SECRET_ACCESS_KEY)" > secrets/aws.secretaccesskey
	@echo -n "$(ROUTE53_ACCESS_KEY)" > secrets/aws.route53accesskey
	@echo -n "$(ROUTE53_SECRET_ACCESS_KEY)" > secrets/aws.route53secretaccesskey
	@echo -n "$(AWS_SECRET_MANAGER_ACCESS_KEY)" > secrets/aws-secret-manager/aws_access_key_id
	@echo -n "$(AWS_SECRET_MANAGER_SECRET_ACCESS_KEY)" > secrets/aws-secret-manager/aws_secret_access_key
.PHONY: aws/setup

# Setup for GCP credentials
gcp/setup/credentials:
	@echo -n "$${GCP_API_CREDENTIALS}" | base64 -d > secrets/gcp.api-credentials
.PHONY: gcp/setup/credentials

# Setup for mas sso credentials
keycloak/setup:
	@echo -n "$(MAS_SSO_CLIENT_ID)" > secrets/keycloak-service.clientId
	@echo -n "$(MAS_SSO_CLIENT_SECRET)" > secrets/keycloak-service.clientSecret
	@echo -n "$(OSD_IDP_MAS_SSO_CLIENT_ID)" > secrets/osd-idp-keycloak-service.clientId
	@echo -n "$(OSD_IDP_MAS_SSO_CLIENT_SECRET)" > secrets/osd-idp-keycloak-service.clientSecret
.PHONY:keycloak/setup

redhatsso/setup:
	@echo -n "$(SSO_CLIENT_ID)" > secrets/redhatsso-service.clientId
	@echo -n "$(SSO_CLIENT_SECRET)" > secrets/redhatsso-service.clientSecret
.PHONY:redhatsso/setup

# Setup for the kafka broker certificate
kafkacert/setup:
	@echo -n "$(KAFKA_TLS_CERT)" > secrets/kafka-tls.crt
	@echo -n "$(KAFKA_TLS_KEY)" > secrets/kafka-tls.key
	@echo -n "$(ACME_ISSUER_ACCOUNT_KEY)" > secrets/kafka-tls-certificate-management-acme-issuer-account-key.pem
.PHONY:kafkacert/setup

observability/cloudwatchlogs/setup:
	@echo -n "$${OBSERVABILITY_CLOUDWATCHLOGS_CONFIG}" > secrets/observability-cloudwatchlogs-config.yaml

observatorium/token-refresher/setup: PORT ?= 8085
observatorium/token-refresher/setup: IMAGE_TAG ?= latest
observatorium/token-refresher/setup: ISSUER_URL ?= https://sso.redhat.com/auth/realms/redhat-external
observatorium/token-refresher/setup: OBSERVATORIUM_URL ?= https://observatorium-mst.api.stage.openshift.com/api/metrics/v1/managedkafka
observatorium/token-refresher/setup:
	@$(DOCKER) run -d -p ${PORT}:${PORT} \
		--restart always \
		--name observatorium-token-refresher quay.io/rhoas/mk-token-refresher:${IMAGE_TAG} \
		/bin/token-refresher \
		--oidc.issuer-url="${ISSUER_URL}" \
		--url="${OBSERVATORIUM_URL}" \
		--oidc.client-id="${CLIENT_ID}" \
		--oidc.client-secret="${CLIENT_SECRET}" \
		--web.listen=":${PORT}"
	@echo The Observatorium token refresher is now running on 'http://localhost:${PORT}'
.PHONY: observatorium/token-refresher/setup

# OCM login
ocm/login: SERVER_URL ?= http://localhost:8000
ocm/login:
	@$(OCM) login --url="$(SERVER_URL)" --token="$(OCM_OFFLINE_TOKEN)"
.PHONY: ocm/login

# Setup OCM_OFFLINE_TOKEN and
# OCM Client ID and Secret should be set only when running inside docker in integration ENV)
ocm/setup: OCM_CLIENT_ID ?= ocm-ams-testing
ocm/setup: OCM_CLIENT_SECRET ?= 8f0c06c5-a558-4a78-a406-02deb1fd3f17
ocm/setup:
	@echo -n "$(OCM_OFFLINE_TOKEN)" > secrets/ocm-service.token
	@echo -n "" > secrets/ocm-service.clientId
	@echo -n "" > secrets/ocm-service.clientSecret
ifeq ($(OCM_ENV), integration)
	@if [[ -n "$(DOCKER_PR_CHECK)" ]]; then echo -n "$(OCM_CLIENT_ID)" > secrets/ocm-service.clientId; echo -n "$(OCM_CLIENT_SECRET)" > secrets/ocm-service.clientSecret; fi;
endif
.PHONY: ocm/setup

# create project where the service will be deployed in an OpenShift cluster
deploy/project:
	@-$(OC) new-project $(NAMESPACE)
.PHONY: deploy/project

# deploy the postgres database required by the service to an OpenShift cluster
deploy/db:
	$(OC) process -f ./templates/db-template.yml | $(OC) apply -f - -n $(NAMESPACE)
	@time timeout --foreground 3m bash -c "until $(OC) get pods -n $(NAMESPACE) | grep kas-fleet-manager-db | grep -v deploy | grep -q Running; do echo 'database is not ready yet'; sleep 10; done"
.PHONY: deploy/db

# deploys the secrets required by the service to an OpenShift cluster
deploy/secrets:
	@$(OC) get service/kas-fleet-manager-db -n $(NAMESPACE) || (echo "Database is not deployed, please run 'make deploy/db'"; exit 1)
	@$(OC) process -f ./templates/secrets-template.yml \
		-p OCM_SERVICE_CLIENT_ID="$(shell ([ -s './secrets/ocm-service.clientId' ] && [ -z '${OCM_SERVICE_CLIENT_ID}' ]) && cat ./secrets/ocm-service.clientId || echo '${OCM_SERVICE_CLIENT_ID}')" \
		-p OCM_SERVICE_CLIENT_SECRET="$(shell ([ -s './secrets/ocm-service.clientSecret' ] && [ -z '${OCM_SERVICE_CLIENT_SECRET}' ]) && cat ./secrets/ocm-service.clientSecret || echo '${OCM_SERVICE_CLIENT_SECRET}')" \
		-p OCM_SERVICE_TOKEN="$(shell ([ -s './secrets/ocm-service.token' ] && [ -z '${OCM_SERVICE_TOKEN}' ]) && cat ./secrets/ocm-service.token || echo '${OCM_SERVICE_TOKEN}')" \
		-p SENTRY_KEY="$(shell ([ -s './secrets/sentry.key' ] && [ -z '${SENTRY_KEY}' ]) && cat ./secrets/sentry.key || echo '${SENTRY_KEY}')" \
		-p AWS_ACCESS_KEY="$(shell ([ -s './secrets/aws.accesskey' ] && [ -z '${AWS_ACCESS_KEY}' ]) && cat ./secrets/aws.accesskey || echo '${AWS_ACCESS_KEY}')" \
		-p AWS_ACCOUNT_ID="$(shell ([ -s './secrets/aws.accountid' ] && [ -z '${AWS_ACCOUNT_ID}' ]) && cat ./secrets/aws.accountid || echo '${AWS_ACCOUNT_ID}')" \
		-p AWS_SECRET_ACCESS_KEY="$(shell ([ -s './secrets/aws.secretaccesskey' ] && [ -z '${AWS_SECRET_ACCESS_KEY}' ]) && cat ./secrets/aws.secretaccesskey || echo '${AWS_SECRET_ACCESS_KEY}')" \
		-p ROUTE53_ACCESS_KEY="$(shell ([ -s './secrets/aws.route53accesskey' ] && [ -z '${ROUTE53_ACCESS_KEY}' ]) && cat ./secrets/aws.route53accesskey || echo '${ROUTE53_ACCESS_KEY}')" \
		-p ROUTE53_SECRET_ACCESS_KEY="$(shell ([ -s './secrets/aws.route53secretaccesskey' ] && [ -z '${ROUTE53_SECRET_ACCESS_KEY}' ]) && cat ./secrets/aws.route53secretaccesskey || echo '${ROUTE53_SECRET_ACCESS_KEY}')" \
		-p AWS_SECRET_MANAGER_ACCESS_KEY="$(shell ([ -s './secrets/aws-secret-manager/aws_access_key_id' ] && [ -z '${AWS_SECRET_MANAGER_ACCESS_KEY}' ]) && cat ./secrets/aws-secret-manager/aws_access_key_id || echo '${AWS_SECRET_MANAGER_ACCESS_KEY}')" \
		-p AWS_SECRET_MANAGER_SECRET_ACCESS_KEY="$(shell ([ -s './secrets/aws-secret-manager/aws_secret_access_key' ] && [ -z '${AWS_SECRET_MANAGER_SECRET_ACCESS_KEY}' ]) && cat ./secrets/aws-secret-manager/aws_secret_access_key || echo '${AWS_SECRET_MANAGER_SECRET_ACCESS_KEY}')" \
		-p GCP_API_CREDENTIALS="$(shell ([ -s './secrets/gcp.api-credentials' ] && [ -z '${GCP_API_CREDENTIALS}' ]) && cat ./secrets/gcp.api-credentials | base64 -w 0 ||  echo '${GCP_API_CREDENTIALS}')" \
		-p MAS_SSO_CLIENT_ID="$(shell ([ -s './secrets/keycloak-service.clientId' ] && [ -z '${MAS_SSO_CLIENT_ID}' ]) && cat ./secrets/keycloak-service.clientId || echo '${MAS_SSO_CLIENT_ID}')" \
		-p MAS_SSO_CLIENT_SECRET="$(shell ([ -s './secrets/keycloak-service.clientSecret' ] && [ -z '${MAS_SSO_CLIENT_SECRET}' ]) && cat ./secrets/keycloak-service.clientSecret || echo '${MAS_SSO_CLIENT_SECRET}')" \
		-p OSD_IDP_MAS_SSO_CLIENT_ID="$(shell ([ -s './secrets/osd-idp-keycloak-service.clientId' ] && [ -z '${OSD_IDP_MAS_SSO_CLIENT_ID}' ]) && cat ./secrets/osd-idp-keycloak-service.clientId || echo '${OSD_IDP_MAS_SSO_CLIENT_ID}')" \
		-p OSD_IDP_MAS_SSO_CLIENT_SECRET="$(shell ([ -s './secrets/osd-idp-keycloak-service.clientSecret' ] && [ -z '${OSD_IDP_MAS_SSO_CLIENT_SECRET}' ]) && cat ./secrets/osd-idp-keycloak-service.clientSecret || echo '${OSD_IDP_MAS_SSO_CLIENT_SECRET}')" \
		-p MAS_SSO_CRT="$(shell ([ -s './secrets/keycloak-service.crt' ] && [ -z '${MAS_SSO_CRT}' ]) && cat ./secrets/keycloak-service.crt || echo '${MAS_SSO_CRT}')" \
		-p KAFKA_TLS_CERT="$(shell ([ -s './secrets/kafka-tls.crt' ] && [ -z '${KAFKA_TLS_CERT}' ]) && cat ./secrets/kafka-tls.crt || echo '${KAFKA_TLS_CERT}')" \
		-p KAFKA_TLS_KEY="$(shell ([ -s './secrets/kafka-tls.key' ] && [ -z '${KAFKA_TLS_KEY}' ]) && cat ./secrets/kafka-tls.key || echo '${KAFKA_TLS_KEY}')" \
		-p ACME_ISSUER_ACCOUNT_KEY="$(shell ([ -s './secrets/kafka-tls-certificate-management-acme-issuer-account-key.pem' ] && [ -z '${ACME_ISSUER_ACCOUNT_KEY}' ]) && cat ./secrets/kafka-tls-certificate-management-acme-issuer-account-key.pem || echo '${ACME_ISSUER_ACCOUNT_KEY}')" \
		-p IMAGE_PULL_DOCKER_CONFIG="$(shell ([ -s './secrets/image-pull.dockerconfigjson' ] && [ -z '${IMAGE_PULL_DOCKER_CONFIG}' ]) && cat ./secrets/image-pull.dockerconfigjson | base64 -w 0 || echo '${IMAGE_PULL_DOCKER_CONFIG}')" \
		-p KUBE_CONFIG="${KUBE_CONFIG}" \
		-p OBSERVABILITY_RHSSO_GRAFANA_CLIENT_ID="${OBSERVABILITY_RHSSO_GRAFANA_CLIENT_ID}" \
		-p OBSERVABILITY_RHSSO_GRAFANA_CLIENT_SECRET="${OBSERVABILITY_RHSSO_GRAFANA_CLIENT_SECRET}" \
		-p OBSERVABILITY_CLOUDWATCHLOGS_CONFIG="$(shell ([ -s './secrets/observability-cloudwatchlogs-config.yaml' ] && [ -z '${OBSERVABILITY_CLOUDWATCHLOGS_CONFIG}' ]) && cat ./secrets/observability-cloudwatchlogs-config.yaml | base64 -w 0 || echo '${OBSERVABILITY_CLOUDWATCHLOGS_CONFIG}' | base64 -w 0)" \
		-p REDHAT_SSO_CLIENT_ID="$(shell ([ -s './secrets/redhatsso-service.clientId' ] && [ -z '${REDHAT_SSO_CLIENT_ID}' ]) && cat ./secrets/redhatsso-service.clientId || echo '${REDHAT_SSO_CLIENT_ID}')" \
		-p REDHAT_SSO_CLIENT_SECRET="$(shell ([ -s './secrets/redhatsso-service.clientSecret' ] && [ -z '${REDHAT_SSO_CLIENT_SECRET}' ]) && cat ./secrets/redhatsso-service.clientSecret || echo '${REDHAT_SSO_CLIENT_SECRET}')" \
		-p DATAPLANE_OBSERVABILITY_CONFIG="$(shell ([ -s './secrets/dataplane-observability-config.yaml' ] && [ -z '${DATAPLANE_OBSERVABILITY_CONFIG}' ]) && cat ./secrets/dataplane-observability-config.yaml | base64 -w 0 || echo '${DATAPLANE_OBSERVABILITY_CONFIG}')" \
		| $(OC) apply -f - -n $(NAMESPACE)
.PHONY: deploy/secrets

deploy/envoy:
	@$(OC) apply -f ./templates/envoy-config-configmap.yml -n $(NAMESPACE)
.PHONY: deploy/envoy

deploy/route:
	@$(OC) process -f ./templates/route-template.yml | $(OC) apply -f - -n $(NAMESPACE)
.PHONY: deploy/route

# deploy service via templates to an OpenShift cluster
deploy/service: IMAGE_REGISTRY ?= $(internal_image_registry)
deploy/service: IMAGE_REPOSITORY ?= $(image_repository)
deploy/service: FLEET_MANAGER_ENV ?= "development"
deploy/service: REPLICAS ?= "1"
deploy/service: ENABLE_KAFKA_EXTERNAL_CERTIFICATE ?= "false"
deploy/service: ENABLE_KAFKA_CNAME_REGISTRATION ?= "false"
deploy/service: OCM_URL ?= "https://api.stage.openshift.com"
deploy/service: AMS_URL ?= "https://api.stage.openshift.com"
deploy/service: MAS_SSO_ENABLE_AUTH ?= "true"
deploy/service: MAS_SSO_INSECURE ?= "false"
deploy/service: MAS_SSO_BASE_URL ?= "https://identity.api.stage.openshift.com"
deploy/service: MAS_SSO_REALM ?= "rhoas"
deploy/service: SSO_SPECIAL_MANAGEMENT_ORG_ID ?= "13640203"
deploy/service: REDHAT_SSO_BASE_URL ?= "https://sso.redhat.com"
deploy/service: SERVICE_ACCOUNT_LIMIT_CHECK_SKIP_ORG_ID_LIST ?= "[]"
deploy/service: USER_NAME_CLAIM ?= "clientId"
deploy/service: FALL_BACK_USER_NAME_CLAIM ?= "preferred_username"
deploy/service: MAX_ALLOWED_SERVICE_ACCOUNTS ?= "2"
deploy/service: MAX_LIMIT_FOR_SSO_GET_CLIENTS ?= "100"
deploy/service: OSD_IDP_MAS_SSO_REALM ?= "rhoas-kafka-sre"
deploy/service: JWKS_VERIFY_INSECURE ?= "false"
deploy/service: TOKEN_ISSUER_URL ?= "https://sso.redhat.com/auth/realms/redhat-external"
deploy/service: SERVICE_PUBLIC_HOST_URL ?= "https://api.openshift.com"
deploy/service: ENABLE_TERMS_ACCEPTANCE ?= "false"
deploy/service: ENABLE_DENY_LIST ?= "false"
deploy/service: ENABLE_ACCESS_LIST ?= "false"
deploy/service: DENIED_USERS ?= "[]"
deploy/service: ACCEPTED_ORGANISATIONS ?= "[]"
deploy/service: ALLOW_DEVELOPER_INSTANCE ?= "true"
deploy/service: QUOTA_TYPE ?= "quota-management-list"
deploy/service: STRIMZI_OLM_INDEX_IMAGE ?= "quay.io/osd-addons/managed-kafka:production-82b42db"
deploy/service: KAS_FLEETSHARD_OLM_INDEX_IMAGE ?= "quay.io/osd-addons/kas-fleetshard-operator:production-82b42db"
deploy/service: OBSERVABILITY_OPERATOR_INDEX_IMAGE ?= "quay.io/rhoas/observability-operator-index:v4.2.0"
deploy/service: OBSERVABILITY_OPERATOR_STARTING_CSV ?= "observability-operator.v4.2.0"
deploy/service: OBSERVABILITY_CONFIG_REPO ?= "quay.io/rhoas/observability-resources-mk"
deploy/service: OBSERVABILITY_CONFIG_CHANNEL ?= "resources"
deploy/service: OBSERVABILITY_CONFIG_TAG ?= "latest"
deploy/service: DATAPLANE_CLUSTER_SCALING_TYPE ?= "manual"
deploy/service: STRIMZI_OPERATOR_ADDON_ID ?= "managed-kafka-qe"
deploy/service: KAS_FLEETSHARD_ADDON_ID ?= "kas-fleetshard-operator-qe"
deploy/service: STRIMZI_OLM_PACKAGE_NAME ?= "managed-kafka"
deploy/service: KAS_FLEETSHARD_OLM_PACKAGE_NAME ?= "kas-fleetshard-operator"
deploy/service: CLUSTER_LIST ?= "[]"
deploy/service: SUPPORTED_CLOUD_PROVIDERS ?= "[{name: aws, default: true, regions: [{name: us-east-1, default: true, supported_instance_type: {standard: {}, developer: {}}}]}]"
deploy/service: KAS_FLEETSHARD_OPERATOR_SUBSCRIPTION_CONFIG ?= "{}"
deploy/service: STRIMZI_OPERATOR_SUBSCRIPTION_CONFIG ?= "{}"
deploy/service: ENABLE_KAFKA_SRE_IDENTITY_PROVIDER_CONFIGURATION ?="true"
deploy/service: ENABLE_KAFKA_OWNER ?="false"
deploy/service: KAFKA_OWNERS ?="[]"
deploy/service: KAFKA_TLS_CERTIFICATE_MANAGEMENT_MUST_STAPLE ?= "false"
deploy/service: KAFKA_TLS_CERTIFICATE_MANAGEMENT_RENEWAL_WINDOW_RATIO ?= "0.3333333333"
deploy/service: KAFKA_TLS_CERTIFICATE_MANAGEMENT_EMAIL ?= ""
deploy/service: KAFKA_TLS_CERTIFICATE_MANAGEMENT_STORAGE_TYPE ?= "secure-storage"
deploy/service: KAFKA_TLS_CERTIFICATE_MANAGEMENT_STRATEGY ?= "manual"
deploy/service: KAFKA_TLS_CERTIFICATE_MANAGEMENT_SECURE_STORAGE_CACHE_TTL ?="10m"
deploy/service: SSO_PROVIDER_TYPE ?= "mas_sso"
deploy/service: REGISTERED_USERS_PER_ORGANISATION ?= "[{id: 13640203, any_user: true, max_allowed_instances: 5, registered_users: [], granted_quota: [{instance_type_id: standard, kafka_billing_models: [{id: standard, max_allowed_instances: 5}, {id: marketplace, max_allowed_instances: 5}, {id: enterprise, max_allowed_instances: 5}]}]}, {id: 12147054, any_user: true, max_allowed_instances: 1, registered_users: [], granted_quota: [{instance_type_id: standard, kafka_billing_models: [{id: standard, max_allowed_instances: 1}, {id: enterprise, max_allowed_instances: 1}]}]}, {id: 13639843, any_user: true, max_allowed_instances: 1, registered_users: [], granted_quota: [{instance_type_id: standard, kafka_billing_models: [{id: standard, max_allowed_instances: 1}, {id: enterprise, max_allowed_instances: 1}]}]}, {id: 13785172, any_user: true, max_allowed_instances: 1, registered_users: [], granted_quota: [{instance_type_id: standard, kafka_billing_models: [{id: standard, max_allowed_instances: 1}, {id: enterprise, max_allowed_instances: 1}]}]}, {id: 13645369, any_user: true, max_allowed_instances: 3, registered_users: [], granted_quota: [{instance_type_id: standard, kafka_billing_models: [{id: standard, max_allowed_instances: 3}, {id: enterprise, max_allowed_instances: 3}]}]}]"
deploy/service: DYNAMIC_SCALING_CONFIG ?= "{new_data_plane_openshift_version: '', enable_dynamic_data_plane_scale_up: false, enable_dynamic_data_plane_scale_down: false, compute_machine_per_cloud_provider: {aws: {cluster_wide_workload: {compute_machine_type: m5.2xlarge, compute_node_autoscaling: {min_compute_nodes: 3, max_compute_nodes: 18}}, kafka_workload_per_instance_type: {standard: {compute_machine_type: r5.xlarge, compute_node_autoscaling: {min_compute_nodes: 3, max_compute_nodes: 18}}, developer: {compute_machine_type: m5.2xlarge, compute_node_autoscaling: {min_compute_nodes: 1, max_compute_nodes: 3}}}}, gcp: {cluster_wide_workload: {compute_machine_type: custom-8-32768, compute_node_autoscaling: {min_compute_nodes: 3, max_compute_nodes: 18}}, kafka_workload_per_instance_type: {standard: {compute_machine_type: custom-8-32768, compute_node_autoscaling: {min_compute_nodes: 3, max_compute_nodes: 18}}, developer: {compute_machine_type: custom-8-32768, compute_node_autoscaling: {min_compute_nodes: 1, max_compute_nodes: 3}}}}}}"
deploy/service: NODE_PREWARMING_CONFIG ?= "{}"
deploy/service: ADMIN_AUTHZ_CONFIG ?= "[{method: GET, roles: [kas-fleet-manager-admin-full, kas-fleet-manager-admin-read, kas-fleet-manager-admin-write]}, {method: PATCH, roles: [kas-fleet-manager-admin-full, kas-fleet-manager-admin-write]}, {method: DELETE, roles: [kas-fleet-manager-admin-full]}]"
deploy/service: MAX_ALLOWED_DEVELOPER_INSTANCES ?= "1"
deploy/service: ADMIN_API_SSO_BASE_URL ?= "https://auth.redhat.com"
deploy/service: ADMIN_API_SSO_ENDPOINT_URI ?= "/auth/realms/EmployeeIDP"
deploy/service: ADMIN_API_SSO_REALM ?= "EmployeeIDP"
deploy/service: OBSERVABILITY_ENABLE_CLOUDWATCHLOGGING ?= "false"
deploy/service: DATAPLANE_OBSERVABILITY_CONFIG_ENABLE ?= "false"
deploy/service: deploy/envoy deploy/route
	@if test -z "$(IMAGE_TAG)"; then echo "IMAGE_TAG was not specified"; exit 1; fi
	@time timeout --foreground 3m bash -c "until $(OC) get routes -n $(NAMESPACE) | grep -q kas-fleet-manager; do echo 'waiting for kas-fleet-manager route to be created'; sleep 1; done"
	@$(OC) process -f ./templates/service-template.yml \
		-p ENVIRONMENT="$(FLEET_MANAGER_ENV)" \
		-p IMAGE_REGISTRY=$(IMAGE_REGISTRY) \
		-p IMAGE_REPOSITORY=$(IMAGE_REPOSITORY) \
		-p IMAGE_TAG=$(IMAGE_TAG) \
		-p REPLICAS="${REPLICAS}" \
		-p ENABLE_KAFKA_OWNER="${ENABLE_KAFKA_OWNER}" \
		-p KAFKA_OWNERS="${KAFKA_OWNERS}" \
		-p ENABLE_KAFKA_EXTERNAL_CERTIFICATE="${ENABLE_KAFKA_EXTERNAL_CERTIFICATE}" \
		-p ENABLE_KAFKA_CNAME_REGISTRATION="${ENABLE_KAFKA_CNAME_REGISTRATION}" \
		-p ENABLE_OCM_MOCK=$(ENABLE_OCM_MOCK) \
		-p OCM_MOCK_MODE=$(OCM_MOCK_MODE) \
		-p OCM_URL="$(OCM_URL)" \
		-p AMS_URL="${AMS_URL}" \
		-p JWKS_VERIFY_INSECURE="${JWKS_VERIFY_INSECURE}" \
		-p JWKS_URL="$(JWKS_URL)" \
		-p MAS_SSO_INSECURE="$(MAS_SSO_INSECURE)" \
		-p MAS_SSO_ENABLE_AUTH="${MAS_SSO_ENABLE_AUTH}" \
		-p MAS_SSO_BASE_URL="$(MAS_SSO_BASE_URL)" \
		-p MAS_SSO_REALM="$(MAS_SSO_REALM)" \
		-p SSO_SPECIAL_MANAGEMENT_ORG_ID="${SSO_SPECIAL_MANAGEMENT_ORG_ID}" \
		-p REDHAT_SSO_BASE_URL="${REDHAT_SSO_BASE_URL}" \
		-p USER_NAME_CLAIM="$(USER_NAME_CLAIM)" \
		-p FALL_BACK_USER_NAME_CLAIM="$(FALL_BACK_USER_NAME_CLAIM)" \
		-p MAX_ALLOWED_SERVICE_ACCOUNTS="${MAX_ALLOWED_SERVICE_ACCOUNTS}" \
		-p SERVICE_ACCOUNT_LIMIT_CHECK_SKIP_ORG_ID_LIST="$(SERVICE_ACCOUNT_LIMIT_CHECK_SKIP_ORG_ID_LIST)"\
		-p MAX_LIMIT_FOR_SSO_GET_CLIENTS="${MAX_LIMIT_FOR_SSO_GET_CLIENTS}" \
		-p ENABLE_DENY_LIST="${ENABLE_DENY_LIST}" \
		-p ENABLE_ACCESS_LIST="${ENABLE_ACCESS_LIST}" \
		-p DENIED_USERS="${DENIED_USERS}" \
		-p ACCEPTED_ORGANISATIONS="${ACCEPTED_ORGANISATIONS}" \
		-p OSD_IDP_MAS_SSO_REALM="$(OSD_IDP_MAS_SSO_REALM)" \
		-p ENABLE_KAFKA_SRE_IDENTITY_PROVIDER_CONFIGURATION="${ENABLE_KAFKA_SRE_IDENTITY_PROVIDER_CONFIGURATION}" \
		-p TOKEN_ISSUER_URL="${TOKEN_ISSUER_URL}" \
		-p SERVICE_PUBLIC_HOST_URL="https://$(shell $(OC) get routes/kas-fleet-manager -o jsonpath="{.spec.host}" -n $(NAMESPACE))" \
		-p OBSERVATORIUM_RHSSO_TENANT="${OBSERVATORIUM_RHSSO_TENANT}" \
		-p OBSERVATORIUM_TOKEN_REFRESHER_URL="http://token-refresher.$(NAMESPACE).svc.cluster.local" \
		-p OBSERVABILITY_CONFIG_REPO="${OBSERVABILITY_CONFIG_REPO}" \
		-p OBSERVABILITY_CONFIG_TAG="${OBSERVABILITY_CONFIG_TAG}" \
		-p OBSERVABILITY_ENABLE_CLOUDWATCHLOGGING="${OBSERVABILITY_ENABLE_CLOUDWATCHLOGGING}" \
		-p DATAPLANE_OBSERVABILITY_CONFIG_ENABLE="${DATAPLANE_OBSERVABILITY_CONFIG_ENABLE}" \
		-p ENABLE_TERMS_ACCEPTANCE="${ENABLE_TERMS_ACCEPTANCE}" \
		-p ALLOW_DEVELOPER_INSTANCE="${ALLOW_DEVELOPER_INSTANCE}" \
		-p QUOTA_TYPE="${QUOTA_TYPE}" \
		-p KAS_FLEETSHARD_OLM_INDEX_IMAGE="${KAS_FLEETSHARD_OLM_INDEX_IMAGE}" \
		-p STRIMZI_OLM_INDEX_IMAGE="${STRIMZI_OLM_INDEX_IMAGE}" \
		-p OBSERVABILITY_OPERATOR_INDEX_IMAGE="${OBSERVABILITY_OPERATOR_INDEX_IMAGE}" \
		-p OBSERVABILITY_OPERATOR_STARTING_CSV="${OBSERVABILITY_OPERATOR_STARTING_CSV}" \
		-p STRIMZI_OPERATOR_ADDON_ID="${STRIMZI_OPERATOR_ADDON_ID}" \
		-p KAS_FLEETSHARD_ADDON_ID="${KAS_FLEETSHARD_ADDON_ID}" \
		-p DATAPLANE_CLUSTER_SCALING_TYPE="${DATAPLANE_CLUSTER_SCALING_TYPE}" \
		-p CLUSTER_LOGGING_OPERATOR_ADDON_ID="${CLUSTER_LOGGING_OPERATOR_ADDON_ID}" \
		-p STRIMZI_OLM_PACKAGE_NAME="${STRIMZI_OLM_PACKAGE_NAME}" \
		-p KAS_FLEETSHARD_OLM_PACKAGE_NAME="${KAS_FLEETSHARD_OLM_PACKAGE_NAME}" \
		-p CLUSTER_LIST="${CLUSTER_LIST}" \
		-p SUPPORTED_CLOUD_PROVIDERS=${SUPPORTED_CLOUD_PROVIDERS} \
		-p STRIMZI_OPERATOR_STARTING_CSV=${STRIMZI_OPERATOR_STARTING_CSV} \
		-p KAS_FLEETSHARD_OPERATOR_STARTING_CSV=${KAS_FLEETSHARD_OPERATOR_STARTING_CSV} \
		-p KAS_FLEETSHARD_OPERATOR_SUBSCRIPTION_CONFIG=${KAS_FLEETSHARD_OPERATOR_SUBSCRIPTION_CONFIG} \
		-p STRIMZI_OPERATOR_SUBSCRIPTION_CONFIG=${STRIMZI_OPERATOR_SUBSCRIPTION_CONFIG} \
		-p SSO_PROVIDER_TYPE=${SSO_PROVIDER_TYPE} \
		-p REGISTERED_USERS_PER_ORGANISATION=${REGISTERED_USERS_PER_ORGANISATION} \
		-p DYNAMIC_SCALING_CONFIG=${DYNAMIC_SCALING_CONFIG} \
		-p NODE_PREWARMING_CONFIG=${NODE_PREWARMING_CONFIG} \
		-p ADMIN_AUTHZ_CONFIG=${ADMIN_AUTHZ_CONFIG} \
		-p MAX_ALLOWED_DEVELOPER_INSTANCES="${MAX_ALLOWED_DEVELOPER_INSTANCES}" \
		-p ADMIN_API_SSO_BASE_URL="${ADMIN_API_SSO_BASE_URL}" \
		-p ADMIN_API_SSO_ENDPOINT_URI="${ADMIN_API_SSO_ENDPOINT_URI}" \
		-p ADMIN_API_SSO_REALM="${ADMIN_API_SSO_REALM}" \
		-p KAFKA_TLS_CERTIFICATE_MANAGEMENT_EMAIL=${KAFKA_TLS_CERTIFICATE_MANAGEMENT_EMAIL} \
		-p KAFKA_TLS_CERTIFICATE_MANAGEMENT_STRATEGY=${KAFKA_TLS_CERTIFICATE_MANAGEMENT_STRATEGY} \
		-p KAFKA_TLS_CERTIFICATE_MANAGEMENT_MUST_STAPLE=${KAFKA_TLS_CERTIFICATE_MANAGEMENT_MUST_STAPLE} \
		-p KAFKA_TLS_CERTIFICATE_MANAGEMENT_STORAGE_TYPE=${KAFKA_TLS_CERTIFICATE_MANAGEMENT_STORAGE_TYPE} \
		-p KAFKA_TLS_CERTIFICATE_MANAGEMENT_RENEWAL_WINDOW_RATIO=${KAFKA_TLS_CERTIFICATE_MANAGEMENT_RENEWAL_WINDOW_RATIO} \
		-p KAFKA_TLS_CERTIFICATE_MANAGEMENT_SECURE_STORAGE_CACHE_TTL=${KAFKA_TLS_CERTIFICATE_MANAGEMENT_SECURE_STORAGE_CACHE_TTL} \
		| $(OC) apply -f - -n $(NAMESPACE)
.PHONY: deploy/service

# remove service deployments from an OpenShift cluster
undeploy: IMAGE_REGISTRY ?= $(internal_image_registry)
undeploy: IMAGE_REPOSITORY ?= $(image_repository)
undeploy:
	@-$(OC) process -f ./templates/observatorium-token-refresher.yml | $(OC) delete -f - -n $(NAMESPACE)
	@-$(OC) process -f ./templates/db-template.yml | $(OC) delete -f - -n $(NAMESPACE)
	@-$(OC) process -f ./templates/secrets-template.yml | $(OC) delete -f - -n $(NAMESPACE)
	@-$(OC) process -f ./templates/route-template.yml | $(OC) delete -f - -n $(NAMESPACE)
	@-$(OC) delete -f ./templates/envoy-config-configmap.yml -n $(NAMESPACE)
	@-$(OC) process -f ./templates/service-template.yml \
		-p IMAGE_REGISTRY=$(IMAGE_REGISTRY) \
		-p IMAGE_REPOSITORY=$(IMAGE_REPOSITORY) \
		| $(OC) delete -f - -n $(NAMESPACE)
.PHONY: undeploy

# Deploys an Observatorium token refresher on an OpenShift cluster
deploy/token-refresher: ISSUER_URL ?= "https://sso.redhat.com/auth/realms/redhat-external"
deploy/token-refresher: OBSERVATORIUM_TOKEN_REFRESHER_IMAGE ?= "quay.io/rhoas/mk-token-refresher"
deploy/token-refresher: OBSERVATORIUM_TOKEN_REFRESHER_IMAGE_TAG ?= "latest"
deploy/token-refresher: OBSERVATORIUM_URL ?= "https://observatorium-mst.api.stage.openshift.com/api/metrics/v1/managedkafka"
deploy/token-refresher: OBSERVATORIUM_TOKEN_REFRESHER_REPLICAS ?= "1"
deploy/token-refresher:
	@-$(OC) process -f ./templates/observatorium-token-refresher.yml \
		-p ISSUER_URL=${ISSUER_URL} \
		-p OBSERVATORIUM_URL=${OBSERVATORIUM_URL} \
		-p OBSERVATORIUM_TOKEN_REFRESHER_IMAGE=${OBSERVATORIUM_TOKEN_REFRESHER_IMAGE} \
		-p OBSERVATORIUM_TOKEN_REFRESHER_IMAGE_TAG=${OBSERVATORIUM_TOKEN_REFRESHER_IMAGE_TAG} \
		-p OBSERVATORIUM_TOKEN_REFRESHER_REPLICAS=${OBSERVATORIUM_TOKEN_REFRESHER_REPLICAS} \
		 | $(OC) apply -f - -n $(NAMESPACE)
.PHONY: deploy/token-refresher

deploy/observability-remote-write-proxy/route:
	@$(OC) process -f $(OCP_TEMPLATES_DIR)/observability-remote-write-proxy-route.yml | $(OC) apply -f - -n $(NAMESPACE)

deploy/observability-remote-write-proxy/secrets: OBSERVABILITY_REMOTE_WRITE_PROXY_OIDC_CONFIG_FILEPATH ?= ""
deploy/observability-remote-write-proxy/secrets:
	@if [ -z "$(OBSERVABILITY_REMOTE_WRITE_PROXY_OIDC_CONFIG_FILEPATH)" ]; then echo "OBSERVABILITY_REMOTE_WRITE_PROXY_OIDC_CONFIG_FILEPATH is required"; exit 1; fi
	@if [ ! -f $(OBSERVABILITY_REMOTE_WRITE_PROXY_OIDC_CONFIG_FILEPATH) ]; then echo "'${OBSERVABILITY_REMOTE_WRITE_PROXY_OIDC_CONFIG_FILEPATH}' file cannot be found"; exit 1; fi

	@$(OC) process -f $(OCP_TEMPLATES_DIR)/observability-remote-write-proxy-oidc-secret.yml \
	-p OBSERVABILITY_REMOTE_WRITE_PROXY_OIDC_CONFIG="$(shell cat $(OBSERVABILITY_REMOTE_WRITE_PROXY_OIDC_CONFIG_FILEPATH) | base64 -w 0)" \
	| $(OC) apply -f - -n $(NAMESPACE)
.PHONY: deploy/observability-remote-write-proxy

deploy/observability-remote-write-proxy: OBSERVABILITY_REMOTE_WRITE_PROXY_FORWARD_URL ?= ""
deploy/observability-remote-write-proxy: OBSERVABILITY_REMOTE_WRITE_PROXY_TOKEN_VERIFICATION_ENABLED ?= "true"
deploy/observability-remote-write-proxy: OBSERVABILITY_REMOTE_WRITE_PROXY_TOKEN_VERIFICATION_URL ?= ""
deploy/observability-remote-write-proxy: OBSERVABILITY_REMOTE_WRITE_PROXY_OIDC_ENABLED ?= "true"
deploy/observability-remote-write-proxy: OBSERVABILITY_REMOTE_WRITE_PROXY_TOKEN_VERIFICATION_SERVICE_SERVING_CA_ENABLED ?= "false"
deploy/observability-remote-write-proxy:
	@if [ -z "$(OBSERVABILITY_REMOTE_WRITE_PROXY_FORWARD_URL)" ]; then echo "OBSERVABILITY_REMOTE_WRITE_PROXY_FORWARD_URL is required"; exit 1; fi
	@if [ ! -z "$(OBSERVABILITY_REMOTE_WRITE_PROXY_TOKEN_VERIFICATION_ENABLED)" ] && [ "$(OBSERVABILITY_REMOTE_WRITE_PROXY_TOKEN_VERIFICATION_ENABLED)" == "true" ] && [ -z "$(OBSERVABILITY_REMOTE_WRITE_PROXY_TOKEN_VERIFICATION_URL)" ]; then echo "OBSERVABILITY_REMOTE_WRITE_PROXY_TOKEN_VERIFICATION_URL is required"; exit 1; fi

	@-$(OC) process -f $(OCP_TEMPLATES_DIR)/observability-remote-write-proxy.yml \
		-p OBSERVABILITY_REMOTE_WRITE_PROXY_FORWARD_URL=${OBSERVABILITY_REMOTE_WRITE_PROXY_FORWARD_URL} \
		-p OBSERVABILITY_REMOTE_WRITE_PROXY_OIDC_ENABLED=${OBSERVABILITY_REMOTE_WRITE_PROXY_OIDC_ENABLED} \
		-p OBSERVABILITY_REMOTE_WRITE_PROXY_TOKEN_VERIFICATION_ENABLED=${OBSERVABILITY_REMOTE_WRITE_PROXY_TOKEN_VERIFICATION_ENABLED} \
		-p OBSERVABILITY_REMOTE_WRITE_PROXY_TOKEN_VERIFICATION_URL=${OBSERVABILITY_REMOTE_WRITE_PROXY_TOKEN_VERIFICATION_URL} \
		-p OBSERVABILITY_REMOTE_WRITE_PROXY_TOKEN_VERIFICATION_SERVICE_SERVING_CA_ENABLED=${OBSERVABILITY_REMOTE_WRITE_PROXY_TOKEN_VERIFICATION_SERVICE_SERVING_CA_ENABLED} \
		 | $(OC) apply -f - -n $(NAMESPACE)
.PHONY: deploy/observability-remote-write-proxy

docs/generate/mermaid:
	@for f in $(shell ls $(DOCS_DIR)/mermaid-diagrams-source/*.mmd); do \
		echo Generating diagram for `basename $${f}`; \
		$(DOCKER) run --rm -u $(shell id -u) -it -v $(DOCS_DIR)/mermaid-diagrams-source:/data -v $(DOCS_DIR)/images:/output minlag/mermaid-cli -i /data/`basename $${f}` -o /output/`basename $${f} .mmd`.png; \
	done
.PHONY: docs/generate/mermaid

dataplane/imagepull/secret/setup:
	@echo -n "$${DATA_PLANE_PULL_SECRET}" > secrets/image-pull.dockerconfigjson
.PHONY: dataplane/imagepull/secret/setup

# TODO CRC Deployment stuff

