MKFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
PROJECT_PATH := $(patsubst %/,%,$(dir $(MKFILE_PATH)))
include $(PROJECT_PATH)/test/performance/Makefile.mk

.DEFAULT_GOAL := help
SHELL = bash

# The details of the application:
binary:=kas-fleet-manager

# The version needs to be different for each deployment because otherwise the
# cluster will not pull the new image from the internal registry:
version:=$(shell date +%s)

# Default namespace for local deployments
NAMESPACE ?= managed-services-${USER}

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
MAS_SSO_BASE_URL ?="https://keycloak-edge-redhat-rhoam-user-sso.apps.mas-sso-stage.1gzl.s1.devshift.org"
MAS_SSO_REALM ?="mas-sso-playground"

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
	$(GO) get -d github.com/matryer/moq@v0.2.0 ;\
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
	TEST_TIMEOUT=5h
endif

# Prints a list of useful targets.
help:
	@echo ""
	@echo "OpenShift Managed Services API"
	@echo ""
	@echo "make verify               	verify source code"
	@echo "make lint                 	run golangci-lint"
	@echo "make binary               	compile binaries"
	@echo "make install              	compile binaries and install in GOPATH bin"
	@echo "make run                  	run the application"
	@echo "make run/docs             	run swagger and host the api spec"
	@echo "make test                 	run unit tests"
	@echo "make test/integration     	run integration tests"
	@echo "make test/performance      run performance tests"
	@echo "make code/fix             	format files"
	@echo "make generate             	generate go and openapi modules"
	@echo "make openapi/generate     	generate openapi modules"
	@echo "make openapi/validate     	validate openapi schema"
	@echo "make image                	build docker image"
	@echo "make push                 	push docker image"
	@echo "make project              	create and use the UHC project"
	@echo "make clean                	delete temporary generated files"
	@echo "make setup/git/hooks      	setup git hooks"
	@echo "make keycloak/setup     	    setup mas sso clientId, clientSecret & crt"
	@echo "make kafkacert/setup     	    setup the kafka certificate used for Kafka Brokers"
	@echo "observatorium/setup"       setup observatorium secret used by CI  
	@echo "make docker/login/internal	login to an openshift cluster image registry"
	@echo "make image/build/push/internal  build and push image to an openshift cluster image registry."
	@echo "make deploy               	deploy the service via templates to an openshift cluster"
	@echo "make undeploy             	remove the service deployments from an openshift cluster"
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
		./test/...
.PHONY: verify

# Runs our linter to verify that everything is following best practices
# Requires golangci-lint to be installed @ $(go env GOPATH)/bin/golangci-lint
lint: golangci-lint
	$(GOLANGCI_LINT) run \
		./cmd/... \
		./pkg/... \
		./test/...
.PHONY: lint

# Build binaries
# NOTE it may be necessary to use CGO_ENABLED=0 for backwards compatibility with centos7 if not using centos7
binary: lint
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
test: install gotestsum
	OCM_ENV=testing $(GOTESTSUM) --junitfile reports/unit-tests.xml --format $(TEST_SUMMARY_FORMAT) -- -p 1 -v -count=1 $(TESTFLAGS) \
		./pkg/... \
		./cmd/...
.PHONY: test

# Precompile everything required for development/test.
test/prepare: install
	$(GO) test -i ./test/integration/...
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
test/integration: test/prepare gotestsum
	$(GOTESTSUM) --junitfile reports/integraton-tests.xml --format $(TEST_SUMMARY_FORMAT) -- -p 1 -ldflags -s -v -timeout $(TEST_TIMEOUT) -count=1 $(TESTFLAGS) \
			./test/integration/...
.PHONY: test/integration

# remove OSD cluster after running tests against real OCM
# requires OCM_OFFLINE_TOKEN env var exported
test/cluster/cleanup:
	./scripts/cleanup_test_cluster.sh
.PHONY: test/cluster/cleanup

# generate files
generate: moq openapi/generate
	$(GO) generate ./...
	$(GOFMT) -w pkg/api/openapi
	$(GOFMT) -w pkg/api/private/openapi
.PHONY: generate

# validate the openapi schema
openapi/validate: openapi-generator
	$(OPENAPI_GENERATOR) validate -i openapi/kas-fleet-manager.yaml
	$(OPENAPI_GENERATOR) validate -i openapi/kas-fleet-manager-private.yaml
.PHONY: openapi/validate

