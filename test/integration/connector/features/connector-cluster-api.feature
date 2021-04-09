Feature: create a a connector
  In order to use connectors api
  As an API user
  I need to be able to manage connectors addon clusters

  Background:
    Given the path prefix is "/api/managed-services-api"
    Given a user named "Greg" in organization "13640203"
    Given a user named "Coworker Sally" in organization "13640203"
    Given a user named "Evil Bob"

  Scenario: Greg creates lists and deletes a connector addon cluster
    Given I am logged in as "Greg"
    Given I have created a kafka cluster as ${kid}
    When I POST path "/v1/kafka-connector-clusters" with json body:
      """
      {}
      """
    Then the response code should be 202
    Given I store the ".id" selection from the response as ${cluster_id}
    And the response should match json:
      """
      {
        "href": "/api/managed-services-api/v1/kafka-connector-clusters/${cluster_id}",
        "id": "${cluster_id}",
        "kind": "ConnectorCluster",
        "metadata": {
          "created_at": "${response.metadata.created_at}",
          "name": "New Cluster",
          "owner": "${response.metadata.owner}",
          "updated_at": "${response.metadata.updated_at}"
        },
        "status": "unconnected"
      }
      """

    When I GET path "/v1/kafka-connector-clusters"
    Then the response code should be 200
    And the ".kind" selection from the response should match "ConnectorClusterList"
    And the ".page" selection from the response should match "1"
    And the ".size" selection from the response should match "1"
    And the ".total" selection from the response should match "1"

    When I GET path "/v1/kafka-connector-clusters/${cluster_id}"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "href": "/api/managed-services-api/v1/kafka-connector-clusters/${cluster_id}",
        "id": "${cluster_id}",
        "kind": "ConnectorCluster",
        "metadata": {
          "created_at": "${response.metadata.created_at}",
          "name": "New Cluster",
          "owner": "${response.metadata.owner}",
          "updated_at": "${response.metadata.updated_at}"
        },
        "status": "unconnected"
      }
      """

    # Before deleting the connector, lets make sure the access control works as expected for other users beside Greg
    Given I am logged in as "Coworker Sally"
    When I GET path "/v1/kafka-connector-clusters/${cluster_id}"
    Then the response code should be 200

    Given I am logged in as "Evil Bob"
    When I GET path "/v1/kafka-connector-clusters/${cluster_id}"
    Then the response code should be 404

    Given I am logged in as "Greg"
    When I DELETE path "/v1/kafka-connector-clusters/${cluster_id}"
    Then the response code should be 204
    And the response should match ""

    When I GET path "/v1/kafka-connector-clusters/${cluster_id}"
    Then the response code should be 404
    And the response should match json:
      """
      {
        "code": "MGD-SERV-API-7",
        "href": "/api/managed-services-api/v1/errors/7",
        "id": "7",
        "kind": "Error",
        "operation_id": "${response.operation_id}",
        "reason": "Connector cluster with id='${cluster_id}' not found"
      }
      """
