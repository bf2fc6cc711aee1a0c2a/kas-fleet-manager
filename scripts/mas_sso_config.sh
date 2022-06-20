#!/bin/bash

TOKEN_PATH="/auth/realms/master/protocol/openid-connect/token"

CLIENT_ID=admin-cli

FLEET_OPERATOR_ROLE=kas_fleetshard_operator
CONNECTOR_OPERATOR_ROLE=connector_fleetshard_operator

# wait for keycloak container to be up or timeout after 2 minutes
ERR_MESSAGE="Keycloak server not running at localhost:8180. No realm configuration will be applied"
timeout 120 bash -c 'while [[ "$(curl -s -o /dev/null -w ''%{http_code}'' http://localhost:8180/auth/realms/master)" != "200" ]]; do echo "Waiting for keycloak server at localhost:8180"; sleep 10; done' || echo "$ERR_MESSAGE"

RESULT=$(curl -sk --data "grant_type=password&client_id=$CLIENT_ID&username=$KEYCLOAK_USER&password=$KEYCLOAK_PASSWORD" "$KEYCLOAK_URL"$TOKEN_PATH)
TOKEN=$(jq -r '.access_token' <<< "$RESULT")
echo "$TOKEN"

CREATE_REALM_RHOAS=$(curl -sk --data-raw '{"enabled":true,"id":"rhoas","realm":"rhoas"}' --header "Content-Type: application/json" --header "Authorization: Bearer $TOKEN" "$KEYCLOAK_URL"/auth/admin/realms)
echo "$CREATE_REALM_RHOAS"
echo "Realm rhoas"

CREATE_REALM_RHOAS_SRE=$(curl -sk --data-raw '{"enabled":true,"id":"rhoas-kafka-sre","realm":"rhoas-kafka-sre"}' --header "Content-Type: application/json" --header "Authorization: Bearer $TOKEN" "$KEYCLOAK_URL"/auth/admin/realms)
echo "$CREATE_REALM_RHOAS_SRE"
echo "Realm rhoas-kafka-sre"

R=$(curl -sk --data-raw '{"name": "'${FLEET_OPERATOR_ROLE}'"}' --header "Content-Type: application/json" --header "Authorization: Bearer $TOKEN" "$KEYCLOAK_URL"/auth/admin/realms/rhoas/roles)
echo "$R"
echo "Realm role fleet"

R=$(curl -sk --data-raw '{"name": "'${CONNECTOR_OPERATOR_ROLE}'"}' --header "Content-Type: application/json" --header "Authorization: Bearer $TOKEN" "$KEYCLOAK_URL"/auth/admin/realms/rhoas/roles)
echo "$R"
echo "Realm role connector fleet"

CREATE=$(curl -sk --data-raw '{
   "authorizationServicesEnabled": false,
   "clientId": "kas-fleet-manager",
   "description": "kas-fleet-manager",
   "name": "kas-fleet-manager",
   "secret":"kas-fleet-manager",
    "directAccessGrantsEnabled": false,
    "serviceAccountsEnabled": true,
    "publicClient": false,
    "protocol": "openid-connect"
}' --header "Content-Type: application/json" --header "Authorization: Bearer $TOKEN" "$KEYCLOAK_URL"/auth/admin/realms/rhoas/clients)
echo "$CREATE"

RE=$(curl -sk --header "Content-Type: application/json" --header "Authorization: Bearer $TOKEN" "$KEYCLOAK_URL"/auth/admin/realms/rhoas/clients?clientId=realm-management)
realmMgmtClientId=$(jq -r '.[].id' <<< "$RE")
echo "$realmMgmtClientId"


ROLES=$(curl -sk --header "Content-Type: application/json" --header "Authorization: Bearer $TOKEN" "$KEYCLOAK_URL"/auth/admin/realms/rhoas/clients/"$realmMgmtClientId"/roles)
manageUser=$(jq -c '.[] | select( .name | contains("manage-users")).id' <<< "$ROLES")
manageClients=$(jq -c '.[] | select( .name | contains("manage-clients")).id' <<< "$ROLES")
manageRealm=$(jq -c '.[] | select( .name | contains("manage-realm")).id' <<< "$ROLES")
echo "$manageUser"
echo "$manageRealm"
echo "$manageClients"


