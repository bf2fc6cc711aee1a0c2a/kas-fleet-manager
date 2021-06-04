#!/bin/bash -ex

declare -x START
declare -x STOP

# check if all required env vars are provided. IF not - exit immediately
check_params () {
  if [[ -z "${OCM_OFFLINE_TOKEN}" || \
      -z "${PERF_TEST_ROUTE_HOST}" || \
      -z "${PERF_TEST_SINGLE_WORKER_KAFKA_CREATE}" || \
      -z "${PERF_TEST_CLEANUP}" || \
      -z "${PERF_TEST_HIT_ENDPOINTS_HOLD_OFF}" || \
      -z "${PERF_TEST_KAFKA_POST_WAIT_TIME}" || \
      -z "${PERF_TEST_GET_ONLY}" || \
      -z "${PERF_TEST_KAFKAS_PER_WORKER}" || \
      -z "${PERF_TEST_PREPOPULATE_DB_KAFKA_PER_WORKER}" || \
      -z "${PERF_TEST_PREPOPULATE_DB}" || \
      -z "${PERF_TEST_WORKERS_NUMBER}" || \
      -z "${PERF_TEST_RUN_TIME}" || \
      -z "${PERF_TEST_BASE_API_URL}" || \
      -z "${PERF_TEST_USER_SPAWN_RATE}" || \
      -z "${PERF_TEST_USERS}" || \
      -z "${HORREUM_URL}" || \
      -z "${TEST_NAME}" || \
      -z "${OWNER}" || \
      -z "${ACCESS}" || \
      -z "${KEYCLOAK_URL}" || \
      -z "${HORREUM_USER}" || \
      -z "${HORREUM_PASSWORD}" || \
      -z "${RESULTS_FILENAME}" ]] &>/dev/null; then

    echo "Required env vars not provided. Exiting...".
    exit 1
  fi
}

# run perf tests
run_perf_test() {
  # test started timestamp (used by horreum)
  START=$(date '+%Y-%m-%dT%H:%M:%S.00Z')

  make test/performance

  # test finished timestamp (used by horreum)
  STOP=$(date '+%Y-%m-%dT%H:%M:%S.00Z')
}

# convert csv results to JSON
process_perf_test_results() {
  make test/performance/process/csv
}

# upload test results to horreum
upload_results() {
  if [ -f "$RESULTS_FILENAME" ]; then
    # get short-living SSO token
    TOKEN=$(curl -s -X POST "$KEYCLOAK_URL"/auth/realms/horreum/protocol/openid-connect/token \
        -H 'content-type: application/x-www-form-urlencoded' \
        -d 'username='"$HORREUM_USER"'&password='"$HORREUM_PASSWORD"'&grant_type=password&client_id=horreum-ui' \
        | jq -r .access_token)

    # post results to horreum instance
    status_code=$(curl "$HORREUM_URL"'/api/run/data?test='"$TEST_NAME"'&start='"$START"'&stop='"$STOP"'&owner='"$OWNER"'&access='"$ACCESS" \
        -s -X POST -o /dev/null -w "%{http_code}" -H 'content-type: application/json' \
        -H 'Authorization: Bearer '"$TOKEN" \
        -d @"$RESULTS_FILENAME")

    if [[ ! "$status_code" == *"200"* ]]; then
      echo "There was an issue with posting results to horreum!"
      exit 1
    fi
  else
    echo "No test results found!"
    exit 1
  fi
}

check_params

run_perf_test

process_perf_test_results

upload_results
