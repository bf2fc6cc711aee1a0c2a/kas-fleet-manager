package migrations

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/services"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/util/validation"
)

const truncatedNamespaceLen = 35
const replacementForSpecialChar = "-"

// All namespace names must conform to DNS1123. This will inverse the validation RE from k8s (https://github.com/kubernetes/apimachinery/blob/master/pkg/util/validation/validation.go#L177)
var dns1123ReplacementRE = regexp.MustCompile(`[^a-z0-9]([^-a-z0-9]*[^a-z0-9])?`)

func migrateOldDinosaurNamespace() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20210809140000",
		Migrate: func(tx *gorm.DB) error {
			var dinosaurs []dbapi.DinosaurRequest
			err := tx.Unscoped().
				Model(&dbapi.DinosaurRequest{}).
				Select("id", "namespace", "owner").
				Where("namespace = ?", "").
				Scan(&dinosaurs).
				Error

			if err != nil {
				return err
			}

			for _, dinosaur := range dinosaurs {
				namespace, err := buildOldDinosaurNamespace(&dinosaur)
				if err != nil {
					return err
				}
				err = tx.Model(&dinosaur).Update("namespace", namespace).Error
				if err != nil {
					return err
				}
			}

			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			return nil
		},
	}
}

func buildOldDinosaurNamespace(dinosaur *dbapi.DinosaurRequest) (string, error) {
	namespaceName := buildDinosaurNamespaceIdentifier(dinosaur)
	namespaceName, err := replaceNamespaceSpecialChar(namespaceName)
	if err != nil {
		return namespaceName, fmt.Errorf("failed to build namespace for dinosaur %s: %w", dinosaur.ID, err)
	}
	return namespaceName, nil
}

// buildDinosaurNamespaceIdentifier creates a unique identifier for a namespace to be used for
// the dinosaur request
func buildDinosaurNamespaceIdentifier(dinosaurRequest *dbapi.DinosaurRequest) string {
	return fmt.Sprintf("%s-%s", services.TruncateString(dinosaurRequest.Owner, truncatedNamespaceLen), strings.ToLower(dinosaurRequest.ID))
}

// replaceNamespaceSpecialChar replaces invalid characters with random char in the namespace name
func replaceNamespaceSpecialChar(name string) (string, error) {
	replacedName := dns1123ReplacementRE.ReplaceAllString(strings.ToLower(name), replacementForSpecialChar)

	replacedName = services.MaskProceedingandTrailingDash(replacedName)

	// This should never fail based on above replacement of invalid characters.
	validationErrors := validation.IsDNS1123Label(replacedName)
	if len(validationErrors) > 0 {
		return replacedName, fmt.Errorf("Namespace name is invalid: %s. A DNS-1123 label must consist of lower case alphanumeric characters or '-', and must start and end with an alphanumeric character (e.g. 'my-name', or '123-abc', regex used for validation is '[a-z0-9]([-a-z0-9]*[a-z0-9])?')", strings.Join(validationErrors[:], ","))
	}

	return replacedName, nil
}
