.DEFAULT_GOAL := help
SHELL = bash

# The details of the application:
binary:=managed-services-api

# The version needs to be different for each deployment because otherwise the
# cluster will not pull the new image from the internal registry:
version:=$(shell date +%s)

# Default namespace for local deployments
namespace:=managed-services-${USER}

# The name of the image repository needs to start with the name of an existing
# namespace because when the image is pushed to the internal registry of a
# cluster it will assume that that namespace exists and will try to create a
# corresponding image stream inside that namespace. If the namespace doesn't
# exist the push fails. This doesn't apply when the image is pushed to a public
# repository, like `docker.io` or `quay.io`.
image_repository:=$(namespace)/managed-services-api

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

ifeq ($(shell uname -s | tr A-Z a-z), darwin)
        PGHOST:="127.0.0.1"
else
        PGHOST:="172.18.0.22"
endif

# Prints a list of useful targets.
help:
	@echo ""
	@echo "OpenShift Managed Services API"
	@echo ""
	@echo "make verify               verify source code"
	@echo "make lint                 run golangci-lint"
	@echo "make binary               compile binaries"
	@echo "make install              compile binaries and install in GOPATH bin"
	@echo "make run                  run the application"
	@echo "make run/docs             run swagger and host the api spec"
	@echo "make test                 run unit tests"
	@echo "make test-integration     run integration tests"
	@echo "make code/fix             format files"
	@echo "make generate             generate go and openapi modules"
	@echo "make openapi/generate     generate openapi modules"
	@echo "make openapi/validate     validate openapi schema"
	@echo "make image                build docker image"
	@echo "make push                 push docker image"
	@echo "make deploy               deploy via templates to local openshift instance"
	@echo "make undeploy             undeploy from local openshift instance"
	@echo "make project              create and use the UHC project"
	@echo "make clean                delete temporary generated files"
	@echo "$(fake)"
.PHONY: help

### Constants:
version:=$(shell date +%s)

### Envrionment-sourced variables with defaults
# Can be overriden by setting environment var before running
# Example:
#   OCM_ENV=testing make run
#   export OCM_ENV=testing; make run
# Set the environment to development by default
ifndef OCM_ENV
	OCM_ENV:=development
endif

ifndef TEST_SUMMARY_FORMAT
	TEST_SUMMARY_FORMAT=short-verbose
endif

ifndef SERVER_URL
	SERVER_URL:=http://localhost:8000
endif

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
		./pkg/...
.PHONY: verify

# Runs our linter to verify that everything is following best practices
# Requires golangci-lint to be installed @ $(go env GOPATH)/bin/golangci-lint
lint:
	$(GOLANGCI_LINT_BIN) run \
		./cmd/... \
		./pkg/...
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
	OCM_ENV=testing gotestsum --format $(TEST_SUMMARY_FORMAT) -- -p 1 -v $(TESTFLAGS) \
		./pkg/... \
		./cmd/...
.PHONY: test

# Precompile everything required for development/test.
test-prepare: install
	OCM_ENV=testing go test -i ./test/integration/...
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
	OCM_ENV=testing gotestsum --format $(TEST_SUMMARY_FORMAT) -- -p 1 -ldflags -s -v -timeout 1h $(TESTFLAGS) \
			./test/integration
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
	docker run -d -p 80:8080 -e SWAGGER_JSON=/managed-services-api.yaml -v $(PWD)/openapi/managed-services-api.yaml:/managed-services-api.yaml swaggerapi/swagger-ui
.PHONY: run/docs

db/setup:
	./local_db_setup.sh
.PHONY: db/setup

db/teardown:
	./local_db_teardown.sh
.PHONY: db/teardown

db/login:
	docker exec -it managed-services-api-db /bin/bash -c "PGPASSWORD=$(shell cat secrets/db.password) psql -d $(shell cat secrets/db.name) -U $(shell cat secrets/db.user)"
.PHONY: db/login

# Login to docker 
docker/login: 
	docker --config="${DOCKER_CONFIG}" login -u "${QUAY_USER}" -p "${QUAY_TOKEN}" quay.io
.PHONE: docker/login

# Build the binary and image
image/build: binary
	docker --config="${DOCKER_CONFIG}" build -t "$(external_image_registry)/$(image_repository):$(image_tag)" .
.PHONY: image/build

# Build and push the image
image/push: image/build
	docker --config="${DOCKER_CONFIG}" push "$(external_image_registry)/$(image_repository):$(image_tag)"
.PHONY: image/push

# Build the binary and test image 
image/build/test: binary
	docker build -t "$(test_image)" -f Dockerfile.integration.test .
.PHONY: image/build/test

# Run the test container
test/run: image/build/test
	docker run -i "$(test_image)"
.PHONY: test/run

# Setup for AWS credentials
aws/setup:
	@echo -n "$(AWS_ACCOUNT_ID)" > secrets/aws.accountid
	@echo -n "$(AWS_ACCESS_KEY)" > secrets/aws.accesskey
	@echo -n "$(AWS_SECRET_ACCESS_KEY)" > secrets/aws.secretaccesskey
.PHONY: aws/setup

# Setup OCM token
ocm/setup:
	@ocm login --url="$(SERVER_URL)" --token="$(OCM_OFFLINE_TOKEN)"
	@echo -n "$(shell ocm token)" > secrets/ocm-service.token
.PHONY: ocm/setup

# TODO CRC Deployment stuff

