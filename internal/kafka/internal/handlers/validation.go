package handlers

import (
	"context"
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"regexp"
)

var ValidKafkaClusterNameRegexp = regexp.MustCompile(`^[a-z]([-a-z0-9]*[a-z0-9])?$`)

var MaxKafkaNameLength = 32

func ValidKafkaClusterName(value *string, field string) handlers.Validate {
	return func() *errors.ServiceError {
		if !ValidKafkaClusterNameRegexp.MatchString(*value) {
			return errors.MalformedKafkaClusterName("%s does not match %s", field, ValidKafkaClusterNameRegexp.String())
		}
		return nil
	}
}

// ValidateKafkaClusterNameIsUnique returns a validator that validates that the kafka cluster name is unique
func ValidateKafkaClusterNameIsUnique(name *string, kafkaService services.KafkaService, context context.Context) handlers.Validate {
	return func() *errors.ServiceError {

		_, pageMeta, err := kafkaService.List(context, &coreServices.ListArguments{Page: 1, Size: 1, Search: fmt.Sprintf("name = %s", *name)})
		if err != nil {
			return err
		}

		if pageMeta.Total > 0 {
			return errors.DuplicateKafkaClusterName()
		}

		return nil
	}
}
