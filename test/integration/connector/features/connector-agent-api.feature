Feature: connector agent API
  In order to deploy connectors to an addon OSD cluster
  As a managed connector agent
  I need to be able update agent status, get assigned connectors
  and update connector status.

  Background:
    Given the path prefix is "/api/managed-services-api"
    Given a user named "Jimmy" in organization "13639843"
    Given a user named "Other"

    Given I am logged in as "Jimmy"
    Given I have created a kafka cluster as ${kid}
    When I POST path "/v1/kafkas/${kid}/connector-deployments?async=true" with json body:
      """
      {
        "kind": "Connector",
        "metadata": {
          "name": "example 1"
        },
        "deployment_location": {
          "kind": "addon",
          "group": "default"
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
    When I POST path "/v1/kafka-connector-clusters" with json body:
      """
      {}
      """
    Then the response code should be 202
    And the ".status" selection from the response should match "unconnected"

    Given I store the ".id" selection from the response as ${cluster_id}
    When I GET path "/v1/kafka-connector-clusters/${cluster_id}/addon-parameters"
    Then the response code should be 200
    And get and store access token using the addon parameter response as ${agent_token}

  Scenario: connector cluster is created and agent processes assigned connectors.

    # Logs in as the agent..
    Given I am logged in as "Jimmy"
    Given I set the Authorization header to "Bearer ${agent_token}"

    # There should be no connectors assigned yet, since the cluster status is unconnected
    When I GET path "/v1/kafka-connector-clusters/${cluster_id}/connectors"
    Then the response code should be 200
    And the ".kind" selection from the response should match "ConnectorList"
    And the ".total" selection from the response should match "0"

    When I GET path "/v1/kafka-connector-clusters/${cluster_id}/connectors?watch=true&gt_version=0" as a json event stream
    Then the response code should be 200
    And the response header "Content-Type" should match "application/json;stream=watch"

    Given I wait up to "2" seconds for a response event
    # yeah.. this event is kinda ugly.. upgrading openapi-generator to 5.0.1 should help us omit more fields.
    Then the response should match json:
      """
      {
        "error": {},
        "object": {
          "deployment_location": {
            "kind": ""
          },
          "metadata": {
            "created_at": "0001-01-01T00:00:00Z",
            "updated_at": "0001-01-01T00:00:00Z"
          }
        },
        "type": "BOOKMARK"
      }
      """

    # switch to another user to avoid reseting Jimmy's event stream..
    Given I am logged in as "Other"
    Given I set the Authorization header to "Bearer ${agent_token}"

    When I PUT path "/v1/kafka-connector-clusters/${cluster_id}/status" with json body:
      """
      {"status":"ready"}
      """
    Then the response code should be 204
    And the response should match ""

    # switch back to the previous Jimmy
    Given I am logged in as "Jimmy"
    Given I set the Authorization header to "Bearer ${agent_token}"

    Given I wait up to "5" seconds for a response event
    Then the response should match json:
      """
      {
        "error": {},
        "object": {
          "connector_spec": {
            "accessKey": "test",
            "queueNameOrArn": "test",
            "region": "east",
            "secretKey": {
              "kind": "base64",
              "value": "dGVzdA=="
            }
          },
          "connector_type_id": "aws-sqs-source-v1alpha1",
          "deployment_location": {
            "group": "default",
            "kind": "addon"
          },
          "href": "${response.object.href}",
          "id": "${response.object.id}",
          "kind": "Connector",
          "metadata": {
            "name": "example 1",
            "created_at": "${response.object.metadata.created_at}",
            "kafka_id": "${response.object.metadata.kafka_id}",
            "owner": "${response.object.metadata.owner}",
            "resource_version": ${response.object.metadata.resource_version},
            "updated_at": "${response.object.metadata.updated_at}"
          },
          "status": "assigned"
        },
        "type": "CHANGE"
      }
      """

    # Now that the cluster is ready, a worker should assign the connector to the cluster for deployment.
    Given I wait up to "5" seconds for a GET on path "/v1/kafka-connector-clusters/${cluster_id}/connectors" response ".total" selection to match "1"
    When I GET path "/v1/kafka-connector-clusters/${cluster_id}/connectors"
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
              "secretKey": {
                "kind": "base64",
                "value": "dGVzdA=="
              }
            },
            "connector_type_id": "aws-sqs-source-v1alpha1",
            "deployment_location": {
              "group": "default",
              "kind": "addon"
            },
            "href": "/api/managed-services-api/v1/kafkas/${kid}/connector-deployments/${cid}",
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
            "status": "assigned"
          }
        ],
        "kind": "ConnectorList",
        "page": 1,
        "size": 1,
        "total": 1
      }
      """

    When I PUT path "/v1/kafka-connector-clusters/${cluster_id}/connectors/${cid}/status" with json body:
      """
      {"status":"ready"}
      """
    Then the response code should be 204
    And the response should match ""
