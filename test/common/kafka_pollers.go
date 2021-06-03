package common

import (
	"context"
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"net/http"
	"time"
)

const (
	defaultKafkaReadyTimeout             = 30 * time.Minute
	defaultKafkaClusterAssignmentTimeout = 2 * time.Minute
)

// WaitForNumberOfKafkaToBeGivenCount - Awaits for the number of kafkas to be exactly X
func WaitForNumberOfKafkaToBeGivenCount(ctx context.Context, client *openapi.APIClient, count int32) error {
	currentCount := int32(-1)

	return NewPollerBuilder().
		IntervalAndTimeout(defaultPollInterval, defaultKafkaPollTimeout).
		RetryLogFunction(func(retry int, maxRetry int) string {
			if currentCount == -1 {
				return fmt.Sprintf("Waiting for kafkas count to become %d", count)
			} else {
				return fmt.Sprintf("Waiting for kafkas count to become %d (current %d)", count, currentCount)
			}
		}).
		OnRetry(func(attempt int, maxRetries int) (done bool, err error) {
			if list, _, err := client.DefaultApi.GetKafkas(ctx, nil); err != nil {
				return false, err
			} else {
				currentCount = list.Size
				return currentCount == count, err
			}
		}).
		Build().Poll()
}

// WaitForKafkaCreateToBeAccepted - Creates a kafka and awaits for the request to be accepted
func WaitForKafkaCreateToBeAccepted(ctx context.Context, client *openapi.APIClient, k openapi.KafkaRequestPayload) (kafka openapi.KafkaRequest, resp *http.Response, err error) {
	currentStatus := ""

	err = NewPollerBuilder().
		IntervalAndTimeout(defaultPollInterval, defaultKafkaPollTimeout).
		RetryLogFunction(func(retry int, maxRetry int) string {
			if currentStatus == "" {
				return "Waiting for kafka creation to be accepted"
			} else {
				return fmt.Sprintf("Waiting for kafka creation to be accepted (current status %s)", currentStatus)
			}
		}).
		OnRetry(func(attempt int, maxRetries int) (done bool, err error) {
			kafka, resp, err = client.DefaultApi.CreateKafka(ctx, true, k)
			if err != nil {
				return true, err
			}
			return resp.StatusCode == http.StatusAccepted, err
		}).
		Build().Poll()
	return kafka, resp, err
}

// WaitForKafkaToReachStatus - Awaits for a kafka to reach a specified status
func WaitForKafkaToReachStatus(ctx context.Context, client *openapi.APIClient, kafkaId string, status constants.KafkaStatus) (kafka openapi.KafkaRequest, err error) {
	currentStatus := ""

	err = NewPollerBuilder().
		IntervalAndTimeout(1*time.Second, defaultKafkaReadyTimeout).
		RetryLogFunction(func(retry int, maxRetry int) string {
			if currentStatus == "" {
				return fmt.Sprintf("Waiting for kafka '%s' to reach status '%s'", kafkaId, status.String())
			} else {
				return fmt.Sprintf("Waiting for kafka '%s' to reach status '%s' (current status %s)", kafkaId, status.String(), currentStatus)
			}
		}).
		OnRetry(func(attempt int, maxRetries int) (done bool, err error) {
			kafka, _, err = client.DefaultApi.GetKafkaById(ctx, kafkaId)
			if err != nil {
				return true, err
			}

			switch kafka.Status {
			case constants.KafkaRequestStatusFailed.String():
				fallthrough
			case constants.KafkaRequestStatusDeprovision.String():
				fallthrough
			case constants.KafkaRequestStatusDeleting.String():
				return false, errors.Errorf("Waiting for kafka '%s' to reach status '%s', but status '%s' has been reached instead", kafkaId, status.String(), kafka.Status)
			}

			currentStatus = kafka.Status
			return constants.KafkaStatus(kafka.Status).CompareTo(status) >= 0, nil
		}).
		Build().Poll()
	return kafka, err
}

// WaitForKafkaToBeDeleted - Awaits for a kafka to be deleted
func WaitForKafkaToBeDeleted(ctx context.Context, client *openapi.APIClient, kafkaId string) error {
	return NewPollerBuilder().
		IntervalAndTimeout(defaultPollInterval, defaultKafkaReadyTimeout).
		RetryLogMessagef("Waiting for kafka '%s' to be deleted", kafkaId).
		OnRetry(func(attempt int, maxRetries int) (done bool, err error) {
			if _, _, err := client.DefaultApi.GetKafkaById(ctx, kafkaId); err != nil {
				if err.Error() == "404 Not Found" {
					return true, nil
				}

				return false, err
			}
			return false, nil
		}).
		Build().Poll()
}

func WaitForKafkaClusterIDToBeAssigned(dbFactory *db.ConnectionFactory, kafkaRequestName string) (*api.KafkaRequest, error) {
	kafkaFound := &api.KafkaRequest{}

	kafkaErr := NewPollerBuilder().
		IntervalAndTimeout(defaultPollInterval, defaultKafkaClusterAssignmentTimeout).
		RetryLogMessagef("Waiting for kafka named '%s' to have a ClusterID", kafkaRequestName).
		OnRetry(func(attempt int, maxRetries int) (done bool, err error) {
			if err := dbFactory.New().Where("name = ?", kafkaRequestName).First(kafkaFound).Error; err != nil {
				return false, err
			}
			glog.Infof("got kafka instance %v", kafkaFound)
			return kafkaFound.ClusterID != "", nil
		}).Build().Poll()

	return kafkaFound, kafkaErr
}
