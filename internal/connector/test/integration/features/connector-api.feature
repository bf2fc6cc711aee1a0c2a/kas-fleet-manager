Feature: create a a connector
  In order to use connectors api
  As an API user
  I need to be able to manage connectors

  Background:
    Given the path prefix is "/api/connector_mgmt"
    # Greg and Coworker Sally will end up in the same org
    Given a user named "Greg" in organization "13640203"
    Given a user named "Coworker Sally" in organization "13640203"
    Given a user named "Evil Bob"
    Given a user named "Jim"

  Scenario: Greg lists all connector types
    Given I am logged in as "Greg"
    When I GET path "/v1/kafka_connector_types"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "items": [
          {
            "description": "Receive data from AWS SQS",
            "href": "/api/connector_mgmt/v1/kafka_connector_types/aws-sqs-source-v1alpha1",
            "id": "aws-sqs-source-v1alpha1",
            "channels": [
              "stable",
              "beta"
            ],
            "json_schema": {
              "description": "Receive data from AWS SQS.",
              "properties": {
                "accessKey": {
                  "description": "The access key obtained from AWS",
                  "title": "Access Key",
                  "type": "string"
                },
                "deleteAfterRead": {
                  "default": true,
                  "description": "Delete messages after consuming them",
                  "title": "Auto-delete messages",
                  "type": "boolean",
                  "x-descriptors": [
                    "urn:alm:descriptor:com.tectonic.ui:checkbox"
                  ]
                },
                "queueNameOrArn": {
                  "description": "The SQS Queue name or ARN",
                  "title": "Queue Name",
                  "type": "string"
                },
                "region": {
                  "description": "The AWS region to connect to",
                  "example": "eu-west-1",
                  "title": "AWS Region",
                  "type": "string"
                },
                "secretKey": {
                  "description": "The secret key obtained from AWS",
                  "oneOf": [
                    {
                      "description": "the secret value",
                      "format": "password",
                      "type": "string"
                    },
                    {
                      "description": "An opaque reference to the secret",
                      "properties": {},
                      "type": "object"
                    }
                  ],
                  "title": "Secret Key",
                  "x-descriptors": [
                    "urn:alm:descriptor:com.tectonic.ui:password"
                  ]
                }
              },
              "required": [
                "queueNameOrArn",
                "accessKey",
                "secretKey",
                "region"
              ],
              "title": "AWS SQS Source"
            },
            "kind": "ConnectorType",
            "name": "aws-sqs-source",
            "version": "v1alpha1"
          },
          {
            "json_schema" : {
              "type" : "object",
              "properties" : {
                "connector" : {
                  "type" : "object",
                  "title" : "Log",
                  "required" : [ ],
                  "properties" : {
                    "multiLine" : {
                      "title" : "Multi Line",
                      "description" : "Multi Line",
                      "type" : "boolean",
                      "default" : false
                    },
                    "showAll" : {
                      "title" : "Show All",
                      "description" : "Show All",
                      "type" : "boolean",
                      "default" : false
                    }
                  }
                },
                "kafka" : {
                  "type" : "object",
                  "title" : "Managed Kafka Source",
                  "required" : [ "topic" ],
                  "properties" : {
                    "topic" : {
                      "title" : "Topic names",
                      "description" : "Comma separated list of Kafka topic names",
                      "type" : "string"
                    }
                  }
                },
                "steps" : {
                  "type" : "array",
                  "items" : {
                    "oneOf" : [ {
                      "type" : "object",
                      "required" : [ "insert-field" ],
                      "properties" : {
                        "insert-field" : {
                          "title" : "Insert Field Action",
                          "description" : "Adds a custom field with a constant value to the message in transit",
                          "required" : [ "field", "value" ],
                          "properties" : {
                            "field" : {
                              "title" : "Field",
                              "description" : "The name of the field to be added",
                              "type" : "string"
                            },
                            "value" : {
                              "title" : "Value",
                              "description" : "The value of the field",
                              "type" : "string"
                            }
                          },
                          "type" : "object"
                        }
                      }
                    }, {
                      "type" : "object",
                      "required" : [ "extract-field" ],
                      "properties" : {
                        "extract-field" : {
                          "title" : "Extract Field Action",
                          "description" : "Extract a field from the body",
                          "required" : [ "field" ],
                          "properties" : {
                            "field" : {
                              "title" : "Field",
                              "description" : "The name of the field to be added",
                              "type" : "string"
                            }
                          },
                          "type" : "object"
                        }
                      }
                    } ]
                  }
                }
              }
            },
            "id" : "log_sink_0.1",
            "href": "/api/connector_mgmt/v1/kafka_connector_types/log_sink_0.1",
            "kind" : "ConnectorType",
            "icon_href" : "TODO",
            "name" : "Log Sink",
            "description" : "Log Sink",
            "version" : "0.1",
            "labels" : [ "sink" ],
            "channels" : [ "stable" ]
          }
        ],
        "kind": "ConnectorTypeList",
        "page": 1,
        "size": 2,
        "total": 2
      }
      """

  Scenario: Greg tries to create a connector with an invalid configuration spec
    Given I am logged in as "Greg"
    When I POST path "/v1/kafka_connectors?async=true" with json body:
      """
      {
        "kind": "Connector",
        "metadata": {
          "name": "example 1",
          "kafka_id":"mykafka"
        },
        "deployment_location": {
          "kind": "addon",
          "cluster_id": "default"
        },
        "kafka": {
          "bootstrap_server": "kafka.hostname",
          "client_id": "myclient",
          "client_secret": "test"
        },
        "connector_type_id": "aws-sqs-source-v1alpha1",
        "connector_spec": {
            "accessKey": "test",
            "secretKey": "test",
            "region": "east"
        }
      }
      """
    Then the response code should be 400
    And the response should match json:
      """
      {
        "code": "CONNECTOR-MGMT-21",
        "href": "/api/connector_mgmt/v1/errors/21",
        "id": "21",
        "kind": "Error",
        "operation_id": "${response.operation_id}",
        "reason": "connector spec not conform to the connector type schema. 1 errors encountered.  1st error: (root): queueNameOrArn is required"
      }
      """

  Scenario: Greg tries to create a connector with an invalid deployment_location.kind
    Given I am logged in as "Greg"
    When I POST path "/v1/kafka_connectors?async=true" with json body:
      """
      {
        "kind": "Connector",
        "metadata": {
          "name": "example 1",
          "kafka_id":"mykafka"
        },
        "deployment_location": {
          "kind": "WRONG",
          "cluster_id": "default"
        },
        "connector_type_id": "aws-sqs-source-v1alpha1",
        "kafka": {
          "bootstrap_server": "kafka.hostname",
          "client_id": "myclient",
          "client_secret": "test"
        },
        "connector_spec": {
            "queueNameOrArn": "test",
            "accessKey": "test",
            "secretKey": "test",
            "region": "east"
        }
      }
      """
    Then the response code should be 400
    And the response should match json:
      """
      {
        "code": "CONNECTOR-MGMT-33",
        "href": "/api/connector_mgmt/v1/errors/33",
        "id": "33",
        "kind": "Error",
        "operation_id": "${response.operation_id}",
        "reason": "deployment_location.kind is not valid. Must be one of: addon"
      }
      """

  Scenario: Greg creates lists and deletes a connector verifying that Evil Bob can't access Gregs Connectors
  but Coworker Sally can.
    Given I am logged in as "Greg"
    When I POST path "/v1/kafka_connectors?async=true" with json body:
      """
      {
        "kind": "Connector",
        "metadata": {
          "name": "example 1",
          "kafka_id":"mykafka"
        },
        "deployment_location": {
          "kind": "addon",
          "cluster_id": "default"
        },
        "connector_type_id": "aws-sqs-source-v1alpha1",
        "kafka": {
          "bootstrap_server": "kafka.hostname",
          "client_id": "myclient",
          "client_secret": "test"
        },
        "connector_spec": {
            "queueNameOrArn": "test",
            "accessKey": "test",
            "secretKey": "test",
            "region": "east"
        }
      }
      """
    Then the response code should be 202
    And the ".status" selection from the response should match "assigning"

    Given I store the ".id" selection from the response as ${connector_id}
    When I GET path "/v1/kafka_connectors?kafka_id=mykafka"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "items": [
          {
            "channel": "stable",
            "kafka": {
              "bootstrap_server": "kafka.hostname",
              "client_id": "myclient"
            },
            "connector_spec": {
              "accessKey": "test",
              "queueNameOrArn": "test",
              "region": "east",
              "secretKey": {}
            },
            "connector_type_id": "aws-sqs-source-v1alpha1",
            "deployment_location": {
              "cluster_id": "default",
              "kind": "addon"
            },
            "href": "/api/connector_mgmt/v1/kafka_connectors/${connector_id}",
            "id": "${connector_id}",
            "kind": "Connector",
            "metadata": {
              "created_at": "${response.items[0].metadata.created_at}",
              "kafka_id": "mykafka",
              "name": "example 1",
              "owner": "${response.items[0].metadata.owner}",
              "resource_version": ${response.items[0].metadata.resource_version},
              "updated_at": "${response.items[0].metadata.updated_at}"
            },
            "desired_state": "ready",
            "status": "assigning"
          }
        ],
        "kind": "ConnectorList",
        "page": 1,
        "size": 1,
        "total": 1
      }
      """

    When I GET path "/v1/kafka_connectors/${connector_id}"
    Then the response code should be 200
    And the ".status" selection from the response should match "assigning"
    And the ".id" selection from the response should match "${connector_id}"
    And the response should match json:
      """
      {
          "id": "${connector_id}",
          "kind": "Connector",
          "href": "/api/connector_mgmt/v1/kafka_connectors/${connector_id}",
          "metadata": {
              "kafka_id": "mykafka",
              "owner": "${response.metadata.owner}",
              "name": "example 1",
              "created_at": "${response.metadata.created_at}",
              "updated_at": "${response.metadata.updated_at}",
              "resource_version": ${response.metadata.resource_version}
          },
          "kafka": {
            "bootstrap_server": "kafka.hostname",
            "client_id": "myclient"
          },
          "deployment_location": {
              "kind": "addon",
              "cluster_id": "default"
          },
          "connector_type_id": "aws-sqs-source-v1alpha1",
          "channel": "stable",
          "connector_spec": {
              "accessKey": "test",
              "queueNameOrArn": "test",
              "region": "east",
              "secretKey": {}
          },
          "desired_state": "ready",
          "status": "assigning"
      }
      """

    Given I set the "Content-Type" header to "application/merge-patch+json"
    When I PATCH path "/v1/kafka-connectors/${connector_id}" with json body:
      """
      {
          "connector_spec": {
              "secretKey": {
                "ref": "hack"
              }
          }
      }
      """
    Then the response code should be 400
    And the response should match json:
      """
      {
        "code": "CONNECTOR-MGMT-21",
        "href": "/api/connector_mgmt/v1/errors/21",
        "id": "21",
        "kind": "Error",
        "operation_id": "${response.operation_id}",
        "reason": "invalid patch: attempting to change opaque connector secret"
      }
      """

    # We can update secrets of a connector
    And the vault delete counter should be 0
    Given I set the "Content-Type" header to "application/merge-patch+json"
    When I PATCH path "/v1/kafka-connectors/${connector_id}" with json body:
      """
      {
          "kafka": {
            "client_secret": "patched_secret 1"
          },
          "connector_spec": {
              "secretKey": "patched_secret 2"
          }
      }
      """

    Then the response code should be 202
    And the vault delete counter should be 2


    # Before deleting the connector, lets make sure the access control work as expected for other users beside Greg
    Given I am logged in as "Coworker Sally"
    When I GET path "/v1/kafka_connectors/${connector_id}"
    Then the response code should be 200

    Given I am logged in as "Evil Bob"
    When I GET path "/v1/kafka_connectors/${connector_id}"
    Then the response code should be 404

    # We are going to delete the connector...
    Given I am logged in as "Greg"
    Given the vault delete counter should be 2
    When I DELETE path "/v1/kafka_connectors/${connector_id}"
    Then the response code should be 204
    And the response should match ""

    Given I wait up to "5" seconds for a GET on path "/v1/kafka_connectors/${connector_id}" response code to match "404"
    When I GET path "/v1/kafka_connectors/${connector_id}"
    Then the response code should be 404
    # This verifies that kafka client secret and the connector_spec.secretKey vault entries are deleted.
    Then the vault delete counter should be 4

  Scenario: Greg can discover the API endpoints
    Given I am logged in as "Greg"
    When I GET path ""
    Then the response code should be 200
    And the response should match json:
      """
      {
        "href": "/api/connector_mgmt",
        "id": "connector_mgmt",
        "kind": "API",
        "versions": [
          {
            "collections": null,
            "href": "/api/connector_mgmt/v1",
            "id": "v1",
            "kind": "APIVersion"
          }
        ]
      }
      """

    When I GET path "/"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "href": "/api/connector_mgmt",
        "id": "connector_mgmt",
        "kind": "API",
        "versions": [
          {
            "kind": "APIVersion",
            "id": "v1",
            "href": "/api/connector_mgmt/v1",
            "collections": null
          }
        ]
      }
      """

    When I GET path "/v1"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "kind": "APIVersion",
        "id": "v1",
        "href": "/api/connector_mgmt/v1",
        "collections": [
          {
            "href": "/api/connector_mgmt/v1/kafka_connector_types",
            "id": "kafka_connector_types",
            "kind": "ConnectorTypeList"
          },
          {
            "href": "/api/connector_mgmt/v1/kafka_connectors",
            "id": "kafka_connectors",
            "kind": "ConnectorList"
          },
          {
            "href": "/api/connector_mgmt/v1/kafka_connector_clusters",
            "id": "kafka_connector_clusters",
            "kind": "ConnectorClusterList"
          }
        ]
      }
      """

    When I GET path "/v1/"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "kind": "APIVersion",
        "id": "v1",
        "href": "/api/connector_mgmt/v1",
        "collections": [
          {
            "href": "/api/connector_mgmt/v1/kafka_connector_types",
            "id": "kafka_connector_types",
            "kind": "ConnectorTypeList"
          },
          {
            "href": "/api/connector_mgmt/v1/kafka_connectors",
            "id": "kafka_connectors",
            "kind": "ConnectorList"
          },
          {
            "href": "/api/connector_mgmt/v1/kafka_connector_clusters",
            "id": "kafka_connector_clusters",
            "kind": "ConnectorClusterList"
          }
        ]
      }
      """

  Scenario: Greg can inspect errors codes
    Given I am logged in as "Greg"
    When I GET path "/v1/errors"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "items": [
          {
            "code": "CONNECTOR-MGMT-4",
            "href": "/api/connector_mgmt/v1/errors/4",
            "id": "4",
            "kind": "Error",
            "reason": "Forbidden to perform this action"
          },
          {
            "code": "CONNECTOR-MGMT-5",
            "href": "/api/connector_mgmt/v1/errors/5",
            "id": "5",
            "kind": "Error",
            "reason": "Forbidden to create more instances than the maximum allowed"
          },
          {
            "code": "CONNECTOR-MGMT-6",
            "href": "/api/connector_mgmt/v1/errors/6",
            "id": "6",
            "kind": "Error",
            "reason": "An entity with the specified unique values already exists"
          },
          {
            "code": "CONNECTOR-MGMT-7",
            "href": "/api/connector_mgmt/v1/errors/7",
            "id": "7",
            "kind": "Error",
            "reason": "Resource not found"
          },
          {
            "code": "CONNECTOR-MGMT-8",
            "href": "/api/connector_mgmt/v1/errors/8",
            "id": "8",
            "kind": "Error",
            "reason": "General validation failure"
          },
          {
            "code": "CONNECTOR-MGMT-9",
            "href": "/api/connector_mgmt/v1/errors/9",
            "id": "9",
            "kind": "Error",
            "reason": "Unspecified error"
          },
          {
            "code": "CONNECTOR-MGMT-10",
            "href": "/api/connector_mgmt/v1/errors/10",
            "id": "10",
            "kind": "Error",
            "reason": "HTTP Method not implemented for this endpoint"
          },
          {
            "code": "CONNECTOR-MGMT-11",
            "href": "/api/connector_mgmt/v1/errors/11",
            "id": "11",
            "kind": "Error",
            "reason": "Account is unauthorized to perform this action"
          },
          {
            "code": "CONNECTOR-MGMT-12",
            "href": "/api/connector_mgmt/v1/errors/12",
            "id": "12",
            "kind": "Error",
            "reason": "Required terms have not been accepted"
          },
          {
            "code": "CONNECTOR-MGMT-15",
            "href": "/api/connector_mgmt/v1/errors/15",
            "id": "15",
            "kind": "Error",
            "reason": "Account authentication could not be verified"
          },
          {
            "code": "CONNECTOR-MGMT-17",
            "href": "/api/connector_mgmt/v1/errors/17",
            "id": "17",
            "kind": "Error",
            "reason": "Unable to read request body"
          },
          {
            "code": "CONNECTOR-MGMT-21",
            "href": "/api/connector_mgmt/v1/errors/21",
            "id": "21",
            "kind": "Error",
            "reason": "Bad request"
          },
          {
            "code": "CONNECTOR-MGMT-23",
            "href": "/api/connector_mgmt/v1/errors/23",
            "id": "23",
            "kind": "Error",
            "reason": "Failed to parse search query"
          },
          {
            "code": "CONNECTOR-MGMT-24",
            "href": "/api/connector_mgmt/v1/errors/24",
            "id": "24",
            "kind": "Error",
            "reason": "The maximum number of allowed kafka instances has been reached"
          },
          {
            "code": "CONNECTOR-MGMT-30",
            "href": "/api/connector_mgmt/v1/errors/30",
            "id": "30",
            "kind": "Error",
            "reason": "Provider not supported"
          },
          {
            "code": "CONNECTOR-MGMT-31",
            "href": "/api/connector_mgmt/v1/errors/31",
            "id": "31",
            "kind": "Error",
            "reason": "Region not supported"
          },
          {
            "code": "CONNECTOR-MGMT-32",
            "href": "/api/connector_mgmt/v1/errors/32",
            "id": "32",
            "kind": "Error",
            "reason": "Kafka cluster name is invalid"
          },
          {
            "code": "CONNECTOR-MGMT-33",
            "href": "/api/connector_mgmt/v1/errors/33",
            "id": "33",
            "kind": "Error",
            "reason": "Minimum field length not reached"
          },
          {
            "code": "CONNECTOR-MGMT-34",
            "href": "/api/connector_mgmt/v1/errors/34",
            "id": "34",
            "kind": "Error",
            "reason": "Maximum field length has been depassed"
          },
          {
            "code": "CONNECTOR-MGMT-35",
            "href": "/api/connector_mgmt/v1/errors/35",
            "id": "35",
            "kind": "Error",
            "reason": "Only multiAZ Kafkas are supported, use multi_az=true"
          },
          {
            "code": "CONNECTOR-MGMT-36",
            "href": "/api/connector_mgmt/v1/errors/36",
            "id": "36",
            "kind": "Error",
            "reason": "Kafka cluster name is already used"
          },
          {
            "code": "CONNECTOR-MGMT-37",
            "href": "/api/connector_mgmt/v1/errors/37",
            "id": "37",
            "kind": "Error",
            "reason": "Field validation failed"
          },
          {
            "code": "CONNECTOR-MGMT-38",
            "href": "/api/connector_mgmt/v1/errors/38",
            "id": "38",
            "kind": "Error",
            "reason": "Service account name is invalid"
          },
          {
            "code": "CONNECTOR-MGMT-39",
            "href": "/api/connector_mgmt/v1/errors/39",
            "id": "39",
            "kind": "Error",
            "reason": "Service account desc is invalid"
          },
          {
            "code": "CONNECTOR-MGMT-40",
            "href": "/api/connector_mgmt/v1/errors/40",
            "id": "40",
            "kind": "Error",
            "reason": "Service account id is invalid"
          },
          {
            "code": "CONNECTOR-MGMT-103",
            "href": "/api/connector_mgmt/v1/errors/103",
            "id": "103",
            "kind": "Error",
            "reason": "Synchronous action is not supported, use async=true parameter"
          },
          {
            "code": "CONNECTOR-MGMT-106",
            "href": "/api/connector_mgmt/v1/errors/106",
            "id": "106",
            "kind": "Error",
            "reason": "Failed to create kafka client in the mas sso"
          },
          {
            "code": "CONNECTOR-MGMT-107",
            "href": "/api/connector_mgmt/v1/errors/107",
            "id": "107",
            "kind": "Error",
            "reason": "Failed to get kafka client secret from the mas sso"
          },
          {
            "code": "CONNECTOR-MGMT-108",
            "href": "/api/connector_mgmt/v1/errors/108",
            "id": "108",
            "kind": "Error",
            "reason": "Failed to get kafka client from the mas sso"
          },
          {
            "code": "CONNECTOR-MGMT-109",
            "href": "/api/connector_mgmt/v1/errors/109",
            "id": "109",
            "kind": "Error",
            "reason": "Failed to delete kafka client from the mas sso"
          },
          {
            "code": "CONNECTOR-MGMT-110",
            "href": "/api/connector_mgmt/v1/errors/110",
            "id": "110",
            "kind": "Error",
            "reason": "Failed to create service account"
          },
          {
            "code": "CONNECTOR-MGMT-111",
            "href": "/api/connector_mgmt/v1/errors/111",
            "id": "111",
            "kind": "Error",
            "reason": "Failed to get service account"
          },
          {
            "code": "CONNECTOR-MGMT-112",
            "href": "/api/connector_mgmt/v1/errors/112",
            "id": "112",
            "kind": "Error",
            "reason": "Failed to delete service account"
          },
          {
            "code": "CONNECTOR-MGMT-113",
            "href": "/api/connector_mgmt/v1/errors/113",
            "id": "113",
            "kind": "Error",
            "reason": "Failed to find service account"
          },
          {
            "code": "CONNECTOR-MGMT-120",
            "href": "/api/connector_mgmt/v1/errors/120",
            "id": "120",
            "kind": "Error",
            "reason": "Insufficient quota"
          },
          {
            "code": "CONNECTOR-MGMT-121",
            "href": "/api/connector_mgmt/v1/errors/121",
            "id": "121",
            "kind": "Error",
            "reason": "Failed to check quota"
          },
          {
            "code": "CONNECTOR-MGMT-429",
            "href": "/api/connector_mgmt/v1/errors/429",
            "id": "429",
            "kind": "Error",
            "reason": "Too Many requests"
          },
          {
            "code": "CONNECTOR-MGMT-1000",
            "href": "/api/connector_mgmt/v1/errors/1000",
            "id": "1000",
            "kind": "Error",
            "reason": "An unexpected error happened, please check the log of the service for details"
          }
        ],
        "kind": "ErrorList",
        "page": 1,
        "size": 38,
        "total": 38
      }
      """

  Scenario: Jim creates a connector but later that connector type is removed from the system.  He should
    still be able to list and get the connector, but it's status should note that it has a bad connector
    type and the connector_spec section should be empty since we have lost the json schema for the field.
    The user should still be able to delete the connector.

    Given this is the only scenario running
    Given I am logged in as "Jim"
    When I POST path "/v1/kafka_connectors?async=true" with json body:
      """
      {
        "kind": "Connector",
        "metadata": {
          "name": "example 1",
          "kafka_id":"mykafka"
        },
        "deployment_location": {
          "kind": "addon",
          "cluster_id": "default"
        },
        "connector_type_id": "aws-sqs-source-v1alpha1",
        "kafka": {
          "bootstrap_server": "kafka.hostname",
          "client_id": "myclient",
          "client_secret": "test"
        },
        "connector_spec": {
            "queueNameOrArn": "test",
            "accessKey": "test",
            "secretKey": "test",
            "region": "east"
        }
      }
      """
    Then the response code should be 202
    And the ".status" selection from the response should match "assigning"
    And I store the ".id" selection from the response as ${connector_id}

    When I run SQL "UPDATE connectors SET connector_type_id='foo' WHERE id = '${connector_id}';" expect 1 row to be affected.

    When I GET path "/v1/kafka_connectors?kafka_id=mykafka"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "items": [
          {
            "channel": "stable",
            "kafka": {
              "bootstrap_server": "kafka.hostname",
              "client_id": "myclient"
            },
            "connector_type_id": "foo",
            "deployment_location": {
              "cluster_id": "default",
              "kind": "addon"
            },
            "href": "/api/connector_mgmt/v1/kafka_connectors/${connector_id}",
            "id": "${connector_id}",
            "kind": "Connector",
            "metadata": {
              "created_at": "${response.items[0].metadata.created_at}",
              "kafka_id": "mykafka",
              "name": "example 1",
              "owner": "${response.items[0].metadata.owner}",
              "resource_version": ${response.items[0].metadata.resource_version},
              "updated_at": "${response.items[0].metadata.updated_at}"
            },
            "desired_state": "ready",
            "status": "bad-connector-type"
          }
        ],
        "kind": "ConnectorList",
        "page": 1,
        "size": 1,
        "total": 1
      }
      """

    When I GET path "/v1/kafka_connectors/${connector_id}"
    Then the response code should be 200
    And the response should match json:
      """
      {
          "id": "${connector_id}",
          "kind": "Connector",
          "href": "/api/connector_mgmt/v1/kafka_connectors/${connector_id}",
          "metadata": {
              "kafka_id": "mykafka",
              "owner": "${response.metadata.owner}",
              "name": "example 1",
              "created_at": "${response.metadata.created_at}",
              "updated_at": "${response.metadata.updated_at}",
              "resource_version": ${response.metadata.resource_version}
          },
          "kafka": {
            "bootstrap_server": "kafka.hostname",
            "client_id": "myclient"
          },
          "deployment_location": {
              "kind": "addon",
              "cluster_id": "default"
          },
          "connector_type_id": "foo",
          "channel": "stable",
          "desired_state": "ready",
          "status": "bad-connector-type"
      }
      """

    When I DELETE path "/v1/kafka_connectors/${connector_id}"
    Then the response code should be 204
    And the response should match ""
