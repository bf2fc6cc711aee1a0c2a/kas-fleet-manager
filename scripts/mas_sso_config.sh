#!/bin/bash

MAX_KEYCLOAK_ATTEMPT_COUNT=24

TOKEN_PATH="/auth/realms/master/protocol/openid-connect/token"

CLIENT_ID=admin-cli

FLEET_OPERATOR_ROLE=kas_fleetshard_operator
CONNECTOR_OPERATOR_ROLE=connector_fleetshard_operator

# wait for keycloak container to be up or timeout after 2 minutes

COUNT=0
while [[ $COUNT -lt $MAX_KEYCLOAK_ATTEMPT_COUNT ]]; do
  ((COUNT+=1))
  if [[ $(curl -s -o /dev/null -w ''%{http_code}'' "$KEYCLOAK_URL"/auth/realms/master) == "200" ]]; then
    echo "[$COUNT/$MAX_KEYCLOAK_ATTEMPT_COUNT] - Keycloak ready"
    KEYCLOAK_READY=1
    break
  fi

  echo "[$COUNT/$MAX_KEYCLOAK_ATTEMPT_COUNT] - Waiting for keycloak server at $KEYCLOAK_URL"
  sleep 5
done

if [[ ! $KEYCLOAK_READY ]]; then
  echo "Max number of attempt reached ($MAX_KEYCLOAK_ATTEMPT_COUNT): Keycloak server not running at $KEYCLOAK_URL. No realm configuration will be applied"
  exit 1
fi

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

REALM=rhoas-kafka-sre
REALM_URL="${KEYCLOAK_URL}/auth/admin/realms/${REALM}"

FLEETMANAGER_ADMIN_CLIENT=kas-fleet-manager
FLEETMANAGER_ADMIN_ROLE=kas-fleet-manager-admin-full

# Obtain admin user access token
RESULT=$(curl -sk --data "grant_type=password&client_id=$CLIENT_ID&username=$KEYCLOAK_USER&password=$KEYCLOAK_PASSWORD" "$KEYCLOAK_URL"$TOKEN_PATH)
ACCESS_TOKEN=$(jq -r '.access_token' <<< "$RESULT")
AUTHN_HEADER="Authorization: Bearer ${ACCESS_TOKEN}"
JSON_CONTENT="Content-Type: application/json"

ADMIN_CLIENT_ID=$(curl --fail --show-error -sk -H"${AUTHN_HEADER}" ${REALM_URL}/clients?clientId=${FLEETMANAGER_ADMIN_CLIENT} | jq -r '.[].id')

# attempt to fetch the role id if role already created
ADMIN_FULL_ROLE_ID=$(curl --fail --show-error -sk -H"${AUTHN_HEADER}" ${REALM_URL}/roles |\
    jq -r -c '.[] | select( .name | contains("'${FLEETMANAGER_ADMIN_ROLE}'")).id')

if [ -z $ADMIN_FULL_ROLE_ID ] ; then
    # Only create "kas-fleet-manager-admin-full" role if not already created
    ROLE_CREATE_RESPONSE=$(curl --fail --show-error -sk -H"${JSON_CONTENT}" -H"${AUTHN_HEADER}" ${REALM_URL}/roles --data-raw '{
    "name": "'${FLEETMANAGER_ADMIN_ROLE}'"
    }')

    if [ $? -ne 0 ] ; then
        exit 1
    fi

    echo "Created role: ${FLEETMANAGER_ADMIN_ROLE}"

    # fetch the created role id
    ADMIN_FULL_ROLE_ID=$(curl --fail --show-error -sk -H"${AUTHN_HEADER}" ${REALM_URL}/roles |\
    jq -r -c '.[] | select( .name | contains("'${FLEETMANAGER_ADMIN_ROLE}'")).id')
fi

echo $ADMIN_FULL_ROLE_ID

if [ $? -ne 0 ] ; then
    exit 1
fi

ADMIN_SERVICE_ACCOUNT_ID=$(curl --fail --show-error -sk -H"${AUTHN_HEADER}" ${REALM_URL}/clients/${ADMIN_CLIENT_ID}/service-account-user | jq -r .id)

FINAL=$(curl --fail --show-error -sk -H"${JSON_CONTENT}" -H"${AUTHN_HEADER}" ${REALM_URL}/users/${ADMIN_SERVICE_ACCOUNT_ID}/role-mappings/realm --data-raw '[
  {
    "id": "'${ADMIN_FULL_ROLE_ID}'",
    "name": "'${FLEETMANAGER_ADMIN_ROLE}'"
  }]')
if [ $? -ne 0 ] ; then
    exit 1
fi

echo "Associated role ${FLEETMANAGER_ADMIN_ROLE}(${ADMIN_FULL_ROLE_ID}) with client ${FLEETMANAGER_ADMIN_CLIENT}(${ADMIN_CLIENT_ID}) service account"


