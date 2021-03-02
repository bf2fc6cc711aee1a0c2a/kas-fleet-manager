Performance tests utilize [locust](https://docs.locust.io/en/stable/api.html) and all the relevant code is available in `./test/performance` folder. Additionally, short living tokens are obtained with help of a small http server running from `test/performance/token_api/main.go`

The docker containers (locust workers, master locust runner and http server) required to run the test can be instantiated using: `test/performance/docker-compose.yml`. 

In order to start the performance test at least these environment variables are required:
- `OCM_OFFLINE_TOKEN` - **long living OCM token** (used to get short-lived access tokens for auth in the http calls)
- `PERF_TEST_ROUTE_HOST` - API route (including `https://`), e.g. `https://managed-services-api-managed-services-pawelpaszki.apps.ppaszki.9wxd.s1.devshift.org` or `https://api.stage.openshift.com` in case of staging OSD cluster (**note that there is no trailing slash**)

Optional parameters (if not provided, they will default to sensible and tested values):

| name                                      | type    | example                                       | description                                                                                                                                                                                                                                                                                |
|-------------------------------------------|---------|-----------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| PERF_TEST_GET_ONLY                        | Boolean | PERF_TEST_GET_ONLY=TRUE                       | If set to TRUE (by default), only GET endpoints will be attached                                                                                                                                                                                                                           |
| PERF_TEST_USERS                           | Integer | PERF_TEST_USERS=100                           | number of locust users per locust worker (more workers means more load can be sent)                                                                                                                                                                                                        |
| PERF_TEST_PREPOPULATE_DB                  | Boolean | PERF_TEST_PREPOPULATE_DB=FALSE                | when set to "TRUE", will pre-seed the database through the API (specified in PERF_TEST_ROUTE_HOST) by creating and then deleting kafka clusters.  **This param should be left out (by default set to "FALSE") when running against staging OSD cluster**. Must be either `TRUE` or `FALSE` |
| PERF_TEST_PREPOPULATE_DB_KAFKA_PER_WORKER | Integer | PERF_TEST_PREPOPULATE_DB_KAFKA_PER_WORKER=500 | number of kafkas to be injected into kafka_requests table per worker  **This param should be left out (by default set to "FALSE") when running against staging OSD cluster**                                                                                                               |
| PERF_TEST_WORKERS_NUMBER                  | Integer | PERF_TEST_WORKERS_NUMBER=125                  | number of locust workers (e.g. docker containers) created during the test (more workers means more load can be sent)                                                                                                                                                                       |
| PERF_TEST_KAFKA_POST_WAIT_TIME            | Integer | PERF_TEST_KAFKA_POST_WAIT_TIME=1              | wait time (in seconds) between creating kafkas by individual locust workers                                                                                                                                                                                                                |
| PERF_TEST_KAFKAS_PER_WORKER               | Integer | PERF_TEST_KAFKAS_PER_WORKER=5                 | number of kafkas created as a part of the performance test execution (per worker). These kafkas will be running for the most of the duration of the test and will be removed one minute before the test completion                                                                         |
| PERF_TEST_RUN_TIME                        | String  | PERF_TEST_RUN_TIME=120m                       | runtime of the performance test. Must be in minutes                                                                                                                                                                                                                                        |
| PERF_TEST_USER_SPAWN_RATE                 | Integer | PERF_TEST_USER_SPAWN_RATE=1                   | The rate per second in which locust users are spawned

To trigger the test (executed from the root of this repo), run:

```
OCM_OFFLINE_TOKEN=<your_ocm_offline_token> PERF_TEST_ROUTE_HOST=https://<your_api_route> make test/performance
```
