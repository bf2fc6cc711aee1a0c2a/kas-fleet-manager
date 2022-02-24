Feature: connector agent API
  In order to deploy connectors to an addon OSD cluster
  As a managed connector agent
  I need to be able to update agent status, get assigned connectors
  and update connector status.

  Background:
    Given the path prefix is "/api/connector_mgmt"
    Given a user named "Jimmy"
    Given a user named "Bobby"
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

    When I GET path "/v1/kafka_connector_namespaces/?search=cluster_id=${connector_cluster_id}"
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
        "deployment_location": {
          "namespace_id": "${connector_namespace_id}"
        },
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
    Given I store the ".id" selection from the response as ${connector_id}

    #-----------------------------------------------------------------------------------------------------------------
    # In this part of the Scenario we test listing/watching connector deployments
    #-----------------------------------------------------------------------------------------------------------------

    # Logs in as the agent..
    Given I am logged in as "Shard"
    Given I set the "Authorization" header to "Bearer ${shard_token}"

    # There should be no deployments assigned yet, since the cluster status is disconnected
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
          "href": "/api/connector_mgmt/v1/kafka_connector_clusters/${connector_cluster_id}/deployments/${connector_deployment_id}",
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
            "namespace_name": "default-connector-namespace",
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
              "namespace_name": "default-connector-namespace",
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
            "namespace_name": "default-connector-namespace",
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
        "deployment_location": {
          "namespace_id": "${connector_namespace_id}"
        },
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
          "href": "/api/connector_mgmt/v1/kafka_connector_clusters/${connector_cluster_id}/deployments/${response.object.id}",
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
            "namespace_name": "default-connector-namespace",
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

    #-----------------------------------------------------------------------------------------------------------------
    # In this part of the Scenario we test out getting connector updates using the admin API
    #-----------------------------------------------------------------------------------------------------------------

    # Now lets verify connector upgrades due to catalog updates
    Given I am logged in as "Ricky Bobby"
    And I GET path "/v1/admin/kafka_connector_clusters"
    And the response code should be 200
    And the ".items[0]" selection from the response should match json:
      """
      {
        "href": "${response.items[0].href}",
        "id": "${response.items[0].id}",
        "kind": "ConnectorCluster",
        "created_at": "${response.items[0].created_at}",
        "name": "${response.items[0].name}",
        "owner": "${response.items[0].owner}",
        "modified_at": "${response.items[0].modified_at}",
        "status": {
          "state": "${response.items[0].status.state}"
        }
      }
      """
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
            "namespace": "default-connector-namespace",
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
    When I GET path "/v1/kafka_connector_clusters/${connector_cluster_id}/deployments"
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

    Then I am logged in as "Ricky Bobby"
    And I GET path "/v1/admin/kafka_connector_clusters/${connector_cluster_id}/upgrades/operator"
    And the response code should be 200
    And the response should match json:
      """
      {
       "items":
          [{
            "connector_id": "${connector_id}",
            "namespace": "default-connector-namespace",
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
    When I GET path "/v1/kafka_connector_clusters/${connector_cluster_id}/deployments"
    Then the response code should be 200
    And the ".items[0].spec.operator_id" selection from the response should match "camel-k-1.0.1"

    # agent removes available operator upgrade status
    Given I am logged in as "Shard"
    And I set the "Authorization" header to "Bearer ${shard_token}"
    When I PUT path "/v1/kafka_connector_clusters/${connector_cluster_id}/deployments/${connector_deployment_id}/status" with json body:
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

    #-----------------------------------------------------------------------------------------------------------------
    # In this part of the Scenario we test out what happens when you have a connector with an invalid connector type
    #-----------------------------------------------------------------------------------------------------------------
    # Validate that there exists 1 connector deployment in the DB...
    Given I run SQL "SELECT count(*) FROM connector_deployments WHERE connector_id='${connector_id}' AND deleted_at IS NULL" gives results:
      | count |
      | 1     |

    Given I am logged in as "Jimmy"
    When I DELETE path "/v1/kafka_connector_clusters/${connector_cluster_id}"
    Then the response code should be 204
    And the response should match ""

    # Connector deployment should be be deleted...
    And I run SQL "SELECT count(*) FROM connector_deployments WHERE connector_id='${connector_id}' AND deleted_at IS NULL" gives results:
      | count |
      | 0     |

    # Connectors that were assigning the cluster get updated to not refer to them.
    When I GET path "/v1/kafka_connectors/${connector_id}"
    Then the response code should be 200
    And the ".status.state" selection from the response should match "assigning"
    And the ".deployment_location" selection from the response should match json:
      """
      {}
      """


  Scenario: Bobby can stop and start and existing connector
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

    #---------------------------------------------------------------------------------------------
    # Create a connector
    # --------------------------------------------------------------------------------------------
    Given I am logged in as "Bobby"
    When I POST path "/v1/kafka_connectors?async=true" with json body:
      """
      {
        "kind": "Connector",
        "name": "example 1",
        "deployment_location": {
          "namespace_id": "${connector_namespace_id}"
        },
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
    Given I wait up to "5" seconds for a GET on path "/v1/kafka_connector_clusters/${connector_cluster_id}/deployments" response ".total" selection to match "1"
    When I GET path "/v1/kafka_connector_clusters/${connector_cluster_id}/deployments"
    Then the ".total" selection from the response should match "1"
    Given I store the ".items[0].id" selection from the response as ${connector_deployment_id}
    When I PUT path "/v1/kafka_connector_clusters/${connector_cluster_id}/deployments/${connector_deployment_id}/status" with json body:
      """
      {
        "phase":"ready",
        "resource_version": 45
      }
      """
    Then the response code should be 204

    Given I am logged in as "Bobby"
    Given I wait up to "5" seconds for a GET on path "/v1/kafka_connectors/${connector_id}" response ".status" selection to match "ready"
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
    Given I wait up to "5" seconds for a GET on path "/v1/kafka_connector_clusters/${connector_cluster_id}/deployments/${connector_deployment_id}" response ".spec.desired_state" selection to match "stopped"
    When I GET path "/v1/kafka_connector_clusters/${connector_cluster_id}/deployments/${connector_deployment_id}"
    Then the ".spec.desired_state" selection from the response should match "stopped"
    When I PUT path "/v1/kafka_connector_clusters/${connector_cluster_id}/deployments/${connector_deployment_id}/status" with json body:
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
    Given I wait up to "5" seconds for a GET on path "/v1/kafka_connector_clusters/${connector_cluster_id}/deployments" response ".total" selection to match "1"
    When I GET path "/v1/kafka_connector_clusters/${connector_cluster_id}/deployments"
    Then the ".total" selection from the response should match "1"
    And the ".items[0].spec.desired_state" selection from the response should match "ready"

    #cleanup
    Given I am logged in as "Bobby"
    When I DELETE path "/v1/kafka_connector_clusters/${connector_cluster_id}"
    Then the response code should be 204
    And the response should match ""
