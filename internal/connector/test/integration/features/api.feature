@api
Feature: create a connector
  In order to use connectors api
  As an API user
  I need to be able to manage connectors

  Background:
    Given the path prefix is "/api/connector_mgmt"
    Given a user named "Gary" in organization "13640203"

  Scenario: Gary can discover the API endpoints
    Given I am logged in as "Gary"
    When I GET path ""
    Then the response code should be 200
    And the response should match json:
      """
      {
        "href": "/api/connector_mgmt",
        "id": "connector_mgmt",
        "kind": "API",
        "versions": [
          {
            "collections": null,
            "href": "/api/connector_mgmt/v1",
            "id": "v1",
            "kind": "APIVersion",
            "server_version": ""
          },
          {
            "collections": null,
            "href": "/api/connector_mgmt/v2alpha1",
            "id": "v2alpha1",
            "kind": "APIVersion",
            "server_version": ""
           }
        ]
      }
      """

    When I GET path "/"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "href": "/api/connector_mgmt",
        "id": "connector_mgmt",
        "kind": "API",
        "versions": [
          {
            "kind": "APIVersion",
            "id": "v1",
            "server_version": "",
            "href": "/api/connector_mgmt/v1",
            "collections": null
          },
          {
            "collections": null,
            "href": "/api/connector_mgmt/v2alpha1",
            "id": "v2alpha1",
            "kind": "APIVersion",
            "server_version": ""
           }
        ]
      }
      """

    When I GET path "/v1"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "kind": "APIVersion",
        "id": "v1",
        "server_version": "",
        "href": "/api/connector_mgmt/v1",
        "collections": [
          {
            "href": "/api/connector_mgmt/v1/kafka_connector_types",
            "id": "kafka_connector_types",
            "kind": "ConnectorTypeList"
          },
          {
            "href": "/api/connector_mgmt/v1/kafka_connectors",
            "id": "kafka_connectors",
            "kind": "ConnectorList"
          },
          {
            "href": "/api/connector_mgmt/v1/kafka_connector_clusters",
            "id": "kafka_connector_clusters",
            "kind": "ConnectorClusterList"
          },
          {
            "href": "/api/connector_mgmt/v1/kafka_connector_namespaces",
            "id": "kafka_connector_namespaces",
            "kind": "ConnectorNamespaceList"
          }
        ]
      }
      """

    When I GET path "/v1/"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "kind": "APIVersion",
        "id": "v1",
        "server_version": "",
        "href": "/api/connector_mgmt/v1",
        "collections": [
          {
            "href": "/api/connector_mgmt/v1/kafka_connector_types",
            "id": "kafka_connector_types",
            "kind": "ConnectorTypeList"
          },
          {
            "href": "/api/connector_mgmt/v1/kafka_connectors",
            "id": "kafka_connectors",
            "kind": "ConnectorList"
          },
          {
            "href": "/api/connector_mgmt/v1/kafka_connector_clusters",
            "id": "kafka_connector_clusters",
            "kind": "ConnectorClusterList"
          },
          {
            "href": "/api/connector_mgmt/v1/kafka_connector_namespaces",
            "id": "kafka_connector_namespaces",
            "kind": "ConnectorNamespaceList"
          }
        ]
      }
      """

    When I GET path "/v2alpha1"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "kind": "APIVersion",
        "id": "v2alpha1",
        "server_version": "",
        "href": "/api/connector_mgmt/v2alpha1",
        "collections": [
          {
            "href": "/api/connector_mgmt/v2alpha1/processors",
            "id": "processors",
            "kind": "ProcessorList"
          },
          {
            "href": "/api/connector_mgmt/v2alpha1/processorTypes",
            "id": "processorTypes",
            "kind": "ProcessorTypesList"
          }
        ]
      }
      """

    When I GET path "/v2alpha1/"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "kind": "APIVersion",
        "id": "v2alpha1",
        "server_version": "",
        "href": "/api/connector_mgmt/v2alpha1",
        "collections": [
          {
            "href": "/api/connector_mgmt/v2alpha1/processors",
            "id": "processors",
            "kind": "ProcessorList"
          },
          {
            "href": "/api/connector_mgmt/v2alpha1/processorTypes",
            "id": "processorTypes",
            "kind": "ProcessorTypesList"
          }
        ]
      }
      """

  Scenario: Gary can inspect errors codes
    Given I am logged in as "Gary"
    When I GET path "/v1/errors"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "items": [
          {
            "code": "CONNECTOR-MGMT-4",
            "href": "/api/connector_mgmt/v1/errors/4",
            "id": "4",
            "kind": "Error",
            "reason": "Forbidden to perform this action"
          },
          {
            "code": "CONNECTOR-MGMT-5",
            "href": "/api/connector_mgmt/v1/errors/5",
            "id": "5",
            "kind": "Error",
            "reason": "Forbidden to create more instances than the maximum allowed"
          },
          {
            "code": "CONNECTOR-MGMT-6",
            "href": "/api/connector_mgmt/v1/errors/6",
            "id": "6",
            "kind": "Error",
            "reason": "An entity with the specified unique values already exists"
          },
          {
            "code": "CONNECTOR-MGMT-7",
            "href": "/api/connector_mgmt/v1/errors/7",
            "id": "7",
            "kind": "Error",
            "reason": "Resource not found"
          },
          {
            "code": "CONNECTOR-MGMT-8",
            "href": "/api/connector_mgmt/v1/errors/8",
            "id": "8",
            "kind": "Error",
            "reason": "General validation failure"
          },
          {
            "code": "CONNECTOR-MGMT-9",
            "href": "/api/connector_mgmt/v1/errors/9",
            "id": "9",
            "kind": "Error",
            "reason": "Unspecified error"
          },
          {
            "code": "CONNECTOR-MGMT-10",
            "href": "/api/connector_mgmt/v1/errors/10",
            "id": "10",
            "kind": "Error",
            "reason": "HTTP Method not implemented for this endpoint"
          },
          {
            "code": "CONNECTOR-MGMT-11",
            "href": "/api/connector_mgmt/v1/errors/11",
            "id": "11",
            "kind": "Error",
            "reason": "Account is unauthorized to perform this action"
          },
          {
            "code": "CONNECTOR-MGMT-12",
            "href": "/api/connector_mgmt/v1/errors/12",
            "id": "12",
            "kind": "Error",
            "reason": "Required terms have not been accepted"
          },
          {
            "code": "CONNECTOR-MGMT-15",
            "href": "/api/connector_mgmt/v1/errors/15",
            "id": "15",
            "kind": "Error",
            "reason": "Account authentication could not be verified"
          },
          {
            "code": "CONNECTOR-MGMT-17",
            "href": "/api/connector_mgmt/v1/errors/17",
            "id": "17",
            "kind": "Error",
            "reason": "Unable to read request body"
          },
          {
            "code": "CONNECTOR-MGMT-18",
            "href": "/api/connector_mgmt/v1/errors/18",
            "id": "18",
            "kind": "Error",
            "reason": "Unable to perform this action, as the service is currently under maintenance"
          },
          {
            "code": "CONNECTOR-MGMT-21",
            "href": "/api/connector_mgmt/v1/errors/21",
            "id": "21",
            "kind": "Error",
            "reason": "Bad request"
          },
          {
            "code": "CONNECTOR-MGMT-23",
            "href": "/api/connector_mgmt/v1/errors/23",
            "id": "23",
            "kind": "Error",
            "reason": "Failed to parse search query"
          },
          {
            "code": "CONNECTOR-MGMT-24",
            "href": "/api/connector_mgmt/v1/errors/24",
            "id": "24",
            "kind": "Error",
            "reason": "The maximum number of allowed kafka instances has been reached"
          },
          {
            "code": "CONNECTOR-MGMT-25",
            "href": "/api/connector_mgmt/v1/errors/25",
            "id": "25",
            "kind": "Error",
            "reason": "Resource gone"
          },
          {
            "code": "CONNECTOR-MGMT-30",
            "href": "/api/connector_mgmt/v1/errors/30",
            "id": "30",
            "kind": "Error",
            "reason": "Provider not supported"
          },
          {
            "code": "CONNECTOR-MGMT-31",
            "href": "/api/connector_mgmt/v1/errors/31",
            "id": "31",
            "kind": "Error",
            "reason": "Region not supported"
          },
          {
            "code": "CONNECTOR-MGMT-32",
            "href": "/api/connector_mgmt/v1/errors/32",
            "id": "32",
            "kind": "Error",
            "reason": "Kafka cluster name is invalid"
          },
          {
            "code": "CONNECTOR-MGMT-33",
            "href": "/api/connector_mgmt/v1/errors/33",
            "id": "33",
            "kind": "Error",
            "reason": "Minimum field length not reached"
          },
          {
            "code": "CONNECTOR-MGMT-34",
            "href": "/api/connector_mgmt/v1/errors/34",
            "id": "34",
            "kind": "Error",
            "reason": "Maximum field length has been depassed"
          },
          {
            "code": "CONNECTOR-MGMT-36",
            "href": "/api/connector_mgmt/v1/errors/36",
            "id": "36",
            "kind": "Error",
            "reason": "Kafka cluster name is already used"
          },
          {
            "code": "CONNECTOR-MGMT-37",
            "href": "/api/connector_mgmt/v1/errors/37",
            "id": "37",
            "kind": "Error",
            "reason": "Field validation failed"
          },
          {
            "code": "CONNECTOR-MGMT-38",
            "href": "/api/connector_mgmt/v1/errors/38",
            "id": "38",
            "kind": "Error",
            "reason": "Service account name is invalid"
          },
          {
            "code": "CONNECTOR-MGMT-39",
            "href": "/api/connector_mgmt/v1/errors/39",
            "id": "39",
            "kind": "Error",
            "reason": "Service account desc is invalid"
          },
          {
            "code": "CONNECTOR-MGMT-40",
            "href": "/api/connector_mgmt/v1/errors/40",
            "id": "40",
            "kind": "Error",
            "reason": "Service account id is invalid"
          },
          {
            "code": "CONNECTOR-MGMT-41",
            "href": "/api/connector_mgmt/v1/errors/41",
            "id": "41",
            "kind": "Error",
            "reason": "Instance Type not supported"
          },
          {
            "code": "CONNECTOR-MGMT-42",
            "href": "/api/connector_mgmt/v1/errors/42",
            "id": "42",
            "kind": "Error",
            "reason": "Instance plan not supported"
          },
          {
            "code": "CONNECTOR-MGMT-43",
            "href": "/api/connector_mgmt/v1/errors/43",
            "id": "43",
            "kind": "Error",
            "reason": "Billing account id missing or invalid"
          },
          {
            "code": "CONNECTOR-MGMT-44",
            "href": "/api/connector_mgmt/v1/errors/44",
            "id": "44",
            "kind": "Error",
            "reason": "Enterprise cluster ID is already used"
          },
          {
            "code": "CONNECTOR-MGMT-45",
            "href": "/api/connector_mgmt/v1/errors/45",
            "id": "45",
            "kind": "Error",
            "reason": "Enterprise cluster ID is invalid"
          },
          {
            "code": "CONNECTOR-MGMT-46",
            "href": "/api/connector_mgmt/v1/errors/46",
            "id": "46",
            "kind": "Error",
            "reason": "Enterprise external cluster ID is invalid"
          },
          {
            "code": "CONNECTOR-MGMT-47",
            "href": "/api/connector_mgmt/v1/errors/47",
            "id": "47",
            "kind": "Error",
            "reason": "Dns name is invalid"
          },
          {
            "code": "CONNECTOR-MGMT-103",
            "href": "/api/connector_mgmt/v1/errors/103",
            "id": "103",
            "kind": "Error",
            "reason": "Synchronous action is not supported, use async=true parameter"
          },
          {
            "code": "CONNECTOR-MGMT-106",
            "href": "/api/connector_mgmt/v1/errors/106",
            "id": "106",
            "kind": "Error",
            "reason": "Failed to create kafka client in the mas sso"
          },
          {
            "code": "CONNECTOR-MGMT-107",
            "href": "/api/connector_mgmt/v1/errors/107",
            "id": "107",
            "kind": "Error",
            "reason": "Failed to get kafka client secret from the mas sso"
          },
          {
            "code": "CONNECTOR-MGMT-108",
            "href": "/api/connector_mgmt/v1/errors/108",
            "id": "108",
            "kind": "Error",
            "reason": "Failed to get kafka client from the mas sso"
          },
          {
            "code": "CONNECTOR-MGMT-109",
            "href": "/api/connector_mgmt/v1/errors/109",
            "id": "109",
            "kind": "Error",
            "reason": "Failed to delete kafka client from the mas sso"
          },
          {
            "code": "CONNECTOR-MGMT-110",
            "href": "/api/connector_mgmt/v1/errors/110",
            "id": "110",
            "kind": "Error",
            "reason": "Failed to create service account"
          },
          {
            "code": "CONNECTOR-MGMT-111",
            "href": "/api/connector_mgmt/v1/errors/111",
            "id": "111",
            "kind": "Error",
            "reason": "Failed to get service account"
          },
          {
            "code": "CONNECTOR-MGMT-112",
            "href": "/api/connector_mgmt/v1/errors/112",
            "id": "112",
            "kind": "Error",
            "reason": "Failed to delete service account"
          },
          {
            "code": "CONNECTOR-MGMT-113",
            "href": "/api/connector_mgmt/v1/errors/113",
            "id": "113",
            "kind": "Error",
            "reason": "Failed to find service account"
          },
           {
             "code": "CONNECTOR-MGMT-115",
              "href": "/api/connector_mgmt/v1/errors/115",
              "id": "115",
              "kind": "Error",
              "reason": "Max limit for the service account creation has reached"
           },
          {
            "code": "CONNECTOR-MGMT-120",
            "href": "/api/connector_mgmt/v1/errors/120",
            "id": "120",
            "kind": "Error",
            "reason": "Insufficient quota"
          },
          {
            "code": "CONNECTOR-MGMT-121",
            "href": "/api/connector_mgmt/v1/errors/121",
            "id": "121",
            "kind": "Error",
            "reason": "Failed to check quota"
          },
          {
            "code": "CONNECTOR-MGMT-429",
            "href": "/api/connector_mgmt/v1/errors/429",
            "id": "429",
            "kind": "Error",
            "reason": "Too many requests"
          },
          {
            "code": "CONNECTOR-MGMT-1000",
            "href": "/api/connector_mgmt/v1/errors/1000",
            "id": "1000",
            "kind": "Error",
            "reason": "An unexpected error happened, please check the log of the service for details"
          }
        ],
        "kind": "ErrorList",
        "page": 1,
        "size": 47,
        "total": 47
      }
      """
