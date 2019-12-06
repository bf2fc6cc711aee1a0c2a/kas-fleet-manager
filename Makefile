.DEFAULT_GOAL := help

# Prints a list of useful targets.
help:
	@echo ""
	@echo "OpenShift CLuster Manager Example Service"
	@echo ""
	@echo "make verify               verify source code"
	@echo "make lint                 run golangci-lint"
	@echo "make binary               compile binaries"
	@echo "make install              compile binaries and install in GOPATH bin"
	@echo "make run                  run the application"
	@echo "make test                 run unit tests"
	@echo "make test-integration     run integration tests"
	@echo "make generate             generate openapi modules"
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
	go build ./cmd/ocm-example-service
.PHONY: binary

# Install
install: check-gopath
	go install ./cmd/ocm-example-service
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


# Regenerate openapi client and models
generate:
	rm -rf pkg/api/openapi
	openapi-generator generate -i openapi/ocm-example-service.yaml -g go -o pkg/api/openapi
	go generate ./cmd/ocm-example-service
	gofmt -w pkg/api/openapi
.PHONY: generate

run: install
	ocm-example-service migrate
	ocm-example-service serve
.PHONY: run

# TODO CRC Deployment stuff

