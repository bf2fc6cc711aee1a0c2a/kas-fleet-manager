# The tests here are to ensure some of the paths are backward compatible. These tests should be removed once we decide the backward compatibility is no longer needed.
Feature: the old connectors path are still valid

  Background:
    Given the path prefix is "/api/connector_mgmt"
    Given a user named "Bob"
    Given a user named "Agent"

  Scenario: Bob lists all connector types
    Given I am logged in as "Bob"
    When I GET path "/v1/kafka-connector-types"
    Then the response code should be 200

  Scenario: Bob tries to create a connector with an invalid configuration spec
    Given I am logged in as "Bob"
    When I POST path "/v1/kafka-connectors?async=true" with json body:
      """
      {
        "kind": "Connector",
        "metadata": {
          "name": "example 1",
          "kafka_id":"mykafka"
        },
        "deployment_location": {
        },
        "kafka": {
          "bootstrap_server": "kafka.hostname",
          "client_id": "myclient",
          "client_secret": "test"
        },
        "connector_type_id": "aws-sqs-source-v1alpha1",
        "connector_spec": {
            "aws_access_key": "test",
            "aws_secret_key": "test",
            "aws_region": "east",
            "kafka_topic": "test"
        }
      }
      """
    Then the response code should be 400

    When I GET path "/v1/kafka-connector-clusters"
    Then the response code should be 200

  Scenario: Agent user should be able to list deployments
    Given I am logged in as "Bob"

    When I POST path "/v1/kafka-connector-clusters" with json body:
      """
      {}
      """
    Then the response code should be 202
    Given I store the ".id" selection from the response as ${connector_cluster_id}

    When I GET path "/v1/kafka-connector-clusters/${connector_cluster_id}/addon_parameters"
    Then the response code should be 200
    And get and store access token using the addon parameter response as ${agent_token} and clientID as ${clientID}
    And I remember keycloak client for cleanup with clientID: ${clientID}

    Given I am logged in as "Agent"
    Given I set the "Authorization" header to "Bearer ${agent_token}"

    # There should be no deployments assigned yet, since the cluster status is disconnected
    When I GET path "/v1/kafka-connector-clusters/${connector_cluster_id}/deployments"
    Then the response code should be 200

    #cleanup
    Given I am logged in as "Bob"
    When I DELETE path "/v1/kafka_connector_clusters/${connector_cluster_id}"
    Then the response code should be 204
    And the response should match ""
