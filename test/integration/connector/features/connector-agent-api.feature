Feature: connector agent API
  In order to deploy connectors to an addon OSD cluster
  As a managed connector agent
  I need to be able update agent status, get assigned connectors
  and update connector status.

  Background:
    Given the path prefix is "/api/managed-services-api"
    Given a user named "Jimmy" in organization "13639843"

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
    Given I set the Authorization header to "Bearer ${agent_token}"

    # There should be no connectors assigned yet, since the cluster status is unconnected
    When I GET path "/v1/kafka-connector-clusters/${cluster_id}/connectors"
    Then the response code should be 200
    And the ".kind" selection from the response should match "ConnectorList"
    And the ".total" selection from the response should match "0"

    When I PUT path "/v1/kafka-connector-clusters/${cluster_id}/status" with json body:
      """
      {"status":"ready"}
      """
    Then the response code should be 204
    And the response should match ""

    # Now that the cluster is ready, a worker should assign the connector to the cluster for deployment.
    Given I wait up to "35" seconds for a GET on path "/v1/kafka-connector-clusters/${cluster_id}/connectors" response ".total" selection to match "1"
    When I GET path "/v1/kafka-connector-clusters/${cluster_id}/connectors"
    Then the response code should be 200
    And the ".total" selection from the response should match "1"

    When I PUT path "/v1/kafka-connector-clusters/${cluster_id}/connectors/${cid}/status" with json body:
      """
      {"status":"ready"}
      """
    Then the response code should be 204
    And the response should match ""