KAS=$(curl -sk --header "Content-Type: application/json" --header "Authorization: Bearer $TOKEN" "$KEYCLOAK_URL"/auth/admin/realms/rhoas/clients?clientId=kas-fleet-manager)
kasClientId=$(jq -r '.[].id' <<< "$KAS")

#/auth/admin/realms/rhoas/clients/de121cf7-a6b2-4d39-a99c-8da787454a66/service-account-user
SVC=$(curl -sk --header "Content-Type: application/json" --header "Authorization: Bearer $TOKEN" "$KEYCLOAK_URL"/auth/admin/realms/rhoas/clients/"$kasClientId"/service-account-user)
svcUserId=$(jq -r '.id' <<< "$SVC")
echo "$svcUserId"

FINAL=$(curl -sk --data-raw '[{"id": '"$manageUser"',"name": "manage-users"},{"id": '"$manageRealm"',"name": "manage-realm"},{"id": '"$manageClients"',"name": "manage-clients"}]' --header "Content-Type: application/json" --header "Authorization: Bearer $TOKEN" "$KEYCLOAK_URL"/auth/admin/realms/rhoas/users/"$svcUserId"/role-mappings/clients/"$realmMgmtClientId")
echo "$FINAL"



CREATE=$(curl -sk --data-raw '{
   "authorizationServicesEnabled": false,
   "clientId": "kas-fleet-manager",
   "description": "kas-fleet-manager",
   "name": "kas-fleet-manager",
   "secret":"kas-fleet-manager",
    "directAccessGrantsEnabled": false,
    "serviceAccountsEnabled": true,
    "publicClient": false,
    "protocol": "openid-connect"
}' --header "Content-Type: application/json" --header "Authorization: Bearer $TOKEN" "$KEYCLOAK_URL"/auth/admin/realms/rhoas-kafka-sre/clients)
echo "$CREATE"

RE=$(curl -sk --header "Content-Type: application/json" --header "Authorization: Bearer $TOKEN" "$KEYCLOAK_URL"/auth/admin/realms/rhoas-kafka-sre/clients?clientId=realm-management)
realmMgmtClientId=$(jq -r '.[].id' <<< "$RE")
echo "$realmMgmtClientId"


ROLES=$(curl -sk --header "Content-Type: application/json" --header "Authorization: Bearer $TOKEN" "$KEYCLOAK_URL"/auth/admin/realms/rhoas-kafka-sre/clients/"$realmMgmtClientId"/roles)
manageUser=$(jq -c '.[] | select( .name | contains("manage-users")).id' <<< "$ROLES")
manageClients=$(jq -c '.[] | select( .name | contains("manage-clients")).id' <<< "$ROLES")
manageRealm=$(jq -c '.[] | select( .name | contains("manage-realm")).id' <<< "$ROLES")
echo "$manageUser"
echo "$manageRealm"
echo "$manageClients"


KAS=$(curl -sk --header "Content-Type: application/json" --header "Authorization: Bearer $TOKEN" "$KEYCLOAK_URL"/auth/admin/realms/rhoas-kafka-sre/clients?clientId=kas-fleet-manager)
kasClientId=$(jq -r '.[].id' <<< "$KAS")

#/auth/admin/realms/rhoas/clients/de121cf7-a6b2-4d39-a99c-8da787454a66/service-account-user
SVC=$(curl -sk --header "Content-Type: application/json" --header "Authorization: Bearer $TOKEN" "$KEYCLOAK_URL"/auth/admin/realms/rhoas-kafka-sre/clients/"$kasClientId"/service-account-user)
svcUserId=$(jq -r '.id' <<< "$SVC")
echo "$svcUserId"

FINAL=$(curl -sk --data-raw '[{"id": '"$manageUser"',"name": "manage-users"},{"id": '"$manageRealm"',"name": "manage-realm"},{"id": '"$manageClients"',"name": "manage-clients"}]' --header "Content-Type: application/json" --header "Authorization: Bearer $TOKEN" "$KEYCLOAK_URL"/auth/admin/realms/rhoas-kafka-sre/users/"$svcUserId"/role-mappings/clients/"$realmMgmtClientId")
echo "$FINAL"
