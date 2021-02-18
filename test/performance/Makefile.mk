# default performance test flags
PERF_TEST_USERS ?= 150 # number of locust test users - more users - more load can be sent
PERF_TEST_USER_SPAWN_RATE ?= 1 # frequency of user spawning (per second)
PERF_TEST_RUN_TIME ?= 120m # running time (in minutes - as our locustfile expects minutes)
PERF_TEST_WORKERS_NUMBER ?= 25 # number of running worker containers - more containers - more load can be sent
PERF_TEST_PREPOPULATE_DB ?= FALSE # whether to prepopulate db with kafka_requests (must be FALSE for staging/producton cluster)
# number of kafkas to prepopulate (in deleted state) per worker (only if PERF_TEST_PREPOPULATE_DB == TRUE)
PERF_TEST_PREPOPULATE_DB_KAFKA_PER_WORKER ?= 500
# number of kafkas to create during performance testing of the API per locust worker
PERF_TEST_KAFKAS_PER_WORKER ?= 5

# run performance tests
# requires at least the api route to be set as "PERF_TEST_ROUTE_HOST" and "OCM_OFFLINE_TOKEN"
test/performance:
ifeq (, $(shell which docker-compose 2> /dev/null))
	@echo "Please install docker-compose to run performance tests"
else
	@if [[ -z "${PERF_TEST_ROUTE_HOST}" ]] || [[ -z "${OCM_OFFLINE_TOKEN}" ]]; \
		then echo "Env vars required to run the performance tests (PERF_TEST_ROUTE_HOST or OCM_OFFLINE_TOKEN not provided!" ; exit 1 ; fi;
	
	PERF_TEST_ROUTE_HOST=$(PERF_TEST_ROUTE_HOST) PERF_TEST_USERS=$(PERF_TEST_USERS) PERF_TEST_USER_SPAWN_RATE=$(PERF_TEST_USER_SPAWN_RATE) \
		PERF_TEST_RUN_TIME=$(PERF_TEST_RUN_TIME) PERF_TEST_PREPOPULATE_DB=$(PERF_TEST_PREPOPULATE_DB) PERF_TEST_KAFKAS_PER_WORKER=$(PERF_TEST_KAFKAS_PER_WORKER) \
			PERF_TEST_PREPOPULATE_DB_KAFKA_PER_WORKER=$(PERF_TEST_PREPOPULATE_DB_KAFKA_PER_WORKER) \
				docker-compose --file test/performance/docker-compose.yml up --scale secondary=$(PERF_TEST_WORKERS_NUMBER) --remove-orphans
endif
.PHONY: test/performance