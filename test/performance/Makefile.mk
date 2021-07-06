# default performance test flags
REGISTRY=quay.io/rhoas
IMAGE_LOCUST=kas-fleet-manager-locust
TAG_LOCUST=latest
IMAGE_TOKEN_REFRESH=kas-fleet-manager-token-refresh
TAG_TOKEN_REFRESH=latest
IMAGE_RESULTS=kas-fleet-manager-perf-results
TAG_RESULTS=latest
IMAGE_BACKUP=kas-fleet-manager-perf-test-backup
TAG_BACKUP=latest

PERF_TEST_USERS ?= 150 # number of locust test users - more users - more load can be sent
PERF_TEST_USER_SPAWN_RATE ?= 1 # frequency of user spawning (per second)
PERF_TEST_RUN_TIME ?= 120m # running time (in minutes - as our locustfile expects minutes)
PERF_TEST_WORKERS_NUMBER ?= 25 # number of running worker containers - more containers - more load can be sent
PERF_TEST_PREPOPULATE_DB ?= FALSE # whether to prepopulate db with kafka_requests (MUST BE FALSE for staging/producton cluster)
# number of kafkas to prepopulate (in deleted state) per worker (only if PERF_TEST_PREPOPULATE_DB == TRUE)
PERF_TEST_PREPOPULATE_DB_KAFKA_PER_WORKER ?= 500
# number of kafkas to create during performance testing of the API per locust worker
PERF_TEST_KAFKAS_PER_WORKER ?= 0
# by default only GET endpoints will be attacked
PERF_TEST_GET_ONLY ?= TRUE
# wait time before creating another kafka by a worker (in seconds)
PERF_TEST_KAFKA_POST_WAIT_TIME ?= 1
# base API url e.g. '/api/kafkas_mgmt/v1'
PERF_TEST_BASE_API_URL ?= /api/kafkas_mgmt/v1
# number of minutes from the test start to wait before attacking endpoints
PERF_TEST_HIT_ENDPOINTS_HOLD_OFF ?= 0
# if set to TRUE - kafka clusters and service accounts created by the tool will be removed in the cleanup stage
PERF_TEST_CLEANUP ?= TRUE
# create kafkas by single worker only
PERF_TEST_SINGLE_WORKER_KAFKA_CREATE ?= FALSE

# run performance tests
# requires at least the api route to be set as "PERF_TEST_ROUTE_HOST" and "OCM_OFFLINE_TOKEN"
test/performance:
ifeq (, $(shell which docker-compose 2> /dev/null))
	@echo "Please install docker-compose to run performance tests"
else
	@if [[ -z "${PERF_TEST_ROUTE_HOST}" ]] || [[ -z "${OCM_OFFLINE_TOKEN}" ]]; \
		then echo "Env vars required to run the performance tests (PERF_TEST_ROUTE_HOST or OCM_OFFLINE_TOKEN not provided!" ; exit 1 ; fi;
	
	export USER_ID=${UID}
	export GROUP_ID=${GID}

	PERF_TEST_ROUTE_HOST=$(PERF_TEST_ROUTE_HOST) PERF_TEST_USERS=$(PERF_TEST_USERS) PERF_TEST_USER_SPAWN_RATE=$(PERF_TEST_USER_SPAWN_RATE) \
		PERF_TEST_RUN_TIME=$(PERF_TEST_RUN_TIME) PERF_TEST_PREPOPULATE_DB=$(PERF_TEST_PREPOPULATE_DB) PERF_TEST_KAFKAS_PER_WORKER=$(PERF_TEST_KAFKAS_PER_WORKER) \
			PERF_TEST_PREPOPULATE_DB_KAFKA_PER_WORKER=$(PERF_TEST_PREPOPULATE_DB_KAFKA_PER_WORKER) PERF_TEST_GET_ONLY=$(PERF_TEST_GET_ONLY) \
				PERF_TEST_KAFKA_POST_WAIT_TIME=$(PERF_TEST_KAFKA_POST_WAIT_TIME) PERF_TEST_BASE_API_URL=$(PERF_TEST_BASE_API_URL) \
					PERF_TEST_HIT_ENDPOINTS_HOLD_OFF=$(PERF_TEST_HIT_ENDPOINTS_HOLD_OFF) PERF_TEST_CLEANUP=$(PERF_TEST_CLEANUP) ADDITIONAL_LOCUST_OPTS=$(ADDITIONAL_LOCUST_OPTS) \
						PERF_TEST_SINGLE_WORKER_KAFKA_CREATE=$(PERF_TEST_SINGLE_WORKER_KAFKA_CREATE) UID=${USER_ID} GID=${GROUP_ID} \
			  			docker-compose --file test/performance/docker-compose.yml up --scale secondary=$(PERF_TEST_WORKERS_NUMBER) --remove-orphans
endif
.PHONY: test/performance

.PHONY: test/performance/process/csv
test/performance/process/csv:
	python3 test/performance/scripts/process_results.py

.PHONY: test/performance/image/build
test/performance/image/build:
	docker build -t ${REGISTRY}/${IMAGE_LOCUST}:${TAG_LOCUST} -f $(PWD)/test/performance/Dockerfile .
	docker build -t ${REGISTRY}/${IMAGE_TOKEN_REFRESH}:${TAG_TOKEN_REFRESH} -f $(PWD)/test/performance/token_api/Dockerfile .
	docker build -t ${REGISTRY}/${IMAGE_RESULTS}:${TAG_RESULTS} -f $(PWD)/test/performance/scripts/Dockerfile .
	docker build -t ${REGISTRY}/${IMAGE_BACKUP}:${TAG_BACKUP} -f $(PWD)/test/performance/backup/Dockerfile .

.PHONY: test/performance/image/push
test/performance/image/push:
	docker push ${REGISTRY}/${IMAGE_LOCUST}:${TAG_LOCUST}
	docker push ${REGISTRY}/${IMAGE_TOKEN_REFRESH}:${TAG_TOKEN_REFRESH}
	docker push ${REGISTRY}/${IMAGE_RESULTS}:${TAG_RESULTS}
	docker push ${REGISTRY}/${IMAGE_BACKUP}:${TAG_BACKUP}

# admin-api params and make commands
ADMIN_API_RUN_TIME ?= 1m # running time (in minutes)

.PHONY: test/performance/admin-api
test/performance/admin-api:
	@if [[ -z "${ADMIN_API_SSO_AUTH_URL}" ]] || [[ -z "${ADMIN_API_SVC_ACC_IDS}" ]] || [[ -z "${ADMIN_API_SVC_ACC_SECRETS}" ]] || [[ -z "${ADMIN_API_HOSTS}" ]]; \
		then echo "Not all env vars required to run the admin-api tests (ADMIN_API_SSO_AUTH_URL, ADMIN_API_SVC_ACC_IDS, ADMIN_API_SVC_ACC_SECRETS, ADMIN_API_HOSTS were provided!" ; exit 1 ; fi;

	ADMIN_API_SSO_AUTH_URL=$(ADMIN_API_SSO_AUTH_URL) ADMIN_API_SVC_ACC_IDS=$(ADMIN_API_SVC_ACC_IDS) ADMIN_API_SVC_ACC_SECRETS=$(ADMIN_API_SVC_ACC_SECRETS) \
		locust -f test/performance/admin-api/locustfile.py --headless -u 1 --run-time $(ADMIN_API_RUN_TIME) --host $(ADMIN_API_HOSTS) --csv-full-history --csv=test/performance/admin-api/reports/admin_api \
		  --logfile=test/performance/admin-api/reports/admin_api_logfile --html test/performance/admin-api/reports/admin_api_report.html
