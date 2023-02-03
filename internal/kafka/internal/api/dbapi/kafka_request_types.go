package dbapi

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"gorm.io/gorm"
)

type KafkaRequest struct {
	api.Meta
	Region                           string `json:"region"`
	ClusterID                        string `json:"cluster_id" gorm:"index"`
	CloudProvider                    string `json:"cloud_provider"`
	MultiAZ                          bool   `json:"multi_az"`
	Name                             string `json:"name" gorm:"index"`
	Status                           string `json:"status" gorm:"index"`
	CanaryServiceAccountClientID     string `json:"canary_service_account_client_id"`
	CanaryServiceAccountClientSecret string `json:"canary_service_account_client_secret"`
	SubscriptionId                   string `json:"subscription_id"`
	Owner                            string `json:"owner" gorm:"index"` // TODO: ocm owner?
	OwnerAccountId                   string `json:"owner_account_id"`
	BootstrapServerHost              string `json:"bootstrap_server_host"`
	AdminApiServerURL                string `json:"admin_api_server_url"`
	OrganisationId                   string `json:"organisation_id" gorm:"index"`
	FailedReason                     string `json:"failed_reason"`
	// PlacementId field should be updated every time when a KafkaRequest is assigned to an OSD cluster (even if it's the same one again)
	PlacementId string `json:"placement_id"`

	DesiredKafkaVersion    string `json:"desired_kafka_version"`
	ActualKafkaVersion     string `json:"actual_kafka_version"`
	DesiredStrimziVersion  string `json:"desired_strimzi_version"`
	ActualStrimziVersion   string `json:"actual_strimzi_version"`
	DesiredKafkaIBPVersion string `json:"desired_kafka_ibp_version"`
	ActualKafkaIBPVersion  string `json:"actual_kafka_ibp_version"`
	KafkaUpgrading         bool   `json:"kafka_upgrading"`
	StrimziUpgrading       bool   `json:"strimzi_upgrading"`
	KafkaIBPUpgrading      bool   `json:"kafka_ibp_upgrading"`
	KafkaStorageSize       string `json:"kafka_storage_size"`
	// The type of kafka instance (developer or standard)
	InstanceType string `json:"instance_type"`
	// the quota service type for the kafka, e.g. ams, quota-management-list
	QuotaType string `json:"quota_type"`
	// Routes routes mapping for the kafka instance. It is an array and each item in the array contains a domain value and the corresponding route url
	Routes api.JSON `json:"routes"`
	// RoutesCreated if the routes mapping have been created in the DNS provider like Route53. Use a separate field to make it easier to query.
	RoutesCreated bool `json:"routes_created"`
	// Namespace is the namespace of the provisioned kafka instance.
	// We store this in the database to ensure that old kafkas whose namespace contained "owner-<kafka-id>" information will continue to work.
	Namespace                string               `json:"namespace"`
	ReauthenticationEnabled  bool                 `json:"reauthentication_enabled"`
	RoutesCreationId         string               `json:"routes_creation_id"`
	SizeId                   string               `json:"size_id"`
	BillingCloudAccountId    string               `json:"billing_cloud_account_id"`
	Marketplace              string               `json:"marketplace"`
	ActualKafkaBillingModel  string               `json:"actual_kafka_billing_model"`
	DesiredKafkaBillingModel string               `json:"desired_kafka_billing_model"`
	PromotionStatus          KafkaPromotionStatus `json:"promotion_status"`
	PromotionDetails         string               `json:"promotion_details"`
	// ExpiresAt contains the timestamp of when a Kafka instance is scheduled to expire.
	// On expiration, the Kafka instance will be marked for deletion, its status will be set to 'deprovision'.
	ExpiresAt sql.NullTime `json:"expires_at"`
}

type KafkaPromotionStatus string

const (
	KafkaPromotionStatusPromoting   KafkaPromotionStatus = "promoting"
	KafkaPromotionStatusFailed      KafkaPromotionStatus = "failed"
	KafkaPromotionStatusNoPromotion KafkaPromotionStatus = ""
)

func (s KafkaPromotionStatus) String() string {
	return string(s)
}

func ParseKafkaPromotionStatus(status string) (KafkaPromotionStatus, error) {
	validPromotionStatuses := map[KafkaPromotionStatus]struct{}{
		KafkaPromotionStatusPromoting:   {},
		KafkaPromotionStatusFailed:      {},
		KafkaPromotionStatusNoPromotion: {},
	}

	parsedStatus := KafkaPromotionStatus(status)
	_, ok := validPromotionStatuses[parsedStatus]
	if !ok {
		return parsedStatus, fmt.Errorf("cannot parse %q as KafkaPromotionStatus: invalid status", parsedStatus)
	}

	return parsedStatus, nil
}

