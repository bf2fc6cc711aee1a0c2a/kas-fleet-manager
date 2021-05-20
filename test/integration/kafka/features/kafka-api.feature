Feature: expose a public api to manage kafka clusters

  Background:
    Given the path prefix is "/api/kafkas_mgmt"
    Given a user named "Greg" in organization "13640203"

  Scenario: Greg can discover the API endpoints
    Given I am logged in as "Greg"
    When I GET path ""
    Then the response code should be 200
    And the response should match json:
      """
      {
        "href": "/api/kafkas_mgmt",
        "id": "kafkas_mgmt",
        "kind": "API",
        "versions": [
          {
            "collections": null,
            "href": "/api/kafkas_mgmt/v1",
            "id": "v1",
            "kind": "APIVersion"
          }
        ]
      }
      """

    When I GET path "/"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "href": "/api/kafkas_mgmt",
        "id": "kafkas_mgmt",
        "kind": "API",
        "versions": [
          {
            "collections": null,
            "href": "/api/kafkas_mgmt/v1",
            "id": "v1",
            "kind": "APIVersion"
          }
        ]
      }
      """

    When I GET path "/v1"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "collections": [
          {
            "href": "/api/kafkas_mgmt/v1/kafkas",
            "id": "kafkas",
            "kind": "KafkaList"
          },
          {
            "href": "/api/kafkas_mgmt/v1/serviceaccounts",
            "id": "serviceaccounts",
            "kind": "ServiceAccountList"
          },
          {
            "href": "/api/kafkas_mgmt/v1/cloud_providers",
            "id": "cloud_providers",
            "kind": "CloudProviderList"
          }
        ],
        "href": "/api/kafkas_mgmt/v1",
        "id": "v1",
        "kind": "APIVersion"
      }
      """

    When I GET path "/v1/"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "collections": [
          {
            "href": "/api/kafkas_mgmt/v1/kafkas",
            "id": "kafkas",
            "kind": "KafkaList"
          },
          {
            "href": "/api/kafkas_mgmt/v1/serviceaccounts",
            "id": "serviceaccounts",
            "kind": "ServiceAccountList"
          },
          {
            "href": "/api/kafkas_mgmt/v1/cloud_providers",
            "id": "cloud_providers",
            "kind": "CloudProviderList"
          }
        ],
        "href": "/api/kafkas_mgmt/v1",
        "id": "v1",
        "kind": "APIVersion"
      }
      """
