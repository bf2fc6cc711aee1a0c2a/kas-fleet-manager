package handlers

import (
	_ "embed"
	"encoding/json"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/onsi/gomega"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

//go:embed processor_0.1.json
var schema string

func GetHandlerParams(method string, url string, body io.Reader, t *testing.T) (*http.Request, *httptest.ResponseRecorder) {
	g := gomega.NewWithT(t)
	req, err := http.NewRequest(method, url, body)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	return req, httptest.NewRecorder()
}

func SetupValidProcessorSchema() (*dbapi.ProcessorType, error) {
	pt := map[string]interface{}{}
	if err := json.Unmarshal([]byte(schema), &pt); err != nil {
		return nil, err
	}
	pt = pt["processor_type"].(map[string]interface{})
	ptSchema := api.JSON{}
	jsonStr, _ := json.Marshal(pt["schema"])
	if err := ptSchema.UnmarshalJSON(jsonStr); err != nil {
		return nil, err
	}
	processorType := dbapi.ProcessorType{
		JsonSchema: ptSchema,
		Channels: []dbapi.ProcessorChannel{{
			Channel: "stable",
		}},
	}
	return &processorType, nil
}
