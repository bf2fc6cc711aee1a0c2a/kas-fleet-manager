#!/bin/bash

set -e

docker network create mas-sso-network || true

docker run \
  --name=mas-sso \
  --net mas-sso-network \
  -p $KEYCLOAK_PORT_NO:8080 \
  -e DB_VENDOR=h2  \
  -e KEYCLOAK_USER=${KEYCLOAK_USER} \
  -e KEYCLOAK_PASSWORD=${KEYCLOAK_PASSWORD} \
  -d quay.io/keycloak/keycloak:17.0.1-legacy