# generate the openapi schema and data/generated/openapi/openapi.go
openapi/generate: go-bindata openapi-generator
	rm -rf pkg/api/openapi
	rm -rf pkg/api/private/openapi
	$(OPENAPI_GENERATOR) generate -i openapi/kas-fleet-manager.yaml -g go -o pkg/api/openapi --ignore-file-override ./.openapi-generator-ignore
	$(OPENAPI_GENERATOR) validate -i openapi/kas-fleet-manager.yaml
	$(OPENAPI_GENERATOR) generate -i openapi/kas-fleet-manager-private.yaml -g go -o pkg/api/private/openapi --ignore-file-override ./.openapi-generator-ignore
	$(OPENAPI_GENERATOR) validate -i openapi/kas-fleet-manager-private.yaml
	$(GOBINDATA) -o ./data/generated/openapi/openapi.go -pkg openapi -prefix ./openapi/ -mode 420 -modtime 1 ./openapi
	$(GOFMT) -w pkg/api/openapi
	$(GOFMT) -w pkg/api/private/openapi
.PHONY: openapi/generate

# clean up code and dependencies
code/fix:
	@$(GO) mod tidy
	@$(GOFMT) -w `find . -type f -name '*.go' -not -path "./vendor/*"`
.PHONY: code/fix

run: install
	kas-fleet-manager migrate
	kas-fleet-manager serve
.PHONY: run

# Run Swagger and host the api docs
run/docs:
	@echo "Please open http://localhost/"
	docker run -u $(shell id -u) --name swagger_ui_docs -d -p 80:8080 -e URLS="[{ url: \"./openapi/kas-fleet-manager.yaml\", name: \"Public API\" },{url: \"./openapi/kas-fleet-manager-private.yaml\", name: \"Private API\"} ]" -v $(PWD)/openapi/:/usr/share/nginx/html/openapi:Z swaggerapi/swagger-ui
.PHONY: run/docs

# Remove Swagger container
run/docs/teardown:
	docker container stop swagger_ui_docs
	docker container rm swagger_ui_docs
.PHONY: run/docs/teardown

cts/setup:
	docker run --name camel-connector-type-service --rm -d -p 9091:8080 -e QUARKUS_LOG_CONSOLE_JSON=false quay.io/lburgazzoli/cts:quarkus
	echo -n "http://localhost:9091/api/managed-services-api/v1/kafka-connector-types" > config/connector-types/cts
.PHONY: cts/setup

cts/teardown:
	docker stop camel-connector-type-service
	rm config/connector-types/cts
.PHONY: cts/teardown

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
	echo -e "Run this command in your database:\n\nINSERT INTO clusters (id, created_at, updated_at, cloud_provider, cluster_id, external_id, multi_az, region, byoc, managed, status) VALUES ('"$$id"', current_timestamp, current_timestamp, '"$$provider"', '"$$id"', '"$$external_id"', "$$multi_az", '"$$region"', true, true, 'cluster_provisioned');";
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
	@echo -n "$(ROUTE53_ACCESS_KEY)" > secrets/aws.route53accesskey
	@echo -n "$(ROUTE53_SECRET_ACCESS_KEY)" > secrets/aws.route53secretaccesskey
.PHONY: aws/setup

# Setup for mas sso credentials
keycloak/setup:
	@echo -n "$(MAS_SSO_CLIENT_ID)" > secrets/keycloak-service.clientId
	@echo -n "$(MAS_SSO_CLIENT_SECRET)" > secrets/keycloak-service.clientSecret
	@echo -n "$(MAS_SSO_CLIENT_ID)" > secrets/osd-idp-keycloak-service.clientId
	@echo -n "$(MAS_SSO_CLIENT_SECRET)" > secrets/osd-idp-keycloak-service.clientSecret
.PHONY:keycloak/setup

# Setup for the kafka broker certificate
kafkacert/setup:
	@echo -n "$(KAFKA_TLS_CERT)" > secrets/kafka-tls.crt
	@echo -n "$(KAFKA_TLS_KEY)" > secrets/kafka-tls.key
.PHONY:kafkacert/setup

observatorium/setup:
ifeq ($(OCM_ENV), development)
	@echo -n "$(OBSERVATORIUM_CONFIG_ACCESS_TOKEN)" > secrets/observability-config-access.token;
endif
.PHONY:observatorium/setup

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
	@time timeout --foreground 3m bash -c "until oc get pods | grep kas-fleet-manager-db | grep -v deploy | grep -q Running; do echo 'database is not ready yet'; sleep 10; done"
.PHONY: deploy/db

