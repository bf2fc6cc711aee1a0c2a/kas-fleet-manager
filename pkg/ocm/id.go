package ocm

import (
	"fmt"
	"github.com/rs/xid"
)

// NOTE: the current mock generation exports to a _test file, if in the future this should be made public, consider
// moving the type into a ocmtest package.
//go:generate moq -out idgenerator_moq_test.go . IDGenerator
// IDGenerator interface for string ID generators.
type IDGenerator interface {
	// Generate create a new string ID.
	Generate() string
}

var _ IDGenerator = idGenerator{}

// idGenerator internal implementation of IDGenerator.
type idGenerator struct {
	// prefix a string to prefix to any generated ID.
	prefix string
}

// NewIDGenerator create a new default implementation of IDGenerator.
func NewIDGenerator(prefix string) IDGenerator {
	return idGenerator{
		prefix: prefix,
	}
}

func (i idGenerator) Generate() string {
	return fmt.Sprintf("%s%s", i.prefix, xid.New().String())
}
