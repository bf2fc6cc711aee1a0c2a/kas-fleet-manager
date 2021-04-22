# Performance tests for Managed Service API for Kafka

Performance tests utilize [locust](https://docs.locust.io/en/stable/api.html) and all the relevant code is available in `./test/performance` folder. Additionally, short living tokens are obtained with help of a small http server running from `test/performance/token_api/main.go`

The docker containers (locust workers, master locust runner and http server) required to run the test can be instantiated using: `test/performance/docker-compose.yml`. 

In order to start the performance test at least these environment variables are required:
- `OCM_OFFLINE_TOKEN` - **long living OCM token** (used to get short-lived access tokens for auth in the http calls)
- `PERF_TEST_ROUTE_HOST` - API route (including `https://`), e.g. `https://managed-services-api-managed-services-pawelpaszki.apps.ppaszki.9wxd.s1.devshift.org` or `https://api.stage.openshift.com` in case of staging OSD cluster (**note that there is no trailing slash**)

Optional parameters (if not provided, they will default to sensible and tested values):

| Parameter name                            | Type    | Example                                             | Description                                                                                                                                                                                                                                                                                |
|-------------------------------------------|---------|-----------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| PERF_TEST_GET_ONLY                        | Boolean | PERF_TEST_GET_ONLY=TRUE                             | If set to TRUE (by default), only GET endpoints will be attached                                                                                                                                                                                                                           |
| PERF_TEST_USERS                           | Integer | PERF_TEST_USERS=100                                 | Number of locust users per locust worker (more workers means more load can be sent)                                                                                                                                                                                                        |
| PERF_TEST_PREPOPULATE_DB                  | Boolean | PERF_TEST_PREPOPULATE_DB=FALSE                      | When set to "TRUE", will pre-seed the database through the API (specified in PERF_TEST_ROUTE_HOST) by creating and then deleting kafka clusters.  **This param should be left out (by default set to "FALSE") when running against staging OSD cluster**. Must be either `TRUE` or `FALSE` |
| PERF_TEST_PREPOPULATE_DB_KAFKA_PER_WORKER | Integer | PERF_TEST_PREPOPULATE_DB_KAFKA_PER_WORKER=500       | Number of kafkas to be injected into kafka_requests table per worker  **This param should be left out (by default set to "FALSE") when running against staging OSD cluster**                                                                                                               |
| PERF_TEST_WORKERS_NUMBER                  | Integer | PERF_TEST_WORKERS_NUMBER=125                        | Number of locust workers (e.g. docker containers) created during the test (more workers means more load can be sent)                                                                                                                                                                       |
| PERF_TEST_KAFKA_POST_WAIT_TIME            | Integer | PERF_TEST_KAFKA_POST_WAIT_TIME=1                    | Wait time (in seconds) between creating kafkas by individual locust worker                                                                                                                                                                                                                 |
| PERF_TEST_KAFKAS_PER_WORKER               | Integer | PERF_TEST_KAFKAS_PER_WORKER=5                       | Number of kafkas created as a part of the performance test execution (per worker). These kafkas will be running for the most of the duration of the test and will be removed one minute before the test completion                                                                         |
| PERF_TEST_RUN_TIME                        | String  | PERF_TEST_RUN_TIME=120m                             | Runtime of the performance test. Must be in minutes                                                                                                                                                                                                                                        |
| PERF_TEST_USER_SPAWN_RATE                 | Integer | PERF_TEST_USER_SPAWN_RATE=1                         | The rate per second in which locust users are spawned                                                                                                                                                                                                                                      |
| PERF_TEST_BASE_API_URL                    | String  | PERF_TEST_BASE_API_URL=/api/managed-services-api/v1 | Base API url (excluding 'PERF_TEST_ROUTE_HOST' param and route suffix representing the resource part of the URL (e.g. 'kafkas'))                                                                                                                                                           |
| PERF_TEST_HIT_ENDPOINTS_HOLD_OFF          | Integer | PERF_TEST_HIT_ENDPOINTS_HOLD_OFF=30                 | Wait time (in minutes) before hitting endpoints (doesn't apply to prepopulating DB and creating kafkas). Counted from the start of the test run                                                                                                                                            |
| PERF_TEST_CLEANUP                         | Boolean | PERF_TEST_CLEANUP=TRUE                              | Determines if a cleanup (of kafka clusters and service accounts) will be performed during last 90 seconds of the test execution                                                                                                                                                            |
| PERF_TEST_SINGLE_WORKER_KAFKA_CREATE      | Boolean | PERF_TEST_SINGLE_WORKER_KAFKA_CREATE=FALSE          | If set to true - only one perf test tool worker will be used to create kafka clusters                                                                                                                                                                                                      |
| ADDITIONAL_LOCUST_OPTS                    | String  | ADDITIONAL_LOCUST_OPTS=--only-summary               | Additional flags supported by locust                                                                                                                                                                                                                                                       |

## Run the performance tests

Note.

By default there will be instantaneous results printed in the terminal. In order to turn it off, add `ADDITIONAL_LOCUST_OPTS=--only-summary` to the make command when running the tests, e.g.:

```
OCM_OFFLINE_TOKEN=<your_ocm_offline_token> PERF_TEST_ROUTE_HOST=https://<your_api_route> ADDITIONAL_LOCUST_OPTS=--only-summary make test/performance
```

To trigger the test (executed from the root of this repo), run:

```
OCM_OFFLINE_TOKEN=<your_ocm_offline_token> PERF_TEST_ROUTE_HOST=https://<your_api_route> make test/performance
```

### Sample parameters combinations and expected results

- Run the test for 30 minutes (PERF_TEST_RUN_TIME=30m). For the first 20 minutes (PERF_TEST_HIT_ENDPOINTS_HOLD_OFF=20) - create 50 kafka clusters (PERF_TEST_WORKERS_NUMBER=10 workers * PERF_TEST_KAFKAS_PER_WORKER=5) and periodically (every ~30 seconds per kafka cluster) check kafkas/[id] GET (to check if kafka cluster is ready) and hit random endpoint. After 20 minutes all endpoints (PERF_TEST_GET_ONLY set to **FALSE**) will be attacked at rate of approx 45-50 requests per second

```
PERF_TEST_RUN_TIME=30m PERF_TEST_WORKERS_NUMBER=10 OCM_OFFLINE_TOKEN=<your_ocm_offline_token> PERF_TEST_ROUTE_HOST=https://<your_api_route> PERF_TEST_USERS=23 PERF_TEST_KAFKA_POST_WAIT_TIME=1 PERF_TEST_KAFKAS_PER_WORKER=5 PERF_TEST_GET_ONLY=FALSE PERF_TEST_HIT_ENDPOINTS_HOLD_OFF=20 make test/performance
```

- Run the test for 30 minutes (PERF_TEST_RUN_TIME=30m). Don't create any kafka requests (by default PERF_TEST_KAFKAS_PER_WORKER is set to **0**) and hit GET only endpoints (by default PERF_TEST_GET_ONLY is set to **TRUE**). All GET endpoints will be attacked at rate of approx 180-200 requests per second

```
PERF_TEST_RUN_TIME=30m PERF_TEST_WORKERS_NUMBER=10 OCM_OFFLINE_TOKEN=<your_ocm_offline_token> PERF_TEST_ROUTE_HOST=https://<your_api_route> PERF_TEST_USERS=100 make test/performance
```

## Generating kafka bootstrap URL and service account credentials config file
If PERF_TEST_KAFKAS_PER_WORKER is set to a value greater than 0, for each of kafka clusters created, its config will be persisted to `test/performance/token_api/config.txt`, so that it can be consumed by Running the Service team for kafka load test.

## Cleaning up created kafka clusters/ service accounts
If any kafka clusters were created during the test run, `test/performance/token_api/kafkas.txt` will be populated with the IDs of those kafka clusters. Assuming that any of those kafka clusters become ready during test execution, `test/performance/token_api/serviceaccounts.txt` file will contain IDs of created service accounts. Those two files can be used to cleanup the resources using `test/performance/scripts/cleanup.py`. The script requires three mandatory parameters:

* `API_HOST` - e.g. `https://api.openshift.com` for production API
* `FILE_PATH` - absolute or relative path to the file with the resources IDs, e.g. `kafkas.txt`
* `RESOURCE` - `kafkas` or `serviceaccounts` (must be the resource name used in the api url)

It was proven that calling kafkas DELETE endpoint with high frequency caused App SRE alerts to fire due to too many simultaneous volume delete attempts. Hence there is a default delay of 2 seconds between each http call. To override it, provide timeout value with the following parameter:

* `DELETE_DELAY`, e.g. DELETE_DELAY="2.0"

### Running the cleanup example
```
API_HOST=https://kas-fleet-manager-managed-services-pawelpaszki.apps.ppaszki.qvfs.s1.devshift.org FILE_PATH=kafkas.txt RESOURCE=kafkas python3 cleanup.py
```

## Build and push the images

Make sure login quay.io using a robot account. The credentials are saved under rhoas/robots/ inside vault. 

```
 make test/performance/image/build 

 make test/performance/image/push
```

## Troubleshooting
Very rarely, after stopping the tests manually and starting them again error similar to one below may appear:

```
secondary_1  | Traceback (most recent call last):
secondary_1  |   File "/usr/local/lib/python3.9/site-packages/urllib3/connection.py", line 159, in _new_conn
secondary_1  |     conn = connection.create_connection(
secondary_1  |   File "/usr/local/lib/python3.9/site-packages/urllib3/util/connection.py", line 84, in create_connection
secondary_1  |     raise err
secondary_1  |   File "/usr/local/lib/python3.9/site-packages/urllib3/util/connection.py", line 74, in create_connection
secondary_1  |     sock.connect(sa)
secondary_1  |   File "/usr/local/lib/python3.9/site-packages/gevent/_socket3.py", line 407, in connect
secondary_1  |     raise error(err, strerror(err))
secondary_1  | ConnectionRefusedError: [Errno 111] Connection refused
secondary_1  | 
secondary_1  | During handling of the above exception, another exception occurred:
secondary_1  | 
secondary_1  | Traceback (most recent call last):
secondary_1  |   File "/usr/local/lib/python3.9/site-packages/urllib3/connectionpool.py", line 670, in urlopen
secondary_1  |     httplib_response = self._make_request(
secondary_1  |   File "/usr/local/lib/python3.9/site-packages/urllib3/connectionpool.py", line 392, in _make_request
secondary_1  |     conn.request(method, url, **httplib_request_kw)
secondary_1  |   File "/usr/local/lib/python3.9/http/client.py", line 1255, in request
secondary_1  |     self._send_request(method, url, body, headers, encode_chunked)
secondary_1  |   File "/usr/local/lib/python3.9/http/client.py", line 1301, in _send_request
secondary_1  |     self.endheaders(body, encode_chunked=encode_chunked)
secondary_1  |   File "/usr/local/lib/python3.9/http/client.py", line 1250, in endheaders
secondary_1  |     self._send_output(message_body, encode_chunked=encode_chunked)
secondary_1  |   File "/usr/local/lib/python3.9/http/client.py", line 1010, in _send_output
secondary_1  |     self.send(msg)
secondary_1  |   File "/usr/local/lib/python3.9/http/client.py", line 950, in send
secondary_1  |     self.connect()
secondary_1  |   File "/usr/local/lib/python3.9/site-packages/urllib3/connection.py", line 187, in connect
secondary_1  |     conn = self._new_conn()
secondary_1  |   File "/usr/local/lib/python3.9/site-packages/urllib3/connection.py", line 171, in _new_conn
secondary_1  |     raise NewConnectionError(
secondary_1  | urllib3.exceptions.NewConnectionError: <urllib3.connection.HTTPConnection object at 0x7fd16826b4f0>: Failed to establish a new connection: [Errno 111] Connection refused
```

This will prevent from hitting the endpoints and to resolve this stop the tests manually again and restart the test

# admin-api performance tests

## Prerequisites
Locust 1.4.2 or newer is required to run admin-api tests

## mandatory parameters

In order to run admin-api tests, these parameters are required:
- `ADMIN_API_SSO_AUTH_URL` - SSO cluster used for getting access token (including `https://`)
- `ADMIN_API_SVC_ACC_IDS` - comma-separated service account ids used to communicate with the admin-api
- `ADMIN_API_SVC_ACC_SECRETS` - comma-separated service account secrets used to communicate with the admin-api
- `ADMIN_API_HOSTS` - comma-separated admin-api hosts (including `https://`, but excluding port number)

## optional parameter
- `ADMIN_API_RUN_TIME` - duration of the test in minutes, if not provided - the test will run for one minute

## Running the tests

```
ADMIN_API_SSO_AUTH_URL=<your_sso_url> ADMIN_API_SVC_ACC_IDS=<your_svc_acc_ids> ADMIN_API_SVC_ACC_SECRETS=<your_svc_acc_secrets> ADMIN_API_HOSTS=<your_admin_api_hosts> make test/performance/admin-api
```