# deploy service via templates to an OpenShift cluster
deploy: IMAGE_REGISTRY ?= $(internal_image_registry)
deploy: IMAGE_REPOSITORY ?= $(image_repository)
deploy: IMAGE_TAG ?= $(image_tag)
deploy: OCM_BASE_URL ?= "https://api.stage.openshift.com"
deploy: MAS_SSO_BASE_URL ?= "https://keycloak-edge-redhat-rhoam-user-sso.apps.mas-sso-stage.1gzl.s1.devshift.org"
deploy: MAS_SSO_REALM ?= "mas-sso-playground"
deploy: OSD_IDP_MAS_SSO_REALM ?= "mas-sso-playground"
deploy: ENABLE_MANAGED_KAFKA_CR ?= "true"
deploy: ENABLE_KAS_FLEETSHARD_OPERATOR_SYNC ?= "false"
deploy: deploy/db
	@oc process -f ./templates/secrets-template.yml \
		-p OCM_SERVICE_CLIENT_ID="$(OCM_SERVICE_CLIENT_ID)" \
		-p OCM_SERVICE_CLIENT_SECRET="$(OCM_SERVICE_CLIENT_SECRET)" \
		-p OCM_SERVICE_TOKEN="$(OCM_SERVICE_TOKEN)" \
		-p OBSERVATORIUM_SERVICE_TOKEN="$(OBSERVATORIUM_SERVICE_TOKEN)" \
		-p AWS_ACCESS_KEY="$(AWS_ACCESS_KEY)" \
		-p AWS_ACCOUNT_ID="$(AWS_ACCOUNT_ID)" \
		-p AWS_SECRET_ACCESS_KEY="$(AWS_SECRET_ACCESS_KEY)" \
		-p MAS_SSO_CLIENT_ID="${MAS_SSO_CLIENT_ID}" \
		-p MAS_SSO_CLIENT_SECRET="${MAS_SSO_CLIENT_SECRET}" \
		-p MAS_SSO_CRT="${MAS_SSO_CRT}" \
		-p DEX_SECRET="${DEX_SECRET}" \
		-p DEX_PASSWORD="${DEX_PASSWORD}" \
		-p ROUTE53_ACCESS_KEY="$(ROUTE53_ACCESS_KEY)" \
		-p ROUTE53_SECRET_ACCESS_KEY="$(ROUTE53_SECRET_ACCESS_KEY)" \
		-p KAFKA_TLS_CERT="$(KAFKA_TLS_CERT)" \
		-p KAFKA_TLS_KEY="$(KAFKA_TLS_KEY)" \
		-p DATABASE_HOST="$(shell oc get service/kas-fleet-manager-db -o jsonpath="{.spec.clusterIP}")" \
		| oc apply -f - -n $(NAMESPACE)
	@oc process -f ./templates/service-template.yml \
		-p ENVIRONMENT="$(OCM_ENV)" \
		-p IMAGE_REGISTRY=$(IMAGE_REGISTRY) \
		-p IMAGE_REPOSITORY=$(IMAGE_REPOSITORY) \
		-p IMAGE_TAG=$(IMAGE_TAG) \
		-p ENABLE_OCM_MOCK=$(ENABLE_OCM_MOCK) \
		-p OCM_MOCK_MODE=$(OCM_MOCK_MODE) \
		-p OCM_BASE_URL="$(OCM_BASE_URL)" \
		-p JWKS_URL="$(JWKS_URL)" \
		-p MAS_SSO_BASE_URL="$(MAS_SSO_BASE_URL)" \
		-p MAS_SSO_REALM="$(MAS_SSO_REALM)" \
		-p OSD_IDP_MAS_SSO_REALM="$(OSD_IDP_MAS_SSO_REALM)" \
		-p ENABLE_MANAGED_KAFKA_CR="$(ENABLE_MANAGED_KAFKA_CR)" \
		-p ENABLE_KAS_FLEETSHARD_OPERATOR_SYNC="$(ENABLE_KAS_FLEETSHARD_OPERATOR_SYNC)" \
		| oc apply -f - -n $(NAMESPACE)
	@oc process -f ./templates/route-template.yml | oc apply -f - -n $(NAMESPACE)
.PHONY: deploy

# remove service deployments from an OpenShift cluster
undeploy: IMAGE_REGISTRY ?= $(internal_image_registry)
undeploy: IMAGE_REPOSITORY ?= $(image_repository)
undeploy:
	@-oc process -f ./templates/db-template.yml | oc delete -f - -n $(NAMESPACE)
	@-oc process -f ./templates/secrets-template.yml | oc delete -f - -n $(NAMESPACE)
	@-oc process -f ./templates/route-template.yml | oc delete -f - -n $(NAMESPACE)
	@-oc process -f ./templates/service-template.yml \
		-p IMAGE_REGISTRY=$(IMAGE_REGISTRY) \
		-p IMAGE_REPOSITORY=$(IMAGE_REPOSITORY) \
		| oc delete -f - -n $(NAMESPACE)
.PHONY: undeploy

# TODO CRC Deployment stuff

