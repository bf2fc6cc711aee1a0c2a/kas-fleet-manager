#!/bin/bash -ex

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
      -z "${PERF_TEST_USERS}" ]] &>/dev/null; then

    echo "Required env vars not provided. Exiting...".
    exit 1
  fi
}

# run perf tests
run_perf_test() {
  make test/performance
}

# convert csv results to JSON
process_perf_test_results() {
  make test/performance/process/csv
}

check_params

run_perf_test

process_perf_test_results
