# Testing utilities

This document describes the utilities that the codebase provides for writing unit and integration tests.

## Polling
When writing integration tests for the Kas Fleet Manager, you will soon or later need to poll for some event.
While polling itself is not an hard task, we require some standard to be followed so that, in case the test will fail, we will be able to understand what happened, why and how.
That's not trivial and requires a number of steps:
* Logging the start of the polling
* Logging the end of the polling
* Logging each poll attempts
* Limit the number of *poll attempts log* so that they won't flood the log
* much more.

To free you from writing all that code, an extensive set of utilities have been created. Those utilities will allow you to create your own poller for scratch or use one of the pre-defined pollers we use more often.

### Custom poller
The poller object exposes only an interface (`Poller`), so the only way to build a `Poller` object is through the usage of the `Poller Builder`.
The simplest poller that can be built must have at least 3 characteristics:
1. a polling interval
2. a number of polling attempts
3. logic to be executed at each attempt

The builder provides 3 methods to configure those values:
* **IntervalAndRetries**: this method allows configuring how much to wait between each attempt and how many attempts to perform before returning an error. It can be used in alternative of `IntervalAndTimeout`
* **IntervalAndTimeout**: this method allows configuring how much to wait between each attempt and how long to keep retrying before returning an error. It can be used in alternative of `IntervalAndRetries`
* **OnRetry**: this method allows to configure a function to be executed on each retry. The function will contain the code to be executed on each poll and must be defined as:
    ```go
    func(attempt int, maxRetries int) (done bool, err error)
    ```
  where:
    * attempt: is the total attempt executed (comprised the current one)
    * maxRetries: is the total number of attempts that will be performed
    * done (return value) must be `true` if the polling reached the desired state
    * err (return value) must contain any error occurred
  
A very simple poller could be:
```go
poller := NewPollerBuilder(&dbConnectionFactory).
    IntervalAndRetries(10 * time.Second, 10).
	OnRetry(func(attempt int, maxAttempts int) (done bool, err error) {
		return attempt > 5, nil
	}).Build
poller.Poll()
```
This will configure the poller to retry 10 times, waiting 10 seconds between each retry and stopping as soon as the `OnRetry` function will return `true`.

Nothing else needs to be done: the poller will automatically log each retry and any eventual error.

A number of other methods are provided by the poller builder to give you in-depth control in configuring your poller:
* OnStart: this allow you to pass a function to be executed one time when the polling starts
* OnFinish: this allows you to pass a function to be executed when the polling has finished (successfully ot not)
* DisableRetryLog: as the name implies, allows to disable the logging of each retry
* RetryLogMessage: Message to be printed on each retry
* RetryLogMessagef: Same as `RetryLogMessage`, but allowing you to pass a string in the `sprintf` format
* RetryLogFunction: A function to be invoked before each retry
* OutputFunction: the function to be used to write the logs instead of `fmt.Printf`
* MaxRetryLogs: the maximum number of retry logs to be printed. This doesn't mean the poller will print only the first `MaxRetryLogs` logs, but the poller will print one `RetryLog` every `MaxRetryLogs/Number of attempts` attemps. This way the logs are evenly distribuited during the whole execution.
* ShouldLog: allows to pass a function to customise how the poller decides if a retry log should be printed or not
* DumpCluster: allows to specify the id a cluster so that its table record is dumped on each retry
* DumpDB: allows to specify a custom DB Table dump on each retry

### Predefined pollers

#### Cluster pollers
A number of predefined pollers are available to perform the most common tasks waiting for Cluster events.
They can be found into the [cluster_pollers.go](../../internal/kafka/test/common/cluster_pollers.go) file

#### Kafka pollers
As for cluster pollers, a number of predefined pollers are available to perform the most common tasks waiting for Kafka cluster events.
They can be found into the [kafka_pollers.go](../../internal/kafka/test/common/kafka_pollers.go) file
