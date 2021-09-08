Feature: expose a public api to manage dinosaur clusters

  Background:
    Given the path prefix is "/api/dinosaurs_mgmt"
    Given a user named "Greg" in organization "13640203"

  Scenario: Greg can discover the API endpoints
    Given I am logged in as "Greg"
    When I GET path ""
    Then the response code should be 200
    And the response should match json:
      """
      {
        "href": "/api/dinosaurs_mgmt",
        "id": "dinosaurs_mgmt",
        "kind": "API",
        "versions": [
          {
            "collections": null,
            "href": "/api/dinosaurs_mgmt/v1",
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
        "href": "/api/dinosaurs_mgmt",
        "id": "dinosaurs_mgmt",
        "kind": "API",
        "versions": [
          {
            "collections": null,
            "href": "/api/dinosaurs_mgmt/v1",
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
            "href": "/api/dinosaurs_mgmt/v1/dinosaurs",
            "id": "dinosaurs",
            "kind": "DinosaurList"
          },
          {
            "href": "/api/dinosaurs_mgmt/v1/service_accounts",
            "id": "service_accounts",
            "kind": "ServiceAccountList"
          },
          {
            "href": "/api/dinosaurs_mgmt/v1/cloud_providers",
            "id": "cloud_providers",
            "kind": "CloudProviderList"
          }
        ],
        "href": "/api/dinosaurs_mgmt/v1",
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
            "href": "/api/dinosaurs_mgmt/v1/dinosaurs",
            "id": "dinosaurs",
            "kind": "DinosaurList"
          },
          {
            "href": "/api/dinosaurs_mgmt/v1/service_accounts",
            "id": "service_accounts",
            "kind": "ServiceAccountList"
          },
          {
            "href": "/api/dinosaurs_mgmt/v1/cloud_providers",
            "id": "cloud_providers",
            "kind": "CloudProviderList"
          }
        ],
        "href": "/api/dinosaurs_mgmt/v1",
        "id": "v1",
        "kind": "APIVersion"
      }
      """
