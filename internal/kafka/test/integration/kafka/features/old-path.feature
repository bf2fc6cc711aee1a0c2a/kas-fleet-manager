# The tests here are to ensure some of the paths are backward compatible. These tests should be removed once we decide the backward compatibility is no longer needed.
Feature: make sure the paths are backward compatible

  Background:
    Given the path prefix is "/api/kafkas_mgmt/v1"
    Given a user named "Greg" in organization "13640203"

  Scenario: Greg can list service accounts via the old "serviceaccont" API endpoint
    Given I am logged in as "Greg"
    When I GET path "/serviceaccounts"
    Then the response code should be 200

    When I POST path "/serviceaccounts/invalidid/reset-credentials" with json body:
      """
      {}
      """
    Then the response code should be 400