@processors
Feature: create a processor
  In order to use processors api
  As an API user
  I need to be able to manage processors

  Background:
    Given the path prefix is "/api/connector_mgmt"

    # User for eval organization id 13640210 configured in internal/connector/test/integration/feature_test.go:81
    Given an org admin user named "Greg" in organization "13640210"
    Given I store userid for "Greg" as ${greg_user_id}
    Given a user named "Coworker Sally" in organization "13640210"
    Given a user named "Evil Bob"
    Given a user named "Greg_shard"

  Scenario: Coworker Sally tries to sql inject processor listing
    Given I am logged in as "Coworker Sally"
    When I GET path "/v2alpha1/processors?orderBy=CAST(CHR(32)||(SELECT+version())+AS+NUMERIC)"
    Then the response code should be 400
    And the response should match json:
      """
      {
        "id":"17",
        "kind":"Error",
        "href":"/api/connector_mgmt/v1/errors/17",
        "code":"CONNECTOR-MGMT-17",
        "reason":"Unable to list processor requests: invalid order by clause 'CAST(CHR(32)||(SELECT version()) AS NUMERIC)'",
        "operation_id": "${response.operation_id}"
      }
      """

    When I GET path "/v2alpha1/processors?search=CAST(CHR(32)||(SELECT+version())+AS+NUMERIC)"
    Then the response code should be 400
      """
      {
        "id":"23",
        "kind":"Error",
        "href":"/api/connector_mgmt/v1/errors/23",
        "code":"CONNECTOR-MGMT-23",
        "reason":"Unable to list processor requests: [1] error parsing the filter: invalid column name: 'CAST'",
        "operation_id": "${response.operation_id}"
      }
      """

  Scenario: Coworker Sally tries to create a processor with an invalid namespace_id
    Given I am logged in as "Coworker Sally"
    When I POST path "/v2alpha1/processors?async=true" with json body:
      """
      {
        "kind": "Processor",
        "name": "example 1",
        "namespace_id": "default",
        "service_account": {
          "client_secret": "test",
          "client_id": "myclient"
        },
        "definition": { "from": { "uri": "kafka:my-topic", "steps": []} }
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
        "reason": "Connector namespace with id='default' not found"
      }
      """

  Scenario: Create eval namespace and create processor
    Given I am logged in as "Greg"
    When I POST path "/v1/kafka_connector_clusters" with json body:
     """
     {
      "name": "Evaluation Cluster"
     }
     """
    Then the response code should be 202
    And the ".status.state" selection from the response should match "disconnected"

    Given I store the ".id" selection from the response as ${connector_cluster_id}
    When I GET path "/v1/kafka_connector_clusters/${connector_cluster_id}/addon_parameters"
    Then the response code should be 200
    And get and store access token using the addon parameter response as ${shard_token} and clientID as ${clientID}
    And I remember keycloak client for cleanup with clientID: ${clientID}

    When I GET path "/v1/kafka_connector_clusters/${connector_cluster_id}/namespaces"
    Then the response code should be 200
    Given I store the ".items[0].id" selection from the response as ${connector_namespace_id}

    # Start the cluster to create eval namespace
    Given I am logged in as "Greg_shard"
    And I set the "Authorization" header to "Bearer ${shard_token}"
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
            },
            {
              "type": "NamespaceDeletionContentFailure",
              "status": "True",
              "lastTransitionTime": "2018-01-01T00:00:00Z",
              "reason": "Testing",
              "message": "This is a test failure message"
            },
            {
              "type": "NamespaceDeletionDiscoveryFailure",
              "status": "True",
              "lastTransitionTime": "2018-01-01T00:00:00Z",
              "reason": "Testing2",
              "message": "This is another test failure message"
            }
          ]
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

   # Create an eval namespace
    Given I am logged in as "Greg"
    When I POST path "/v1/kafka_connector_namespaces/eval" with json body:
    """
    {
      "name": "Greg_namespace",
      "annotations": { "cos.bf2.org/profile": "evaluation-profile" }
    }
    """
    Then the response code should be 201
    And the response should match json:
    """
    {
      "id": "${response.id}",
      "kind": "ConnectorNamespace",
      "href": "/api/connector_mgmt/v1/kafka_connector_namespaces/${response.id}",
      "name": "Greg_namespace",
      "owner": "${greg_user_id}",
      "resource_version": ${response.resource_version},
      "cluster_id": "${response.cluster_id}",
      "created_at": "${response.created_at}",
      "modified_at": "${response.modified_at}",
      "expiration": "${response.expiration}",
      "quota": {
        "connectors": 4,
        "cpu_limits": "2",
        "cpu_requests": "1",
        "memory_limits": "2Gi",
        "memory_requests": "1Gi"
      },
      "annotations": {
          "cos.bf2.org/profile": "evaluation-profile"
      },
      "status": {
        "state": "disconnected",
        "connectors_deployed": 0
      },
      "tenant": {
        "kind": "user",
        "id": "${greg_user_id}"
      }
    }
    """
    And I store the ".id" selection from the response as ${namespace_id}

    # Create a Processor in the provisioned namespace
    When I POST path "/v2alpha1/processors?async=true" with json body:
      """
      {
        "kind": "Processor",
        "name": "example 1",
        "namespace_id": "${namespace_id}",
        "kafka": {
          "id": "mykafka-id",
          "url": "mykafka-url"
        },
        "service_account": {
          "client_id": "myclient",
          "client_secret": "myclient_secret"
        },
        "definition": { "from": { "uri": "kafka:my-topic", "steps": []} }
      }
      """
    Then the response code should be 202
    And the response should match json:
      """
      {
        "id": "${response.id}",
        "kind": "Processor",
        "href": "/api/connector_mgmt/v2alpha1/processors/${response.id}",
        "owner": "${greg_user_id}",
        "created_at": "${response.created_at}",
        "modified_at": "${response.modified_at}",
        "name": "example 1",
        "namespace_id": "${namespace_id}",
        "processor_type_id": "processor_v1",
        "desired_state": "ready",
        "annotations": {
          "cos.bf2.org/organisation-id": "13640210"
        },
        "resource_version": 1,
        "kafka": {
          "id": "mykafka-id",
          "url": "mykafka-url"
        },
        "service_account": {
          "client_id": "myclient",
          "client_secret": ""
        },
        "definition": { "from": { "uri": "kafka:my-topic", "steps": []} },
        "error_handler": {
          "dead_letter_queue": {},
          "log": null,
          "stop": {}
        },
        "status": {
          "state":"preparing"
        }
      }
      """
    And I store the ".id" selection from the response as ${processor_id}

    # Get processor
    #Given I wait up to "35" seconds for a GET on path "/v2alpha1/processors/${processor_id}" response ".status.state" selection to match "prepared"
    When I GET path "/v2alpha1/processors/${processor_id}"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "id": "${processor_id}",
        "kind": "Processor",
        "href": "/api/connector_mgmt/v2alpha1/processors/${processor_id}",
        "created_at": "${response.created_at}",
        "modified_at": "${response.modified_at}",
        "name": "example 1",
        "namespace_id": "${namespace_id}",
        "processor_type_id": "processor_v1",
        "owner": "${greg_user_id}",
        "desired_state": "ready",
        "resource_version": 1,
        "definition": { "from": { "uri": "kafka:my-topic", "steps": []} },
        "error_handler": {
          "dead_letter_queue": {},
          "log": null,
          "stop": {}
        },
        "kafka": {
          "id": "mykafka-id",
          "url": "mykafka-url"
        },
        "service_account": {
          "client_id": "myclient",
          "client_secret": ""
        },
        "status": {
          "state": "${response.status.state}"
        },
        "annotations": {
          "cos.bf2.org/organisation-id": "13640210"
        }
      }
      """

    # List processors
    When I GET path "/v2alpha1/processors"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "items": [
      {
        "id": "${processor_id}",
        "kind": "Processor",
        "href": "${response.items[0].href}",
        "created_at": "${response.items[0].created_at}",
        "modified_at": "${response.items[0].modified_at}",
        "name": "example 1",
        "namespace_id": "${namespace_id}",
        "processor_type_id": "processor_v1",
        "owner": "${greg_user_id}",
        "desired_state": "ready",
        "resource_version": 1,
        "definition": { "from": { "uri": "kafka:my-topic", "steps": []} },
        "error_handler": {
          "dead_letter_queue": {},
          "log": null,
          "stop": {}
        },
        "kafka": {
          "id": "mykafka-id",
          "url": "mykafka-url"
        },
        "service_account": {
          "client_id": "myclient",
          "client_secret": ""
        },
        "status": {
          "state": "${response.items[0].status.state}"
        },
        "annotations": {
          "cos.bf2.org/organisation-id": "13640210"
        }
      }
        ],
        "kind": "ProcessorList",
        "page": 1,
        "size": 1,
        "total": 1
      }
      """

    # Try to create another Processor with an invalid processor_type_id
    When I POST path "/v2alpha1/processors?async=true" with json body:
      """
      {
        "kind": "Processor",
        "name": "example 1",
        "namespace_id": "${namespace_id}",
        "processor_type_id": "invalid",
        "kafka": {
          "id": "mykafka-id",
          "url": "mykafka-url"
        },
        "service_account": {
          "client_id": "myclient",
          "client_secret": "myclient_secret"
        },
        "definition": { "from": { "uri": "kafka:my-topic", "steps": []} }
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
        "reason": "processor_type_id is not valid. Must be one of: processor_v1"
      }
      """

    # Check immutable fields
    Given I set the "Content-Type" header to "application/merge-patch+json"
    When I PATCH path "/v2alpha1/processors/${processor_id}" with json body:
      """
      { "processor_type_id": "new-processor-type-id" }
      """
    Then the response code should be 409
    And the response should match json:
      """
      {
        "code": "CONNECTOR-MGMT-21",
        "href": "/api/connector_mgmt/v1/errors/21",
        "id": "21",
        "kind": "Error",
        "operation_id": "${response.operation_id}",
        "reason": "An attempt was made to modify one or more immutable field(s): processor_type_id"
      }
      """

    Given I set the "Content-Type" header to "application/merge-patch+json"
    When I PATCH path "/v2alpha1/processors/${processor_id}" with json body:
      """
      { "namespace_id": "new-namespace_id" }
      """
    Then the response code should be 409
    And the response should match json:
      """
      {
        "code": "CONNECTOR-MGMT-21",
        "href": "/api/connector_mgmt/v1/errors/21",
        "id": "21",
        "kind": "Error",
        "operation_id": "${response.operation_id}",
        "reason": "An attempt was made to modify one or more immutable field(s): namespace_id"
      }
      """

    Given I set the "Content-Type" header to "application/merge-patch+json"
    When I PATCH path "/v2alpha1/processors/${processor_id}" with json body:
      """
      { "channel": "beta" }
      """
    Then the response code should be 409
    And the response should match json:
      """
      {
        "code": "CONNECTOR-MGMT-21",
        "href": "/api/connector_mgmt/v1/errors/21",
        "id": "21",
        "kind": "Error",
        "operation_id": "${response.operation_id}",
        "reason": "An attempt was made to modify one or more immutable field(s): channel"
      }
      """

    Given I set the "Content-Type" header to "application/merge-patch+json"
    When I PATCH path "/v2alpha1/processors/${processor_id}" with json body:
      """
      {
        "processor_type_id": "new-processor-type-id",
        "namespace_id": "new-namespace_id",
        "channel": "beta"
      }
      """
    Then the response code should be 409
    And the response should match json:
      """
      {
        "code": "CONNECTOR-MGMT-21",
        "href": "/api/connector_mgmt/v1/errors/21",
        "id": "21",
        "kind": "Error",
        "operation_id": "${response.operation_id}",
        "reason": "An attempt was made to modify one or more immutable field(s): namespace_id, processor_type_id, channel"
      }
      """

    # Check update
    Given I set the "Content-Type" header to "application/json-patch+json"
    When I PATCH path "/v2alpha1/processors/${processor_id}" with json body:
      """
      [
        { "op": "replace", "path": "/name", "value": "my-new-name" }
      ]
      """
    Then the response code should be 202
    And the ".name" selection from the response should match "my-new-name"

    # Check annotations - Add protected annotation
    Given I set the "Content-Type" header to "application/merge-patch+json"
    When I PATCH path "/v2alpha1/processors/${processor_id}" with json body:
      """
      {
          "annotations": {
            "cos.bf2.org/organisation-id": "13640210",
            "cos.bf2.org/pricing-tier": "free"
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
        "reason": "cannot override reserved annotation cos.bf2.org/pricing-tier"
      }
      """

    # Check annotations - Update protected annotation
    Given I set the "Content-Type" header to "application/merge-patch+json"
    When I PATCH path "/v2alpha1/processors/${processor_id}" with json body:
      """
      {
          "annotations": {
            "cos.bf2.org/organisation-id": "666"
          }
      }
      """
    Then the response code should be 400
    And the ".reason" selection from the response should match "cannot override reserved annotation cos.bf2.org/organisation-id"

    # Check annotations - Add custom annotation
    Given I set the "Content-Type" header to "application/merge-patch+json"
    When I PATCH path "/v2alpha1/processors/${processor_id}" with json body:
      """
      {
          "annotations": {
            "cos.bf2.org/organisation-id": "13640210",
            "custom/my-key": "my-value"
          }
      }
      """
    Then the response code should be 202
    And the ".annotations" selection from the response should match json:
      """
      {
          "cos.bf2.org/organisation-id": "13640210",
          "custom/my-key": "my-value"
      }
      """

    # Check annotations - Remove custom annotation
    Given I set the "Content-Type" header to "application/merge-patch+json"
    When I PATCH path "/v2alpha1/processors/${processor_id}" with json body:
      """
      {
          "annotations": {
            "cos.bf2.org/organisation-id": "13640210",
            "custom/my-key": null
          }
      }
      """
    Then the response code should be 202
    And the ".annotations" selection from the response should match json:
      """
      {
          "cos.bf2.org/organisation-id": "13640210"
      }
      """

    # Check overriding service account credentials
    Given I set the "Content-Type" header to "application/merge-patch+json"
    When I PATCH path "/v2alpha1/processors/${processor_id}" with json body:
      """
      {
          "service_account": {
            "client_secret": {
                "ref": "hack"
            },
            "client_id": "myclient"
          }
      }
      """
    Then the response code should be 500
    And the response should match json:
      """
      {
        "code": "CONNECTOR-MGMT-9",
        "href": "/api/connector_mgmt/v1/errors/9",
        "id": "9",
        "kind": "Error",
        "operation_id": "${response.operation_id}",
        "reason": "failed to decode patched resource: json: cannot unmarshal object into Go struct field ServiceAccount.service_account.client_secret of type string"
      }
      """

    # Check that we can update secrets of a processor
    Given LOCK--------------------------------------------------------------
    Given I reset the vault counters
    Given I set the "Content-Type" header to "application/json"
    When I PATCH path "/v2alpha1/processors/${processor_id}" with json body:
        """
        {
            "service_account": {
              "client_secret": "patched_client_secret"
            }
        }
        """
    Then the response code should be 202
    And the vault insert counter should be 1
    And UNLOCK---------------------------------------------------------------

    # Before deleting the processor, lets make sure the access control work as expected for other users beside Greg
    Given I am logged in as "Greg"
    When I GET path "/v2alpha1/processors/${processor_id}"
    Then the response code should be 200

    Given I am logged in as "Evil Bob"
    When I GET path "/v2alpha1/processors/${processor_id}"
    Then the response code should be 404

    # We are going to delete the connector...
    Given LOCK--------------------------------------------------------------
    Given I reset the vault counters
    Given I am logged in as "Greg"
    When I DELETE path "/v2alpha1/processors/${processor_id}"
    Then the response code should be 204
    And the response should match ""

    # The delete occurs async in a worker, so we have to wait a little for the counters to update.
    Given I sleep for 10 seconds
    Then the vault delete counter should be 1
    Given UNLOCK--------------------------------------------------------------

    Given I wait up to "10" seconds for a GET on path "/v2alpha1/processors/${processor_id}" response code to match "404"
    When I GET path "/v2alpha1/processors/${processor_id}"
    Then the response code should be 404
    And the response should match json:
      """
      {
        "code": "CONNECTOR-MGMT-7",
        "href": "/api/connector_mgmt/v1/errors/7",
        "id": "7",
        "kind": "Error",
        "operation_id": "${response.operation_id}",
        "reason": "Processor with id='${processor_id}' not found"
      }
      """