type KafkaList []*KafkaRequest
type KafkaIndex map[string]*KafkaRequest

func (l KafkaList) Index() KafkaIndex {
	index := KafkaIndex{}
	for _, o := range l {
		index[o.ID] = o
	}
	return index
}

func (k *KafkaRequest) BeforeCreate(scope *gorm.DB) error {
	// To allow the id set on the KafkaRequest object to be used. This is useful for testing purposes.
	id := k.ID
	if id == "" {
		k.ID = api.NewID()
	}
	return nil
}

func (k *KafkaRequest) GetRoutes() ([]DataPlaneKafkaRoute, error) {
	var routes []DataPlaneKafkaRoute
	if k.Routes == nil {
		return routes, nil
	}
	if err := json.Unmarshal(k.Routes, &routes); err != nil {
		return nil, err
	} else {
		return routes, nil
	}
}

func (k *KafkaRequest) SetRoutes(routes []DataPlaneKafkaRoute) error {
	if r, err := json.Marshal(routes); err != nil {
		return err
	} else {
		k.Routes = r
		return nil
	}
}

// GetExpirationTime returns when the Kafka request will expire based on the
// provided lifespanSeconds value. lifespanSeconds is assumed to be greater
// than 0
func (k *KafkaRequest) GetExpirationTime(lifespanSeconds int) *time.Time {
	expireTime := k.CreatedAt.Add(time.Duration(lifespanSeconds) * time.Second)
	return &expireTime
}

// The RemainingLifeSpan is the remaining life a kafka instance have before it expires and gets deleted
type RemainingLifeSpan interface {
	// Infinite returns whether the Kafka instance has infinite (no expiration) remaining life span
	Infinite() bool

	// Days returns the number of remaining life span days a Kafka instance has.
	// The value can be negative if the instance has expired by 1 day or more to indicate how many days ago the instance
	// has expired, or it can be `-1` if the remaining life is `infinite`.
	// Caller shouldn't rely on this function returning `-1` to check for infinity, since this function will return `-1`
	// even when the instance has expired one day ago.
	// To reliable check for infinity, check the value returned by `Infinite`
	Days() float64

	// LessThanOrEqual returns whether the remaining number of days is less or equal then the specified one.
	// If the remaining life is `infinte` this method always returns `false`.
	LessThanOrEqual(days float64) bool
}

var _ RemainingLifeSpan = &remainingLifeSpan{}

// remainingLifeSpan is an implementation of the RemainingLifeSpan interface
type remainingLifeSpan struct {
	// remainingLifeSpanDays is the number of days before the Kafka instance will be deleted. It is a `float64` to be able to manage fraction of days.
	remainingLifeSpanDays float64
	// infinite represent whether the Kafka instance has an expiration
	infinite bool
}

func (l *remainingLifeSpan) Infinite() bool {
	return l.infinite
}

func (l *remainingLifeSpan) Days() float64 {
	return l.remainingLifeSpanDays
}

func (l *remainingLifeSpan) LessThanOrEqual(days float64) bool {
	if l.infinite {
		return false
	}
	return l.remainingLifeSpanDays <= days
}

// IsExpired returns whether a kafka is expired and how many days the instance will live before expiring
func (k *KafkaRequest) IsExpired() (bool, RemainingLifeSpan) {
	if !k.ExpiresAt.Valid {
		return false, &remainingLifeSpan{
			remainingLifeSpanDays: -1,
			infinite:              true,
		}
	}

	now := time.Now()
	if now.After(k.ExpiresAt.Time) {
		return true, &remainingLifeSpan{
			remainingLifeSpanDays: 0,
			infinite:              false,
		}
	}

	remainingLife := k.ExpiresAt.Time.Sub(now)
	return false, &remainingLifeSpan{
		remainingLifeSpanDays: remainingLife.Hours() / 24,
		infinite:              false,
	}
}

// CanBeAutomaticallySuspended returns whether the kafka instance can be suspended or not
// This method is used when the Kafka instance enters its grace period
func (k *KafkaRequest) CanBeAutomaticallySuspended() bool {
	validSuspensionStatuses := []string{
		constants.KafkaRequestStatusAccepted.String(),
		constants.KafkaRequestStatusPreparing.String(),
		constants.KafkaRequestStatusReady.String(),
		constants.KafkaRequestStatusProvisioning.String(),
		constants.KafkaRequestStatusResuming.String(),
	}

	return arrays.Contains(validSuspensionStatuses, k.Status)
}

// DesiredBillingModelIsEnterprise returns true if the Kafka has enterprise billing model.
// Otherwise returns false.
func (k *KafkaRequest) DesiredBillingModelIsEnterprise() bool {
	return shared.StringEqualsIgnoreCase(k.DesiredKafkaBillingModel, constants.BillingModelEnterprise.String())
}
