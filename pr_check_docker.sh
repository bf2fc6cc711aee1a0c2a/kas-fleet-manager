#!/bin/bash -ex

# reflects defaults found in pkg/config/db.go
export GORM_DIALECT="postgres"
export GORM_HOST="localhost"
export GORM_PORT="5432"
export GORM_NAME="managed-services-api"
export GORM_USERNAME="managed-services-api"
export GORM_PASSWORD="secret"
export GORM_SSLMODE="disable"
export GORM_DEBUG="false"

export LOGLEVEL="1"
export TEST_SUMMARY_FORMAT="standard-verbose"

# cd /uhc/src/gitlab.cee.redhat.com/service/managed-services-api
# ls -la 
# go version

# install linter
test -f golangci-lint.sh || wget https://install.goreleaser.com/github.com/golangci/golangci-lint.sh
sh ./golangci-lint.sh -d -b "$(go env GOPATH)/bin" v1.18.0

# install gotestsum
which gotestsum || curl -sSL "https://github.com/gotestyourself/gotestsum/releases/download/v0.3.5/gotestsum_0.3.5_linux_amd64.tar.gz" | tar -xz -C "$(go env GOPATH)/bin" gotestsum

# start postgres
which pg_ctl
PGDATA=/var/lib/postgresql/data /usr/lib/postgresql/*/bin/pg_ctl -w stop
PGDATA=/var/lib/postgresql/data /usr/lib/postgresql/*/bin/pg_ctl start -o "-c listen_addresses='*' -p 5432"

# check the code. Then run the unit and integration tests and cleanup cluster (if running against real OCM)
make -k verify test test-integration cluster/cleanup

# required for entrypoint script run by docker to exit and stop container
exit 0