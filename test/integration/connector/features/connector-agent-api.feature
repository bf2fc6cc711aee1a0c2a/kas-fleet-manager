Feature: connector agent API
  In order to deploy connectors to an addon OSD cluster
  As a managed connector agent
  I need to be able update agent status, get assigned connectors
  and update connector status.

  Background:
    Given the path prefix is "/api/connector_mgmt"
    Given a user named "Jimmy"
    Given a user named "Agent"
    Given a user named "Agent2"

    Given I am logged in as "Jimmy"

    When I POST path "/v1/kafka_connector_clusters" with json body:
      """
      {}
      """
    Then the response code should be 202
    And the ".status" selection from the response should match "unconnected"
    Given I store the ".id" selection from the response as ${connector_cluster_id}

    Given I have created a kafka cluster as ${kafka_id}
    When I POST path "/v1/kafka_connectors?async=true" with json body:
      """
      {
        "kind": "Connector",
        "metadata": {
          "name": "example 1",
          "kafka_id": "${kafka_id}"
        },
        "deployment_location": {
          "kind": "addon",
          "cluster_id": "${connector_cluster_id}"
        },
        "channel":"stable",
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


    When I GET path "/v1/kafka_connector_clusters/${connector_cluster_id}/addon_parameters"
    Then the response code should be 200
    And get and store access token using the addon parameter response as ${agent_token}

  Scenario: connector cluster is created and agent processes assigned a deployment.

    # Logs in as the agent..
    Given I am logged in as "Agent"
    Given I set the "Authorization" header to "Bearer ${agent_token}"

    # There should be no deployments assigned yet, since the cluster status is unconnected
    When I GET path "/v1/kafka_connector_clusters/${connector_cluster_id}/deployments"
    Then the response code should be 200
    And the ".kind" selection from the response should match "ConnectorDeploymentList"
    And the ".total" selection from the response should match "0"

    When I GET path "/v1/kafka_connector_clusters/${connector_cluster_id}/deployments?watch=true&gt_version=0" as a json event stream
    Then the response code should be 200
    And the response header "Content-Type" should match "application/json;stream=watch"

    Given I wait up to "2" seconds for a response event
    # yeah.. this event is kinda ugly.. upgrading openapi-generator to 5.0.1 should help us omit more fields.
    Then the response should match json:
      """
      {
        "error": {},
        "object": {
          "metadata": {
            "created_at": "0001-01-01T00:00:00Z",
            "updated_at": "0001-01-01T00:00:00Z"
          },
          "spec": {
            "kafka": {
            }
          },
          "status": {
            "operators": {
              "assigned": {},
              "available": {}
            }
          }
        },
        "type": "BOOKMARK"
      }
      """

    # switch to another user session to avoid resetting the event stream.
    Given I am logged in as "Agent2"
    Given I set the "Authorization" header to "Bearer ${agent_token}"

    When I PUT path "/v1/kafka_connector_clusters/${connector_cluster_id}/status" with json body:
      """
      {
        "phase":"ready",
        "version": "0.0.1",
        "conditions": [{
          "type": "Ready",
          "status": "True",
          "lastTransitionTime": "2018-01-01T00:00:00Z"
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

    # switch back to the previous session
    Given I am logged in as "Agent"
    Given I set the "Authorization" header to "Bearer ${agent_token}"

    Given I wait up to "5" seconds for a response event
    Given I store the ".object.id" selection from the response as ${connector_deployment_id}
    Then the response should match json:
      """
      {
        "type": "CHANGE",
        "error": {},
        "object": {
          "href": "/api/connector_mgmt/v1/kafka_connector_clusters/${connector_cluster_id}/deployments/${connector_deployment_id}",
          "id": "${connector_deployment_id}",
          "kind": "ConnectorDeployment",
          "metadata": {
            "created_at": "${response.object.metadata.created_at}",
            "resource_version": ${response.object.metadata.resource_version},
            "updated_at": "${response.object.metadata.updated_at}"
          },
          "spec": {
            "kafka_id": "${kafka_id}",
            "kafka": {
              "bootstrap_server": "kafka.hostname",
              "client_id": "myclient",
              "client_secret": "dGVzdA=="
            },
            "shard_metadata": {
              "meta_image": "quay.io/mock-image:77c0b8763729a9167ddfa19266d83a3512b7aa8124ca53e381d5d05f7d197a24",
              "operators": [
                {
                  "type": "camel-k",
                  "versions": "[1.0.0,2.0.0]"
                }
              ]
            },
            "connector_id": "${connector_id}",
            "connector_resource_version": ${response.object.spec.connector_resource_version},
            "connector_type_id": "aws-sqs-source-v1alpha1",
            "connector_spec": {
              "accessKey": "test",
              "queueNameOrArn": "test",
              "region": "east",
              "secretKey": {
                "kind": "base64",
                "value": "dGVzdA=="
              }
            },
            "desired_state": "ready"
          },
          "status": {
            "operators": {
              "assigned": {},
              "available": {}
            }
          }
        }
      }
      """

    # Now that the cluster is ready, a worker should assign the connector to the cluster for deployment.
    Given I am logged in as "Agent2"
    Given I set the "Authorization" header to "Bearer ${agent_token}"
    When I GET path "/v1/kafka_connector_clusters/${connector_cluster_id}/deployments"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "items": [
          {
            "href": "/api/connector_mgmt/v1/kafka_connector_clusters/${connector_cluster_id}/deployments/${connector_deployment_id}",
            "kind": "ConnectorDeployment",
            "id": "${response.items[0].id}",
            "metadata": {
              "created_at": "${response.items[0].metadata.created_at}",
              "resource_version": ${response.items[0].metadata.resource_version},
              "updated_at": "${response.items[0].metadata.updated_at}"
            },
            "spec": {
              "kafka_id": "${kafka_id}",
              "kafka": {
                "bootstrap_server": "kafka.hostname",
                "client_id": "myclient",
                "client_secret": "dGVzdA=="
              },
              "shard_metadata": {
                "meta_image": "quay.io/mock-image:77c0b8763729a9167ddfa19266d83a3512b7aa8124ca53e381d5d05f7d197a24",
                "operators": [
                  {
                    "type": "camel-k",
                    "versions": "[1.0.0,2.0.0]"
                  }
                ]
              },
              "connector_id": "${connector_id}",
              "connector_resource_version": ${response.items[0].spec.connector_resource_version},
              "connector_type_id": "aws-sqs-source-v1alpha1",
              "connector_spec": {
                "accessKey": "test",
                "queueNameOrArn": "test",
                "region": "east",
                "secretKey": {
                  "kind": "base64",
                  "value": "dGVzdA=="
                }
              },
              "desired_state": "ready"
            },
            "status": {
              "operators": {
                "assigned": {},
                "available": {}
              }
            }
          }
        ],
        "kind": "ConnectorDeploymentList",
        "page": 1,
        "size": 1,
        "total": 1
      }
      """
    When I GET path "/v1/kafka_connector_clusters/${connector_cluster_id}/deployments/${connector_deployment_id}"
    Then the response code should be 200
    And the response should match json:
      """
      {
          "href": "/api/connector_mgmt/v1/kafka_connector_clusters/${connector_cluster_id}/deployments/${connector_deployment_id}",
          "kind": "ConnectorDeployment",
          "id": "${response.id}",
          "metadata": {
            "created_at": "${response.metadata.created_at}",
            "resource_version": ${response.metadata.resource_version},
            "updated_at": "${response.metadata.updated_at}"
          },
          "spec": {
            "kafka_id": "${kafka_id}",
            "kafka": {
              "bootstrap_server": "kafka.hostname",
              "client_id": "myclient",
              "client_secret": "dGVzdA=="
            },
            "shard_metadata": {
              "meta_image": "quay.io/mock-image:77c0b8763729a9167ddfa19266d83a3512b7aa8124ca53e381d5d05f7d197a24",
              "operators": [
                {
                  "type": "camel-k",
                  "versions": "[1.0.0,2.0.0]"
                }
              ]
            },
            "connector_id": "${connector_id}",
            "connector_resource_version": ${response.spec.connector_resource_version},
            "connector_type_id": "aws-sqs-source-v1alpha1",
            "connector_spec": {
              "accessKey": "test",
              "queueNameOrArn": "test",
              "region": "east",
              "secretKey": {
                "kind": "base64",
                "value": "dGVzdA=="
              }
            },
            "desired_state": "ready"
          },
          "status": {
            "operators": {
              "assigned": {},
              "available": {}
            }
          }
      }
      """
    When I PUT path "/v1/kafka_connector_clusters/${connector_cluster_id}/deployments/${connector_deployment_id}/status" with json body:
      """
      {
        "phase":"ready",
        "resource_version": 45,
        "operators": {
          "assigned": {
            "id": "camel-k-1.0.0",
            "type": "camel-k",
            "version": "1.0.0"
          }
        },
        "conditions": [{
          "type": "Ready",
          "status": "True",
          "lastTransitionTime": "2018-01-01T00:00:00Z"
        }]
      }
      """
    Then the response code should be 204
    And the response should match ""

    # Verify the connector deployment status is updated.
    When I GET path "/v1/kafka_connector_clusters/${connector_cluster_id}/deployments"
    Then the response code should be 200
    And the ".items[0].status.phase" selection from the response should match "ready"

    # Jimmy should now see his connector's status update.
    Given I am logged in as "Jimmy"
    When I GET path "/v1/kafka_connectors/${connector_id}"
    Then the response code should be 200
    And the ".status" selection from the response should match "ready"

    # Updating the connector config should update the deployment.
    Given I set the "Content-Type" header to "application/merge-patch+json"
    When I PATCH path "/v1/kafka_connectors/${connector_id}" with json body:
      """
      {
        "connector_spec": {
            "queueNameOrArn": "I-GOT-PATCHED"
        }
      }
      """

    Then the response code should be 202
    And the response should match json:
      """
      {
        "connector_spec": {
          "accessKey": "test",
          "queueNameOrArn": "I-GOT-PATCHED",
          "region": "east",
          "secretKey": {}
        },
        "connector_type_id": "aws-sqs-source-v1alpha1",
        "deployment_location": {
          "kind": "addon",
          "cluster_id": "${connector_cluster_id}"
        },
        "href": "/api/connector_mgmt/v1/kafka_connectors/${connector_id}",
        "id": "${connector_id}",
        "kafka": {
          "bootstrap_server": "kafka.hostname",
          "client_id": "myclient"
        },
        "kind": "Connector",
        "metadata": {
          "name": "example 1",
          "owner": "${response.metadata.owner}",
          "created_at": "${response.metadata.created_at}",
          "kafka_id": "${kafka_id}",
          "updated_at": "${response.metadata.updated_at}",
          "resource_version": ${response.metadata.resource_version}
        },
        "channel": "stable",
        "desired_state": "ready",
        "status": "updating"
      }
      """

    # switch back to the previous session
    Given I am logged in as "Agent"
    Given I set the "Authorization" header to "Bearer ${agent_token}"

    Given I wait up to "5" seconds for a response event
    Given I store the ".object.metadata.spec_checksum" selection from the response as ${deployment_spec_checksum}
    Given I store the ".object.id" selection from the response as ${connector_deployment_id}
    Then the response should match json:
      """
      {
        "type": "CHANGE",
        "error": {},
        "object": {
          "href": "/api/connector_mgmt/v1/kafka_connector_clusters/${connector_cluster_id}/deployments/${response.object.id}",
          "id": "${response.object.id}",
          "kind": "ConnectorDeployment",
          "metadata": {
            "created_at": "${response.object.metadata.created_at}",
            "resource_version": ${response.object.metadata.resource_version},
            "updated_at": "${response.object.metadata.updated_at}"
          },
          "spec": {
            "kafka_id": "${kafka_id}",
            "kafka": {
              "bootstrap_server": "kafka.hostname",
              "client_id": "myclient",
              "client_secret": "dGVzdA=="
            },
            "shard_metadata": {
              "meta_image": "quay.io/mock-image:77c0b8763729a9167ddfa19266d83a3512b7aa8124ca53e381d5d05f7d197a24",
              "operators": [
                {
                  "type": "camel-k",
                  "versions": "[1.0.0,2.0.0]"
                }
              ]
            },
            "connector_id": "${connector_id}",
            "connector_resource_version": ${response.object.spec.connector_resource_version},
            "connector_type_id": "aws-sqs-source-v1alpha1",
            "connector_spec": {
              "accessKey": "test",
              "queueNameOrArn": "I-GOT-PATCHED",
              "region": "east",
              "secretKey": {
                "kind": "base64",
                "value": "dGVzdA=="
              }
            },
            "desired_state": "ready"
          },
          "status": {
            "conditions": [
              {
                "lastTransitionTime": "2018-01-01T00:00:00Z",
                "status": "True",
                "type": "Ready"
              }
            ],
            "operators": {
              "assigned": {
                "id": "camel-k-1.0.0",
                "type": "camel-k",
                "version": "1.0.0"
              },
              "available": {}
            },
            "phase": "ready",
            "resource_version": 45
          }
        }
      }
      """

    # Now lets verify connector upgrades due to catalog updates
    Given connector deployment upgrades available are:
      """
      []
      """

    # Simulate the catalog getting an update
    When update connector catalog of type "aws-sqs-source-v1alpha1" and channel "stable" with shard metadata:
      """
      {
        "meta_image": "quay.io/mock-image:1.0.0",
        "operators": [
          {
            "type": "camel-k",
            "versions": "[2.0.0]"
          }
        ]
      }
      """
    Then connector deployment upgrades available are:
      """
      [{
        "deployment_id": "${connector_deployment_id}",
        "connector_type_id": "aws-sqs-source-v1alpha1",
        "channel": "stable",
        "shard_metadata": {
          "assigned_id": ${response[0].shard_metadata.assigned_id},
          "available_id": ${response[0].shard_metadata.available_id}
        }
      }]
      """

    # Simulate the agent telling us there is an operator upgrade available for the deployment...
    When I PUT path "/v1/kafka_connector_clusters/${connector_cluster_id}/deployments/${connector_deployment_id}/status" with json body:
      """
      {
        "phase":"ready",
        "resource_version": 45,
        "conditions": [{
          "type": "Ready",
          "status": "True",
          "lastTransitionTime": "2018-01-01T00:00:00Z"
        }],
        "operators": {
          "assigned": {
            "id": "camel-k-1.0.0",
            "type": "camel-k",
            "version": "1.0.0"
          },
          "available": {
            "id": "camel-k-1.0.1",
            "type": "camel-k",
            "version": "1.0.1"
          }
        }
      }
      """
    Then the response code should be 204
    And the response should match ""
    And connector deployment upgrades available are:
      """
      [{
        "deployment_id": "${connector_deployment_id}",
        "connector_type_id": "aws-sqs-source-v1alpha1",
        "channel": "stable",
        "operator": {
          "assigned": {
            "id": "camel-k-1.0.0",
            "type": "camel-k",
            "version": "1.0.0"
          },
          "available": {
            "id": "camel-k-1.0.1",
            "type": "camel-k",
            "version": "1.0.1"
          }
        },
        "shard_metadata": {
          "assigned_id": ${response[0].shard_metadata.assigned_id},
          "available_id": ${response[0].shard_metadata.available_id}
        }
      }]
      """

