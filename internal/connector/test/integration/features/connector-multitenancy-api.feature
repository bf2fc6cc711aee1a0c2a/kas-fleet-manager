Feature: connector namespaces API
  In order to deploy connectors to an addon OSD cluster
  As a regular user, I need to be able to create and manage namespaces in both evaluation and non-evaluation clusters
  As an admin user, I need to be able to create and manage namespaces for other users and clusters

  Background:
    Given the path prefix is "/api/connector_mgmt"

    # User for eval organization id 13640210 configured in internal/connector/test/integration/feature_test.go:27
    Given a user named "Gru" in organization "13640210"

    # eval users used in public API
    Given a user named "Stuart" in organization "13640221"
    Given I store userid for "Stuart" as ${stuart_user_id}
    Given a user named "Kevin" in organization "13640222"
    Given I store userid for "Kevin" as ${kevin_user_id}
    Given a user named "Carl" in organization "13640223"
    Given I store userid for "Carl" as ${carl_user_id}

    # eval users used in admin API
    Given a user named "Dave" in organization "13640224"
    Given I store userid for "Dave" as ${dave_user_id}
    Given a user named "Phil" in organization "13640225"
    Given I store userid for "Phil" as ${phil_user_id}
    Given a user named "Tim" in organization "13640226"
    Given I store userid for "Tim" as ${tim_user_id}

    # users in organization 13640230
    Given a user named "Dusty" in organization "13640230"
    Given I store userid for "Dusty" as ${dusty_user_id}
    Given a user named "Lucky" in organization "13640230"
    Given I store userid for "Lucky" as ${lucky_user_id}
    Given a user named "Ned" in organization "13640230"
    Given I store userid for "Ned" as ${ned_user_id}

    # users in organization 13640231
    Given a user named "El Guapo" in organization "13640231"
    Given I store userid for "El Guapo" as ${guapo_user_id}

  Scenario Outline: Create eval namespace
    Given I am logged in as "Gru"
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

   # There should be no namespaces at first for user
    Given I am logged in as "<user>"
    When I GET path "/v1/kafka_connector_namespaces/"
    Then the response code should be 200
    And the response should match json:
     """
     {
       "items": [],
       "kind": "ConnectorNamespaceList",
       "page": 1,
       "size": 0,
       "total": 0
     }
     """

   # Create an eval namespace
    Given I am logged in as "<user>"
    When I POST path "/v1/kafka_connector_namespaces/eval" with json body:
    """
    {
      "name": "<user>_namespace",
      "annotations": [
        {
          "name": "connector_mgmt.api.openshift.com/profile",
          "value": "default-profile"
        }
      ]
    }
    """
    Then the response code should be 201
    And the response should match json:
    """
    {
      "id": "${response.id}",
      "kind": "ConnectorNamespace",
      "href": "/api/connector_mgmt/v1/kafka_connector_namespaces/${response.id}",
      "name": "<user>_namespace",
      "owner": "${<user_id>}",
      "cluster_id": "${response.cluster_id}",
      "created_at": "${response.created_at}",
      "modified_at": "${response.modified_at}",
      "expiration": "${response.expiration}",
      "annotations": [
        {
          "name": "connector_mgmt.api.openshift.com/profile",
          "value": "default-profile"
        }
      ],
      "status": {
        "state": "disconnected",
        "connectors_deployed": 0
      },
      "tenant": {
        "kind": "user",
        "id": "${<user_id>}"
      }
    }
    """
    And I store the ".id" selection from the response as ${namespace_id}

    # eval namespace MUST be in list of user's namespaces
    Given I GET path "/v1/kafka_connector_namespaces/"
    Then the response code should be 200
     """
     {
       "items": [
         {
           "cluster_id": "${connector_cluster_id}",
           "href": "${response.items[0].href}",
           "id": "${namespace_id}",
           "kind": "ConnectorNamespace",
           "name": "<user>_namespace",
           "owner": "${<user_id>}",
           "created_at": "${response.items[0].created_at}",
           "modified_at": "${response.items[0].modified_at}",
           "expiration": "${response.expiration}",
           "tenant": {
             "kind": "user",
             "id": "${<user_id>}"
           },
           "annotations": [
             {
               "name": "connector_mgmt.api.openshift.com/profile",
               "value": "default-profile"
             }
           ],
           "status": {
             "state": "disconnected",
             "connectors_deployed": 0
           },
         }
       ],
       "kind": "ConnectorNamespaceList",
       "page": 1,
       "size": 1,
       "total": 1
     }
     """

    # Eval namespace should expire and get deleted after 2 seconds as configured in internal/connector/test/integration/feature_test.go:27
    Given I sleep for 3 seconds
    And I GET path "/v1/kafka_connector_namespaces/"
    Then the response code should be 200
    And the ".total" selection from the response should match "0"

   # cleanup eval cluster
    Given I am logged in as "Gru"
    When I DELETE path "/v1/kafka_connector_clusters/${connector_cluster_id}"
    Then the response code should be 204
    And the response should match ""

    Examples:
      | user   | user_id        |
      | Stuart | stuart_user_id |
      | Kevin  | kevin_user_id  |
      | Carl    | carl_user_id    |

  Scenario: Create namespaces in cluster for organization 13640230
    Given I am logged in as "Dusty"
    When I POST path "/v1/kafka_connector_clusters" with json body:
     """
     {
      "name": "Dusty's Cluster"
     }
     """
    Then the response code should be 202
    And the ".status.state" selection from the response should match "disconnected"

    Given I store the ".id" selection from the response as ${connector_cluster_id}
    When I GET path "/v1/kafka_connector_clusters/${connector_cluster_id}/addon_parameters"
    Then the response code should be 200
    And get and store access token using the addon parameter response as ${shard_token} and clientID as ${clientID}
    And I remember keycloak client for cleanup with clientID: ${clientID}

   # There should be default namespace at first for organization 13640230
    Given I am logged in as "Lucky"
    When I GET path "/v1/kafka_connector_namespaces"
    Then the response code should be 200
    And the response should match json:
     """
     {
       "items": [
         {
           "cluster_id": "${connector_cluster_id}",
           "created_at": "${response.items[0].created_at}",
           "href": "${response.items[0].href}",
           "id": "${response.items[0].id}",
           "kind": "ConnectorNamespace",
           "modified_at": "${response.items[0].modified_at}",
           "name": "default-connector-namespace",
           "owner": "${dusty_user_id}",
           "tenant": {
             "kind": "organisation",
             "id": "13640230"
           },
           "annotations": [
             {
               "name": "connector_mgmt.api.openshift.com/profile",
               "value": "default-profile"
             }
           ],
           "status": {
             "state": "disconnected",
             "connectors_deployed": 0
           }
         }
       ],
       "kind": "ConnectorNamespaceList",
       "page": 1,
       "size": 1,
       "total": 1
     }
     """

   # Create a namespace
    Given I am logged in as "Lucky"
    When I POST path "/v1/kafka_connector_namespaces/" with json body:
    """
    {
      "name": "shared_namespace",
      "cluster_id": "${connector_cluster_id}",
      "kind": "organisation",
      "annotations": [
        {
          "name": "connector_mgmt.api.openshift.com/profile",
          "value": "default-profile"
        }
      ]
    }
    """
    Then the response code should be 201
    And the response should match json:
    """
    {
      "id": "${response.id}",
      "kind": "ConnectorNamespace",
      "href": "/api/connector_mgmt/v1/kafka_connector_namespaces/${response.id}",
      "name": "shared_namespace",
      "owner": "${lucky_user_id}",
      "cluster_id": "${connector_cluster_id}",
      "created_at": "${response.created_at}",
      "modified_at": "${response.modified_at}",
      "annotations": [
        {
          "name": "connector_mgmt.api.openshift.com/profile",
          "value": "default-profile"
        }
      ],
      "tenant": {
        "kind": "organisation",
        "id": "13640230"
      },
      "status": {
        "state": "disconnected",
        "connectors_deployed": 0
      }
    }
    """
    And I store the ".id" selection from the response as ${namespace_id}

   # All organization members MUST be able to see the org tenant namespace
    Given I am logged in as "Ned"
    When I GET path "/v1/kafka_connector_namespaces/?orderBy=name"
    Then the response code should be 200
    And the response should match json:
     """
     {
       "items": [
         {
           "cluster_id": "${connector_cluster_id}",
           "created_at": "${response.items[0].created_at}",
           "href": "${response.items[0].href}",
           "id": "${response.items[0].id}",
           "kind": "ConnectorNamespace",
           "modified_at": "${response.items[0].modified_at}",
           "name": "default-connector-namespace",
           "owner": "${dusty_user_id}",
           "tenant": {
             "kind": "organisation",
             "id": "13640230"
           },
           "annotations": [
             {
               "name": "connector_mgmt.api.openshift.com/profile",
               "value": "default-profile"
             }
           ],
           "status": {
             "state": "disconnected",
             "connectors_deployed": 0
           }
         },
         {
           "id": "${namespace_id}",
           "kind": "ConnectorNamespace",
           "href": "/api/connector_mgmt/v1/kafka_connector_namespaces/${namespace_id}",
           "name": "shared_namespace",
           "owner": "${lucky_user_id}",
           "cluster_id": "${connector_cluster_id}",
           "created_at": "${response.items[1].created_at}",
           "modified_at": "${response.items[1].modified_at}",
           "tenant": {
             "kind": "organisation",
             "id": "13640230"
           },
           "annotations": [
             {
               "name": "connector_mgmt.api.openshift.com/profile",
               "value": "default-profile"
             }
           ],
           "status": {
             "state": "disconnected",
             "connectors_deployed": 0
           }
         }
       ],
       "kind": "ConnectorNamespaceList",
       "page": 1,
       "size": 2,
       "total": 2
     }
     """

   # Delete namespace
    Given I am logged in as "Lucky"
    When I DELETE path "/v1/kafka_connector_namespaces/${namespace_id}"
    Then the response code should be 204
    And I GET path "/v1/kafka_connector_namespaces"
    And the response code should be 200
    And the response should match json:
     """
     {
       "items": [
         {
           "cluster_id": "${connector_cluster_id}",
           "created_at": "${response.items[0].created_at}",
           "href": "${response.items[0].href}",
           "id": "${response.items[0].id}",
           "kind": "ConnectorNamespace",
           "modified_at": "${response.items[0].modified_at}",
           "name": "default-connector-namespace",
           "owner": "${dusty_user_id}",
           "tenant": {
             "kind": "organisation",
             "id": "13640230"
           },
           "annotations": [
             {
               "name": "connector_mgmt.api.openshift.com/profile",
               "value": "default-profile"
             }
           ],
           "status": {
             "state": "disconnected",
             "connectors_deployed": 0
           }
         }
       ],
       "kind": "ConnectorNamespaceList",
       "page": 1,
       "size": 1,
       "total": 1
     }
     """

    # cleanup cluster
    Given I am logged in as "Dusty"
    When I DELETE path "/v1/kafka_connector_clusters/${connector_cluster_id}"
    Then the response code should be 204
    And the response should match ""

  Scenario Outline: Use Admin API to create namespaces for end users
    Given I am logged in as "<user>"

   #-----------------------------------------------------------------------------------
   # Create a target cluster, and get the shard access token.
   # -----------------------------------------------------------------------------------
    When I POST path "/v1/kafka_connector_clusters" with json body:
     """
     {
      "name": "User <user>'s Cluster"
     }
     """
    Then the response code should be 202
    And the ".status.state" selection from the response should match "disconnected"

    Given I store the ".id" selection from the response as ${connector_cluster_id}
    When I GET path "/v1/kafka_connector_clusters/${connector_cluster_id}/addon_parameters"
    Then the response code should be 200
    And get and store access token using the addon parameter response as ${shard_token} and clientID as ${clientID}
    And I remember keycloak client for cleanup with clientID: ${clientID}

   #-----------------------------------------------------------------------------------------------------------------
   # In this part of the Scenario we create connector namespaces
   #-----------------------------------------------------------------------------------------------------------------

   # There should be default namespace first
    Given I am logged in as "Ricky Bobby"
    When I GET path "/v1/admin/kafka_connector_namespaces/?search=cluster_id=${connector_cluster_id}"
    Then the response code should be 200
    And the response should match json:
     """
     {
       "items": [
         {
           "cluster_id": "${connector_cluster_id}",
           "created_at": "${response.items[0].created_at}",
           "href": "${response.items[0].href}",
           "id": "${response.items[0].id}",
           "kind": "ConnectorNamespace",
           "modified_at": "${response.items[0].modified_at}",
           "name": "default-connector-namespace",
           "owner": "${<user_id>}",
           "tenant": {
             "kind": "organisation",
             "id": "${response.items[0].tenant.id}"
           },
           "annotations": [
             {
               "name": "connector_mgmt.api.openshift.com/profile",
               "value": "default-profile"
             }
           ],
           "status": {
             "state": "disconnected",
             "connectors_deployed": 0
           }
         }
       ],
       "kind": "ConnectorNamespaceList",
       "page": 1,
       "size": 1,
       "total": 1
     }
     """

   # Create a namespace
    Given I am logged in as "Ricky Bobby"
    When I POST path "/v1/admin/kafka_connector_namespaces/" with json body:
    """
    {
      "name": "<user>_namespace",
      "cluster_id": "${connector_cluster_id}",
      "annotations": [
        {
          "name": "connector_mgmt.api.openshift.com/profile",
          "value": "default-profile"
        }
      ],
      "tenant": {
        "kind": "user",
        "id": "${<user_id>}"
      }
    }
    """
    Then the response code should be 201
    And the response should match json:
    """
    {
      "id": "${response.id}",
      "kind": "ConnectorNamespace",
      "href": "/api/connector_mgmt/v1/kafka_connector_namespaces/${response.id}",
      "name": "<user>_namespace",
      "owner": "${<user_id>}",
      "cluster_id": "${connector_cluster_id}",
      "created_at": "${response.created_at}",
      "modified_at": "${response.modified_at}",
      "tenant": {
        "kind": "user",
        "id": "${<user_id>}"
      },
      "annotations": [
        {
          "name": "connector_mgmt.api.openshift.com/profile",
          "value": "default-profile"
        }
      ],
      "status": {
        "state": "disconnected",
        "connectors_deployed": 0
      }
    }
    """
    And I store the ".id" selection from the response as ${namespace_id}

   # Delete namespace
    Given I am logged in as "Ricky Bobby"
    When I DELETE path "/v1/admin/kafka_connector_namespaces/${namespace_id}"
    Then the response code should be 204
    And I GET path "/v1/admin/kafka_connector_clusters/${connector_cluster_id}/namespaces"
    And the response code should be 200
    And the response should match json:
     """
     {
       "items": [
         {
           "cluster_id": "${connector_cluster_id}",
           "created_at": "${response.items[0].created_at}",
           "href": "${response.items[0].href}",
           "id": "${response.items[0].id}",
           "kind": "ConnectorNamespace",
           "modified_at": "${response.items[0].modified_at}",
           "name": "default-connector-namespace",
           "owner": "${<user_id>}",
           "tenant": {
             "kind": "organisation",
             "id": "${response.items[0].tenant.id}"
           },
           "annotations": [
             {
               "name": "connector_mgmt.api.openshift.com/profile",
               "value": "default-profile"
             }
           ],
           "status": {
             "state": "disconnected",
             "connectors_deployed": 0
           }
         }
       ],
       "kind": "ConnectorNamespaceList",
       "page": 1,
       "size": 1,
       "total": 1
     }
     """

   #cleanup
    Given I am logged in as "<user>"
    When I DELETE path "/v1/kafka_connector_clusters/${connector_cluster_id}"
    Then the response code should be 204
    And the response should match ""

    Examples:
      | user | user_id      |
      | Dave | dave_user_id |
      | Phil | phil_user_id |
      | Tim  | tim_user_id  |

  Scenario: Use Admin API to create namespace for organization 13640231.
    Given I am logged in as "El Guapo"

   #-----------------------------------------------------------------------------------
   # Create a target cluster, and get the shard access token.
   # -----------------------------------------------------------------------------------
    When I POST path "/v1/kafka_connector_clusters" with json body:
     """
     {
      "name": "El Guapo's Cluster"
     }
     """
    Then the response code should be 202
    And the ".status.state" selection from the response should match "disconnected"

    Given I store the ".id" selection from the response as ${connector_cluster_id}
    When I GET path "/v1/kafka_connector_clusters/${connector_cluster_id}/addon_parameters"
    Then the response code should be 200
    And get and store access token using the addon parameter response as ${shard_token} and clientID as ${clientID}
    And I remember keycloak client for cleanup with clientID: ${clientID}

   #-----------------------------------------------------------------------------------------------------------------
   # In this part of the Scenario we create connector namespace
   #-----------------------------------------------------------------------------------------------------------------

   # There should be default namespace in cluster ${connector_cluster_id}
    Given I am logged in as "Ricky Bobby"
    When I GET path "/v1/admin/kafka_connector_namespaces/?search=cluster_id=${connector_cluster_id}"
    Then the response code should be 200
    And the response should match json:
     """
     {
       "items": [
         {
           "cluster_id": "${connector_cluster_id}",
           "created_at": "${response.items[0].created_at}",
           "href": "${response.items[0].href}",
           "id": "${response.items[0].id}",
           "kind": "ConnectorNamespace",
           "modified_at": "${response.items[0].modified_at}",
           "name": "default-connector-namespace",
           "owner": "${guapo_user_id}",
           "tenant": {
             "kind": "organisation",
             "id": "13640231"
           },
           "annotations": [
             {
               "name": "connector_mgmt.api.openshift.com/profile",
               "value": "default-profile"
             }
           ],
           "status": {
             "state": "disconnected",
             "connectors_deployed": 0
           }
         }
       ],
       "kind": "ConnectorNamespaceList",
       "page": 1,
       "size": 1,
       "total": 1
     }
     """

   # Create a namespace
    Given I am logged in as "Ricky Bobby"
    When I POST path "/v1/admin/kafka_connector_namespaces/" with json body:
    """
    {
      "name": "amigos_namespace",
      "cluster_id": "${connector_cluster_id}",
      "tenant": {
        "kind": "organisation",
        "id": "13640231"
      },
      "annotations": [
        {
          "name": "connector_mgmt.api.openshift.com/profile",
          "value": "default-profile"
        }
      ],
      "status": {
        "state": "disconnected",
        "connectors_deployed": 0
      }
    }
    """
    Then the response code should be 201
    And the response should match json:
    """
    {
      "id": "${response.id}",
      "kind": "ConnectorNamespace",
      "href": "/api/connector_mgmt/v1/kafka_connector_namespaces/${response.id}",
      "name": "amigos_namespace",
      "owner": "Ricky Bobby",
      "cluster_id": "${connector_cluster_id}",
      "created_at": "${response.created_at}",
      "modified_at": "${response.modified_at}",
      "tenant": {
        "kind": "organisation",
        "id": "13640231"
      },
      "annotations": [
        {
          "name": "connector_mgmt.api.openshift.com/profile",
          "value": "default-profile"
        }
      ],
      "status": {
        "state": "disconnected",
        "connectors_deployed": 0
      }
    }
    """
    And I store the ".id" selection from the response as ${namespace_id}

   # Delete namespace
    Given I am logged in as "Ricky Bobby"
    When I DELETE path "/v1/admin/kafka_connector_namespaces/${namespace_id}"
    Then the response code should be 204
    And I GET path "/v1/admin/kafka_connector_namespaces?search=cluster_id=${connector_cluster_id}"
    And the response code should be 200
    And the response should match json:
     """
     {
       "items": [
         {
           "cluster_id": "${connector_cluster_id}",
           "created_at": "${response.items[0].created_at}",
           "href": "${response.items[0].href}",
           "id": "${response.items[0].id}",
           "kind": "ConnectorNamespace",
           "modified_at": "${response.items[0].modified_at}",
           "name": "default-connector-namespace",
           "owner": "${guapo_user_id}",
           "tenant": {
             "kind": "organisation",
             "id": "13640231"
           },
           "annotations": [
             {
               "name": "connector_mgmt.api.openshift.com/profile",
               "value": "default-profile"
             }
           ],
           "status": {
             "state": "disconnected",
             "connectors_deployed": 0
           }
         }
       ],
       "kind": "ConnectorNamespaceList",
       "page": 1,
       "size": 1,
       "total": 1
     }
     """

   #cleanup
    Given I am logged in as "El Guapo"
    When I DELETE path "/v1/kafka_connector_clusters/${connector_cluster_id}"
    Then the response code should be 204
    And the response should match ""
