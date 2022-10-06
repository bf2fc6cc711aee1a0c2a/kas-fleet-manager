Feature: connector cluster admin API

  Background:
    Given the path prefix is "/api/connector_mgmt"
    Given an org admin user named "Jimmy"
    Given a user named "Shard"
    Given an admin user named "Ricky Bobby" with roles "connector-fleet-manager-admin-full"

  Scenario: connector cluster is created and agent update status.
    Given I am logged in as "Jimmy"
    Given I store an UID as ${openshift.id.1}
    Given I store an UID as ${openshift.id.2}

    #------------------------------------------------------------------------------------
    # Create a target cluster, and get the shard access token.
    # -----------------------------------------------------------------------------------

    When I POST path "/v1/kafka_connector_clusters" with json body:
      """
      {}
      """
    Then the response code should be 202
    And the ".status.state" selection from the response should match "disconnected"
    Given I store the ".id" selection from the response as ${connector_cluster_id}

    When I GET path "/v1/kafka_connector_clusters/${connector_cluster_id}/namespaces"
    Then the response code should be 200

    Given I store the ".items[0].id" selection from the response as ${connector_namespace_id}
    When I GET path "/v1/kafka_connector_clusters/${connector_cluster_id}/addon_parameters"
    Then the response code should be 200
    And get and store access token using the addon parameter response as ${shard_token} and clientID as ${clientID}
    And I remember keycloak client for cleanup with clientID: ${clientID}

    Given I am logged in as "Shard"
    Given I set the "Authorization" header to "Bearer ${shard_token}"

    When I PUT path "/v1/agent/kafka_connector_clusters/${connector_cluster_id}/status" with json body:
      """
      {
        "phase":"ready",
        "version": "0.0.1",
        "conditions": [{
          "type": "Ready",
          "status": "True",
          "lastTransitionTime": "2018-01-01T00:00:00Z"
        }],
        "namespaces": [{
          "id": "${connector_namespace_id}",
          "phase": "ready",
          "version": "0.0.1",
          "connectors_deployed": 0,
          "conditions": [
            {
              "type": "Ready",
              "status": "True",
              "lastTransitionTime": "2018-01-01T00:00:00Z"
            }
          ]
        }],
        "operators": [{
          "id":"camelk",
          "version": "1.0",
          "namespace": "openshift-mcs-camelk-1.0",
          "status": "ready"
        }],
        "platform": {
          "type": "OpenShift",
          "id": "${openshift.id.1}",
          "version": "4.10.1"
        }
      }
      """
    Then the response code should be 204
    And the response should match ""

    #------------------------------------------------------------------------------------
    # Get connector cluster status with connector-fleet-manager-admin-full role
    # -----------------------------------------------------------------------------------

    Given I am logged in as "Ricky Bobby"
    When I GET path "/v1/admin/kafka_connector_clusters/${connector_cluster_id}"
    Then the response code should be 200
    And the ".status.conditions[0].type" selection from the response should match "Ready"
    And the ".status.operators[0].status" selection from the response should match "ready"
    And the ".status.platform" selection from the response should match json:
      """
      {
        "type": "OpenShift",
        "id": "${openshift.id.1}",
        "version": "4.10.1"
      }
      """

    Given I am logged in as "Shard"
    Given I set the "Authorization" header to "Bearer ${shard_token}"

    When I PUT path "/v1/agent/kafka_connector_clusters/${connector_cluster_id}/status" with json body:
      """
      {
        "phase":"ready",
        "version": "0.0.1",
        "conditions": [{
          "type": "Ready",
          "status": "True",
          "lastTransitionTime": "2018-01-01T00:00:00Z"
        }],
        "namespaces": [{
          "id": "${connector_namespace_id}",
          "phase": "ready",
          "version": "0.0.1",
          "connectors_deployed": 0,
          "conditions": [
            {
              "type": "Ready",
              "status": "True",
              "lastTransitionTime": "2018-01-01T00:00:00Z"
            }
          ]
        }],
        "operators": [{
          "id":"camelk",
          "version": "1.0",
          "namespace": "openshift-mcs-camelk-1.0",
          "status": "ready"
        }],
        "platform": {
          "type": "OpenShift",
          "id": "${openshift.id.2}"
        }
      }
      """
    Then the response code should be 204
    And the response should match ""

    #------------------------------------------------------------------------------------
    # Get connector cluster status with connector-fleet-manager-admin-full role
    # -----------------------------------------------------------------------------------

    Given I am logged in as "Ricky Bobby"
    When I GET path "/v1/admin/kafka_connector_clusters/${connector_cluster_id}"
    Then the response code should be 200
    And the ".status.conditions[0].type" selection from the response should match "Ready"
    And the ".status.operators[0].status" selection from the response should match "ready"
    And the ".status.platform" selection from the response should match json:
      """
      {
        "type": "OpenShift",
        "id": "${openshift.id.2}",
        "version": "4.10.1"
      }
      """

    #------------------------------------------------------------------------------------
    # Get connector cluster status as org admin
    # -----------------------------------------------------------------------------------

    Given I am logged in as "Jimmy"
    When I GET path "/v1/kafka_connector_clusters/${connector_cluster_id}"
    Then the response code should be 200
    And the ".status.platform" selection from the response should match "null"
    And the ".status.state" selection from the response should match "ready"

    #------------------------------------------------------------------------------------
    # Cleanup
    # -----------------------------------------------------------------------------------

    # delete namespace
    Given I am logged in as "Ricky Bobby"

    When I DELETE path "/v1/admin/kafka_connector_namespaces/${connector_namespace_id}"
    Then the response code should be 204
    And the response should match ""
    When I DELETE path "/v1/admin/kafka_connector_namespaces/${connector_namespace_id}?force=true"
    Then the response code should be 204
    And the response should match ""

    # wait for cluster cleanup
    Given I am logged in as "Jimmy"
    Given I wait up to "30" seconds for a GET on path "/v1/kafka_connector_namespaces/${connector_namespace_id}" response code to match "410"
    When I GET path "/v1/kafka_connector_namespaces/${connector_namespace_id}"
    Then the response code should be 410
    And the response should match json:
      """
      {
        "code": "CONNECTOR-MGMT-25",
        "href": "/api/connector_mgmt/v1/errors/25",
        "id": "25",
        "kind": "Error",
        "operation_id": "${response.operation_id}",
        "reason": "Connector namespace with id='${connector_namespace_id}' has been deleted"
      }
      """

    # delete cluster
    Given I am logged in as "Jimmy"
    When I DELETE path "/v1/kafka_connector_clusters/${connector_cluster_id}"
    Then the response code should be 204
    And the response should match ""

    # wait for cluster cleanup
    Given I am logged in as "Jimmy"
    Given I wait up to "30" seconds for a GET on path "/v1/kafka_connector_clusters/${connector_cluster_id}" response code to match "410"
    When I GET path "/v1/kafka_connector_clusters/${connector_cluster_id}"
    Then the response code should be 410
    And the response should match json:
      """
      {
        "code": "CONNECTOR-MGMT-25",
        "href": "/api/connector_mgmt/v1/errors/25",
        "id": "25",
        "kind": "Error",
        "operation_id": "${response.operation_id}",
        "reason": "Connector cluster with id='${connector_cluster_id}' has been deleted"
      }
      """

    And I can forget keycloak clientID: ${clientID}
