# Performance tests for Managed Service API for Dinosaur

Performance tests utilize [locust](https://docs.locust.io/en/stable/api.html) and all the relevant code is available in `./test/performance` folder. The main logic is driven by `test/performance/locustfile.py` file and any changes to the logic should be implemented in this file (or files called by the `locustfile`) 

Additionally, short-living tokens are obtained with help of a small http server running from `test/performance/api_helper/main.go`

The docker containers (locust workers, master locust runner and http server) required to run the test can be instantiated using: `test/performance/docker-compose.yml`. 

In order to start the performance test at least these environment variables are required:
- `OCM_OFFLINE_TOKEN` - **long living OCM token** (used to get short-lived access tokens for auth in the http calls)
- `PERF_TEST_ROUTE_HOST` - API route (including `https://`), e.g. `https://my-generic-service.9wxd.s1.devshift.org` or `https://api.stage.openshift.com` in case of staging OSD cluster (**note that there is no trailing slash**)

Optional parameters (if not provided, they will default to sensible and tested values):

| Parameter name                            | Type    | Example                                             | Description                                                                                                                                                                                                                                                                                |
|-------------------------------------------|---------|-----------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| PERF_TEST_GET_ONLY                        | Boolean | PERF_TEST_GET_ONLY=TRUE                             | If set to TRUE (by default), only GET endpoints will be attached                                                                                                                                                                                                                           |
| PERF_TEST_USERS                           | Integer | PERF_TEST_USERS=100                                 | Number of locust users per locust worker (more workers means more load can be sent)                                                                                                                                                                                                        |
| PERF_TEST_PREPOPULATE_DB                  | Boolean | PERF_TEST_PREPOPULATE_DB=FALSE                      | When set to "TRUE", will pre-seed the database through the API (specified in PERF_TEST_ROUTE_HOST) by creating and then deleting dinosaur clusters.  **This param should be left out (by default set to "FALSE") when running against non-development environments**. Must be either `TRUE` or `FALSE` |
| PERF_TEST_PREPOPULATE_DB_DINOSAUR_PER_WORKER | Integer | PERF_TEST_PREPOPULATE_DB_DINOSAUR_PER_WORKER=500       | Number of dinosaurs to be injected into dinosaur_requests table per worker  **This param should be left out (by default set to "FALSE") when non-development environments**                                                                                                               |
| PERF_TEST_WORKERS_NUMBER                  | Integer | PERF_TEST_WORKERS_NUMBER=125                        | Number of locust workers (e.g. docker containers) created during the test (more workers means more load can be sent)                                                                                                                                                                       |
| PERF_TEST_DINOSAUR_POST_WAIT_TIME            | Integer | PERF_TEST_DINOSAUR_POST_WAIT_TIME=1                    | Wait time (in seconds) between creating dinosaurs by individual locust worker                                                                                                                                                                                                                 |
| PERF_TEST_DINOSAURS_PER_WORKER               | Integer | PERF_TEST_DINOSAURS_PER_WORKER=5                       | Number of dinosaurs created as a part of the performance test execution (per worker). These dinosaurs will be running for the most of the duration of the test and will be removed one minute before the test completion                                                                         |
| PERF_TEST_RUN_TIME                        | String  | PERF_TEST_RUN_TIME=120m                             | Runtime of the performance test. Must be in minutes                                                                                                                                                                                                                                        |
| PERF_TEST_USER_SPAWN_RATE                 | Integer | PERF_TEST_USER_SPAWN_RATE=1                         | The rate per second in which locust users are spawned                                                                                                                                                                                                                                      |
| PERF_TEST_BASE_API_URL                    | String  | PERF_TEST_BASE_API_URL=/api/managed-services-api/v1 | Base API url (excluding 'PERF_TEST_ROUTE_HOST' param and route suffix representing the resource part of the URL (e.g. 'dinosaurs'))                                                                                                                                                           |
| PERF_TEST_HIT_ENDPOINTS_HOLD_OFF          | Integer | PERF_TEST_HIT_ENDPOINTS_HOLD_OFF=30                 | Wait time (in minutes) before hitting endpoints (doesn't apply to prepopulating DB and creating dinosaurs). Counted from the start of the test run                                                                                                                                            |
| PERF_TEST_CLEANUP                         | Boolean | PERF_TEST_CLEANUP=TRUE                              | Determines if a cleanup (of dinosaur clusters and service accounts) will be performed during last 90 seconds of the test execution                                                                                                                                                            |
| PERF_TEST_SINGLE_WORKER_DINOSAUR_CREATE      | Boolean | PERF_TEST_SINGLE_WORKER_DINOSAUR_CREATE=FALSE          | If set to true - only one perf test tool worker will be used to create dinosaur clusters                                                                                                                                                                                                      |
| PERF_TEST_USE_CONFIG_FILE_DISTRIBUTION    | Boolean | PERF_TEST_USE_CONFIG_FILE_DISTRIBUTION=FALSE        | If set to true - file from test/performance/config/load.json will be used to determine the load distribution                                                                                                                                                                               |
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

- Run the test for 30 minutes (PERF_TEST_RUN_TIME=30m). For the first 20 minutes (PERF_TEST_HIT_ENDPOINTS_HOLD_OFF=20) - create 50 dinosaur clusters (PERF_TEST_WORKERS_NUMBER=10 workers * PERF_TEST_DINOSAURS_PER_WORKER=5) and periodically (every ~30 seconds per dinosaur cluster) check dinosaurs/[id] GET (to check if dinosaur cluster is ready) and hit random endpoint. After 20 minutes all endpoints (PERF_TEST_GET_ONLY set to **FALSE**) will be attacked at rate of approx 45-50 requests per second

```
PERF_TEST_RUN_TIME=30m PERF_TEST_WORKERS_NUMBER=10 OCM_OFFLINE_TOKEN=<your_ocm_offline_token> PERF_TEST_ROUTE_HOST=https://<your_api_route> PERF_TEST_USERS=23 PERF_TEST_DINOSAUR_POST_WAIT_TIME=1 PERF_TEST_DINOSAURS_PER_WORKER=5 PERF_TEST_GET_ONLY=FALSE PERF_TEST_HIT_ENDPOINTS_HOLD_OFF=20 make test/performance
```

- Run the test for 30 minutes (PERF_TEST_RUN_TIME=30m). Use `test/performance/config/load.json` file for load distribution

```
PERF_TEST_RUN_TIME=30m PERF_TEST_WORKERS_NUMBER=10 OCM_OFFLINE_TOKEN=<your_ocm_offline_token> PERF_TEST_ROUTE_HOST=https://<your_api_route> PERF_TEST_USERS=100 PERF_TEST_USE_CONFIG_FILE_DISTRIBUTION=TRUE make test/performance
```

## Cleaning up created dinosaur clusters/ service accounts
If any dinosaur clusters were created during the test run, `test/performance/api_helper/dinosaurs.txt` will be populated with the IDs of those dinosaur clusters. `test/performance/scripts/cleanup.py` can be used to cleanup those dinosaurs. The script requires two mandatory parameters:

* `API_HOST` - e.g. `https://api.openshift.com` for production API
* `FILE_PATH` - absolute or relative path to the file with the resources IDs, e.g. `dinosaurs.txt`

Increase `DELETE_DELAY`, if deleting dinosaurs at fast rate causes any issues (e.g. attempting to release OpenShift resources too quickly):

* `DELETE_DELAY`, e.g. DELETE_DELAY="2.0"

### Running the cleanup example
```
API_HOST=https://my-generic-service.qvfs.s1.devshift.org FILE_PATH=dinosaurs.txt python3 cleanup.py
```

## Convert csv results to JSON format accepted by horreum
Test results generated by the testing tool are in CSV format. In order to convert them to JSON schema accepted by Horreum instance run the following command:

```
make test/performance/process/csv
```

This make target runs a script that:

* takes `test/performance/reports/perf_test_stats.csv` file in the following format:

```
Type,Name,Request Count,Failure Count,Median Response Time,Average Response Time,Min Response Time,Max Response Time,Average Content Size,Requests/s,Failures/s,50%,66%,75%,80%,90%,95%,98%,99%,99.9%,99.99%,100%
<http_method>,<endpoint>,<numeric_value>,<numeric_value>,<numeric_value>,<numeric_value>,<numeric_value>,<numeric_value>,<numeric_value>,<numeric_value>,<numeric_value>,<numeric_value>,<numeric_value>,<numeric_value>,<numeric_value>,<numeric_value>,<numeric_value>,<numeric_value>,<numeric_value>,<numeric_value>,<numeric_value>,<numeric_value>
```

* reads all of the data from it and assigns read values to the values in the schema template: `test/performance/templates/results.json` (no data for given endpoint results in all values set to `undefined`). 

* Creates new JSON file `test/performance/reports/perf_test_stats.json` that can be pushed to Horreum instance

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

## Automating API performance tests
`perf_test.sh` file in the root of this repository can be used to run the tests in automated fashion on Jenkins. It requires all of the env vars used across the scripts/ docker images to be specified, otherwise it will fail. Some of the steps could be removed from the script (if desired), e.g. pushing results to Horreum.

If desired, Horreum instance could be setup to upload and store performance tests results. More info can be found [here](https://horreum.hyperfoil.io/)
