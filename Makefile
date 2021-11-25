MKFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
PROJECT_PATH := $(patsubst %/,%,$(dir $(MKFILE_PATH)))
DOCS_DIR := $(PROJECT_PATH)/docs

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

DOCKER_CONFIG="${PWD}/.docker"

# Default Variables
ENABLE_OCM_MOCK ?= false
OCM_MOCK_MODE ?= emulate-server
JWKS_URL ?= "https://sso.redhat.com/auth/realms/redhat-external/protocol/openid-connect/certs"
MAS_SSO_BASE_URL ?="https://identity.api.stage.openshift.com"
MAS_SSO_REALM ?="rhoas"
VAULT_KIND ?= tmp

GO := go
GOFMT := gofmt
# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell $(GO) env GOBIN))
GOBIN=$(shell $(GO) env GOPATH)/bin
else
GOBIN=$(shell $(GO) env GOBIN)
endif

LOCAL_BIN_PATH := ${PROJECT_PATH}/bin
# Add the project-level bin directory into PATH. Needed in order
# for `go generate` to use project-level bin directory binaries first
export PATH := ${LOCAL_BIN_PATH}:$(PATH)

GOLANGCI_LINT ?= $(LOCAL_BIN_PATH)/golangci-lint
golangci-lint:
ifeq (, $(shell which $(LOCAL_BIN_PATH)/golangci-lint 2> /dev/null))
	@{ \
	set -e ;\
	VERSION="v1.33.0" ;\
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/$${VERSION}/install.sh | sh -s -- -b ${LOCAL_BIN_PATH} $${VERSION} ;\
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
	$(GO) get -d gotest.tools/gotestsum@v0.6.0 ;\
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
	$(GO) get -d github.com/matryer/moq@v0.2.1 ;\
	mkdir -p ${LOCAL_BIN_PATH} ;\
	$(GO) build -o ${LOCAL_BIN_PATH}/moq github.com/matryer/moq ;\
	rm -rf $$MOQ_TMP_DIR ;\
	}
endif

