Feature: create a a connector
  In order to use connectors api
  As an API user
  I need to be able to manage connectors

  Background:
    Given the path prefix is "/api/managed-services-api"
    # Greg and Coworker Sally will end up in the same org
    Given a user named "Greg" in organization "13640203"
    Given a user named "Coworker Sally" in organization "13640203"
    Given a user named "Evil Bob"

  Scenario: Greg lists all connector types
    Given I am logged in as "Greg"
    When I GET path "/v1/kafka-connector-types"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "items": [
          {
            "description": "Receive data from AWS SQS",
            "href": "/api/managed-services-api/v1/kafka-connector-types/aws-sqs-source-v1alpha1",
            "id": "aws-sqs-source-v1alpha1",
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
          }
        ],
        "kind": "ConnectorTypeList",
        "page": 1,
        "size": 1,
        "total": 1
      }
      """

  Scenario: Greg tries to create a connector with an invalid configuration spec
    Given I am logged in as "Greg"
    Given I have created a kafka cluster as ${kid}
    When I POST path "/v1/kafka-connectors?async=true" with json body:
      """
      {
        "kind": "Connector",
        "metadata": {
          "name": "example 1",
          "kafka_id":"${kid}"
        },
        "deployment_location": {
          "kind": "addon",
          "cluster_id": "default"
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
        "code": "MGD-SERV-API-21",
        "href": "/api/managed-services-api/v1/errors/21",
        "id": "21",
        "kind": "Error",
        "operation_id": "${response.operation_id}",
        "reason": "connector spec not conform to the connector type schema. 1 errors encountered.  1st error: (root): queueNameOrArn is required"
      }
      """

  Scenario: Greg creates lists and deletes a connector verifying that Evil Bob can't access Gregs Connectors
  but Coworker Sally can.
    Given I am logged in as "Greg"
    Given I have created a kafka cluster as ${kid}
    When I POST path "/v1/kafka-connectors?async=true" with json body:
      """
      {
        "kind": "Connector",
        "metadata": {
          "name": "example 1",
          "kafka_id":"${kid}"
        },
        "deployment_location": {
          "kind": "addon",
          "cluster_id": "default"
        },
        "connector_type_id": "aws-sqs-source-v1alpha1",
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

    Given I store the ".id" selection from the response as ${cid}
    When I GET path "/v1/kafka-connectors?kafka_id=${kid}"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "items": [
          {
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
            "href": "/api/managed-services-api/v1/kafka-connectors/${cid}",
            "id": "${cid}",
            "kind": "Connector",
            "metadata": {
              "created_at": "${response.items[0].metadata.created_at}",
              "kafka_id": "${kid}",
              "name": "example 1",
              "owner": "${response.items[0].metadata.owner}",
              "resource_version": ${response.items[0].metadata.resource_version},
              "updated_at": "${response.items[0].metadata.updated_at}"
            },
            "status": "assigning"
          }
        ],
        "kind": "ConnectorList",
        "page": 1,
        "size": 1,
        "total": 1
      }
      """

    When I GET path "/v1/kafka-connectors/${cid}"
    Then the response code should be 200
    And the ".status" selection from the response should match "assigning"
    And the ".id" selection from the response should match "${cid}"
    And the response should match json:
      """
      {
          "id": "${cid}",
          "kind": "Connector",
          "href": "/api/managed-services-api/v1/kafka-connectors/${cid}",
          "metadata": {
              "kafka_id": "${kid}",
              "owner": "${response.metadata.owner}",
              "name": "example 1",
              "created_at": "${response.metadata.created_at}",
              "updated_at": "${response.metadata.updated_at}",
              "resource_version": ${response.metadata.resource_version}
          },
          "deployment_location": {
              "kind": "addon",
              "cluster_id": "default"
          },
          "connector_type_id": "aws-sqs-source-v1alpha1",
          "connector_spec": {
              "accessKey": "test",
              "queueNameOrArn": "test",
              "region": "east",
              "secretKey": {}
          },
          "status": "assigning"
      }
      """

    # Before deleting the connector, lets make sure the access control work as expected for other users beside Greg
    Given I am logged in as "Coworker Sally"
    When I GET path "/v1/kafka-connectors/${cid}"
    Then the response code should be 200

    Given I am logged in as "Evil Bob"
    When I GET path "/v1/kafka-connectors/${cid}"
    Then the response code should be 404

    Given I am logged in as "Greg"
    When I DELETE path "/v1/kafka-connectors/${cid}"
    Then the response code should be 204
    And the response should match ""

    When I GET path "/v1/kafka-connectors/${cid}"
    Then the response code should be 404
    And the response should match json:
      """
      {
        "code": "MGD-SERV-API-7",
        "href": "/api/managed-services-api/v1/errors/7",
        "id": "7",
        "kind": "Error",
        "operation_id": "${response.operation_id}",
        "reason": "Connector with id='${cid}' not found"
      }
      """

