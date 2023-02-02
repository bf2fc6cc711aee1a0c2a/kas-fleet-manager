package presenters

import (
	"strings"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/private"
)

func getStatusError(conditions []private.MetaV1Condition) string {
	var result string
	for _, c := range conditions {
		if c.Type == "Ready" {
			if c.Status == "False" {
				finalError := c.Message
				start := strings.Index(c.Message, "error.message")
				end := strings.Index(c.Message, "failure.count")
				if start > -1 && end > -1 {
					finalError = c.Message[start+14 : end]
				}
				result = c.Reason + ": " + finalError
			}
			break
		}
	}
	return result
}