GOBINDATA ?= ${LOCAL_BIN_PATH}/go-bindata
go-bindata:
ifeq (, $(shell which ${LOCAL_BIN_PATH}/go-bindata 2> /dev/null))
	@{ \
	set -e ;\
	GOBINDATA_TMP_DIR=$$(mktemp -d) ;\
	cd $$GOBINDATA_TMP_DIR ;\
	$(GO) mod init tmp ;\
	$(GO) get -d github.com/go-bindata/go-bindata/v3/...@v3.1.3 ;\
	mkdir -p ${LOCAL_BIN_PATH} ;\
	$(GO) build -o ${LOCAL_BIN_PATH}/go-bindata github.com/go-bindata/go-bindata/v3/go-bindata ;\
	rm -rf $$GOBINDATA_TMP_DIR ;\
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

SPECTRAL ?= ${LOCAL_BIN_PATH}/spectral
NPM ?= "$(shell which npm)"
specinstall:
ifeq (, $(shell which ${NPM} 2> /dev/null))
	@echo "npm is not available please install it to be able to install spectral"
	exit 1
endif
ifeq (, $(shell which ${LOCAL_BIN_PATH}/spectral 2> /dev/null))
	@{ \
	set -e ;\
	mkdir -p ${LOCAL_BIN_PATH} ;\
	mkdir -p ${LOCAL_BIN_PATH}/spectral-installation ;\
	cd ${LOCAL_BIN_PATH} ;\
	${NPM} install --prefix ${LOCAL_BIN_PATH}/spectral-installation @stoplight/spectral ;\
	${NPM} i --prefix ${LOCAL_BIN_PATH}/spectral-installation @rhoas/spectral-ruleset ;\
	ln -s spectral-installation/node_modules/.bin/spectral spectral ;\
	}
endif
openapi/spec/validate: specinstall
	spectral lint openapi/kas-fleet-manager.yaml openapi/kas-fleet-manager-private-admin.yaml


ifeq ($(shell uname -s | tr A-Z a-z), darwin)
        PGHOST:="127.0.0.1"
else
        PGHOST:="172.18.0.22"
endif

### Environment-sourced variables with defaults
# Can be overriden by setting environment var before running
# Example:
#   OCM_ENV=testing make run
#   export OCM_ENV=testing; make run
# Set the environment to development by default
ifndef OCM_ENV
	OCM_ENV:=integration
endif

ifndef TEST_SUMMARY_FORMAT
	TEST_SUMMARY_FORMAT=short-verbose
endif

# Enable Go modules:
export GO111MODULE=on
export GOPROXY=https://proxy.golang.org
export GOPRIVATE=gitlab.cee.redhat.com

ifndef SERVER_URL
	SERVER_URL:=http://localhost:8000
endif

ifndef TEST_TIMEOUT
	ifeq ($(OCM_ENV), integration)
		TEST_TIMEOUT=15m
	else
		TEST_TIMEOUT=5h
	endif
endif

# Prints a list of useful targets.
help:
	@echo "Kafka Service Fleet Manager make targets"
	@echo ""
	@echo "make verify                      verify source code"
	@echo "make lint                        lint go files and .yaml templates"
	@echo "make binary                      compile binaries"
	@echo "make install                     compile binaries and install in GOPATH bin"
	@echo "make run                         run the application"
	@echo "make run/docs                    run swagger and host the api spec"
	@echo "make test                        run unit tests"
	@echo "make test/integration            run integration tests"
	@echo "make code/fix                    format files"
	@echo "make generate                    generate go and openapi modules"
	@echo "make openapi/generate            generate openapi modules"
	@echo "make openapi/validate            validate openapi schema"
	@echo "make image                       build docker image"
	@echo "make push                        push docker image"
	@echo "make project                     create and use the kas-fleet-manager project"
	@echo "make clean                       delete temporary generated files"
	@echo "make setup/git/hooks             setup git hooks"
	@echo "make keycloak/setup              setup mas sso clientId, clientSecret & crt"
	@echo "make kafkacert/setup             setup the kafka certificate used for Kafka Brokers"
	@echo "make observatorium/setup         setup observatorium secrets used by CI"
	@echo "make observatorium/token-refresher/setup" setup a local observatorium token refresher
	@echo "make docker/login/internal       login to an openshift cluster image registry"
	@echo "make image/build/push/internal   build and push image to an openshift cluster image registry."
	@echo "make deploy                      deploy the service via templates to an openshift cluster"
	@echo "make undeploy                    remove the service deployments from an openshift cluster"
	@echo "openapi/spec/validate            validate OpenAPI spec using spectral"
	@echo "$(fake)"
.PHONY: help

# Set git hook path to .githooks/
.PHONY: setup/git/hooks
setup/git/hooks:
	git config core.hooksPath .githooks

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

# Runs linter against go files and .y(a)ml files in the templates directory
# Requires golangci-lint to be installed @ $(go env GOPATH)/bin/golangci-lint
# and spectral installed via npm
lint: golangci-lint specinstall
	$(GOLANGCI_LINT) run \
		./cmd/... \
		./pkg/... \
		./internal/... \
		./test/...

	spectral lint templates/*.yml templates/*.yaml --ignore-unknown-format --ruleset .validate-templates.yaml
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

# Runs the unit tests.
#
# Args:
#   TESTFLAGS: Flags to pass to `go test`. The `-v` argument is always passed.
#
# Examples:
#   make test TESTFLAGS="-run TestSomething"
test: gotestsum
	OCM_ENV=testing $(GOTESTSUM) --junitfile data/results/unit-tests.xml --format $(TEST_SUMMARY_FORMAT) -- -p 1 -v -count=1 $(TESTFLAGS) \
		$(shell go list ./... | grep -v /test)
.PHONY: test

# Precompile everything required for development/test.
test/prepare:
	$(GO) test -i ./internal/kafka/test/integration/... -i ./internal/connector/test/integration/...
.PHONY: test/prepare

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
test/integration/kafka: test/prepare gotestsum
	$(GOTESTSUM) --junitfile data/results/kas-fleet-manager-integration-tests.xml --format $(TEST_SUMMARY_FORMAT) -- -p 1 -ldflags -s -v -timeout $(TEST_TIMEOUT) -count=1 $(TESTFLAGS) \
				./internal/kafka/test/integration/...
.PHONY: test/integration/kafka

test/integration/connector: test/prepare gotestsum
	$(GOTESTSUM) --junitfile data/results/integraton-tests-connector.xml --format $(TEST_SUMMARY_FORMAT) -- -p 1 -ldflags -s -v -timeout $(TEST_TIMEOUT) -count=1 $(TESTFLAGS) \
				./internal/connector/test/integration/...
.PHONY: test/integration/connector

test/integration/connector/cleanup:
	#delete expired keycloak test clients
	$(GO) run ./internal/connector/test/integration/cleanup/main.go
.PHONY: test/integration/connector/cleanup

test/integration: test/integration/kafka test/integration/connector
.PHONY: test/integration

# remove OSD cluster after running tests against real OCM
# requires OCM_OFFLINE_TOKEN env var exported
test/cluster/cleanup:
	./scripts/cleanup_test_cluster.sh
.PHONY: test/cluster/cleanup

# generate files
generate: moq openapi/generate
	$(GO) generate ./...
	$(GO) mod vendor
	$(MOQ) -out ./pkg/client/keycloak/gocloak_moq.go -pkg keycloak vendor/github.com/Nerzal/gocloak/v8 GoCloak:GoCloakMock
.PHONY: generate

# validate the openapi schema
openapi/validate: openapi-generator
	$(OPENAPI_GENERATOR) validate -i openapi/kas-fleet-manager.yaml
	$(OPENAPI_GENERATOR) validate -i openapi/kas-fleet-manager-private.yaml
	$(OPENAPI_GENERATOR) validate -i openapi/kas-fleet-manager-private-admin.yaml
	$(OPENAPI_GENERATOR) validate -i openapi/connector_mgmt.yaml
.PHONY: openapi/validate

# generate the openapi schema and generated package
openapi/generate: openapi/generate/kas-public openapi/generate/kas-private openapi/generate/kas-admin openapi/generate/connector-public openapi/generate/connector-private
.PHONY: openapi/generate

openapi/generate/kas-public: go-bindata openapi-generator
	rm -rf internal/kafka/internal/api/public
	$(OPENAPI_GENERATOR) validate -i openapi/kas-fleet-manager.yaml
	$(OPENAPI_GENERATOR) generate -i openapi/kas-fleet-manager.yaml -g go -o internal/kafka/internal/api/public --package-name public -t openapi/templates --ignore-file-override ./.openapi-generator-ignore
	$(GOFMT) -w internal/kafka/internal/api/public

	mkdir -p .generate/openapi
	cp ./openapi/kas-fleet-manager.yaml .generate/openapi
	$(GOBINDATA) -o ./internal/kafka/internal/generated/bindata.go -pkg generated -mode 420 -modtime 1 -prefix .generate/openapi/ .generate/openapi
	$(GOFMT) -w internal/kafka/internal/generated
	rm -rf .generate/openapi
.PHONY: openapi/generate/kas-public

openapi/generate/kas-private: go-bindata openapi-generator
	rm -rf internal/kafka/internal/api/private
	$(OPENAPI_GENERATOR) validate -i openapi/kas-fleet-manager-private.yaml
	$(OPENAPI_GENERATOR) generate -i openapi/kas-fleet-manager-private.yaml -g go -o internal/kafka/internal/api/private --package-name private -t openapi/templates --ignore-file-override ./.openapi-generator-ignore
	$(GOFMT) -w internal/kafka/internal/api/private
.PHONY: openapi/generate/kas-private

openapi/generate/kas-admin: go-bindata openapi-generator
	rm -rf internal/kafka/internal/api/admin/private
	$(OPENAPI_GENERATOR) validate -i openapi/kas-fleet-manager-private-admin.yaml
	$(OPENAPI_GENERATOR) generate -i openapi/kas-fleet-manager-private-admin.yaml -g go -o internal/kafka/internal/api/admin/private --package-name private -t openapi/templates --ignore-file-override ./.openapi-generator-ignore
	$(GOFMT) -w internal/kafka/internal/api/admin/private
.PHONY: openapi/generate/kas-admin

openapi/generate/connector-public: go-bindata openapi-generator
	rm -rf internal/connector/internal/api/public
	$(OPENAPI_GENERATOR) validate -i openapi/connector_mgmt.yaml
	$(OPENAPI_GENERATOR) generate -i openapi/connector_mgmt.yaml -g go -o internal/connector/internal/api/public --package-name public -t openapi/templates --ignore-file-override ./.openapi-generator-ignore
	$(GOFMT) -w internal/connector/internal/api/public

	mkdir -p .generate/openapi
	cp ./openapi/connector_mgmt.yaml .generate/openapi
	$(GOBINDATA) -o ./internal/connector/internal/generated/bindata.go -pkg generated -mode 420 -modtime 1 -prefix .generate/openapi/ .generate/openapi
	$(GOFMT) -w internal/connector/internal/generated
	rm -rf .generate/openapi
.PHONY: openapi/generate/connector-public

openapi/generate/connector-private: go-bindata openapi-generator
	rm -rf internal/connector/internal/api/private
	$(OPENAPI_GENERATOR) validate -i openapi/connector_mgmt-private.yaml
	$(OPENAPI_GENERATOR) generate -i openapi/connector_mgmt-private.yaml -g go -o internal/connector/internal/api/private --package-name private -t openapi/templates --ignore-file-override ./.openapi-generator-ignore
	$(GOFMT) -w internal/connector/internal/api/private
.PHONY: openapi/generate/connector-private

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
	@echo "Please open http://localhost/"
	docker run -u $(shell id -u) --rm --name swagger_ui_docs -d -p 80:8080 -e URLS="[ \
		{ url: \"./openapi/kas-fleet-manager.yaml\", name: \"Public API\" },\
		{ url: \"./openapi/connector_mgmt.yaml\", name: \"Connector Management API\"},\
		{ url: \"./openapi/kas-fleet-manager-private.yaml\", name: \"Private API\"},\
		{ url: \"./openapi/kas-fleet-manager-private-admin.yaml\", name: \"Private Admin API\"}]"\
		  -v $(PWD)/openapi/:/usr/share/nginx/html/openapi:Z swaggerapi/swagger-ui
.PHONY: run/docs

# Remove Swagger container
run/docs/teardown:
	docker container stop swagger_ui_docs
	docker container rm swagger_ui_docs
.PHONY: run/docs/teardown

cos-fleet-catalog-camel/setup:
	docker run --name cos-fleet-catalog-camel --rm -d -p 9091:8080 quay.io/lburgazzoli/ccs:latest
	mkdir -p config/connector-types
	echo -n "http://localhost:9091" > config/connector-types/cos-fleet-catalog-camel
.PHONY: cos-fleet-catalog-camel/setup

cos-fleet-catalog-camel/teardown:
	docker stop cos-fleet-catalog-camel
	rm config/connector-types/cos-fleet-catalog-camel
.PHONY: cos-fleet-catalog-camel/teardown

db/setup:
	./scripts/local_db_setup.sh
.PHONY: db/setup

db/migrate:
	OCM_ENV=integration $(GO) run ./cmd/kas-fleet-manager migrate
.PHONY: db/migrate

db/teardown:
	./scripts/local_db_teardown.sh
.PHONY: db/teardown

db/login:
	docker exec -u $(shell id -u) -it kas-fleet-manager-db /bin/bash -c "PGPASSWORD=$(shell cat secrets/db.password) psql -d $(shell cat secrets/db.name) -U $(shell cat secrets/db.user)"
.PHONY: db/login

db/generate/insert/cluster:
	@read -r id external_id provider region multi_az<<<"$(shell ocm get /api/clusters_mgmt/v1/clusters/${CLUSTER_ID} | jq '.id, .external_id, .cloud_provider.id, .region.id, .multi_az' | tr -d \" | xargs -n2 echo)";\
	echo -e "Run this command in your database:\n\nINSERT INTO clusters (id, created_at, updated_at, cloud_provider, cluster_id, external_id, multi_az, region, status, provider_type) VALUES ('"$$id"', current_timestamp, current_timestamp, '"$$provider"', '"$$id"', '"$$external_id"', "$$multi_az", '"$$region"', 'cluster_provisioned', 'ocm');";
.PHONY: db/generate/insert/cluster

# Login to docker
docker/login:
	docker --config="${DOCKER_CONFIG}" login -u "${QUAY_USER}" -p "${QUAY_TOKEN}" quay.io
.PHONY: docker/login

# Login to the OpenShift internal registry
docker/login/internal:
	docker login -u kubeadmin -p $(shell oc whoami -t) $(shell oc get route default-route -n openshift-image-registry -o jsonpath="{.spec.host}")
.PHONY: docker/login/internal

# Build the binary and image
image/build: binary
	docker --config="${DOCKER_CONFIG}" build -t "$(external_image_registry)/$(image_repository):$(image_tag)" .
.PHONY: image/build

# Build and push the image
image/push: image/build
	docker --config="${DOCKER_CONFIG}" push "$(external_image_registry)/$(image_repository):$(image_tag)"
.PHONY: image/push

# build binary and image for OpenShift deployment
image/build/internal: IMAGE_TAG ?= $(image_tag)
image/build/internal: binary
	docker build -t "$(shell oc get route default-route -n openshift-image-registry -o jsonpath="{.spec.host}")/$(image_repository):$(IMAGE_TAG)" .
.PHONY: image/build/internal

# push the image to the OpenShift internal registry
image/push/internal: IMAGE_TAG ?= $(image_tag)
image/push/internal:
	docker push "$(shell oc get route default-route -n openshift-image-registry -o jsonpath="{.spec.host}")/$(image_repository):$(IMAGE_TAG)"
.PHONY: image/push/internal

# build and push the image to an OpenShift cluster's internal registry
# namespace used in the image repository must exist on the cluster before running this command. Run `make deploy/project` to create the namespace if not available.
image/build/push/internal: image/build/internal image/push/internal
.PHONY: image/build/push/internal

# Build the binary and test image
image/build/test: binary
	docker build -t "$(test_image)" -f Dockerfile.integration.test .
.PHONY: image/build/test

# Run the test container
test/run: image/build/test
	docker run -u $(shell id -u) --net=host -p 9876:9876 -i "$(test_image)"
.PHONY: test/run

# Setup for AWS credentials
aws/setup:
	@echo -n "$(AWS_ACCOUNT_ID)" > secrets/aws.accountid
	@echo -n "$(AWS_ACCESS_KEY)" > secrets/aws.accesskey
	@echo -n "$(AWS_SECRET_ACCESS_KEY)" > secrets/aws.secretaccesskey
	@echo -n "$(VAULT_ACCESS_KEY)" > secrets/vault.accesskey
	@echo -n "$(VAULT_SECRET_ACCESS_KEY)" > secrets/vault.secretaccesskey
	@echo -n "$(ROUTE53_ACCESS_KEY)" > secrets/aws.route53accesskey
	@echo -n "$(ROUTE53_SECRET_ACCESS_KEY)" > secrets/aws.route53secretaccesskey
.PHONY: aws/setup

# Setup for mas sso credentials
keycloak/setup:
	@echo -n "$(MAS_SSO_CLIENT_ID)" > secrets/keycloak-service.clientId
	@echo -n "$(MAS_SSO_CLIENT_SECRET)" > secrets/keycloak-service.clientSecret
	@echo -n "$(OSD_IDP_MAS_SSO_CLIENT_ID)" > secrets/osd-idp-keycloak-service.clientId
	@echo -n "$(OSD_IDP_MAS_SSO_CLIENT_SECRET)" > secrets/osd-idp-keycloak-service.clientSecret
.PHONY:keycloak/setup

# Setup for the kafka broker certificate
kafkacert/setup:
	@echo -n "$(KAFKA_TLS_CERT)" > secrets/kafka-tls.crt
	@echo -n "$(KAFKA_TLS_KEY)" > secrets/kafka-tls.key
.PHONY:kafkacert/setup

observatorium/setup:
	@echo -n "$(OBSERVATORIUM_CONFIG_ACCESS_TOKEN)" > secrets/observability-config-access.token;
	@echo -n "$(RHSSO_LOGS_CLIENT_ID)" > secrets/rhsso-logs.clientId;
	@echo -n "$(RHSSO_LOGS_CLIENT_SECRET)" > secrets/rhsso-logs.clientSecret;
	@echo -n "$(RHSSO_METRICS_CLIENT_ID)" > secrets/rhsso-metrics.clientId;
	@echo -n "$(RHSSO_METRICS_CLIENT_SECRET)" > secrets/rhsso-metrics.clientSecret;
.PHONY:observatorium/setup

observatorium/token-refresher/setup: PORT ?= 8085
observatorium/token-refresher/setup: IMAGE_TAG ?= latest
observatorium/token-refresher/setup: ISSUER_URL ?= https://sso.redhat.com/auth/realms/redhat-external
observatorium/token-refresher/setup: OBSERVATORIUM_URL ?= https://observatorium-mst.api.stage.openshift.com/api/metrics/v1/managedkafka
observatorium/token-refresher/setup:
	@docker run -d -p ${PORT}:${PORT} \
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
ocm/login:
	@ocm login --url="$(SERVER_URL)" --token="$(OCM_OFFLINE_TOKEN)"
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
	@-oc new-project $(NAMESPACE)
.PHONY: deploy/project

# deploy the postgres database required by the service to an OpenShift cluster
deploy/db:
	oc process -f ./templates/db-template.yml | oc apply -f - -n $(NAMESPACE)
	@time timeout --foreground 3m bash -c "until oc get pods -n $(NAMESPACE) | grep kas-fleet-manager-db | grep -v deploy | grep -q Running; do echo 'database is not ready yet'; sleep 10; done"
.PHONY: deploy/db

# deploys the secrets required by the service to an OpenShift cluster
deploy/secrets:
	@oc get service/kas-fleet-manager-db -n $(NAMESPACE) || (echo "Database is not deployed, please run 'make deploy/db'"; exit 1)
	@oc process -f ./templates/secrets-template.yml \
		-p DATABASE_HOST="$(shell oc get service/kas-fleet-manager-db -o jsonpath="{.spec.clusterIP}")" \
		-p OCM_SERVICE_CLIENT_ID="$(shell ([ -s './secrets/ocm-service.clientId' ] && [ -z '${OCM_SERVICE_CLIENT_ID}' ]) && cat ./secrets/ocm-service.clientId || echo '${OCM_SERVICE_CLIENT_ID}')" \
		-p OCM_SERVICE_CLIENT_SECRET="$(shell ([ -s './secrets/ocm-service.clientSecret' ] && [ -z '${OCM_SERVICE_CLIENT_SECRET}' ]) && cat ./secrets/ocm-service.clientSecret || echo '${OCM_SERVICE_CLIENT_SECRET}')" \
		-p OCM_SERVICE_TOKEN="$(shell ([ -s './secrets/ocm-service.token' ] && [ -z '${OCM_SERVICE_TOKEN}' ]) && cat ./secrets/ocm-service.token || echo '${OCM_SERVICE_TOKEN}')" \
		-p OBSERVATORIUM_SERVICE_TOKEN="$(shell ([ -s './secrets/observatorium.token' ] && [ -z '${OBSERVATORIUM_SERVICE_TOKEN}' ]) && cat ./secrets/observatorium.token || echo '${OBSERVATORIUM_SERVICE_TOKEN}')" \
		-p SENTRY_KEY="$(shell ([ -s './secrets/sentry.key' ] && [ -z '${SENTRY_KEY}' ]) && cat ./secrets/sentry.key || echo '${SENTRY_KEY}')" \
		-p AWS_ACCESS_KEY="$(shell ([ -s './secrets/aws.accesskey' ] && [ -z '${AWS_ACCESS_KEY}' ]) && cat ./secrets/aws.accesskey || echo '${AWS_ACCESS_KEY}')" \
		-p AWS_ACCOUNT_ID="$(shell ([ -s './secrets/aws.accountid' ] && [ -z '${AWS_ACCOUNT_ID}' ]) && cat ./secrets/aws.accountid || echo '${AWS_ACCOUNT_ID}')" \
		-p AWS_SECRET_ACCESS_KEY="$(shell ([ -s './secrets/aws.secretaccesskey' ] && [ -z '${AWS_SECRET_ACCESS_KEY}' ]) && cat ./secrets/aws.secretaccesskey || echo '${AWS_SECRET_ACCESS_KEY}')" \
		-p ROUTE53_ACCESS_KEY="$(shell ([ -s './secrets/aws.route53accesskey' ] && [ -z '${ROUTE53_ACCESS_KEY}' ]) && cat ./secrets/aws.route53accesskey || echo '${ROUTE53_ACCESS_KEY}')" \
		-p ROUTE53_SECRET_ACCESS_KEY="$(shell ([ -s './secrets/aws.route53secretaccesskey' ] && [ -z '${ROUTE53_SECRET_ACCESS_KEY}' ]) && cat ./secrets/aws.route53secretaccesskey || echo '${ROUTE53_SECRET_ACCESS_KEY}')" \
		-p VAULT_ACCESS_KEY="$(shell ([ -s './secrets/vault.accesskey' ] && [ -z '${VAULT_ACCESS_KEY}' ]) && cat ./secrets/vault.accesskey || echo '${VAULT_ACCESS_KEY}')" \
		-p VAULT_SECRET_ACCESS_KEY="$(shell ([ -s './secrets/vault.secretaccesskey' ] && [ -z '${VAULT_SECRET_ACCESS_KEY}' ]) && cat ./secrets/vault.secretaccesskey || echo '${VAULT_SECRET_ACCESS_KEY}')" \
		-p DEX_SECRET="$(shell ([ -s './secrets/dex.secret' ] && [ -z '${DEX_SECRET}' ]) && cat ./secrets/dex.secret || echo '${DEX_SECRET}')" \
		-p DEX_PASSWORD="$(shell ([ -s './secrets/dex.password' ] && [ -z '${DEX_PASSWORD}' ]) && cat ./secrets/dex.password || echo '${DEX_PASSWORD}')" \
		-p MAS_SSO_CLIENT_ID="$(shell ([ -s './secrets/keycloak-service.clientId' ] && [ -z '${MAS_SSO_CLIENT_ID}' ]) && cat ./secrets/keycloak-service.clientId || echo '${MAS_SSO_CLIENT_ID}')" \
		-p MAS_SSO_CLIENT_SECRET="$(shell ([ -s './secrets/keycloak-service.clientSecret' ] && [ -z '${MAS_SSO_CLIENT_SECRET}' ]) && cat ./secrets/keycloak-service.clientSecret || echo '${MAS_SSO_CLIENT_SECRET}')" \
		-p OSD_IDP_MAS_SSO_CLIENT_ID="$(shell ([ -s './secrets/osd-idp-keycloak-service.clientId' ] && [ -z '${OSD_IDP_MAS_SSO_CLIENT_ID}' ]) && cat ./secrets/osd-idp-keycloak-service.clientId || echo '${OSD_IDP_MAS_SSO_CLIENT_ID}')" \
		-p OSD_IDP_MAS_SSO_CLIENT_SECRET="$(shell ([ -s './secrets/osd-idp-keycloak-service.clientSecret' ] && [ -z '${OSD_IDP_MAS_SSO_CLIENT_SECRET}' ]) && cat ./secrets/osd-idp-keycloak-service.clientSecret || echo '${OSD_IDP_MAS_SSO_CLIENT_SECRET}')" \
		-p MAS_SSO_CRT="$(shell ([ -s './secrets/keycloak-service.crt' ] && [ -z '${MAS_SSO_CRT}' ]) && cat ./secrets/keycloak-service.crt || echo '${MAS_SSO_CRT}')" \
		-p KAFKA_TLS_CERT="$(shell ([ -s './secrets/kafka-tls.crt' ] && [ -z '${KAFKA_TLS_CERT}' ]) && cat ./secrets/kafka-tls.crt || echo '${KAFKA_TLS_CERT}')" \
		-p KAFKA_TLS_KEY="$(shell ([ -s './secrets/kafka-tls.key' ] && [ -z '${KAFKA_TLS_KEY}' ]) && cat ./secrets/kafka-tls.key || echo '${KAFKA_TLS_KEY}')" \
		-p OBSERVABILITY_CONFIG_ACCESS_TOKEN="$(shell ([ -s './secrets/observability-config-access.token' ] && [ -z '${OBSERVABILITY_CONFIG_ACCESS_TOKEN}' ]) && cat ./secrets/observability-config-access.token || echo '${OBSERVABILITY_CONFIG_ACCESS_TOKEN}')" \
		-p IMAGE_PULL_DOCKER_CONFIG="$(shell ([ -s './secrets/image-pull.dockerconfigjson' ] && [ -z '${IMAGE_PULL_DOCKER_CONFIG}' ]) && cat ./secrets/image-pull.dockerconfigjson || echo '${IMAGE_PULL_DOCKER_CONFIG}')" \
		-p KUBE_CONFIG="${KUBE_CONFIG}" \
		-p OBSERVABILITY_RHSSO_LOGS_CLIENT_ID="$(shell ([ -s './secrets/rhsso-logs.clientId' ] && [ -z '${OBSERVABILITY_RHSSO_LOGS_CLIENT_ID}' ]) && cat ./secrets/rhsso-logs.clientId || echo '${OBSERVABILITY_RHSSO_LOGS_CLIENT_ID}')" \
		-p OBSERVABILITY_RHSSO_LOGS_SECRET="$(shell ([ -s './secrets/rhsso-logs.clientSecret' ] && [ -z '${OBSERVABILITY_RHSSO_LOGS_SECRET}' ]) && cat ./secrets/rhsso-logs.clientSecret || echo '${OBSERVABILITY_RHSSO_LOGS_SECRET}')" \
		-p OBSERVABILITY_RHSSO_METRICS_CLIENT_ID="$(shell ([ -s './secrets/rhsso-metrics.clientId' ] && [ -z '${OBSERVABILITY_RHSSO_METRICS_CLIENT_ID}' ]) && cat ./secrets/rhsso-metrics.clientId || echo '${OBSERVABILITY_RHSSO_METRICS_CLIENT_ID}')" \
		-p OBSERVABILITY_RHSSO_METRICS_SECRET="$(shell ([ -s './secrets/rhsso-metrics.clientSecret' ] && [ -z '${OBSERVABILITY_RHSSO_METRICS_SECRET}' ]) && cat ./secrets/rhsso-metrics.clientSecret || echo '${OBSERVABILITY_RHSSO_METRICS_SECRET}')" \
		-p OBSERVABILITY_RHSSO_GRAFANA_CLIENT_ID="${OBSERVABILITY_RHSSO_GRAFANA_CLIENT_ID}" \
		-p OBSERVABILITY_RHSSO_GRAFANA_CLIENT_SECRET="${OBSERVABILITY_RHSSO_GRAFANA_CLIENT_SECRET}" \
		| oc apply -f - -n $(NAMESPACE)
.PHONY: deploy/secrets

deploy/envoy:
	@oc apply -f ./templates/envoy-config-configmap.yml -n $(NAMESPACE)
.PHONY: deploy/envoy

deploy/route:
	@oc process -f ./templates/route-template.yml | oc apply -f - -n $(NAMESPACE)
.PHONY: deploy/route

# deploy service via templates to an OpenShift cluster
deploy/service: IMAGE_REGISTRY ?= $(internal_image_registry)
deploy/service: IMAGE_REPOSITORY ?= $(image_repository)
deploy/service: ENV ?= "development"
deploy/service: REPLICAS ?= "1"
deploy/service: ENABLE_KAFKA_EXTERNAL_CERTIFICATE ?= "false"
deploy/service: ENABLE_KAFKA_LIFE_SPAN ?= "false"
deploy/service: KAFKA_LIFE_SPAN ?= "48"
deploy/service: OCM_URL ?= "https://api.stage.openshift.com"
deploy/service: MAS_SSO_ENABLE_AUTH ?= "true"
deploy/service: MAS_SSO_BASE_URL ?= "https://identity.api.stage.openshift.com"
deploy/service: MAS_SSO_REALM ?= "rhoas"
deploy/service: USER_NAME_CLAIM ?= "clientId"
deploy/service: FALL_BACK_USER_NAME_CLAIM ?= "preferred_username"
deploy/service: MAX_ALLOWED_SERVICE_ACCOUNTS ?= "2"
deploy/service: MAX_LIMIT_FOR_SSO_GET_CLIENTS ?= "100"
deploy/service: OSD_IDP_MAS_SSO_REALM ?= "rhoas-kafka-sre"
deploy/service: TOKEN_ISSUER_URL ?= "https://sso.redhat.com/auth/realms/redhat-external"
deploy/service: SERVICE_PUBLIC_HOST_URL ?= "https://api.openshift.com"
deploy/service: ENABLE_TERMS_ACCEPTANCE ?= "false"
deploy/service: ENABLE_DENY_LIST ?= "false"
deploy/service: ALLOW_EVALUATOR_INSTANCE ?= "true"
deploy/service: QUOTA_TYPE ?= "quota-management-list"
deploy/service: STRIMZI_OLM_INDEX_IMAGE ?= "quay.io/osd-addons/managed-kafka:production-82b42db"
deploy/service: KAS_FLEETSHARD_OLM_INDEX_IMAGE ?= "quay.io/osd-addons/kas-fleetshard-operator:production-82b42db"
deploy/service: DEX_USERNAME ?= "admin@example.com"
deploy/service: DEX_URL ?= "http://dex-dex.apps.pbraun-observatorium.observability.rhmw.io"
deploy/service: OBSERVATORIUM_GATEWAY ?= "https://observatorium-observatorium.apps.pbraun-observatorium.observability.rhmw.io"
deploy/service: OBSERVATORIUM_TENANT ?= "test"
deploy/service: OBSERVABILITY_CONFIG_REPO ?= "https://api.github.com/repos/bf2fc6cc711aee1a0c2a/observability-resources-mk/contents"
deploy/service: OBSERVATORIUM_TENANT ?= "test"
deploy/service: OBSERVABILITY_CONFIG_CHANNEL ?= "resources"
deploy/service: OBSERVABILITY_CONFIG_TAG ?= "v1.15.0-staging"
deploy/service: DATAPLANE_CLUSTER_SCALING_TYPE ?= "manual"
deploy/service: STRIMZI_OPERATOR_ADDON_ID ?= "managed-kafka-qe"
deploy/service: KAS_FLEETSHARD_ADDON_ID ?= "kas-fleetshard-operator-qe"
deploy/service: VAULT_KIND ?= "tmp"
deploy/service: deploy/envoy deploy/route
	@if test -z "$(IMAGE_TAG)"; then echo "IMAGE_TAG was not specified"; exit 1; fi
	@time timeout --foreground 3m bash -c "until oc get routes -n $(NAMESPACE) | grep -q kas-fleet-manager; do echo 'waiting for kas-fleet-manager route to be created'; sleep 1; done"
	@oc process -f ./templates/service-template.yml \
		-p ENVIRONMENT="$(ENV)" \
		-p IMAGE_REGISTRY=$(IMAGE_REGISTRY) \
		-p IMAGE_REPOSITORY=$(IMAGE_REPOSITORY) \
		-p IMAGE_TAG=$(IMAGE_TAG) \
		-p REPLICAS="${REPLICAS}" \
		-p ENABLE_KAFKA_EXTERNAL_CERTIFICATE="${ENABLE_KAFKA_EXTERNAL_CERTIFICATE}" \
		-p ENABLE_KAFKA_LIFE_SPAN="${ENABLE_KAFKA_LIFE_SPAN}" \
		-p KAFKA_LIFE_SPAN="${KAFKA_LIFE_SPAN}" \
		-p ENABLE_OCM_MOCK=$(ENABLE_OCM_MOCK) \
		-p OCM_MOCK_MODE=$(OCM_MOCK_MODE) \
		-p OCM_URL="$(OCM_URL)" \
		-p AMS_URL="${AMS_URL}" \
		-p JWKS_URL="$(JWKS_URL)" \
		-p MAS_SSO_ENABLE_AUTH="${MAS_SSO_ENABLE_AUTH}" \
		-p MAS_SSO_BASE_URL="$(MAS_SSO_BASE_URL)" \
		-p MAS_SSO_REALM="$(MAS_SSO_REALM)" \
		-p USER_NAME_CLAIM="$(USER_NAME_CLAIM)" \
		-p FALL_BACK_USER_NAME_CLAIM="$(FALL_BACK_USER_NAME_CLAIM)" \
		-p MAX_ALLOWED_SERVICE_ACCOUNTS="${MAX_ALLOWED_SERVICE_ACCOUNTS}" \
		-p MAX_LIMIT_FOR_SSO_GET_CLIENTS="${MAX_LIMIT_FOR_SSO_GET_CLIENTS}" \
		-p OSD_IDP_MAS_SSO_REALM="$(OSD_IDP_MAS_SSO_REALM)" \
		-p TOKEN_ISSUER_URL="${TOKEN_ISSUER_URL}" \
		-p VAULT_KIND=$(VAULT_KIND) \
		-p SERVICE_PUBLIC_HOST_URL="https://$(shell oc get routes/kas-fleet-manager -o jsonpath="{.spec.host}" -n $(NAMESPACE))" \
		-p OBSERVATORIUM_AUTH_TYPE="${OBSERVATORIUM_AUTH_TYPE}" \
		-p DEX_USERNAME="${DEX_USERNAME}" \
		-p DEX_URL="${DEX_URL}" \
		-p OBSERVATORIUM_GATEWAY="${OBSERVATORIUM_GATEWAY}" \
		-p OBSERVATORIUM_TENANT="${OBSERVATORIUM_TENANT}" \
		-p OBSERVATORIUM_RHSSO_GATEWAY="${OBSERVATORIUM_RHSSO_GATEWAY}" \
		-p OBSERVATORIUM_RHSSO_REALM="${OBSERVATORIUM_RHSSO_REALM}" \
		-p OBSERVATORIUM_RHSSO_TENANT="${OBSERVATORIUM_RHSSO_TENANT}" \
		-p OBSERVATORIUM_RHSSO_AUTH_SERVER_URL="${OBSERVATORIUM_RHSSO_AUTH_SERVER_URL}" \
		-p OBSERVATORIUM_TOKEN_REFRESHER_URL="http://token-refresher.$(NAMESPACE).svc.cluster.local" \
		-p OBSERVABILITY_CONFIG_REPO="${OBSERVABILITY_CONFIG_REPO}" \
		-p OBSERVABILITY_CONFIG_TAG="${OBSERVABILITY_CONFIG_TAG}" \
		-p ENABLE_TERMS_ACCEPTANCE="${ENABLE_TERMS_ACCEPTANCE}" \
		-p ALLOW_EVALUATOR_INSTANCE="${ALLOW_EVALUATOR_INSTANCE}" \
		-p QUOTA_TYPE="${QUOTA_TYPE}" \
		-p KAS_FLEETSHARD_OLM_INDEX_IMAGE="${KAS_FLEETSHARD_OLM_INDEX_IMAGE}" \
		-p STRIMZI_OLM_INDEX_IMAGE="${STRIMZI_OLM_INDEX_IMAGE}" \
		-p STRIMZI_OPERATOR_ADDON_ID="${STRIMZI_OPERATOR_ADDON_ID}" \
		-p KAS_FLEETSHARD_ADDON_ID="${KAS_FLEETSHARD_ADDON_ID}" \
		-p DATAPLANE_CLUSTER_SCALING_TYPE="${DATAPLANE_CLUSTER_SCALING_TYPE}" \
		-p CLUSTER_LOGGING_OPERATOR_ADDON_ID="${CLUSTER_LOGGING_OPERATOR_ADDON_ID}" \
		| oc apply -f - -n $(NAMESPACE)
.PHONY: deploy/service



# remove service deployments from an OpenShift cluster
undeploy: IMAGE_REGISTRY ?= $(internal_image_registry)
undeploy: IMAGE_REPOSITORY ?= $(image_repository)
undeploy:
	@-oc process -f ./templates/observatorium-token-refresher.yml | oc delete -f - -n $(NAMESPACE)
	@-oc process -f ./templates/db-template.yml | oc delete -f - -n $(NAMESPACE)
	@-oc process -f ./templates/secrets-template.yml | oc delete -f - -n $(NAMESPACE)
	@-oc process -f ./templates/route-template.yml | oc delete -f - -n $(NAMESPACE)
	@-oc delete -f ./templates/envoy-config-configmap.yml -n $(NAMESPACE)
	@-oc process -f ./templates/service-template.yml \
		-p IMAGE_REGISTRY=$(IMAGE_REGISTRY) \
		-p IMAGE_REPOSITORY=$(IMAGE_REPOSITORY) \
		| oc delete -f - -n $(NAMESPACE)
.PHONY: undeploy

# Deploys an Observatorium token refresher on an OpenShift cluster
deploy/token-refresher: ISSUER_URL ?= "https://sso.redhat.com/auth/realms/redhat-external"
deploy/token-refresher: OBSERVATORIUM_TOKEN_REFRESHER_IMAGE ?= "quay.io/rhoas/mk-token-refresher"
deploy/token-refresher: OBSERVATORIUM_TOKEN_REFRESHER_IMAGE_TAG ?= "latest"
deploy/token-refresher:
	@-oc process -f ./templates/observatorium-token-refresher.yml \
		-p ISSUER_URL=${ISSUER_URL} \
		-p OBSERVATORIUM_URL=${OBSERVATORIUM_URL} \
		-p OBSERVATORIUM_TOKEN_REFRESHER_IMAGE=${OBSERVATORIUM_TOKEN_REFRESHER_IMAGE} \
		-p OBSERVATORIUM_TOKEN_REFRESHER_IMAGE_TAG=${OBSERVATORIUM_TOKEN_REFRESHER_IMAGE_TAG} \
		 | oc apply -f - -n $(NAMESPACE)
.PHONY: deploy/token-refresher

docs/generate/mermaid:
	@for f in $(shell ls $(DOCS_DIR)/mermaid-diagrams-source/*.mmd); do \
		echo Generating diagram for `basename $${f}`; \
		docker run -it -v $(DOCS_DIR)/mermaid-diagrams-source:/data -v $(DOCS_DIR)/images:/output minlag/mermaid-cli -i /data/`basename $${f}` -o /output/`basename $${f} .mmd`.png; \
	done
.PHONY: docs/generate/mermaid

# TODO CRC Deployment stuff

