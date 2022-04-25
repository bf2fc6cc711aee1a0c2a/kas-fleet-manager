Feature: connector agent API
  In order to deploy connectors to an addon OSD cluster
  As a managed connector agent
  I need to be able to update agent status, get assigned connectors
  and update connector status.

  Background:
    Given the path prefix is "/api/connector_mgmt"
    Given an org admin user named "Jimmy"
    Given an org admin user named "Bobby" in organization "${Jimmy.OrgId}"
    Given a user named "Dr. Evil"
    Given a user named "Shard"
    Given a user named "Shard2"
    Given an admin user named "Ricky Bobby" with roles "connector-fleet-manager-admin-full"
    Given an admin user named "Cal Naughton Jr." with roles "connector-fleet-manager-admin-write"
    Given an admin user named "Carley Bobby" with roles "connector-fleet-manager-admin-read"

  Scenario: connector cluster is created and agent processes assigned a deployment.
    Given I am logged in as "Jimmy"

    #-----------------------------------------------------------------------------------
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

    When I POST path "/v1/kafka_connectors?async=true" with json body:
      """
      {
        "kind": "Connector",
        "name": "example 1",
        "namespace_id": "${connector_namespace_id}",
        "channel":"stable",
        "connector_type_id": "aws-sqs-source-v1alpha1",
        "kafka": {
          "id": "mykafka",
          "url": "kafka.hostname"
        },
        "service_account": {
          "client_id": "myclient",
          "client_secret": "test"
        },
        "schema_registry": {
          "id": "myregistry",
          "url": "registry.hostname"
        },
        "connector": {
            "aws_queue_name_or_arn": "test",
            "aws_secret_key": "test",
            "aws_access_key": "test",
            "aws_region": "east",
            "kafka_topic": "test"
        }
      }
      """
    Then the response code should be 202
    And the ".status.state" selection from the response should match "assigning"
    And I store the ".id" selection from the response as ${connector_id}

    #-----------------------------------------------------------------------------------------------------------------
    # In this part of the Scenario we test that Dr. Evil is not allowed to create a connector in Jimmy's namespace
    #-----------------------------------------------------------------------------------------------------------------
    Given I am logged in as "Dr. Evil"
    When I POST path "/v1/kafka_connectors?async=true" with json body:
      """
      {
        "kind": "Connector",
        "name": "example 1",
        "namespace_id": "${connector_namespace_id}",
        "channel":"stable",
        "connector_type_id": "aws-sqs-source-v1alpha1",
        "kafka": {
          "id": "mykafka",
          "url": "kafka.hostname"
        },
        "service_account": {
          "client_id": "myclient",
          "client_secret": "test"
        },
        "schema_registry": {
          "id": "myregistry",
          "url": "registry.hostname"
        },
        "connector": {
            "aws_queue_name_or_arn": "test",
            "aws_secret_key": "test",
            "aws_access_key": "test",
            "aws_region": "east",
            "kafka_topic": "test"
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
        "reason": "Connector namespace with id='${connector_namespace_id}' not found"
      }
      """

    #-----------------------------------------------------------------------------------------------------------------
    # In this part of the Scenario we test listing/watching connector deployments
    #-----------------------------------------------------------------------------------------------------------------

    # Logs in as the agent..
    Given I am logged in as "Shard"
    Given I set the "Authorization" header to "Bearer ${shard_token}"

    # agent should be able to get namespace details for creating K8s namespaces on data plane
    When I GET path "/v1/agent/kafka_connector_clusters/${connector_cluster_id}/namespaces?gt_version=0"
    Then the response code should be 200
    And the ".total" selection from the response should match "1"
    And the ".items[0].id" selection from the response should match "${connector_namespace_id}"

    When I GET path "/v1/agent/kafka_connector_clusters/${connector_cluster_id}/namespaces/${connector_namespace_id}"
    Then the response code should be 200
    And the ".id" selection from the response should match "${connector_namespace_id}"

    # There should be no deployments assigned yet, since the cluster status is disconnected
    When I GET path "/v1/agent/kafka_connector_clusters/${connector_cluster_id}/deployments"
    Then the response code should be 200
    And the ".kind" selection from the response should match "ConnectorDeploymentList"
    And the ".total" selection from the response should match "0"

    When I GET path "/v1/agent/kafka_connector_clusters/${connector_cluster_id}/deployments?watch=true&gt_version=0" as a json event stream
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
              "id": "",
              "url": ""
            },
            "service_account": {
              "client_id": "",
              "client_secret": ""
            },
            "schema_registry": {
              "id": "",
              "url": ""
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
    Given I am logged in as "Shard2"
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

    # check that the test failure condition is reported in namespace status error
    Given I am logged in as "Jimmy"
    When I GET path "/v1/kafka_connector_namespaces/${connector_namespace_id}"
    Then the response code should be 200
    And the ".status.error" selection from the response should match "Testing: This is a test failure message; Testing2: This is another test failure message"

    # switch back to the previous session
    Given I am logged in as "Shard"
    Given I set the "Authorization" header to "Bearer ${shard_token}"

    Given I wait up to "5" seconds for a response event
    Given I store the ".object.id" selection from the response as ${connector_deployment_id}
    Then the response should match json:
      """
      {
        "type": "CHANGE",
        "error": {},
        "object": {
          "href": "/api/connector_mgmt/v1/agent/kafka_connector_clusters/${connector_cluster_id}/deployments/${connector_deployment_id}",
          "id": "${connector_deployment_id}",
          "kind": "ConnectorDeployment",
          "metadata": {
            "created_at": "${response.object.metadata.created_at}",
            "resource_version": ${response.object.metadata.resource_version},
            "updated_at": "${response.object.metadata.updated_at}"
          },
          "spec": {
            "kafka": {
              "id": "mykafka",
              "url": "kafka.hostname"
            },
            "service_account": {
              "client_id": "myclient",
              "client_secret": "dGVzdA=="
            },
            "schema_registry": {
              "id": "myregistry",
              "url": "registry.hostname"
            },
            "shard_metadata": {
              "connector_image": "quay.io/mock-image:77c0b8763729a9167ddfa19266d83a3512b7aa8124ca53e381d5d05f7d197a24",
              "connector_revision": "5",
              "connector_type": "source",
              "kamelets": {
                "adapter": {
                  "name": "aws-sqs-source",
                  "prefix": "aws"
                },
                "kafka": {
                  "name": "managed-kafka-sink",
                  "prefix": "kafka"
                },
                "processors": {
                  "extract_field": "extract-field-action",
                  "has_header_filter": "has-header-filter-action",
                  "insert_field": "insert-field-action",
                  "throttle": "throttle-action"
                }
              },
              "operators": [
                {
                  "type": "camel-k",
                  "version": "[1.0.0,2.0.0)"
                }
              ]
            },
            "connector_id": "${connector_id}",
            "namespace_id": "${connector_namespace_id}",
            "connector_resource_version": ${response.object.spec.connector_resource_version},
            "connector_type_id": "aws-sqs-source-v1alpha1",
            "connector_spec": {
              "aws_access_key": {
                "kind": "base64",
                "value": "dGVzdA=="
              },
              "aws_queue_name_or_arn": "test",
              "aws_region": "east",
              "aws_secret_key": {
                "kind": "base64",
                "value": "dGVzdA=="
              },
              "kafka_topic": "test"
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

    #-----------------------------------------------------------------------------------------------------------------
    # In this part of the Scenario we test updating the connector status to ready
    #-----------------------------------------------------------------------------------------------------------------

    # at this stage the user will see that the connector is assigned to the cluster.
    Given I am logged in as "Jimmy"
    When I GET path "/v1/kafka_connectors/${connector_id}"
    Then the response code should be 200
    And the ".status.state" selection from the response should match "assigned"

    # Now that the cluster is ready, a worker should assign the connector to the cluster for deployment.
    Given I am logged in as "Shard2"
    Given I set the "Authorization" header to "Bearer ${shard_token}"
    When I GET path "/v1/agent/kafka_connector_clusters/${connector_cluster_id}/deployments"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "items": [
          {
            "href": "/api/connector_mgmt/v1/agent/kafka_connector_clusters/${connector_cluster_id}/deployments/${connector_deployment_id}",
            "kind": "ConnectorDeployment",
            "id": "${response.items[0].id}",
            "metadata": {
              "created_at": "${response.items[0].metadata.created_at}",
              "resource_version": ${response.items[0].metadata.resource_version},
              "updated_at": "${response.items[0].metadata.updated_at}"
            },
            "spec": {
              "kafka": {
                "id": "mykafka",
                "url": "kafka.hostname"
              },
              "service_account": {
                "client_id": "myclient",
                "client_secret": "dGVzdA=="
              },
              "schema_registry": {
                "id": "myregistry",
                "url": "registry.hostname"
              },
              "shard_metadata": {
                "connector_image": "quay.io/mock-image:77c0b8763729a9167ddfa19266d83a3512b7aa8124ca53e381d5d05f7d197a24",
                "connector_revision": "5",
                "connector_type": "source",
                "kamelets": {
                  "adapter": {
                    "name": "aws-sqs-source",
                    "prefix": "aws"
                  },
                  "kafka": {
                    "name": "managed-kafka-sink",
                    "prefix": "kafka"
                  },
                  "processors": {
                    "extract_field": "extract-field-action",
                    "has_header_filter": "has-header-filter-action",
                    "insert_field": "insert-field-action",
                    "throttle": "throttle-action"
                  }
                },
                "operators": [
                  {
                    "type": "camel-k",
                    "version": "[1.0.0,2.0.0)"
                  }
                ]
              },
              "connector_id": "${connector_id}",
              "namespace_id": "${connector_namespace_id}",
              "connector_resource_version": ${response.items[0].spec.connector_resource_version},
              "connector_type_id": "aws-sqs-source-v1alpha1",
              "connector_spec": {
                "aws_access_key": {
                  "kind": "base64",
                  "value": "dGVzdA=="
                },
                "aws_queue_name_or_arn": "test",
                "aws_region": "east",
                "aws_secret_key": {
                  "kind": "base64",
                  "value": "dGVzdA=="
                },
                "kafka_topic": "test"
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
    When I GET path "/v1/agent/kafka_connector_clusters/${connector_cluster_id}/deployments/${connector_deployment_id}"
    Then the response code should be 200
    And the response should match json:
      """
      {
          "href": "/api/connector_mgmt/v1/agent/kafka_connector_clusters/${connector_cluster_id}/deployments/${connector_deployment_id}",
          "kind": "ConnectorDeployment",
          "id": "${response.id}",
          "metadata": {
            "created_at": "${response.metadata.created_at}",
            "resource_version": ${response.metadata.resource_version},
            "updated_at": "${response.metadata.updated_at}"
          },
          "spec": {
            "kafka": {
              "id": "mykafka",
              "url": "kafka.hostname"
            },
            "service_account": {
              "client_id": "myclient",
              "client_secret": "dGVzdA=="
            },
            "schema_registry": {
              "id": "myregistry",
              "url": "registry.hostname"
            },
            "shard_metadata": {
              "connector_image": "quay.io/mock-image:77c0b8763729a9167ddfa19266d83a3512b7aa8124ca53e381d5d05f7d197a24",
              "connector_revision": "5",
              "connector_type": "source",
              "kamelets": {
                "adapter": {
                  "name": "aws-sqs-source",
                  "prefix": "aws"
                },
                "kafka": {
                  "name": "managed-kafka-sink",
                  "prefix": "kafka"
                },
                "processors": {
                  "extract_field": "extract-field-action",
                  "has_header_filter": "has-header-filter-action",
                  "insert_field": "insert-field-action",
                  "throttle": "throttle-action"
                }
              },
              "operators": [
                {
                  "type": "camel-k",
                  "version": "[1.0.0,2.0.0)"
                }
              ]
            },
            "connector_id": "${connector_id}",
            "namespace_id": "${connector_namespace_id}",
            "connector_resource_version": ${response.spec.connector_resource_version},
            "connector_type_id": "aws-sqs-source-v1alpha1",
            "connector_spec": {
              "aws_access_key": {
                "kind": "base64",
                "value": "dGVzdA=="
              },
              "aws_queue_name_or_arn": "test",
              "aws_region": "east",
              "aws_secret_key": {
                "kind": "base64",
                "value": "dGVzdA=="
              },
              "kafka_topic": "test"
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

    # Simulate an error condition when deploying the connector to the data plane
    Given I am logged in as "Shard2"
    Given I set the "Authorization" header to "Bearer ${shard_token}"
    When I PUT path "/v1/agent/kafka_connector_clusters/${connector_cluster_id}/deployments/${connector_deployment_id}/status" with json body:
      """
      {
        "phase":"failed",
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
          "status": "False",
          "lastTransitionTime": "2018-01-01T00:00:00Z",
          "reason": "BadDeploy",
          "message": "error authenticating against kafka"
        }]
      }
      """
    Then the response code should be 204
    And the response should match ""

    # Verify the connector deployment status is updated.
    When I GET path "/v1/agent/kafka_connector_clusters/${connector_cluster_id}/deployments"
    Then the response code should be 200
    And the ".items[0].status.phase" selection from the response should match "failed"

    # Jimmy should now see his connector's error status
    Given I am logged in as "Jimmy"
    When I GET path "/v1/kafka_connectors/${connector_id}"
    Then the response code should be 200
    And the ".status.error" selection from the response should match "BadDeploy: error authenticating against kafka"

    # Move on with a working connector
    Given I am logged in as "Shard2"
    Given I set the "Authorization" header to "Bearer ${shard_token}"
    When I PUT path "/v1/agent/kafka_connector_clusters/${connector_cluster_id}/deployments/${connector_deployment_id}/status" with json body:
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
          "last_transition_time": "2018-01-01T00:00:00Z"
        }]
      }
      """
    Then the response code should be 204
    And the response should match ""

    # Verify the connector deployment status is updated.
    When I GET path "/v1/agent/kafka_connector_clusters/${connector_cluster_id}/deployments"
    Then the response code should be 200
    And the ".items[0].status.phase" selection from the response should match "ready"

    # Jimmy should now see his connector's status update.
    Given I am logged in as "Jimmy"
    When I GET path "/v1/kafka_connectors/${connector_id}"
    Then the response code should be 200
    And the ".status.state" selection from the response should match "ready"

    #-----------------------------------------------------------------------------------------------------------------
    # In this part of the Scenario we test having an user update a connector configuration
    #-----------------------------------------------------------------------------------------------------------------

    # Updating the connector config should update the deployment.
    Given I set the "Content-Type" header to "application/merge-patch+json"
    When I PATCH path "/v1/kafka_connectors/${connector_id}" with json body:
      """
      {
        "connector": {
            "aws_queue_name_or_arn": "I-GOT-PATCHED"
        }
      }
      """

    Then the response code should be 202
    And the response should match json:
      """
      {
        "connector": {
          "aws_access_key": {},
          "aws_queue_name_or_arn": "I-GOT-PATCHED",
          "aws_region": "east",
          "aws_secret_key": {},
          "kafka_topic": "test"
        },
        "connector_type_id": "aws-sqs-source-v1alpha1",
        "namespace_id": "${connector_namespace_id}",
        "href": "/api/connector_mgmt/v1/kafka_connectors/${connector_id}",
        "id": "${connector_id}",
        "kafka": {
          "id": "mykafka",
          "url": "kafka.hostname"
        },
        "service_account": {
          "client_id": "myclient",
          "client_secret": ""
        },
        "schema_registry": {
          "id": "myregistry",
          "url": "registry.hostname"
        },
        "kind": "Connector",
        "name": "example 1",
        "owner": "${response.owner}",
        "created_at": "${response.created_at}",
        "modified_at": "${response.modified_at}",
        "resource_version": ${response.resource_version},
        "channel": "stable",
        "desired_state": "ready",
        "status": {
          "state": "updating"
        }
      }
      """

    # switch back to the previous session
    Given I am logged in as "Shard"
    And I set the "Authorization" header to "Bearer ${shard_token}"

    And I wait up to "5" seconds for a response event
    And I store the ".object.metadata.spec_checksum" selection from the response as ${deployment_spec_checksum}
    And I store the ".object.id" selection from the response as ${connector_deployment_id}
    Then the response should match json:
      """
      {
        "type": "CHANGE",
        "error": {},
        "object": {
          "href": "/api/connector_mgmt/v1/agent/kafka_connector_clusters/${connector_cluster_id}/deployments/${response.object.id}",
          "id": "${response.object.id}",
          "kind": "ConnectorDeployment",
          "metadata": {
            "created_at": "${response.object.metadata.created_at}",
            "resource_version": ${response.object.metadata.resource_version},
            "updated_at": "${response.object.metadata.updated_at}"
          },
          "spec": {
            "kafka": {
              "id": "mykafka",
              "url": "kafka.hostname"
            },
            "service_account": {
              "client_id": "myclient",
              "client_secret": "dGVzdA=="
            },
            "schema_registry": {
              "id": "myregistry",
              "url": "registry.hostname"
            },
            "shard_metadata": {
              "connector_image": "quay.io/mock-image:77c0b8763729a9167ddfa19266d83a3512b7aa8124ca53e381d5d05f7d197a24",
              "connector_revision": "5",
              "connector_type": "source",
              "kamelets": {
                "adapter": {
                  "name": "aws-sqs-source",
                  "prefix": "aws"
                },
                "kafka": {
                  "name": "managed-kafka-sink",
                  "prefix": "kafka"
                },
                "processors": {
                  "extract_field": "extract-field-action",
                  "has_header_filter": "has-header-filter-action",
                  "insert_field": "insert-field-action",
                  "throttle": "throttle-action"
                }
              },
              "operators": [
                {
                  "type": "camel-k",
                  "version": "[1.0.0,2.0.0)"
                }
              ]
            },
            "connector_id": "${connector_id}",
            "namespace_id": "${connector_namespace_id}",
            "connector_resource_version": ${response.object.spec.connector_resource_version},
            "connector_type_id": "aws-sqs-source-v1alpha1",
            "connector_spec": {
              "aws_access_key": {
                "kind": "base64",
                "value": "dGVzdA=="
              },
              "aws_queue_name_or_arn": "I-GOT-PATCHED",
              "aws_region": "east",
              "aws_secret_key": {
                "kind": "base64",
                "value": "dGVzdA=="
              },
              "kafka_topic": "test"
            },
            "desired_state": "ready"
          },
          "status": {
            "conditions": [
              {
                "last_transition_time": "2018-01-01T00:00:00Z",
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

    #-----------------------------------------------------------------------------------------------------------------
    # In this part of the Scenario we test out getting connectors using the admin API
    #-----------------------------------------------------------------------------------------------------------------
    # get cluster
    Given I am logged in as "Ricky Bobby"
    When I GET path "/v1/admin/kafka_connector_clusters"
    Then the response code should be 200
    And the ".items[0]" selection from the response should match json:
      """
      {
        "created_at": "${response.items[0].created_at}",
        "href": "${response.items[0].href}",
        "id": "${response.items[0].id}",
        "kind": "ConnectorCluster",
        "modified_at": "${response.items[0].modified_at}",
        "name": "${response.items[0].name}",
        "owner": "${response.items[0].owner}",
        "status": {
          "state": "${response.items[0].status.state}"
        }
      }
      """

    # get cluster namespaces
    When I GET path "/v1/admin/kafka_connector_clusters/${connector_cluster_id}/namespaces"
    Then the response code should be 200
    And the ".items[0].id" selection from the response should match "${connector_namespace_id}"

    # get cluster connectors
    When I GET path "/v1/admin/kafka_connector_clusters/${connector_cluster_id}/connectors"
    Then the response code should be 200
    And the ".kind" selection from the response should match "ConnectorAdminViewList"
    And the ".items[0].kind" selection from the response should match "ConnectorAdminView"
    And the ".items[0].kafka" selection from the response should match "null"
    And the ".items[0].service_account" selection from the response should match "null"
    And the ".items[0].schema_registry" selection from the response should match "null"
    And the ".items[0].connector" selection from the response should match "null"

    # get cluster deployments
    When I GET path "/v1/admin/kafka_connector_clusters/${connector_cluster_id}/deployments"
    And the response code should be 200
    And the ".kind" selection from the response should match "ConnectorDeploymentAdminViewList"
    And the ".items[0].kind" selection from the response should match "ConnectorDeploymentAdminView"
    And the ".items[0].spec.cluster_id" selection from the response should match "${connector_cluster_id}"
    And the ".items[0].kafka" selection from the response should match "null"
    And the ".items[0].service_account" selection from the response should match "null"
    And the ".items[0].schema_registry" selection from the response should match "null"
    And the ".items[0].connector_spec" selection from the response should match "null"

    # get cluster deployment by id
    When I GET path "/v1/admin/kafka_connector_clusters/${connector_cluster_id}/deployments/${connector_deployment_id}"
    And the response code should be 200
    And the response should match json:
    """
    {
      "href": "/api/connector_mgmt/v1/admin/kafka_connector_clusters/${connector_cluster_id}/deployments/${connector_deployment_id}",
      "id": "${connector_deployment_id}",
      "kind": "ConnectorDeploymentAdminView",
      "metadata": {
        "created_at": "${response.metadata.created_at}",
        "resource_version": ${response.metadata.resource_version},
        "updated_at": "${response.metadata.updated_at}"
      },
      "spec": {
        "cluster_id": "${connector_cluster_id}",
        "connector_id": "${connector_id}",
        "connector_resource_version": ${response.spec.connector_resource_version},
        "connector_type_id": "aws-sqs-source-v1alpha1",
        "desired_state": "ready",
        "namespace_id": "${connector_namespace_id}",
        "shard_metadata": {
          "connector_image": "quay.io/mock-image:77c0b8763729a9167ddfa19266d83a3512b7aa8124ca53e381d5d05f7d197a24",
          "connector_revision": "5",
          "connector_type": "source",
          "kamelets": {
            "adapter": {
              "name": "aws-sqs-source",
              "prefix": "aws"
            },
            "kafka": {
              "name": "managed-kafka-sink",
              "prefix": "kafka"
            },
            "processors": {
              "extract_field": "extract-field-action",
              "has_header_filter": "has-header-filter-action",
              "insert_field": "insert-field-action",
              "throttle": "throttle-action"
            }
          },
          "operators": [
            {
              "type": "camel-k",
              "version": "[1.0.0,2.0.0)"
            }
          ]
        }
      },
      "status": {
        "operators": {
          "assigned": {},
          "available": {}
        }
      }
    }
    """

    # get namespaces
    When I GET path "/v1/admin/kafka_connector_namespaces"
    Then the response code should be 200
    And the ".kind" selection from the response should match "ConnectorNamespaceList"
    And the ".items[0].kind" selection from the response should match "ConnectorNamespace"

    # get cluster namespace
    When I GET path "/v1/admin/kafka_connector_namespaces/${connector_namespace_id}"
    Then the response code should be 200
    And the response should match json:
    """
    {
      "annotations": [
        {
          "key": "connector_mgmt.bf2.org/profile",
          "value": "default-profile"
        }
      ],
      "cluster_id": "${connector_cluster_id}",
      "created_at": "${response.created_at}",
      "href": "/api/connector_mgmt/v1/kafka_connector_namespaces/${connector_namespace_id}",
      "id": "${connector_namespace_id}",
      "kind": "ConnectorNamespace",
      "modified_at": "${response.modified_at}",
      "name": "default-connector-namespace",
      "owner": "${response.owner}",
      "quota": {},
      "resource_version": ${response.resource_version},
      "status": {
        "connectors_deployed": 0,
        "error": "Testing: This is a test failure message; Testing2: This is another test failure message",
        "state": "ready",
        "version": "0.0.1"
      },
      "tenant": {
        "id": "${response.tenant.id}",
        "kind": "organisation"
      }
    }
    """

    # namespace connectors
    And I GET path "/v1/admin/kafka_connector_namespaces/${connector_namespace_id}/connectors"
    And the response code should be 200
    And the ".kind" selection from the response should match "ConnectorAdminViewList"
    And the ".items[0].kind" selection from the response should match "ConnectorAdminView"
    And the ".items[0].kafka" selection from the response should match "null"
    And the ".items[0].service_account" selection from the response should match "null"
    And the ".items[0].schema_registry" selection from the response should match "null"
    And the ".items[0].connector" selection from the response should match "null"

    # namespace deployments
    When I GET path "/v1/admin/kafka_connector_namespaces/${connector_namespace_id}/deployments"
    Then the response code should be 200
    And the ".kind" selection from the response should match "ConnectorDeploymentAdminViewList"
    And the ".items[0].kind" selection from the response should match "ConnectorDeploymentAdminView"
    And the ".items[0].spec.cluster_id" selection from the response should match "${connector_cluster_id}"
    And the ".items[0].kafka" selection from the response should match "null"
    And the ".items[0].service_account" selection from the response should match "null"
    And the ".items[0].schema_registry" selection from the response should match "null"
    And the ".items[0].connector_spec" selection from the response should match "null"

    # connector by id
    When I GET path "/v1/admin/kafka_connectors/${connector_id}"
    Then the response code should be 200
    And the response should match json:
    """
    {
      "id":"${connector_id}",
      "kind":"ConnectorAdminView",
      "href":"/api/connector_mgmt/v1/admin/kafka_connectors/${connector_id}",
      "owner":"${response.owner}",
      "created_at":"${response.created_at}",
      "modified_at":"${response.modified_at}",
      "name":"example 1",
      "connector_type_id":"aws-sqs-source-v1alpha1",
      "namespace_id":"${connector_namespace_id}",
      "channel":"stable",
      "desired_state":"ready",
      "resource_version":${response.resource_version},
      "status":{
        "state":"updating"
      }
    }
    """

    #-----------------------------------------------------------------------------------------------------------------
    # In this part of the Scenario we test out getting connector updates using the admin API
    #-----------------------------------------------------------------------------------------------------------------

    # Now lets verify connector upgrades due to catalog updates
    Given I am logged in as "Ricky Bobby"
    And I GET path "/v1/admin/kafka_connector_clusters/${connector_cluster_id}/upgrades/type"
    And the response code should be 200
    And the response should match json:
      """
      {
       "items": [],
       "kind": "",
       "page": 0,
       "size": 0,
       "total": 0
      }
      """
    Given I am logged in as "Carley Bobby"
    And I GET path "/v1/admin/kafka_connector_clusters/${connector_cluster_id}/upgrades/operator"
    And the response code should be 200
    And the response should match json:
      """
      {
       "items": [],
       "kind": "",
       "page": 0,
       "size": 0,
       "total": 0
      }
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
    Then I GET path "/v1/admin/kafka_connector_clusters/${connector_cluster_id}/upgrades/type"
    And the response code should be 200
    And the response should match json:
      """
      {
       "items":
          [{
            "connector_id": "${connector_id}",
            "namespace_id": "${connector_namespace_id}",
            "connector_type_id": "aws-sqs-source-v1alpha1",
            "channel": "stable",
            "shard_metadata": {
              "assigned_id": ${response.items[0].shard_metadata.assigned_id},
              "available_id": ${response.items[0].shard_metadata.available_id}
            }
          }],
       "kind": "",
       "page": 0,
       "size": 1,
       "total": 1
      }
      """
    And I store the ".items[0].shard_metadata.available_id" selection from the response as ${connector_resource_version}
    And I store the ".items" selection from the response as ${upgrade_items}

    # Upgrade by type
    # Should fail for Carley, who can't write
    Then I PUT path "/v1/admin/kafka_connector_clusters/${connector_cluster_id}/upgrades/type" with json body:
      """
      ${upgrade_items}
      """
    And the response code should be 404

    # Should work for Cal, who can write
    Given I am logged in as "Cal Naughton Jr."
    Then I PUT path "/v1/admin/kafka_connector_clusters/${connector_cluster_id}/upgrades/type" with json body:
      """
      ${upgrade_items}
      """
    And the response code should be 204
    And the response should match ""

    # agent should get updated connector type version
    Given I am logged in as "Shard"
    And I set the "Authorization" header to "Bearer ${shard_token}"
    When I GET path "/v1/agent/kafka_connector_clusters/${connector_cluster_id}/deployments"
    Then the response code should be 200
    And the ".items[0].spec.shard_metadata" selection from the response should match json:
      """"
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

    # type upgrade is not available anymore
    Then I am logged in as "Ricky Bobby"
    And I GET path "/v1/admin/kafka_connector_clusters/${connector_cluster_id}/upgrades/type"
    And the response code should be 200
    And the response should match json:
      """
      {
       "items": [],
       "kind": "",
       "page": 0,
       "size": 0,
       "total": 0
      }
      """

    # Simulate the agent telling us there is an operator upgrade available for the deployment...
    Given I am logged in as "Shard"
    And I set the "Authorization" header to "Bearer ${shard_token}"
    When I PUT path "/v1/agent/kafka_connector_clusters/${connector_cluster_id}/deployments/${connector_deployment_id}/status" with json body:
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

    Then I am logged in as "Ricky Bobby"
    And I GET path "/v1/admin/kafka_connector_clusters/${connector_cluster_id}/upgrades/operator"
    And the response code should be 200
    And the response should match json:
      """
      {
       "items":
          [{
            "connector_id": "${connector_id}",
            "namespace_id": "${connector_namespace_id}",
            "connector_type_id": "aws-sqs-source-v1alpha1",
            "channel": "stable",
            "operator": {
              "assigned_id": "camel-k-1.0.0",
              "available_id": "camel-k-1.0.1"
            }
          }],
       "kind": "",
       "page": 0,
       "size": 1,
       "total": 1
      }
      """
    And I store the ".items" selection from the response as ${upgrade_items}

    # Upgrade by operator
    Then I PUT path "/v1/admin/kafka_connector_clusters/${connector_cluster_id}/upgrades/operator" with json body:
      """
      ${upgrade_items}
      """
    And the response code should be 204
    And the response should match ""

    # agent should get updated operator id
    Given I am logged in as "Shard"
    And I set the "Authorization" header to "Bearer ${shard_token}"
    When I GET path "/v1/agent/kafka_connector_clusters/${connector_cluster_id}/deployments"
    Then the response code should be 200
    And the ".items[0].spec.operator_id" selection from the response should match "camel-k-1.0.1"

    # agent removes available operator upgrade status
    Given I am logged in as "Shard"
    And I set the "Authorization" header to "Bearer ${shard_token}"
    When I PUT path "/v1/agent/kafka_connector_clusters/${connector_cluster_id}/deployments/${connector_deployment_id}/status" with json body:
      """
      {
        "phase":"ready",
        "resource_version": 45,
        "conditions": [{
          "type": "Ready",
          "status": "True",
          "lastTransitionTime": "2018-01-01T00:00:00Z"
        }]
        }
      }
      """
    Then the response code should be 204
    And the response should match ""

    # upgrade is not available anymore
    Then I am logged in as "Ricky Bobby"
    And I GET path "/v1/admin/kafka_connector_clusters/${connector_cluster_id}/upgrades/operator"
    And the response code should be 200
    And the response should match json:
      """
      {
       "items": [],
       "kind": "",
       "page": 0,
       "size": 0,
       "total": 0
      }
      """

    #--------------------------------------------------------------
    # In this part of the Scenario we test admin connector deletion
    #--------------------------------------------------------------
    # create connector to test force delete
    Given I am logged in as "Jimmy"
    When I POST path "/v1/kafka_connectors?async=true" with json body:
      """
      {
        "kind": "Connector",
        "name": "example 2",
        "namespace_id": "${connector_namespace_id}",
        "channel":"stable",
        "connector_type_id": "aws-sqs-source-v1alpha1",
        "kafka": {
          "id": "mykafka",
          "url": "kafka.hostname"
        },
        "service_account": {
          "client_id": "myclient",
          "client_secret": "test"
        },
        "schema_registry": {
          "id": "myregistry",
          "url": "registry.hostname"
        },
        "connector": {
            "aws_queue_name_or_arn": "test",
            "aws_secret_key": "test",
            "aws_access_key": "test",
            "aws_region": "east",
            "kafka_topic": "test"
        }
      }
      """
    Then the response code should be 202
    And the ".status.state" selection from the response should match "assigning"
    And I store the ".id" selection from the response as ${forced_connector_id}

    # api admin should be able to see new connector deployment
    Given I am logged in as "Ricky Bobby"
    When I wait up to "10" seconds for a GET on path "/v1/admin/kafka_connector_clusters/${connector_cluster_id}/deployments" response ".total" selection to match "2"
    Then I GET path "/v1/admin/kafka_connector_clusters/${connector_cluster_id}/deployments"
    And the ".total" selection from the response should match "2"

    # soft delete connector using admin API
    When I DELETE path "/v1/admin/kafka_connectors/${forced_connector_id}"
    Then the response code should be 204

    # assigned connector and deployment should get marked for deletion
    When I GET path "/v1/admin/kafka_connectors/${forced_connector_id}"
    Then the ".desired_state" selection from the response should match "deleted"
    When I GET path "/v1/admin/kafka_connector_clusters/${connector_cluster_id}/deployments"
    Then the response code should be 200
    And the ".total" selection from the response should match "2"

    # force delete connector using admin API
    When I DELETE path "/v1/admin/kafka_connectors/${forced_connector_id}?force=true"
    Then the response code should be 204

    # assigned connector and deployment should get deleted immediately
    When I GET path "/v1/admin/kafka_connectors/${forced_connector_id}"
    Then the response code should be 410
    When I GET path "/v1/admin/kafka_connector_clusters/${connector_cluster_id}/deployments"
    Then the response code should be 200
    And the ".total" selection from the response should match "1"

    # Validate that there exists 1 connector deployment in the DB...
    Given I run SQL "SELECT count(*) FROM connector_deployments WHERE connector_id='${connector_id}' AND deleted_at IS NULL" gives results:
      | count |
      | 1     |

    # Bobby should be able to delete cluster as an org admin
    Given I am logged in as "Bobby"
    When I DELETE path "/v1/kafka_connector_clusters/${connector_cluster_id}"
    Then the response code should be 204
    And the response should match ""

    # agent deletes deployment
    Given I am logged in as "Shard"
    And I set the "Authorization" header to "Bearer ${shard_token}"
    When I PUT path "/v1/agent/kafka_connector_clusters/${connector_cluster_id}/deployments/${connector_deployment_id}/status" with json body:
      """
      {
        "phase":"deleted",
        "resource_version": 45,
        "conditions": [{
          "type": "Ready",
          "status": "True",
          "lastTransitionTime": "2018-01-01T00:00:00Z"
        }]
        }
      }
      """
    Then the response code should be 204
    And the response should match ""

    # Connector deployment should be deleted...
    And I run SQL "SELECT count(*) FROM connector_deployments WHERE connector_id='${connector_id}' AND deleted_at IS NULL" gives results:
      | count |
      | 0     |

    # Connectors that were assigning the cluster get updated to not refer to them.
    Given I am logged in as "Jimmy"
    When I wait up to "10" seconds for a GET on path "/v1/kafka_connectors/${connector_id}" response ".namespace_id" selection to match ""
    Then I GET path "/v1/kafka_connectors/${connector_id}"
    And the response code should be 200
    And the ".desired_state" selection from the response should match "unassigned"
    And the ".namespace_id" selection from the response should match ""

    # delete connector using admin API
    Given I am logged in as "Ricky Bobby"
    When I DELETE path "/v1/admin/kafka_connectors/${connector_id}"
    Then the response code should be 204
    # unassigned connector should get reconciled and deleted
    When I wait up to "10" seconds for a GET on path "/v1/admin/kafka_connectors/${connector_id}" response code to match "410"
    Then I GET path "/v1/admin/kafka_connectors/${connector_id}"
    And the response code should be 410


  Scenario: Bobby can stop and start an existing connector
    Given I am logged in as "Bobby"

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

    # agent should be able to post individual namespace status
    When I PUT path "/v1/agent/kafka_connector_clusters/${connector_cluster_id}/namespaces/${connector_namespace_id}/status" with json body:
      """
      {
        "id": "${connector_namespace_id}",
        "phase": "ready",
        "version": "0.0.1",
        "connectors_deployed": 3,
        "conditions": [{
          "type": "Ready",
          "status": "True",
          "lastTransitionTime": "2018-01-01T00:00:00Z"
        }]
      }
      """
    Then the response code should be 204
    And the response should match ""
    And I run SQL "SELECT status_connectors_deployed FROM connector_namespaces WHERE id='${connector_namespace_id}'" gives results:
      | status_connectors_deployed |
      | 3                          |

    #---------------------------------------------------------------------------------------------
    # Create a connector
    # --------------------------------------------------------------------------------------------
    Given I am logged in as "Bobby"
    When I POST path "/v1/kafka_connectors?async=true" with json body:
      """
      {
        "kind": "Connector",
        "name": "example 1",
        "namespace_id": "${connector_namespace_id}",
        "channel":"stable",
        "connector_type_id": "log_sink_0.1",
        "kafka": {
          "id": "mykafka",
          "url": "kafka.hostname"
        },
        "service_account": {
          "client_id": "myclient",
          "client_secret": "test"
        },
        "connector": {
          "log_multi_line": true,
          "kafka_topic":"test",
          "processors":[]
        }
      }
      """
    Then the response code should be 202
    Given I store the ".id" selection from the response as ${connector_id}

    #-----------------------------------------------------------------------------------------------------------------
    # Shard waits for the deployment, marks it ready, Bobby waits to see ready status.
    #-----------------------------------------------------------------------------------------------------------------
    Given I am logged in as "Shard"
    Given I set the "Authorization" header to "Bearer ${shard_token}"
    Given I wait up to "10" seconds for a GET on path "/v1/agent/kafka_connector_clusters/${connector_cluster_id}/deployments" response ".total" selection to match "1"
    When I GET path "/v1/agent/kafka_connector_clusters/${connector_cluster_id}/deployments"
    Then the ".total" selection from the response should match "1"
    Given I store the ".items[0].id" selection from the response as ${connector_deployment_id}
    When I PUT path "/v1/agent/kafka_connector_clusters/${connector_cluster_id}/deployments/${connector_deployment_id}/status" with json body:
      """
      {
        "phase":"ready",
        "resource_version": 45
      }
      """
    Then the response code should be 204

    Given I am logged in as "Bobby"
    Given I wait up to "10" seconds for a GET on path "/v1/kafka_connectors/${connector_id}" response ".status" selection to match "ready"
    When I GET path "/v1/kafka_connectors/${connector_id}"
    Then the ".status.state" selection from the response should match "ready"

    #-----------------------------------------------------------------------------------------------------------------
    # Bobby sets desired state to stopped.. Agent sees deployment stopped, it updates status to stopped,, Bobby then see stopped status
    #-----------------------------------------------------------------------------------------------------------------
    # Updating the connector config should update the deployment.
    Given I set the "Content-Type" header to "application/merge-patch+json"
    When I PATCH path "/v1/kafka_connectors/${connector_id}" with json body:
      """
      {
        "desired_state": "stopped"
      }
      """
    Then the response code should be 202

    Given I am logged in as "Shard"
    Given I set the "Authorization" header to "Bearer ${shard_token}"
    Given I wait up to "10" seconds for a GET on path "/v1/agent/kafka_connector_clusters/${connector_cluster_id}/deployments/${connector_deployment_id}" response ".spec.desired_state" selection to match "stopped"
    When I GET path "/v1/agent/kafka_connector_clusters/${connector_cluster_id}/deployments/${connector_deployment_id}"
    Then the ".spec.desired_state" selection from the response should match "stopped"
    When I PUT path "/v1/agent/kafka_connector_clusters/${connector_cluster_id}/deployments/${connector_deployment_id}/status" with json body:
      """
      {
        "phase":"stopped",
        "resource_version": 45
      }
      """
    Then the response code should be 204

    Given I am logged in as "Bobby"
    When I GET path "/v1/kafka_connectors/${connector_id}"
    Then the ".status.state" selection from the response should match "stopped"

    #-----------------------------------------------------------------------------------------------------------------
    # Bobby sets desired state to ready.. Agent sees new deployment
    #-----------------------------------------------------------------------------------------------------------------
    Given I set the "Content-Type" header to "application/merge-patch+json"
    When I PATCH path "/v1/kafka_connectors/${connector_id}" with json body:
      """
      {
        "desired_state": "ready"
      }
      """
    Then the response code should be 202

    Given I am logged in as "Shard"
    Given I set the "Authorization" header to "Bearer ${shard_token}"
    Given I wait up to "10" seconds for a GET on path "/v1/agent/kafka_connector_clusters/${connector_cluster_id}/deployments" response ".total" selection to match "1"
    When I GET path "/v1/agent/kafka_connector_clusters/${connector_cluster_id}/deployments"
    Then the ".total" selection from the response should match "1"
    And the ".items[0].spec.desired_state" selection from the response should match "ready"

    When I PUT path "/v1/agent/kafka_connector_clusters/${connector_cluster_id}/deployments/${connector_deployment_id}/status" with json body:
      """
      {
        "phase":"ready",
        "resource_version": 46
      }
      """
    Then the response code should be 204

    #-----------------------------------------------------------------------------------------------------------------
    # Agent try to update status with a stale version and get 500
    #-----------------------------------------------------------------------------------------------------------------
    Given I am logged in as "Shard"
    Given I set the "Authorization" header to "Bearer ${shard_token}"
    When I PUT path "/v1/agent/kafka_connector_clusters/${connector_cluster_id}/deployments/${connector_deployment_id}/status" with json body:
      """
      {
        "phase":"stopped",
        "resource_version": 1
      }
      """
    Then the response code should be 409
    
    #-----------------------------------------------------------------------------------------------------------------
    # Bobby sets desired state to deleted.. Agent sees deployment deleted, it updates status to deleted, Bobby can not see the connector anymore
    #-----------------------------------------------------------------------------------------------------------------
    Given I am logged in as "Bobby"
    When I DELETE path "/v1/kafka_connectors/${connector_id}"
    Then the response code should be 204

    Given I am logged in as "Shard"
    And I set the "Authorization" header to "Bearer ${shard_token}"
    And I wait up to "10" seconds for a GET on path "/v1/agent/kafka_connector_clusters/${connector_cluster_id}/deployments/${connector_deployment_id}" response ".spec.desired_state" selection to match "deleted"
    When I GET path "/v1/agent/kafka_connector_clusters/${connector_cluster_id}/deployments/${connector_deployment_id}"
    Then the ".spec.desired_state" selection from the response should match "deleted"
    When I PUT path "/v1/agent/kafka_connector_clusters/${connector_cluster_id}/deployments/${connector_deployment_id}/status" with json body:
      """
      {
        "phase":"deleted",
        "resource_version": 46
      }
      """
    Then the response code should be 204

    Given I am logged in as "Bobby"
    And I wait up to "10" seconds for a GET on path "/v1/kafka_connectors/${connector_id}" response code to match "410"
    When I GET path "/v1/kafka_connectors/${connector_id}"
    Then the response code should be 410


    #---------------------------------------------------------------------------------------------
    # Validate cluster delete completes with empty namespace removed after agent ack
    # --------------------------------------------------------------------------------------------
    # expire the existing namespace
    Given I run SQL "UPDATE connector_namespaces SET expiration='1000-01-01 10:10:10+00' WHERE id = '${connector_namespace_id}';" expect 1 row to be affected.
    # check that the namespace state is now deleting
    Given I am logged in as "Bobby"
    When I wait up to "10" seconds for a GET on path "/v1/kafka_connector_namespaces/${connector_namespace_id}" response ".status.state" selection to match "deleting"
    Then I GET path "/v1/kafka_connector_namespaces/${connector_namespace_id}"
    And the response code should be 200
    And the ".status.state" selection from the response should match "deleting"

    # delete the cluster
    Given I DELETE path "/v1/kafka_connector_clusters/${connector_cluster_id}"
    Then the response code should be 204
    And the response should match ""

    # check that the cluster state is now `deleting`
    Given I wait up to "10" seconds for a GET on path "/v1/kafka_connector_clusters/${connector_cluster_id}" response ".status.state" selection to match "deleting"
    When I GET path "/v1/kafka_connector_clusters/${connector_cluster_id}"
    Then the response code should be 200
    And the ".status.state" selection from the response should match "deleting"

    # validate namespace processing can handle namespace status update errors and removes deleting namespace
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
          "id": "invalid_namespace_id",
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
    Then the response code should be 500
    And the response should match json:
    """
    {
      "code": "CONNECTOR-MGMT-9",
      "href": "/api/connector_mgmt/v1/errors/9",
      "id": "9",
      "kind": "Error",
      "operation_id": "${response.operation_id}",
      "reason": "CONNECTOR-MGMT-9: Unable to update Connector namespace: CONNECTOR-MGMT-7: Connector namespace with id='invalid_namespace_id' not found"
    }
    """

    # wait for cluster to be deleted
    Given I am logged in as "Bobby"
    Given I wait up to "10" seconds for a GET on path "/v1/kafka_connector_clusters/${connector_cluster_id}" response code to match "410"
    When I GET path "/v1/kafka_connector_clusters/${connector_cluster_id}"
    Then the response code should be 410

    # agent should get 410 for deleted cluster
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
