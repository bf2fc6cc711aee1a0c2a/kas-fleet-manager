Feature: connector admin api
  In order to support the service
  As an Admin API user
  I need to be able to manage the service

  Background:
    Given the path prefix is "/api/connector_mgmt"

    # users with fleet role
    Given an admin user named "Ricky Bobby" with roles "connector-fleet-manager-admin-full"

    # org admins
    Given an org admin user named "Stuart Admin" in organization "13640240"
    Given an org admin user named "Kevin Admin" in organization "13640241"

    # regular users
    Given a user named "Regular Bob"

  Scenario: Ricky lists connector types
    Given LOCK--------------------------------------------------------------

    Given I am logged in as "Ricky Bobby"

    When I GET path "/v1/admin/kafka_connector_types"
    Then the response code should be 200
    And the ".kind" selection from the response should match "ConnectorTypeAdminViewList"
    And the ".page" selection from the response should match "1"

    # there is a fake connector
    And the ".size" selection from the response should match "3"
    And the ".total" selection from the response should match "3"
    And the ".items | length" selection from the response should match "3"

    # log_sink_0.1
    And the ".items[] | select(.id == "log_sink_0.1") | has("schema")" selection from the response should match "true"
    And the ".items[] | select(.id == "log_sink_0.1") | .kind" selection from the response should match "ConnectorTypeAdminView"
    And the ".items[] | select(.id == "log_sink_0.1") | .href" selection from the response should match "/api/connector_mgmt/v1/admin/kafka_connector_types/log_sink_0.1"
    And the ".items[] | select(.id == "log_sink_0.1") | .channels.stable.shard_metadata.connector_image" selection from the response should match "quay.io/mcs_dev/log-sink:0.0.1"

    # aws-sqs-source-v1alpha1
    And the ".items[] | select(.id == "aws-sqs-source-v1alpha1") | has("schema")" selection from the response should match "true"
    And the ".items[] | select(.id == "aws-sqs-source-v1alpha1") | .kind" selection from the response should match "ConnectorTypeAdminView"
    And the ".items[] | select(.id == "aws-sqs-source-v1alpha1") | .href" selection from the response should match "/api/connector_mgmt/v1/admin/kafka_connector_types/aws-sqs-source-v1alpha1"

    # We cannot test this as the catalog is manipulated in the connector-agent-api.feature, hence depending on the execution
    # order fo the tests, the result may be different
    #And the ".items[] | select(.id == "aws-sqs-source-v1alpha1") | .channels.stable.shard_metadata.connector_image" selection from the response should match "quay.io/mock-image:1.0.0"

    When I GET path "/v1/admin/kafka_connector_types?search=name=aws-sqs-source"
    Then the response code should be 200
    And the ".items[] | select(.id == "aws-sqs-source-v1alpha1") | has("schema")" selection from the response should match "true"
    And the ".items[] | select(.id == "aws-sqs-source-v1alpha1") | .kind" selection from the response should match "ConnectorTypeAdminView"
    And the ".items[] | select(.id == "aws-sqs-source-v1alpha1") | .href" selection from the response should match "/api/connector_mgmt/v1/admin/kafka_connector_types/aws-sqs-source-v1alpha1"

    # We cannot test this as the catalog is manipulated in the connector-agent-api.feature, hence depending on the execution
    # order fo the tests, the result may be different
    #And the ".items[] | select(.id == "aws-sqs-source-v1alpha1") | .channels.stable.shard_metadata.connector_image" selection from the response should match "quay.io/mock-image:1.0.0"

    When I GET path "/v1/admin/kafka_connector_types/log_sink_0.1"
    Then the response code should be 200
    And the ".kind" selection from the response should match "ConnectorTypeAdminView"
    And the "has("schema")" selection from the response should match "true"
    And the ".href" selection from the response should match "/api/connector_mgmt/v1/admin/kafka_connector_types/log_sink_0.1"
    And the ".channels.stable.shard_metadata.connector_image" selection from the response should match "quay.io/mcs_dev/log-sink:0.0.1"

    When I GET path "/v1/admin/kafka_connector_types/aws-sqs-source-v1alpha1"
    Then the response code should be 200
    And the ".kind" selection from the response should match "ConnectorTypeAdminView"
    And the "has("schema")" selection from the response should match "true"
    And the ".href" selection from the response should match "/api/connector_mgmt/v1/admin/kafka_connector_types/aws-sqs-source-v1alpha1"
    # We cannot test this as the catalog is manipulated in the connector-agent-api.feature, hence depending on the execution
    # order fo the tests, the result may be different
    #And the ".channels.stable.shard_metadata.connector_image" selection from the response should match "quay.io/mock-image:1.0.0"

    When I GET path "/v1/admin/kafka_connector_types/not_existing"
    Then the response code should be 404

    When I GET path "/v1/admin/kafka_connector_types?page=2"
    Then the response code should be 200
    And the ".kind" selection from the response should match "ConnectorTypeAdminViewList"
    And the ".items | length" selection from the response should match "0"

    When I GET path "/v1/admin/kafka_connector_types?size=1"
    Then the response code should be 200
    And the ".kind" selection from the response should match "ConnectorTypeAdminViewList"
    And the ".items | length" selection from the response should match "1"

    Given I am logged in as "Regular Bob"
    When I GET path "/v1/admin/kafka_connector_types"
    Then the response code should be 404

    And UNLOCK--------------------------------------------------------------

  Scenario: Ricky can lists all connector clusters, others can't
    Given LOCK--------------------------------------------------------------

    Given I am logged in as "Stuart Admin"
    When I POST path "/v1/kafka_connector_clusters" with json body:
      """
      { "name": "stuart_cluster" }
      """
    Then the response code should be 202
    Given I store the ".id" selection from the response as ${stuart_cluster_id}

    When I GET path "/v1/kafka_connector_clusters/${stuart_cluster_id}/addon_parameters"
    Then the response code should be 200
    And get and store access token using the addon parameter response as ${stuart_shard_token} and clientID as ${stuart_clientID}
    And I remember keycloak client for cleanup with clientID: ${stuart_clientID}

    Given I am logged in as "Kevin Admin"
    When I POST path "/v1/kafka_connector_clusters" with json body:
      """
      { "name": "kevin_cluster" }
      """
    Then the response code should be 202
    Given I store the ".id" selection from the response as ${kevin_cluster_id}

    When I GET path "/v1/kafka_connector_clusters/${kevin_cluster_id}/addon_parameters"
    Then the response code should be 200
    And get and store access token using the addon parameter response as ${kevin_shard_token} and clientID as ${kevin_clientID}
    And I remember keycloak client for cleanup with clientID: ${kevin_clientID}

    Given I am logged in as "Ricky Bobby"

    When I GET path "/v1/admin/kafka_connector_clusters"
    Then the response code should be 200
    And the ".kind" selection from the response should match "ConnectorClusterList"
    And the ".items[] | select(.name == "stuart_cluster") | .id" selection from the response should match "${stuart_cluster_id}"
    And the ".items[] | select(.name == "kevin_cluster") | .id" selection from the response should match "${kevin_cluster_id}"

    When I GET path "/v1/admin/kafka_connector_clusters/${stuart_cluster_id}"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "href": "/api/connector_mgmt/v1/kafka_connector_clusters/${stuart_cluster_id}",
        "id": "${stuart_cluster_id}",
        "kind": "ConnectorCluster",
        "name": "stuart_cluster",
        "created_at": "${response.created_at}",
        "owner": "${response.owner}",
        "modified_at": "${response.modified_at}",
        "status": {
          "state": "disconnected"
        }
      }
      """

    When I GET path "/v1/admin/kafka_connector_clusters/${kevin_cluster_id}"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "href": "/api/connector_mgmt/v1/kafka_connector_clusters/${kevin_cluster_id}",
        "id": "${kevin_cluster_id}",
        "kind": "ConnectorCluster",
        "name": "kevin_cluster",
        "created_at": "${response.created_at}",
        "owner": "${response.owner}",
        "modified_at": "${response.modified_at}",
        "status": {
          "state": "disconnected"
        }
      }
      """

    Given I am logged in as "Regular Bob"
    When I GET path "/v1/admin/kafka_connector_clusters"
    Then the response code should be 404
    When I GET path "/v1/admin/kafka_connector_clusters/${stuart_cluster_id}"
    Then the response code should be 404
    When I GET path "/v1/admin/kafka_connector_clusters/${kevin_cluster_id}"
    Then the response code should be 404

    And UNLOCK--------------------------------------------------------------

  Scenario: Ricky Bobby can stop and start a connector
    Given LOCK--------------------------------------------------------------

    Given I am logged in as "Stuart Admin"
    #---------------------------------------------------------------------------------------------
    # Create a target cluster, and get the shard access token, and connect it using the Shard user
    # --------------------------------------------------------------------------------------------
    When I POST path "/v1/kafka_connector_clusters" with json body:
      """
      {}
      """
    Then the response code should be 202
    And the ".status.state" selection from the response should match "disconnected"
    Given I store the ".id" selection from the response as ${connector_cluster_id}

    When I GET path "/v1/kafka_connector_namespaces/?search=cluster_id=${connector_cluster_id}"
    Then the response code should be 200
    Given I store the ".items[0].id" selection from the response as ${connector_namespace_id}

    When I GET path "/v1/kafka_connector_clusters/${connector_cluster_id}/addon_parameters"
    Then the response code should be 200
    And get and store access token using the addon parameter response as ${shard_token} and clientID as ${clientID}
    And I remember keycloak client for cleanup with clientID: ${clientID}

    # agent should be able to post individual namespace status
    Given I am logged in as "Regular Bob"
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
          "conditions": [{
            "type": "Ready",
            "status": "True",
            "lastTransitionTime": "2018-01-01T00:00:00Z"
          }]
        }],
        "operators": [{
          "id":"camelk",
          "version": "1.0",
          "namespace": "openshift-mcs-camelk-1.0",
          "status": "ready"
        }]
      }
      """
    Then the response code should be 204
    And the response should match ""

    Given I am logged in as "Stuart Admin"
    When I POST path "/v1/kafka_connectors?async=true" with json body:
      """
      {
        "kind": "Connector",
        "name": "example 1",
        "namespace_id": "${connector_namespace_id}",
        "connector_type_id": "aws-sqs-source-v1alpha1",
        "kafka": {
          "id":"mykafka",
          "url": "kafka.hostname"
        },
        "schema_registry": {
          "id":"myregistry",
          "url": "registry.hostname"
        },
        "service_account": {
          "client_secret": "test",
          "client_id": "myclient"
        },
        "connector": {
            "aws_queue_name_or_arn": "test",
            "aws_access_key": "test",
            "aws_secret_key": "test",
            "aws_region": "east",
            "kafka_topic": "test"
        }
      }
      """
    Then the response code should be 202
    And the ".status.state" selection from the response should match "assigning"
    Given I store the ".id" selection from the response as ${connector_id}

    Given I am logged in as "Ricky Bobby"
    When I GET path "/v1/admin/kafka_connectors/${connector_id}"
    Then the response code should be 200

    When I PATCH path "/v1/admin/kafka_connectors/${connector_id}" with json body:
        """
        {
            "desired_state": "stopped"
        }
        """
    Then the response code should be 202

    When I GET path "/v1/admin/kafka_connectors/${connector_id}"
    Then the response code should be 200
    And the ".desired_state" selection from the response should match "stopped"

    When I PATCH path "/v1/admin/kafka_connectors/${connector_id}" with json body:
        """
        {
            "desired_state": "ready"
        }
        """
    Then the response code should be 202

    When I GET path "/v1/admin/kafka_connectors/${connector_id}"
    Then the response code should be 200
    And the ".desired_state" selection from the response should match "ready"

    And UNLOCK--------------------------------------------------------------