package mocks

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/private/openapi"
)

const SQSConnectorSchemaText = `
{
  "id": "aws-sqs-source-v1alpha1",
  "kind": "ConnectorType",
  "href": "/api/kafkas_mgmt/v1/kafka-connector-types/aws-sqs-source-v1alpha1",
  "name": "aws-sqs-source",
  "version": "v1alpha1",
  "title": "AWS SQS Source",
  "description": "Receive data from AWS SQS",
  "json_schema": {
    "title": "AWS SQS Source",
    "description": "Receive data from AWS SQS.",
    "required": [
      "queueNameOrArn",
      "accessKey",
      "secretKey",
      "region"
    ],
    "properties": {
      "queueNameOrArn": {
        "title": "Queue Name",
        "description": "The SQS Queue name or ARN",
        "type": "string"
      },
      "deleteAfterRead": {
        "title": "Auto-delete messages",
        "description": "Delete messages after consuming them",
        "type": "boolean",
        "x-descriptors": [
          "urn:alm:descriptor:com.tectonic.ui:checkbox"
        ],
        "default": true
      },
      "accessKey": {
        "title": "Access Key",
        "description": "The access key obtained from AWS",
        "type": "string"
      },
      "secretKey": {
        "title": "Secret Key",
        "description": "The secret key obtained from AWS",
        "x-descriptors": [
          "urn:alm:descriptor:com.tectonic.ui:password"
        ],
        "oneOf": [
          {
            "description": "the secret value",
            "type": "string",
            "format": "password"
          },
          {
            "description": "An opaque reference to the secret",
            "type": "object",
            "properties": {}
          }
        ]
      },
      "region": {
        "title": "AWS Region",
        "description": "The AWS region to connect to",
        "type": "string",
        "example": "eu-west-1"
      }
    }
  }
}`

func NewConnectorTypeMock(t *testing.T) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/",
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintln(w, `
{
  "connector_type_ids": [
	"aws-sqs-source-v1alpha1"
  ]
}
			`)
		},
	)
	mux.HandleFunc("/api/kafkas_mgmt/v1/kafka-connector-types/aws-sqs-source-v1alpha1",
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintln(w, SQSConnectorSchemaText)
		},
	)
	mux.HandleFunc("/api/kafkas_mgmt/v1/kafka-connector-types/aws-sqs-source-v1alpha1/reify/spec",
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			input := openapi.ConnectorReifyRequest{}
			err := json.NewDecoder(r.Body).Decode(&input)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			data, err := json.Marshal(input.ConnectorSpec)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			_, _ = fmt.Fprintf(w, `
{
  "operator_ids":["example-operator:1.0.0"],
  "resources":[
	{
	   "apiVersion": "v1",
	   "kind": "Secret",
	   "metadata": {
		  "name": "secret-sa-sample",
		  "annotations": {
			 "kubernetes.io/service-account.name": "sa-name"
		  }
	   },
	   "type": "kubernetes.io/service-account-token",
	   "data": {
		  "extra": "YmFyCg==",
          "spec": %s
	   }
	}
   ],
  "status_extractors":[{
    "apiVersion": "v1",
    "kind": "Secret",
    "name": "secret-sa-sample",
    "jsonPath":	"status",
	"conditionType": "Secret"
  }]
}`, string(data))
		},
	)
	return httptest.NewServer(mux)
}
