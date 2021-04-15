package mocks

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

const SQSConnectorSchemaText = `
{
  "id": "aws-sqs-source-v1alpha1",
  "kind": "ConnectorType",
  "href": "/api/managed-services-api/v1/kafka-connector-types/aws-sqs-source-v1alpha1",
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
	mux.HandleFunc("/v1/kafka-connector-catalog",
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintln(w, `
[
    {
        "id": "aws-sqs-source-v1alpha1",
        "channel": "stable",
        "shard_metadata": {
            "meta_image": "quay.io/mock-image:77c0b8763729a9167ddfa19266d83a3512b7aa8124ca53e381d5d05f7d197a24",
            "operators": [
                {
                    "type": "camel-k",
                    "versions": "[1.0.0,2.0.0]"
                }
            ]
        }
    },
    {
        "id": "aws-sqs-source-v1alpha1",
        "channel": "beta",
        "shard_metadata": {
            "meta_image": "quay.io/mock-image:beta",
            "operators": [
                {
                    "type": "camel-k",
                    "versions": "[2.0.0]"
                }
            ]
        }
    }
]
			`)
		},
	)
	mux.HandleFunc("/v1/kafka-connector-types/aws-sqs-source-v1alpha1",
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintln(w, SQSConnectorSchemaText)
		},
	)
	return httptest.NewServer(mux)
}
