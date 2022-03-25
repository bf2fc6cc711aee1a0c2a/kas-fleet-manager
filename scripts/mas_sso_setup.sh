#!/bin/bash

set -e

docker network create mas-sso-network || true

docker run \
  --name=mas-sso \
  --net mas-sso-network \
  -p 8180:8080 \
  -e DB_VENDOR=h2  \
  -e KEYCLOAK_USER=admin \
  -e KEYCLOAK_PASSWORD=admin \
  -d quay.io/keycloak/keycloak:17.0.1-legacy

