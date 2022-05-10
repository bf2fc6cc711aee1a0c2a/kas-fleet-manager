package config

import (
	"fmt"
	"strings"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/senseyeio/duration"
	"k8s.io/apimachinery/pkg/api/resource"
)

type KafkaInstanceType struct {
	Id          string              `yaml:"id"`
	DisplayName string              `yaml:"display_name"`
	Sizes       []KafkaInstanceSize `yaml:"sizes"`
}

// validates kafka instance type config to ensure the following:
// - id must be defined and included in the valid instance type id list
// - display_name must be defined and included in the valid instance type list
// - sizes cannot be an empty list and each size id must be unique
func (kp *KafkaInstanceType) validate() error {
	if kp.Id == "" || kp.DisplayName == "" || len(kp.Sizes) == 0 {
		return fmt.Errorf("Kafka instance type '%s' is missing required parameters.", kp.Id)
	}

	if !arrays.Contains(types.ValidKafkaInstanceTypes, kp.Id) {
		return fmt.Errorf("kafka instance type id '%s' is not valid. Valid kafka instance types are: '%v'", kp.Id, types.ValidKafkaInstanceTypes)
	}

	existingSizes := make(map[string]int, len(kp.Sizes))

	for _, kafkaInstanceSize := range kp.Sizes {
		if _, ok := existingSizes[kafkaInstanceSize.Id]; ok {
			return fmt.Errorf("Kafka instance size '%s' for instance type '%s' was defined more than once.", kafkaInstanceSize.Id, kp.Id)
		}
		existingSizes[kafkaInstanceSize.Id]++

		if err := kafkaInstanceSize.validate(kp.Id); err != nil {
			return err
		}
	}

	return nil
}

type KafkaInstanceSize struct {
	Id                          string   `yaml:"id"`
	DisplayName                 string   `yaml:"display_name"`
	IngressThroughputPerSec     Quantity `yaml:"ingressThroughputPerSec"`
	EgressThroughputPerSec      Quantity `yaml:"egressThroughputPerSec"`
	TotalMaxConnections         int      `yaml:"totalMaxConnections"`
	MaxDataRetentionSize        Quantity `yaml:"maxDataRetentionSize"`
	MaxPartitions               int      `yaml:"maxPartitions"`
	MaxDataRetentionPeriod      string   `yaml:"maxDataRetentionPeriod"`
	MaxConnectionAttemptsPerSec int      `yaml:"maxConnectionAttemptsPerSec"`
	MaxMessageSize              Quantity `yaml:"maxMessageSize"`
	QuotaConsumed               int      `yaml:"quotaConsumed"`
	QuotaType                   string   `yaml:"quotaType"`
	CapacityConsumed            int      `yaml:"capacityConsumed"`
	SupportedAZModes            []string `yaml:"supportedAZModes"`
	MinInSyncReplicas           int      `yaml:"minInSyncReplicas"` // also abbreviated as ISR in Kafka terminology
	ReplicationFactor           int      `yaml:"replicationFactor"` // also abbreviated as RF in Kafka terminology
	LifespanSeconds             *int     `yaml:"lifespanSeconds"`
}

// validates Kafka instance size configuration to ensure the following:
//
// - all properties must be defined
// - any non-id string values must be parseable
// - any int values must not be less than or equal to zero
func (k *KafkaInstanceSize) validate(instanceTypeId string) error {
	if k.EgressThroughputPerSec.IsEmpty() || k.IngressThroughputPerSec.IsEmpty() ||
		k.MaxDataRetentionPeriod == "" || k.MaxDataRetentionSize.IsEmpty() || k.Id == "" || k.QuotaType == "" ||
		k.DisplayName == "" || k.MaxMessageSize.IsEmpty() || k.SupportedAZModes == nil {
		return fmt.Errorf("Kafka instance size '%s' for instance type '%s' is missing required parameters.", k.Id, instanceTypeId)
	}

	egressThroughputQuantity, err := k.EgressThroughputPerSec.ToK8Quantity()
	if err != nil {
		return fmt.Errorf("egressThroughputPerSec for Kafka instance type '%s', size '%s' is invalid: %s", k.Id, instanceTypeId, err.Error())
	}

	ingressThroughputQuantity, err := k.IngressThroughputPerSec.ToK8Quantity()
	if err != nil {
		return fmt.Errorf("ingressThroughputPerSec for Kafka instance type '%s', size '%s' is invalid: %s", k.Id, instanceTypeId, err.Error())
	}

	maxDataRetentionSize, err := k.MaxDataRetentionSize.ToK8Quantity()
	if err != nil {
		return fmt.Errorf("maxDataRetentionSize for Kafka instance type '%s', size '%s' is invalid: %s", k.Id, instanceTypeId, err.Error())
	}

	maxDataRetentionPeriod, err := duration.ParseISO8601(k.MaxDataRetentionPeriod)
	if err != nil {
		return fmt.Errorf("maxDataRetentionPeriod for Kafka instance type '%s', size '%s' is invalid: %s", k.Id, instanceTypeId, err.Error())
	}

	maxMessageSize, err := k.MaxMessageSize.ToK8Quantity()
	if err != nil {
		return fmt.Errorf("maxMessageSize for Kafka instance type '%s', size '%s' is invalid: %s", k.Id, instanceTypeId, err.Error())
	}

	validSupportedAZModes := map[string]struct{}{
		"single": {},
		"multi":  {},
	}
	for _, supportedAZMode := range k.SupportedAZModes {
		if _, ok := validSupportedAZModes[supportedAZMode]; !ok {
			return fmt.Errorf("value '%s' in supportedAZModes for Kafka instance type '%s', size '%s' is invalid", supportedAZMode, k.Id, instanceTypeId)
		}
	}

	if k.LifespanSeconds != nil && *k.LifespanSeconds <= 0 {
		return fmt.Errorf("Kafka instance size '%s' for instance type '%s' specifies a lifespanSeconds seconds value less than or equals to Zero.", k.Id, instanceTypeId)
	}

	if maxDataRetentionPeriod.IsZero() || egressThroughputQuantity.CmpInt64(1) < 0 ||
		ingressThroughputQuantity.CmpInt64(1) < 0 || maxDataRetentionSize.CmpInt64(1) < 0 ||
		k.TotalMaxConnections <= 0 || k.MaxPartitions <= 0 || k.MaxConnectionAttemptsPerSec <= 0 ||
		k.QuotaConsumed < 1 || k.CapacityConsumed < 1 || k.MinInSyncReplicas < 1 ||
		k.ReplicationFactor < 1 || maxMessageSize.CmpInt64(0) < 0 || len(k.SupportedAZModes) == 0 {
		return fmt.Errorf("Kafka instance size '%s' for instance type '%s' specifies a property value less than or equals to Zero.", k.Id, instanceTypeId)
	}

	return nil
}

