#!/bin/bash

# This bash script is meant to be run only in a development environment.
# DO NOT run this in production for any reason. There are risks associated with running SQL commands
# based off of defined variables in bash.
# Additionally, because the password for the user is passed through the command line, its possible
# for other users on the system to see the password in `ps` output.

set -e

docker stop managed-services-api-db

docker rm managed-services-api-db

docker network rm managed-services-api-network