# default performance test flags
REGISTRY=quay.io/generic
IMAGE_LOCUST=fleet-manager-locust
TAG_LOCUST=latest
IMAGE_TOKEN_REFRESH=fleet-manager-api-helper
TAG_TOKEN_REFRESH=latest
IMAGE_RESULTS=fleet-manager-perf-results
TAG_RESULTS=latest

PERF_TEST_USERS ?= 150 # number of locust test users - more users - more load can be sent
PERF_TEST_USER_SPAWN_RATE ?= 1 # frequency of user spawning (per second)
PERF_TEST_RUN_TIME ?= 120m # running time (in minutes - as our locustfile expects minutes)
PERF_TEST_WORKERS_NUMBER ?= 25 # number of running worker containers - more containers - more load can be sent
PERF_TEST_PREPOPULATE_DB ?= FALSE # whether to prepopulate db with dinosaur_requests (MUST BE FALSE unless running against non-production environment)
# number of dinosaurs to prepopulate (in deleted state) per worker (only if PERF_TEST_PREPOPULATE_DB == TRUE)
PERF_TEST_PREPOPULATE_DB_DINOSAUR_PER_WORKER ?= 500
# number of dinosaurs to create during performance testing of the API per locust worker
PERF_TEST_DINOSAURS_PER_WORKER ?= 0
# by default only GET endpoints will be attacked
PERF_TEST_GET_ONLY ?= TRUE
# wait time before creating another dinosaur by a worker (in seconds)
PERF_TEST_DINOSAUR_POST_WAIT_TIME ?= 1
# base API url e.g. '/api/dinosaurs_mgmt/v1'
PERF_TEST_BASE_API_URL ?= /api/dinosaurs_mgmt/v1
# number of minutes from the test start to wait before attacking endpoints
PERF_TEST_HIT_ENDPOINTS_HOLD_OFF ?= 0
# if set to TRUE - dinosaur clusters created by the tool will be removed in the cleanup stage
PERF_TEST_CLEANUP ?= TRUE
# create dinosaurs by single worker only
PERF_TEST_SINGLE_WORKER_DINOSAUR_CREATE ?= FALSE
# whether the load should be distributed as per config file
PERF_TEST_USE_CONFIG_FILE_DISTRIBUTION=FALSE

# run performance tests
# requires at least the api route to be set as "PERF_TEST_ROUTE_HOST" and "OCM_OFFLINE_TOKEN"
test/performance:
ifeq (, $(shell which docker-compose 2> /dev/null))
	@echo "Please install docker-compose to run performance tests"
else
	@if [[ -z "${PERF_TEST_ROUTE_HOST}" ]] || [[ -z "${OCM_OFFLINE_TOKEN}" ]]; \
		then echo "Env vars required to run the performance tests (PERF_TEST_ROUTE_HOST or OCM_OFFLINE_TOKEN not provided!" ; exit 1 ; fi;
	
	# might be required by Jenkins to prevent from permissions errors
	export USER_ID=${UID}
	export GROUP_ID=${GID}

	PERF_TEST_ROUTE_HOST=$(PERF_TEST_ROUTE_HOST) PERF_TEST_USERS=$(PERF_TEST_USERS) PERF_TEST_USER_SPAWN_RATE=$(PERF_TEST_USER_SPAWN_RATE) \
		PERF_TEST_RUN_TIME=$(PERF_TEST_RUN_TIME) PERF_TEST_PREPOPULATE_DB=$(PERF_TEST_PREPOPULATE_DB) PERF_TEST_DINOSAURS_PER_WORKER=$(PERF_TEST_DINOSAURS_PER_WORKER) \
			PERF_TEST_PREPOPULATE_DB_DINOSAUR_PER_WORKER=$(PERF_TEST_PREPOPULATE_DB_DINOSAUR_PER_WORKER) PERF_TEST_GET_ONLY=$(PERF_TEST_GET_ONLY) \
				PERF_TEST_DINOSAUR_POST_WAIT_TIME=$(PERF_TEST_DINOSAUR_POST_WAIT_TIME) PERF_TEST_BASE_API_URL=$(PERF_TEST_BASE_API_URL) \
					PERF_TEST_HIT_ENDPOINTS_HOLD_OFF=$(PERF_TEST_HIT_ENDPOINTS_HOLD_OFF) PERF_TEST_CLEANUP=$(PERF_TEST_CLEANUP) ADDITIONAL_LOCUST_OPTS=$(ADDITIONAL_LOCUST_OPTS) \
						PERF_TEST_SINGLE_WORKER_DINOSAUR_CREATE=$(PERF_TEST_SINGLE_WORKER_DINOSAUR_CREATE) UID=${USER_ID} GID=${GROUP_ID} PERF_TEST_USE_CONFIG_FILE_DISTRIBUTION=${PERF_TEST_USE_CONFIG_FILE_DISTRIBUTION} \
			  			docker-compose --file test/performance/docker-compose.yml up --scale secondary=$(PERF_TEST_WORKERS_NUMBER) --remove-orphans
endif
.PHONY: test/performance

.PHONY: test/performance/process/csv
test/performance/process/csv:
	python3 test/performance/scripts/process_results.py

.PHONY: test/performance/image/build
test/performance/image/build:
	docker build -t ${REGISTRY}/${IMAGE_LOCUST}:${TAG_LOCUST} -f $(PWD)/test/performance/Dockerfile .
	docker build -t ${REGISTRY}/${IMAGE_TOKEN_REFRESH}:${TAG_TOKEN_REFRESH} -f $(PWD)/test/performance/api_helper/Dockerfile .
	docker build -t ${REGISTRY}/${IMAGE_RESULTS}:${TAG_RESULTS} -f $(PWD)/test/performance/scripts/Dockerfile .

.PHONY: test/performance/image/push
test/performance/image/push:
	docker push ${REGISTRY}/${IMAGE_LOCUST}:${TAG_LOCUST}
	docker push ${REGISTRY}/${IMAGE_TOKEN_REFRESH}:${TAG_TOKEN_REFRESH}
	docker push ${REGISTRY}/${IMAGE_RESULTS}:${TAG_RESULTS}
