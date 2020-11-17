.DEFAULT_GOAL := help
SHELL = bash

# The details of the application:
binary:=managed-services-api

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
image_repository:=$(NAMESPACE)/managed-services-api

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
test_image:=test/managed-services-api

DOCKER_CONFIG="${PWD}/.docker"

# Default Variables
ENABLE_OCM_MOCK ?= false
OCM_MOCK_MODE ?= emulated-server
JWKS_URL ?= "https://api.openshift.com/.well-known/jwks.json"

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
	@echo "make test-integration     	run integration tests"
	@echo "make code/fix             	format files"
	@echo "make generate             	generate go and openapi modules"
	@echo "make openapi/generate     	generate openapi modules"
	@echo "make openapi/validate     	validate openapi schema"
	@echo "make image                	build docker image"
	@echo "make push                 	push docker image"
	@echo "make project              	create and use the UHC project"
	@echo "make clean                	delete temporary generated files"
	@echo "make setup/git/hooks      	setup git hooks"
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
verify: check-gopath
	go vet \
		./cmd/... \
		./pkg/... \
		./test/...
.PHONY: verify

# Runs our linter to verify that everything is following best practices
# Requires golangci-lint to be installed @ $(go env GOPATH)/bin/golangci-lint
lint:
	$(GOLANGCI_LINT_BIN) run \
		./cmd/... \
		./pkg/... \
		./test/...
.PHONY: lint

# Build binaries
# NOTE it may be necessary to use CGO_ENABLED=0 for backwards compatibility with centos7 if not using centos7
binary: check-gopath
	go build ./cmd/managed-services-api
.PHONY: binary

# Install
install: check-gopath
	go install ./cmd/managed-services-api
.PHONY: install

# Runs the unit tests.
#
# Args:
#   TESTFLAGS: Flags to pass to `go test`. The `-v` argument is always passed.
#
# Examples:
#   make test TESTFLAGS="-run TestSomething"
test: install
	OCM_ENV=testing gotestsum --junitfile reports/unit-tests.xml --format $(TEST_SUMMARY_FORMAT) -- -p 1 -v -count=1 $(TESTFLAGS) \
		./pkg/... \
		./cmd/...
.PHONY: test

# Precompile everything required for development/test.
test-prepare: install
	go test -i ./test/integration/...
.PHONY: test-prepare

# Runs the integration tests.
#
# Args:
#   TESTFLAGS: Flags to pass to `go test`. The `-v` argument is always passed.
#
# Example:
#   make test-integration
#   make test-integration TESTFLAGS="-run TestAccounts"     acts as TestAccounts* and run TestAccountsGet, TestAccountsPost, etc.
#   make test-integration TESTFLAGS="-run TestAccountsGet"  runs TestAccountsGet
#   make test-integration TESTFLAGS="-short"                skips long-run tests
test-integration: test-prepare
	gotestsum --junitfile reports/integraton-tests.xml --format $(TEST_SUMMARY_FORMAT) -- -p 1 -ldflags -s -v -timeout 5h -count=1 $(TESTFLAGS) \
			./test/integration || true
	./scripts/cleanup_test_cluster.sh
.PHONY: test-integration

# generate files
generate: openapi/generate
	go generate ./...
	gofmt -w pkg/api/openapi
.PHONY: generate

# validate the openapi schema
openapi/validate:
	openapi-generator validate -i openapi/managed-services-api.yaml
.PHONY: openapi/validate

# generate the openapi schema
openapi/generate:
	rm -rf pkg/api/openapi
	openapi-generator generate -i openapi/managed-services-api.yaml -g go -o pkg/api/openapi --ignore-file-override ./.openapi-generator-ignore
	openapi-generator validate -i openapi/managed-services-api.yaml
	gofmt -w pkg/api/openapi
.PHONY: openapi/generate

# clean up code and dependencies
code/fix:
	@go mod tidy
	@gofmt -w `find . -type f -name '*.go' -not -path "./vendor/*"`
.PHONY: code/fix

run: install
	managed-services-api migrate
	managed-services-api serve
.PHONY: run

# Run Swagger and host the api docs
run/docs:
	@echo "Please open http://localhost/"
	docker run --name swagger_ui_docs -d -p 80:8080 -e SWAGGER_JSON=/managed-services-api.yaml -v $(PWD)/openapi/managed-services-api.yaml:/managed-services-api.yaml swaggerapi/swagger-ui
