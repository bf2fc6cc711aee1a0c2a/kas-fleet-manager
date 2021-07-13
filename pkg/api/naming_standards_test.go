package api

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestKasFleetManagerApi(t *testing.T) {
	checkFile(t, "../../openapi/kas-fleet-manager.yaml")
}

func TestConnectorManagerApi(t *testing.T) {
	checkFile(t, "../../openapi/connector_mgmt.yaml")
}

func checkFile(t *testing.T, file string) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	doc, err := loader.LoadFromFile(file)
	if err != nil {
		t.Fatalf("%v", err)
	}

	errors := checkAPIDoc(doc)
	if len(errors) > 0 {
		fmt.Printf("found the following non standard naming issues in: %s\n", file)
		for _, s := range errors {
			fmt.Println("  " + s)
		}
		t.FailNow()
	}
}

var pathRegex = regexp.MustCompile("^[a-z{][a-z0-9_{}]*$")
var propertyRegex = regexp.MustCompile("^[a-z][a-z0-9_]*$")

func checkAPIDoc(t *openapi3.T) (errors []string) {

	for key, path := range t.Paths {
		split := strings.Split(key, "/")
		for i, s := range split {
			if s == "" {
				continue
			}
			if !pathRegex.MatchString(s) {
				errors = append(errors, fmt.Sprintf("bad api path name: '%s', sub path %d invalid: %s", key, i, s))
			}
		}

		for where, op := range path.Operations() {
			if op.RequestBody != nil {
				where := fmt.Sprintf("operation '%s' request body", where)
				errors = append(errors, checkRequestBody(where, op.RequestBody)...)
			}
			if op.Responses != nil {
				for status, ref := range op.Responses {
					if ref.Ref == "" {
						where := fmt.Sprintf("operation '%s' status '%s'", where, status)
						errors = append(errors, checkContent(where, ref.Value.Content)...)
					}
				}
			}
		}

	}

	for where, schema := range t.Components.Schemas {
		errors = append(errors, checkSchemaRef(where, schema)...)
	}
	for where, response := range t.Components.Responses {
		errors = append(errors, checkResponse(where, response)...)
	}
	for where, body := range t.Components.RequestBodies {
		errors = append(errors, checkRequestBody(where, body)...)
	}

	return
}

func checkResponse(where string, response *openapi3.ResponseRef) (errors []string) {
	if response.Ref == "" {
		errors = append(errors, checkContent(where, response.Value.Content)...)
	}
	return
}

func checkRequestBody(where string, body *openapi3.RequestBodyRef) (errors []string) {
	if body.Ref == "" {
		errors = append(errors, checkContent(where, body.Value.Content)...)
	}
	return
}

func checkContent(where string, content openapi3.Content) (errors []string) {
	for typeName, mediaType := range content {
		where := fmt.Sprintf("%s: media type '%s'", where, typeName)
		errors = append(errors, checkSchemaRef(where, mediaType.Schema)...)
	}
	return
}

func checkSchemas(where string, schemas openapi3.Schemas) (errors []string) {
	for name, schema := range schemas {
		where := fmt.Sprintf("%s.%s", where, name)

		if !propertyRegex.MatchString(name) && !schema.Value.Deprecated {
			errors = append(errors, fmt.Sprintf("bad property name: '%s'", where))
		}

		errors = append(errors, checkSchemaRef(where, schema)...)
	}
	return
}

func checkSchemaRef(where string, schema *openapi3.SchemaRef) (errors []string) {
	if schema.Ref == "" {
		errors = append(errors, checkSchema(where, schema.Value)...)
	}
	return
}

func checkSchemaRefs(where string, schemas openapi3.SchemaRefs) (errors []string) {
	for _, schema := range schemas {
		errors = append(errors, checkSchemaRef(where, schema)...)
	}
	return
}

func checkSchema(where string, schema *openapi3.Schema) (errors []string) {
	if schema.Properties != nil {
		errors = append(errors, checkSchemas(where, schema.Properties)...)
	}
	errors = append(errors, checkSchemaRefs(where, schema.OneOf)...)
	errors = append(errors, checkSchemaRefs(where, schema.AllOf)...)
	errors = append(errors, checkSchemaRefs(where, schema.AnyOf)...)
	if schema.Not != nil {
		errors = append(errors, checkSchemaRef(where, schema.Not)...)
	}
	if schema.Items != nil {
		errors = append(errors, checkSchemaRef(where, schema.Items)...)
	}
	if schema.AdditionalProperties != nil {
		errors = append(errors, checkSchemaRef(where, schema.AdditionalProperties)...)
	}
	return
}
