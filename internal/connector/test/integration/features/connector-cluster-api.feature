Feature: create a connector
  In order to use connectors api
  As an API user
  I need to be able to manage connectors addon clusters

  Background:
    Given the path prefix is "/api/connector_mgmt"
    Given an org admin user named "Greg" in organization "13640203"
    Given a user named "Coworker Sally" in organization "13640203"
    Given a user named "Evil Bob"

  Scenario: Greg creates lists and deletes a connector addon cluster
    Given I am logged in as "Greg"
    When I POST path "/v1/kafka_connector_clusters" with json body:
      """
      {}
      """
    Then the response code should be 202
    Given I store the ".id" selection from the response as ${cluster_id}
    And the response should match json:
      """
      {
        "href": "/api/connector_mgmt/v1/kafka_connector_clusters/${cluster_id}",
        "id": "${cluster_id}",
        "kind": "ConnectorCluster",
        "created_at": "${response.created_at}",
        "name": "New Cluster",
        "owner": "${response.owner}",
        "modified_at": "${response.modified_at}",
        "annotations": {
          "cos.bf2.org/organisation-id": "13640203"
        },
        "status": {
          "state": "disconnected"
        }
      }
      """

    When I GET path "/v1/kafka_connector_clusters/${cluster_id}/addon_parameters"
    Then the response code should be 200
    And get and store access token using the addon parameter response as ${shard_token} and clientID as ${clientID}
    And I remember keycloak client for cleanup with clientID: ${clientID}

    When I GET path "/v1/kafka_connector_clusters"
    Then the response code should be 200
    And the ".kind" selection from the response should match "ConnectorClusterList"
    And the ".page" selection from the response should match "1"
    And the ".size" selection from the response should match "1"
    And the ".total" selection from the response should match "1"

    When I GET path "/v1/kafka_connector_clusters/${cluster_id}"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "href": "/api/connector_mgmt/v1/kafka_connector_clusters/${cluster_id}",
        "id": "${cluster_id}",
        "kind": "ConnectorCluster",
        "name": "New Cluster",
        "created_at": "${response.created_at}",
        "owner": "${response.owner}",
        "modified_at": "${response.modified_at}",
        "annotations": {
          "cos.bf2.org/organisation-id": "13640203"
        },
        "status": {
          "state": "disconnected"
        }
      }
      """

    #
    # Validate that cluster updates work.
    When I PUT path "/v1/kafka_connector_clusters/${cluster_id}" with json body:
      """
      {
        "name": "My Cluster Name",
        "annotations": {
          "cos.bf2.org/organisation-id": "13640203",
          "test/key": "test-value"
        }
      }
      """
    Then the response code should be 204
    And the response should match ""

    # Check invalid annotations update that tries to change organisation-id
    When I PUT path "/v1/kafka_connector_clusters/${cluster_id}" with json body:
      """
      {
        "name": "My Cluster Name",
        "annotations": {
          "cos.bf2.org/organisation-id": "666",
          "test/key": "test-value"
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
        "reason": "cannot override reserved annotation cos.bf2.org/organisation-id"
      }
      """

    # Check invalid annotations update that try to use reserved domains
    When I PUT path "/v1/kafka_connector_clusters/${cluster_id}" with json body:
      """
      {
        "name": "My Cluster Name",
        "annotations": {
          "cos.bf2.org/organisation-id": "13640203",
          "app.kubernetes.io/component": "my-component"
        }
      }
      """
    Then the response code should be 400
    And the ".reason" selection from the response should match "cannot use reserved annotation app.kubernetes.io/component from domain kubernetes.io/"

    When I PUT path "/v1/kafka_connector_clusters/${cluster_id}" with json body:
      """
      {
        "name": "My Cluster Name",
        "annotations": {
          "cos.bf2.org/organisation-id": "13640203",
          "authorization.k8s.io/decision": "my-decision"
        }
      }
      """
    Then the response code should be 400
    And the ".reason" selection from the response should match "cannot use reserved annotation authorization.k8s.io/decision from domain k8s.io/"

    When I PUT path "/v1/kafka_connector_clusters/${cluster_id}" with json body:
      """
      {
        "name": "My Cluster Name",
        "annotations": {
          "cos.bf2.org/organisation-id": "13640203",
          "openshift.io/cluster-monitoring": "false"
        }
      }
      """
    Then the response code should be 400
    And the ".reason" selection from the response should match "cannot use reserved annotation openshift.io/cluster-monitoring from domain openshift.io/"

    When I GET path "/v1/kafka_connector_clusters/${cluster_id}"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "href": "/api/connector_mgmt/v1/kafka_connector_clusters/${cluster_id}",
        "id": "${cluster_id}",
        "kind": "ConnectorCluster",
        "name": "My Cluster Name",
        "created_at": "${response.created_at}",
        "owner": "${response.owner}",
        "modified_at": "${response.modified_at}",
        "annotations": {
          "cos.bf2.org/organisation-id": "13640203",
          "test/key": "test-value"
        },
        "status": {
          "state": "disconnected"
        }
      }
      """

    # Before deleting the cluster, lets make sure the access control works as expected for other users beside Greg
    Given I am logged in as "Coworker Sally"
    # a user who is part of the org should be able to get cluster details
    When I GET path "/v1/kafka_connector_clusters/${cluster_id}"
    Then the response code should be 200

    # a user who is not org admin should not be able to get addon parameters even if it is part of the org
    When I GET path "/v1/kafka_connector_clusters/${cluster_id}/addon_parameters"
    Then the response code should be 403

    # a user who is not org admin should not be able to delete a cluster even if it is part of the org
    When I DELETE path "/v1/kafka_connector_clusters/${cluster_id}"
    Then the response code should be 403

    Given I am logged in as "Evil Bob"
    # a user who is not part of the org should not be able to get cluster details
    When I GET path "/v1/kafka_connector_clusters/${cluster_id}"
    Then the response code should be 404

    Given I am logged in as "Greg"
    # a user who is org admin should be able to get addon parameters
    When I GET path "/v1/kafka_connector_clusters/${cluster_id}/addon_parameters"
    Then the response code should be 200

    # a user who is org admin should be able to delete a cluster
    When I DELETE path "/v1/kafka_connector_clusters/${cluster_id}"
    Then the response code should be 204
    And the response should match ""

    # wait for cluster cleanup
    Given I wait up to "30" seconds for a GET on path "/v1/kafka_connector_clusters/${cluster_id}" response code to match "410"
    And the response should match json:
      """
      {
        "code": "CONNECTOR-MGMT-25",
        "href": "/api/connector_mgmt/v1/errors/25",
        "id": "25",
        "kind": "Error",
        "operation_id": "${response.operation_id}",
        "reason": "Connector cluster with id='${cluster_id}' has been deleted"
      }
      """
    And I can forget keycloak clientID: ${clientID}