type SupportedKafkaInstanceTypesConfig struct {
	SupportedKafkaInstanceTypes []KafkaInstanceType `yaml:"supported_instance_types"`
}

func (s *SupportedKafkaInstanceTypesConfig) validate() error {
	existingInstanceTypes := make(map[string]int, len(s.SupportedKafkaInstanceTypes))

	for _, KafkaInstanceType := range s.SupportedKafkaInstanceTypes {
		if _, ok := existingInstanceTypes[KafkaInstanceType.Id]; ok {
			return fmt.Errorf("Kafka instance type id '%s' was defined more than once.", KafkaInstanceType.Id)
		}
		existingInstanceTypes[KafkaInstanceType.Id]++

		if err := KafkaInstanceType.validate(); err != nil {
			return err
		}
	}

	return nil
}

type KafkaSupportedInstanceTypesConfig struct {
	Configuration     SupportedKafkaInstanceTypesConfig
	ConfigurationFile string
}

func NewKafkaSupportedInstanceTypesConfig() *KafkaSupportedInstanceTypesConfig {
	return &KafkaSupportedInstanceTypesConfig{
		ConfigurationFile: "config/kafka-instance-types-configuration.yaml",
	}
}

func (s *SupportedKafkaInstanceTypesConfig) GetKafkaInstanceTypeByID(instanceType string) (*KafkaInstanceType, error) {
	for _, t := range s.SupportedKafkaInstanceTypes {
		if t.Id == instanceType {
			ret := t
			return &ret, nil
		}
	}
	return nil, fmt.Errorf("Unable to find kafka instance type for '%s'", instanceType)
}

func (kp *KafkaInstanceType) GetKafkaInstanceSizeByID(sizeId string) (*KafkaInstanceSize, error) {
	for _, size := range kp.Sizes {
		if size.Id == sizeId {
			ret := size
			return &ret, nil
		}
	}
	return nil, fmt.Errorf("Kafka instance size id: '%s' not found for '%s' instance type", sizeId, kp.Id)
}

type Plan string

func (p Plan) String() string {
	return string(p)
}

const numComponentsInPlanFormat = 2

func (p Plan) GetInstanceType() (string, error) {
	t := strings.Split(strings.TrimSpace(p.String()), ".")
	if len(t) != numComponentsInPlanFormat {
		return "", errors.New(errors.ErrorGeneral, fmt.Sprintf("Unsupported plan provided: '%s'", p))
	}
	return t[0], nil
}

func (p Plan) GetSizeID() (string, error) {
	t := strings.Split(strings.TrimSpace(p.String()), ".")
	if len(t) != numComponentsInPlanFormat {
		return "", errors.New(errors.ErrorGeneral, fmt.Sprintf("Unsupported plan provided: '%s'", p))
	}
	return t[1], nil
}

type Quantity string

func (q *Quantity) String() string {
	return string(*q)
}

func (q *Quantity) ToInt64() (int64, error) {
	if p, err := resource.ParseQuantity(string(*q)); err != nil {
		return 0, err
	} else {
		return p.Value(), nil
	}
}

func (q *Quantity) ToK8Quantity() (*resource.Quantity, error) {
	if p, err := resource.ParseQuantity(string(*q)); err != nil {
		return nil, err
	} else {
		return &p, nil
	}
}

func (q *Quantity) IsEmpty() bool {
	return q == nil
}
