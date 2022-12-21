package segment

import (
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/buildinformation"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/logger"

	analytics "github.com/segmentio/analytics-go/v3"
)

type SegmentClient interface {
	// Adds a message to the queue.
	// Messages are sent asynchronously to Segment when batch upload conditions are met,
	// see https://segment.com/docs/connections/sources/catalog/libraries/server/go/#batching for further information.
	//
	// Returns an error if the message cannot be added to the queue which may happen if the client was closed at the time of the method
	// call or if the message is malformed.
	Track(event, userId, result string)
}

type SegmentClientFactory struct {
	// Client used for interacting with Segment
	client analytics.Client
	// KAS Fleet Manager build info.
	// This can be used for setting information about KFM in the event context.
	kfmBuildInfo *buildinformation.BuildInfo
}

var _ SegmentClient = &SegmentClientFactory{}

func NewClientWithConfig(apiKey string, config analytics.Config) (*SegmentClientFactory, error) {
	// https://pkg.go.dev/gopkg.in/segmentio/analytics-go.v3#Client
	client, err := analytics.NewWithConfig(apiKey, config)
	if err != nil {
		return nil, err
	}

	kfmBuildInfo, err := buildinformation.GetBuildInfo()
	if err != nil {
		logger.Logger.Error(err)
		return nil, err
	}

	return &SegmentClientFactory{
		client:       client,
		kfmBuildInfo: kfmBuildInfo,
	}, nil
}

func (sc *SegmentClientFactory) Track(event, userId, result string) {
	// The message to be sent to Segment. This is defined by an analytics.Track object
	// The analytics.Track spec is described here: https://segment.com/docs/connections/spec/common/#structure
	message := analytics.Track{
		// Required Fields
		// The event name/descriptor.
		// Event naming convention: must be Upper Case i.e. Kafka Instance Creation (based on https://github.com/bf2fc6cc711aee1a0c2a/kafka-ui/pull/1004#discussion_r1048680467)
		Event: event,
		// A unique identifier for a user
		UserId: userId,

		// Optional Fields
		// Time when the event occurred
		// https://segment.com/docs/connections/spec/common/#timestamps
		Timestamp: time.Now(),
		// Additional information related to the event can be added to properties.
		// Any key-value data can be set here.
		// https://segment.com/docs/connections/spec/track/#properties
		Properties: analytics.NewProperties().
			Set("result", result),
		// Additional information not directly related to the event can be added here
		// https://segment.com/docs/connections/spec/common/#context
		Context: &analytics.Context{
			App: analytics.AppInfo{
				Name:      "KAS Fleet Manager",
				Version:   sc.kfmBuildInfo.GetCommitSHA(),
				Namespace: environments.GetEnvironmentStrFromEnv(),
			},
		},
	}

	// If userId can't be set, 'anonymousId' must be specified
	if message.UserId == "" {
		// Note that the segment client uses google/uuid for generating ids
		// https://github.com/segmentio/analytics-go/blob/v3.0/config.go#L172
		message.AnonymousId = api.NewID()
	}

	// Adds message to queue, the client handles the upload to Segment when it meets the upload requirements
	// based on how the client is configured.
	//
	// This returns an error if the message can't be added to the queue.
	if err := sc.client.Enqueue(message); err != nil {
		logger.Logger.Error(err)
	}
}
