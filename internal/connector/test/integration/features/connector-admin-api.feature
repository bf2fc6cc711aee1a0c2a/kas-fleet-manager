Feature: connector admin api
  In order to support the service
  As an Admin API user
  I need to be able to manage the service

  Background:
    Given the path prefix is "/api/connector_mgmt"

    # org admin user used in admin API
    Given an admin user named "Ricky Bobby" with roles "connector-fleet-manager-admin-full"

    # regular user without roles
    Given a user named "Evil Bob"

  Scenario: Ricky lists connector types
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
    And the ".items[] | select(.id == "aws-sqs-source-v1alpha1") | .channels.stable.shard_metadata.connector_image" selection from the response should match "quay.io/mock-image:77c0b8763729a9167ddfa19266d83a3512b7aa8124ca53e381d5d05f7d197a24"

    When I GET path "/v1/admin/kafka_connector_types?search=name=aws-sqs-source"
    Then the response code should be 200
    And the ".items[] | select(.id == "aws-sqs-source-v1alpha1") | has("schema")" selection from the response should match "true"
    And the ".items[] | select(.id == "aws-sqs-source-v1alpha1") | .kind" selection from the response should match "ConnectorTypeAdminView"
    And the ".items[] | select(.id == "aws-sqs-source-v1alpha1") | .href" selection from the response should match "/api/connector_mgmt/v1/admin/kafka_connector_types/aws-sqs-source-v1alpha1"
    And the ".items[] | select(.id == "aws-sqs-source-v1alpha1") | .channels.stable.shard_metadata.connector_image" selection from the response should match "quay.io/mock-image:77c0b8763729a9167ddfa19266d83a3512b7aa8124ca53e381d5d05f7d197a24"

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
    And the ".channels.stable.shard_metadata.connector_image" selection from the response should match "quay.io/mock-image:77c0b8763729a9167ddfa19266d83a3512b7aa8124ca53e381d5d05f7d197a24"

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

  Scenario: Bob can't lists connector types
    Given I am logged in as "Evil Bob"
    When I GET path "/v1/admin/kafka_connector_types"
    Then the response code should be 404

