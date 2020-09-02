#!/bin/bash

# This bash script is meant to be run only in a development environment.
# DO NOT run this in production for any reason. There are risks associated with running SQL commands
# based off of defined variables in bash.
# Additionally, because the password for the user is passed through the command line, its possible
# for other users on the system to see the password in `ps` output.

set -e

docker network create --subnet=172.18.0.0/16 ocm-managed-service-api-network || true

docker run \
  --name=ocm-managed-service-api-db \
  --net ocm-managed-service-api-network \
  --ip 172.18.0.22 \
  -e POSTGRES_PASSWORD=$(cat secrets/db.password) \
  -e POSTGRES_USER=$(cat secrets/db.user) \
  -e POSTGRES_DB=$(cat secrets/db.name) \
  -p $(cat secrets/db.port):5432 \
  -d postgres:13