.PHONY: run/docs

# Remove Swagger container
run/docs/teardown:
	docker container stop swagger_ui_docs
	docker container rm swagger_ui_docs
.PHONY: run/docs/teardown

db/setup:
	./scripts/local_db_setup.sh
.PHONY: db/setup

db/teardown:
	./scripts/local_db_teardown.sh
.PHONY: db/teardown

db/login:
	docker exec -it managed-services-api-db /bin/bash -c "PGPASSWORD=$(shell cat secrets/db.password) psql -d $(shell cat secrets/db.name) -U $(shell cat secrets/db.user)"
.PHONY: db/login

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
	docker run --net=host -p 9876:9876 -i "$(test_image)"
.PHONY: test/run

# Setup for AWS credentials
aws/setup:
	@echo -n "$(AWS_ACCOUNT_ID)" > secrets/aws.accountid
	@echo -n "$(AWS_ACCESS_KEY)" > secrets/aws.accesskey
	@echo -n "$(AWS_SECRET_ACCESS_KEY)" > secrets/aws.secretaccesskey
.PHONY: aws/setup

# OCM login
ocm/login:
	@ocm login --url="$(SERVER_URL)" --token="$(OCM_OFFLINE_TOKEN)"
.PHONY: ocm/login

# Setup OCM Client ID and Secret
ocm/setup: OCM_CLIENT_ID ?= ocm-ams-testing
ocm/setup: OCM_CLIENT_SECRET ?= 8f0c06c5-a558-4a78-a406-02deb1fd3f17
ocm/setup:
	@echo -n "$(OCM_OFFLINE_TOKEN)" > secrets/ocm-service.token
ifeq ($(OCM_ENV), integration)
	@echo -n "$(OCM_CLIENT_ID)" > secrets/ocm-service.clientId
	@echo -n "$(OCM_CLIENT_SECRET)" > secrets/ocm-service.clientSecret
endif
.PHONY: ocm/setup

# create project where the service will be deployed in an OpenShift cluster
deploy/project:
	@-oc new-project $(NAMESPACE)
.PHONY: deploy/project

# deploy the postgres database required by the service to an OpenShift cluster
deploy/db:
	oc process -f ./templates/db-template.yml | oc apply -f - -n $(NAMESPACE)
	@time timeout --foreground 1m bash -c "until oc get pods | grep managed-services-api-db | grep -v deploy | grep -q Running; do echo 'database is not ready yet'; sleep 10; done"
.PHONY: deploy/db

# deploy service via templates to an OpenShift cluster
deploy: IMAGE_REGISTRY ?= $(internal_image_registry)
deploy: IMAGE_REPOSITORY ?= $(image_repository)
deploy: IMAGE_TAG ?= $(image_tag)
deploy: deploy/db
	@oc process -f ./templates/secrets-template.yml \
		-p OCM_SERVICE_CLIENT_ID="$(OCM_SERVICE_CLIENT_ID)" \
		-p OCM_SERVICE_CLIENT_SECRET="$(OCM_SERVICE_CLIENT_SECRET)" \
		-p OCM_SERVICE_TOKEN="$(OCM_SERVICE_TOKEN)" \
		-p AWS_ACCESS_KEY="$(AWS_ACCESS_KEY)" \
		-p AWS_ACCOUNT_ID="$(AWS_ACCOUNT_ID)" \
		-p AWS_SECRET_ACCESS_KEY="$(AWS_SECRET_ACCESS_KEY)" \
		-p DATABASE_HOST="$(shell oc get service/managed-services-api-db -o jsonpath="{.spec.clusterIP}")" \
		| oc apply -f - -n $(NAMESPACE)
	@oc process -f ./templates/service-template.yml \
		-p IMAGE_REGISTRY=$(IMAGE_REGISTRY) \
		-p IMAGE_REPOSITORY=$(IMAGE_REPOSITORY) \
		-p IMAGE_TAG=$(IMAGE_TAG) \
		-p ENABLE_OCM_MOCK=$(ENABLE_OCM_MOCK) \
		-p OCM_MOCK_MODE=$(OCM_MOCK_MODE) \
		-p JWKS_URL="$(JWKS_URL)" \
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

