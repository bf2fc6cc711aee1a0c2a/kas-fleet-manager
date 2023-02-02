@connectors
Feature: create a connector
  In order to use connectors api
  As an API user
  I need to be able to manage connectors

  Background:
    Given the path prefix is "/api/connector_mgmt"
    # Gary and Coworker Sally will end up in the same org
    Given a user named "Gary" in organization "13640203"
    Given a user named "Coworker Sally" in organization "13640203"
    Given a user named "Evil Bob"
    Given a user named "Jim"
    Given a user named "Tommy"
    Given I store orgid for "Jim" as ${jim_org_id}

  Scenario: check that the connector types have been reconciled
    Given I am logged in as "Gary"
    When I GET path "/v1/kafka_connector_types"
    Then the response code should be 200
    And the ".total" selection from the response should match "3"
    And I run SQL "SELECT count(*) FROM connector_types WHERE checksum IS NULL AND deleted_at IS NULL" gives results:
      | count |
      | 0     |

  Scenario: Gary tries to sql inject connector type listing
    Given I am logged in as "Gary"
    When I GET path "/v1/kafka_connector_types?orderBy=CAST(CHR(32)||(SELECT+version())+AS+NUMERIC)"
    Then the response code should be 400
    And the response should match json:
      """
      {
        "id":"17",
        "kind":"Error",
        "href":"/api/connector_mgmt/v1/errors/17",
        "code":"CONNECTOR-MGMT-17",
        "reason":"Unable to list connector type requests: invalid order by clause 'CAST(CHR(32)||(SELECT version()) AS NUMERIC)'",
        "operation_id": "${response.operation_id}"
      }
      """

    When I GET path "/v1/kafka_connector_types?search=CAST(CHR(32)||(SELECT+version())+AS+NUMERIC)"
    Then the response code should be 400
      """
      {
        "id":"23",
        "kind":"Error",
        "href":"/api/connector_mgmt/v1/errors/23",
        "code":"CONNECTOR-MGMT-23",
        "reason":"Unable to list connector type requests: [1] error parsing the filter: invalid column name: 'CAST'",
        "operation_id": "${response.operation_id}"
      }
      """

  Scenario: Gary lists connector type labels
    Given I am logged in as "Gary"
    When I GET path "/v1/kafka_connector_types/labels"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "items": [
          {
            "count": 1,
            "label": "category-amazon"
          },
          {
            "count": 1,
            "label": "category-featured"
          },
          {
            "count": 1,
            "label": "category-streaming-and-messaging"
          },
          {
            "count": 1,
            "label": "sink"
          },
          {
            "count": 1,
            "label": "source"
          }
        ]
      }
      """

    # check search param
    When I GET path "/v1/kafka_connector_types/labels?search=name+ilike+%25log%25"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "items": [
          {
            "count": 0,
            "label": "category-amazon"
          },
          {
            "count": 0,
            "label": "category-featured"
          },
          {
            "count": 0,
            "label": "category-streaming-and-messaging"
          },
          {
            "count": 1,
            "label": "sink"
          },
          {
            "count": 0,
            "label": "source"
          }
        ]
      }
      """

    # check search param
    When I GET path "/v1/kafka_connector_types/labels?search=channel+like+beta"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "items": [
          {
            "count": 1,
            "label": "category-amazon"
          },
          {
            "count": 1,
            "label": "category-featured"
          },
          {
            "count": 1,
            "label": "category-streaming-and-messaging"
          },
          {
            "count": 0,
            "label": "sink"
          },
          {
            "count": 1,
            "label": "source"
          }
        ]
      }
      """

    # check search param
    When I GET path "/v1/kafka_connector_types/labels?search=label+like+category-%25"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "items": [
          {
            "count": 1,
            "label": "category-amazon"
          },
          {
            "count": 1,
            "label": "category-featured"
          },
          {
            "count": 1,
            "label": "category-streaming-and-messaging"
          },
          {
            "count": 0,
            "label": "sink"
          },
          {
            "count": 1,
            "label": "source"
          }
        ]
      }
      """

    # check AND condition search for labels using like
    When I GET path "/v1/kafka_connector_types/labels?search=label+like+%25category-featured%25source%25"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "items": [
          {
            "count": 1,
            "label": "category-amazon"
          },
          {
            "count": 1,
            "label": "category-featured"
          },
          {
            "count": 1,
            "label": "category-streaming-and-messaging"
          },
          {
            "count": 0,
            "label": "sink"
          },
          {
            "count": 1,
            "label": "source"
          }
        ]
      }
      """

  Scenario: Gary lists all connector types
    Given I am logged in as "Gary"
    When I GET path "/v1/kafka_connector_types"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "items": [
          {
            "channels": [
              "stable",
              "beta"
            ],
            "description": "AWS SQS Source",
            "featured_rank": 10,
            "href": "/api/connector_mgmt/v1/kafka_connector_types/aws-sqs-source-v1alpha1",
            "icon_href": "TODO",
            "id": "aws-sqs-source-v1alpha1",
            "kind": "ConnectorType",
            "labels": [
              "source",
              "category-streaming-and-messaging",
              "category-amazon",
              "category-featured"
            ],
            "annotations": {
              "cos.bf2.org/pricing-tier": "essentials"
            },
            "capabilities": [
              "processors"
            ],
            "name": "aws-sqs-source",
            "schema": {
              "$defs": {
                "processors": {
                  "extract_field": {
                    "description": "Extract a field from the body",
                    "properties": {
                      "field": {
                        "description": "The name of the field to be added",
                        "title": "Field",
                        "type": "string"
                      }
                    },
                    "required": [
                      "field"
                    ],
                    "title": "Extract Field Action",
                    "type": "object"
                  },
                  "has_header_filter": {
                    "description": "Filter based on the presence of one header",
                    "properties": {
                      "name": {
                        "description": "The header name to evaluate",
                        "example": "headerName",
                        "title": "Header Name",
                        "type": "string"
                      },
                      "value": {
                        "description": "An optional header value to compare the header to",
                        "example": "headerValue",
                        "title": "Header Value",
                        "type": "string"
                      }
                    },
                    "required": [
                      "name"
                    ],
                    "title": "Has Header Filter Action",
                    "type": "object"
                  },
                  "insert_field": {
                    "description": "Adds a custom field with a constant value to the message in transit.\n\nThis action works with Json Object. So it will expect a Json Array or a Json Object.\n\nIf for example you have an array like '{ \"foo\":\"John\", \"bar\":30 }' and your action has been configured with field as 'element' and value as 'hello', you'll get '{ \"foo\":\"John\", \"bar\":30, \"element\":\"hello\" }'\n\nNo headers mapping supported, only constant values.",
                    "properties": {
                      "field": {
                        "description": "The name of the field to be added",
                        "title": "Field",
                        "type": "string"
                      },
                      "value": {
                        "description": "The value of the field",
                        "title": "Value",
                        "type": "string"
                      }
                    },
                    "required": [
                      "field",
                      "value"
                    ],
                    "title": "Insert Field Action",
                    "type": "object"
                  },
                  "throttle": {
                    "description": "The Throttle action allows to ensure that a specific sink does not get overloaded.",
                    "properties": {
                      "messages": {
                        "description": "The number of messages to send in the time period set",
                        "example": 10,
                        "title": "Messages Number",
                        "type": "integer"
                      },
                      "timePeriod": {
                        "default": "1000",
                        "description": "Sets the time period during which the maximum request count is valid for, in milliseconds",
                        "title": "Time Period",
                        "type": "string"
                      }
                    },
                    "required": [
                      "messages"
                    ],
                    "title": "Throttle Action",
                    "type": "object"
                  }
                }
              },
              "properties": {
                "aws_access_key": {
                  "oneOf": [
                    {
                      "description": "The access key obtained from AWS",
                      "format": "password",
                      "title": "Access Key",
                      "type": "string"
                    },
                    {
                      "description": "An opaque reference to the aws_access_key",
                      "properties": {},
                      "type": "object"
                    }
                  ],
                  "title": "Access Key",
                  "x-group": "credentials"
                },
                "aws_amazon_a_w_s_host": {
                  "description": "The hostname of the Amazon AWS cloud.",
                  "title": "AWS Host",
                  "type": "string"
                },
                "aws_auto_create_queue": {
                  "default": false,
                  "description": "Setting the autocreation of the SQS queue.",
                  "title": "Autocreate Queue",
                  "type": "boolean"
                },
                "aws_delete_after_read": {
                  "default": true,
                  "description": "Delete messages after consuming them",
                  "title": "Auto-delete Messages",
                  "type": "boolean"
                },
                "aws_protocol": {
                  "default": "https",
                  "description": "The underlying protocol used to communicate with SQS",
                  "example": "http or https",
                  "title": "Protocol",
                  "type": "string"
                },
                "aws_queue_name_or_arn": {
                  "description": "The SQS Queue Name or ARN",
                  "title": "Queue Name",
                  "type": "string"
                },
                "aws_region": {
                  "description": "The AWS region to connect to",
                  "example": "eu-west-1",
                  "title": "AWS Region",
                  "type": "string"
                },
                "aws_secret_key": {
                  "oneOf": [
                    {
                      "description": "The secret key obtained from AWS",
                      "format": "password",
                      "title": "Secret Key",
                      "type": "string"
                    },
                    {
                      "description": "An opaque reference to the aws_secret_key",
                      "properties": {},
                      "type": "object"
                    }
                  ],
                  "title": "Secret Key",
                  "x-group": "credentials"
                },
                "kafka_topic": {
                  "description": "Comma separated list of Kafka topic names",
                  "title": "Topic Names",
                  "type": "string"
                },
                "processors": {
                  "items": {
                    "oneOf": [
                      {
                        "properties": {
                          "insert_field": {
                            "$ref": "#/$defs/processors/insert_field"
                          }
                        },
                        "required": [
                          "insert_field"
                        ],
                        "type": "object"
                      },
                      {
                        "properties": {
                          "extract_field": {
                            "$ref": "#/$defs/processors/extract_field"
                          }
                        },
                        "required": [
                          "extract_field"
                        ],
                        "type": "object"
                      },
                      {
                        "properties": {
                          "has_header_filter": {
                            "$ref": "#/$defs/processors/has_header_filter"
                          }
                        },
                        "required": [
                          "has_header_filter"
                        ],
                        "type": "object"
                      },
                      {
                        "properties": {
                          "throttle": {
                            "$ref": "#/$defs/processors/throttle"
                          }
                        },
                        "required": [
                          "throttle"
                        ],
                        "type": "object"
                      }
                    ]
                  },
                  "type": "array"
                }
              },
              "required": [
                "aws_queue_name_or_arn",
                "aws_access_key",
                "aws_secret_key",
                "aws_region",
                "kafka_topic"
              ],
              "type": "object"
            },
            "version": "v1alpha1"
          },
          {
            "channels": [
              "stable"
            ],
            "description": "Log Sink",
            "href": "/api/connector_mgmt/v1/kafka_connector_types/log_sink_0.1",
            "icon_href": "TODO",
            "id": "log_sink_0.1",
            "kind": "ConnectorType",
            "labels": [
              "sink"
            ],
            "annotations": {
              "cos.bf2.org/pricing-tier": "free"
            },
            "capabilities": [
              "processors"
            ],
            "name": "Log Sink",
            "schema": {
             "$defs": {
                "processors": {
                  "extract_field": {
                    "description": "Extract a field from the body",
                    "properties": {
                      "field": {
                        "description": "The name of the field to be added",
                        "title": "Field",
                        "type": "string"
                      }
                    },
                    "required": [
                      "field"
                    ],
                    "title": "Extract Field Action",
                    "type": "object"
                  },
                  "has_header_filter": {
                    "description": "Filter based on the presence of one header",
                    "properties": {
                      "name": {
                        "description": "The header name to evaluate",
                        "example": "headerName",
                        "title": "Header Name",
                        "type": "string"
                      },
                      "value": {
                        "description": "An optional header value to compare the header to",
                        "example": "headerValue",
                        "title": "Header Value",
                        "type": "string"
                      }
                    },
                    "required": [
                      "name"
                    ],
                    "title": "Has Header Filter Action",
                    "type": "object"
                  },
                  "insert_field": {
                    "description": "Adds a custom field with a constant value to the message in transit.\n\nThis action works with Json Object. So it will expect a Json Array or a Json Object.\n\nIf for example you have an array like '{ \"foo\":\"John\", \"bar\":30 }' and your action has been configured with field as 'element' and value as 'hello', you'll get '{ \"foo\":\"John\", \"bar\":30, \"element\":\"hello\" }'\n\nNo headers mapping supported, only constant values.",
                    "properties": {
                      "field": {
                        "description": "The name of the field to be added",
                        "title": "Field",
                        "type": "string"
                      },
                      "value": {
                        "description": "The value of the field",
                        "title": "Value",
                        "type": "string"
                      }
                    },
                    "required": [
                      "field",
                      "value"
                    ],
                    "title": "Insert Field Action",
                    "type": "object"
                  },
                  "throttle": {
                    "description": "The Throttle action allows to ensure that a specific sink does not get overloaded.",
                    "properties": {
                      "messages": {
                        "description": "The number of messages to send in the time period set",
                        "example": 10,
                        "title": "Messages Number",
                        "type": "integer"
                      },
                      "timePeriod": {
                        "default": "1000",
                        "description": "Sets the time period during which the maximum request count is valid for, in milliseconds",
                        "title": "Time Period",
                        "type": "string"
                      }
                    },
                    "required": [
                      "messages"
                    ],
                    "title": "Throttle Action",
                    "type": "object"
                  }
                }
              },
              "properties": {
                "kafka_topic": {
                  "description": "Comma separated list of Kafka topic names",
                  "title": "Topic Names",
                  "type": "string"
                },
                "log_multi_line": {
                  "default": false,
                  "description": "Multi Line",
                  "title": "Multi Line",
                  "type": "boolean",
                  "x-group": "common"
                },
                "log_show_all": {
                  "default": false,
                  "description": "Show All",
                  "title": "Show All",
                  "type": "boolean",
                  "x-group": "common"
                },
                "processors": {
                  "items": {
                    "oneOf": [
                      {
                        "properties": {
                          "insert_field": {
                            "$ref": "#/$defs/processors/insert_field"
                          }
                        },
                        "required": [
                          "insert_field"
                        ],
                        "type": "object"
                      },
                      {
                        "properties": {
                          "extract_field": {
                            "$ref": "#/$defs/processors/extract_field"
                          }
                        },
                        "required": [
                          "extract_field"
                        ],
                        "type": "object"
                      },
                      {
                        "properties": {
                          "has_header_filter": {
                            "$ref": "#/$defs/processors/has_header_filter"
                          }
                        },
                        "required": [
                          "has_header_filter"
                        ],
                        "type": "object"
                      },
                      {
                        "properties": {
                          "throttle": {
                            "$ref": "#/$defs/processors/throttle"
                          }
                        },
                        "required": [
                          "throttle"
                        ],
                        "type": "object"
                      }
                    ]
                  },
                  "type": "array"
                }
              },
              "required": [
                "kafka_topic"
              ],
              "type": "object"
            },
            "version": "0.1"
          },
          {
            "channels": [
                "stable"
            ],
            "href": "/api/connector_mgmt/v1/kafka_connector_types/OldConnectorTypeStillInUseId",
            "id": "OldConnectorTypeStillInUseId",
            "kind": "ConnectorType",
            "name": "Old Connector Type still in use",
            "schema": null,
            "deprecated": true,
            "version": ""
          }
        ],
        "kind": "ConnectorTypeList",
        "page": 1,
        "size": 3,
        "total": 3
      }
      """

  Scenario: Gary lists deprecated connector types
    Given I am logged in as "Gary"
    When I GET path "/v1/kafka_connector_types?search=deprecated=true"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "items": [
          {
            "channels": [
                "stable"
            ],
            "deprecated": true,
            "href": "/api/connector_mgmt/v1/kafka_connector_types/OldConnectorTypeStillInUseId",
            "id": "OldConnectorTypeStillInUseId",
            "kind": "ConnectorType",
            "name": "Old Connector Type still in use",
            "schema": null,
            "version": ""
          }
        ],
        "kind": "ConnectorTypeList",
        "page": 1,
        "size": 1,
        "total": 1
      }
      """

  Scenario: Gary searches for connector types using name and ilike
    Given I am logged in as "Gary"
    When I GET path "/v1/kafka_connector_types/?search=name+ilike+%25AWS%25&orderBy=id%2Ccreated_at+desc"
    Then the response code should be 200
    And the ".items[0].name" selection from the response should match "aws-sqs-source"

  Scenario: Gary searches for sink connector types
    Given I am logged in as "Gary"
    When I GET path "/v1/kafka_connector_types?search=label=sink"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "items": [
          {
            "channels": [
              "stable"
            ],
            "description": "Log Sink",
            "href": "/api/connector_mgmt/v1/kafka_connector_types/log_sink_0.1",
            "icon_href": "TODO",
            "id": "log_sink_0.1",
            "kind": "ConnectorType",
            "labels": [
              "sink"
            ],
            "annotations": {
              "cos.bf2.org/pricing-tier": "free"
            },
            "capabilities": [
              "processors"
            ],
            "name": "Log Sink",
            "schema": {
              "$defs": {
                "processors": {
                  "extract_field": {
                    "description": "Extract a field from the body",
                    "properties": {
                      "field": {
                        "description": "The name of the field to be added",
                        "title": "Field",
                        "type": "string"
                      }
                    },
                    "required": [
                      "field"
                    ],
                    "title": "Extract Field Action",
                    "type": "object"
                  },
                  "has_header_filter": {
                    "description": "Filter based on the presence of one header",
                    "properties": {
                      "name": {
                        "description": "The header name to evaluate",
                        "example": "headerName",
                        "title": "Header Name",
                        "type": "string"
                      },
                      "value": {
                        "description": "An optional header value to compare the header to",
                        "example": "headerValue",
                        "title": "Header Value",
                        "type": "string"
                      }
                    },
                    "required": [
                      "name"
                    ],
                    "title": "Has Header Filter Action",
                    "type": "object"
                  },
                  "insert_field": {
                    "description": "Adds a custom field with a constant value to the message in transit.\n\nThis action works with Json Object. So it will expect a Json Array or a Json Object.\n\nIf for example you have an array like '{ \"foo\":\"John\", \"bar\":30 }' and your action has been configured with field as 'element' and value as 'hello', you'll get '{ \"foo\":\"John\", \"bar\":30, \"element\":\"hello\" }'\n\nNo headers mapping supported, only constant values.",
                    "properties": {
                      "field": {
                        "description": "The name of the field to be added",
                        "title": "Field",
                        "type": "string"
                      },
                      "value": {
                        "description": "The value of the field",
                        "title": "Value",
                        "type": "string"
                      }
                    },
                    "required": [
                      "field",
                      "value"
                    ],
                    "title": "Insert Field Action",
                    "type": "object"
                  },
                  "throttle": {
                    "description": "The Throttle action allows to ensure that a specific sink does not get overloaded.",
                    "properties": {
                      "messages": {
                        "description": "The number of messages to send in the time period set",
                        "example": 10,
                        "title": "Messages Number",
                        "type": "integer"
                      },
                      "timePeriod": {
                        "default": "1000",
                        "description": "Sets the time period during which the maximum request count is valid for, in milliseconds",
                        "title": "Time Period",
                        "type": "string"
                      }
                    },
                    "required": [
                      "messages"
                    ],
                    "title": "Throttle Action",
                    "type": "object"
                  }
                }
              },
              "properties": {
                "kafka_topic": {
                  "description": "Comma separated list of Kafka topic names",
                  "title": "Topic Names",
                  "type": "string"
                },
                "log_multi_line": {
                  "default": false,
                  "description": "Multi Line",
                  "title": "Multi Line",
                  "type": "boolean",
                  "x-group": "common"
                },
                "log_show_all": {
                  "default": false,
                  "description": "Show All",
                  "title": "Show All",
                  "type": "boolean",
                  "x-group": "common"
                },
                "processors": {
                  "items": {
                    "oneOf": [
                      {
                        "properties": {
                          "insert_field": {
                            "$ref": "#/$defs/processors/insert_field"
                          }
                        },
                        "required": [
                          "insert_field"
                        ],
                        "type": "object"
                      },
                      {
                        "properties": {
                          "extract_field": {
                            "$ref": "#/$defs/processors/extract_field"
                          }
                        },
                        "required": [
                          "extract_field"
                        ],
                        "type": "object"
                      },
                      {
                        "properties": {
                          "has_header_filter": {
                            "$ref": "#/$defs/processors/has_header_filter"
                          }
                        },
                        "required": [
                          "has_header_filter"
                        ],
                        "type": "object"
                      },
                      {
                        "properties": {
                          "throttle": {
                            "$ref": "#/$defs/processors/throttle"
                          }
                        },
                        "required": [
                          "throttle"
                        ],
                        "type": "object"
                      }
                    ]
                  },
                  "type": "array"
                }
              },
              "required": [
                "kafka_topic"
              ],
              "type": "object"
            },
            "version": "0.1"
          }
        ],
        "kind": "ConnectorTypeList",
        "page": 1,
        "size": 1,
        "total": 1
      }
      """

  Scenario: Gary searches for essentials connector types
    Given I am logged in as "Gary"
    # also, ignoring order by channel, label, and pricing tier should not cause any errors
    When I GET path "/v1/kafka_connector_types?search=pricing_tier=essentials&orderBy=label+ASC,pricing_tier+ASC,channel+DESC"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "items": [
          {
            "channels": [
              "stable",
              "beta"
            ],
            "description": "AWS SQS Source",
            "featured_rank": 10,
            "href": "/api/connector_mgmt/v1/kafka_connector_types/aws-sqs-source-v1alpha1",
            "icon_href": "TODO",
            "id": "aws-sqs-source-v1alpha1",
            "kind": "ConnectorType",
            "labels": [
              "source",
              "category-streaming-and-messaging",
              "category-amazon",
              "category-featured"
            ],
            "annotations": {
              "cos.bf2.org/pricing-tier": "essentials"
            },
            "capabilities": [
              "processors"
            ],
            "name": "aws-sqs-source",
            "schema": {
              "$defs": {
                "processors": {
                  "extract_field": {
                    "description": "Extract a field from the body",
                    "properties": {
                      "field": {
                        "description": "The name of the field to be added",
                        "title": "Field",
                        "type": "string"
                      }
                    },
                    "required": [
                      "field"
                    ],
                    "title": "Extract Field Action",
                    "type": "object"
                  },
                  "has_header_filter": {
                    "description": "Filter based on the presence of one header",
                    "properties": {
                      "name": {
                        "description": "The header name to evaluate",
                        "example": "headerName",
                        "title": "Header Name",
                        "type": "string"
                      },
                      "value": {
                        "description": "An optional header value to compare the header to",
                        "example": "headerValue",
                        "title": "Header Value",
                        "type": "string"
                      }
                    },
                    "required": [
                      "name"
                    ],
                    "title": "Has Header Filter Action",
                    "type": "object"
                  },
                  "insert_field": {
                    "description": "Adds a custom field with a constant value to the message in transit.\n\nThis action works with Json Object. So it will expect a Json Array or a Json Object.\n\nIf for example you have an array like '{ \"foo\":\"John\", \"bar\":30 }' and your action has been configured with field as 'element' and value as 'hello', you'll get '{ \"foo\":\"John\", \"bar\":30, \"element\":\"hello\" }'\n\nNo headers mapping supported, only constant values.",
                    "properties": {
                      "field": {
                        "description": "The name of the field to be added",
                        "title": "Field",
                        "type": "string"
                      },
                      "value": {
                        "description": "The value of the field",
                        "title": "Value",
                        "type": "string"
                      }
                    },
                    "required": [
                      "field",
                      "value"
                    ],
                    "title": "Insert Field Action",
                    "type": "object"
                  },
                  "throttle": {
                    "description": "The Throttle action allows to ensure that a specific sink does not get overloaded.",
                    "properties": {
                      "messages": {
                        "description": "The number of messages to send in the time period set",
                        "example": 10,
                        "title": "Messages Number",
                        "type": "integer"
                      },
                      "timePeriod": {
                        "default": "1000",
                        "description": "Sets the time period during which the maximum request count is valid for, in milliseconds",
                        "title": "Time Period",
                        "type": "string"
                      }
                    },
                    "required": [
                      "messages"
                    ],
                    "title": "Throttle Action",
                    "type": "object"
                  }
                }
              },
              "properties": {
                "aws_access_key": {
                  "oneOf": [
                    {
                      "description": "The access key obtained from AWS",
                      "format": "password",
                      "title": "Access Key",
                      "type": "string"
                    },
                    {
                      "description": "An opaque reference to the aws_access_key",
                      "properties": {},
                      "type": "object"
                    }
                  ],
                  "title": "Access Key",
                  "x-group": "credentials"
                },
                "aws_amazon_a_w_s_host": {
                  "description": "The hostname of the Amazon AWS cloud.",
                  "title": "AWS Host",
                  "type": "string"
                },
                "aws_auto_create_queue": {
                  "default": false,
                  "description": "Setting the autocreation of the SQS queue.",
                  "title": "Autocreate Queue",
                  "type": "boolean"
                },
                "aws_delete_after_read": {
                  "default": true,
                  "description": "Delete messages after consuming them",
                  "title": "Auto-delete Messages",
                  "type": "boolean"
                },
                "aws_protocol": {
                  "default": "https",
                  "description": "The underlying protocol used to communicate with SQS",
                  "example": "http or https",
                  "title": "Protocol",
                  "type": "string"
                },
                "aws_queue_name_or_arn": {
                  "description": "The SQS Queue Name or ARN",
                  "title": "Queue Name",
                  "type": "string"
                },
                "aws_region": {
                  "description": "The AWS region to connect to",
                  "example": "eu-west-1",
                  "title": "AWS Region",
                  "type": "string"
                },
                "aws_secret_key": {
                  "oneOf": [
                    {
                      "description": "The secret key obtained from AWS",
                      "format": "password",
                      "title": "Secret Key",
                      "type": "string"
                    },
                    {
                      "description": "An opaque reference to the aws_secret_key",
                      "properties": {},
                      "type": "object"
                    }
                  ],
                  "title": "Secret Key",
                  "x-group": "credentials"
                },
                "kafka_topic": {
                  "description": "Comma separated list of Kafka topic names",
                  "title": "Topic Names",
                  "type": "string"
                },
                "processors": {
                  "items": {
                    "oneOf": [
                      {
                        "properties": {
                          "insert_field": {
                            "$ref": "#/$defs/processors/insert_field"
                          }
                        },
                        "required": [
                          "insert_field"
                        ],
                        "type": "object"
                      },
                      {
                        "properties": {
                          "extract_field": {
                            "$ref": "#/$defs/processors/extract_field"
                          }
                        },
                        "required": [
                          "extract_field"
                        ],
                        "type": "object"
                      },
                      {
                        "properties": {
                          "has_header_filter": {
                            "$ref": "#/$defs/processors/has_header_filter"
                          }
                        },
                        "required": [
                          "has_header_filter"
                        ],
                        "type": "object"
                      },
                      {
                        "properties": {
                          "throttle": {
                            "$ref": "#/$defs/processors/throttle"
                          }
                        },
                        "required": [
                          "throttle"
                        ],
                        "type": "object"
                      }
                    ]
                  },
                  "type": "array"
                }
              },
              "required": [
                "aws_queue_name_or_arn",
                "aws_access_key",
                "aws_secret_key",
                "aws_region",
                "kafka_topic"
              ],
              "type": "object"
            },
            "version": "v1alpha1"
          }
        ],
        "kind": "ConnectorTypeList",
        "page": 1,
        "size": 1,
        "total": 1
      }
      """

  Scenario: Gary searches for connector types matching multiple label values
    Given I am logged in as "Gary"
    When I GET path "/v1/kafka_connector_types?search=label+like+%25category-featured%25source%25"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "items": [
          {
            "channels": [
              "stable",
              "beta"
            ],
            "description": "AWS SQS Source",
            "featured_rank": 10,
            "href": "/api/connector_mgmt/v1/kafka_connector_types/aws-sqs-source-v1alpha1",
            "icon_href": "TODO",
            "id": "aws-sqs-source-v1alpha1",
            "kind": "ConnectorType",
            "labels": [
              "source",
              "category-streaming-and-messaging",
              "category-amazon",
              "category-featured"
            ],
            "annotations": {
              "cos.bf2.org/pricing-tier": "essentials"
            },
            "capabilities": [
              "processors"
            ],
            "name": "aws-sqs-source",
            "schema": {
              "$defs": {
                "processors": {
                  "extract_field": {
                    "description": "Extract a field from the body",
                    "properties": {
                      "field": {
                        "description": "The name of the field to be added",
                        "title": "Field",
                        "type": "string"
                      }
                    },
                    "required": [
                      "field"
                    ],
                    "title": "Extract Field Action",
                    "type": "object"
                  },
                  "has_header_filter": {
                    "description": "Filter based on the presence of one header",
                    "properties": {
                      "name": {
                        "description": "The header name to evaluate",
                        "example": "headerName",
                        "title": "Header Name",
                        "type": "string"
                      },
                      "value": {
                        "description": "An optional header value to compare the header to",
                        "example": "headerValue",
                        "title": "Header Value",
                        "type": "string"
                      }
                    },
                    "required": [
                      "name"
                    ],
                    "title": "Has Header Filter Action",
                    "type": "object"
                  },
                  "insert_field": {
                    "description": "Adds a custom field with a constant value to the message in transit.\n\nThis action works with Json Object. So it will expect a Json Array or a Json Object.\n\nIf for example you have an array like '{ \"foo\":\"John\", \"bar\":30 }' and your action has been configured with field as 'element' and value as 'hello', you'll get '{ \"foo\":\"John\", \"bar\":30, \"element\":\"hello\" }'\n\nNo headers mapping supported, only constant values.",
                    "properties": {
                      "field": {
                        "description": "The name of the field to be added",
                        "title": "Field",
                        "type": "string"
                      },
                      "value": {
                        "description": "The value of the field",
                        "title": "Value",
                        "type": "string"
                      }
                    },
                    "required": [
                      "field",
                      "value"
                    ],
                    "title": "Insert Field Action",
                    "type": "object"
                  },
                  "throttle": {
                    "description": "The Throttle action allows to ensure that a specific sink does not get overloaded.",
                    "properties": {
                      "messages": {
                        "description": "The number of messages to send in the time period set",
                        "example": 10,
                        "title": "Messages Number",
                        "type": "integer"
                      },
                      "timePeriod": {
                        "default": "1000",
                        "description": "Sets the time period during which the maximum request count is valid for, in milliseconds",
                        "title": "Time Period",
                        "type": "string"
                      }
                    },
                    "required": [
                      "messages"
                    ],
                    "title": "Throttle Action",
                    "type": "object"
                  }
                }
              },
              "properties": {
                "aws_access_key": {
                  "oneOf": [
                    {
                      "description": "The access key obtained from AWS",
                      "format": "password",
                      "title": "Access Key",
                      "type": "string"
                    },
                    {
                      "description": "An opaque reference to the aws_access_key",
                      "properties": {},
                      "type": "object"
                    }
                  ],
                  "title": "Access Key",
                  "x-group": "credentials"
                },
                "aws_amazon_a_w_s_host": {
                  "description": "The hostname of the Amazon AWS cloud.",
                  "title": "AWS Host",
                  "type": "string"
                },
                "aws_auto_create_queue": {
                  "default": false,
                  "description": "Setting the autocreation of the SQS queue.",
                  "title": "Autocreate Queue",
                  "type": "boolean"
                },
                "aws_delete_after_read": {
                  "default": true,
                  "description": "Delete messages after consuming them",
                  "title": "Auto-delete Messages",
                  "type": "boolean"
                },
                "aws_protocol": {
                  "default": "https",
                  "description": "The underlying protocol used to communicate with SQS",
                  "example": "http or https",
                  "title": "Protocol",
                  "type": "string"
                },
                "aws_queue_name_or_arn": {
                  "description": "The SQS Queue Name or ARN",
                  "title": "Queue Name",
                  "type": "string"
                },
                "aws_region": {
                  "description": "The AWS region to connect to",
                  "example": "eu-west-1",
                  "title": "AWS Region",
                  "type": "string"
                },
                "aws_secret_key": {
                  "oneOf": [
                    {
                      "description": "The secret key obtained from AWS",
                      "format": "password",
                      "title": "Secret Key",
                      "type": "string"
                    },
                    {
                      "description": "An opaque reference to the aws_secret_key",
                      "properties": {},
                      "type": "object"
                    }
                  ],
                  "title": "Secret Key",
                  "x-group": "credentials"
                },
                "kafka_topic": {
                  "description": "Comma separated list of Kafka topic names",
                  "title": "Topic Names",
                  "type": "string"
                },
                "processors": {
                  "items": {
                    "oneOf": [
                      {
                        "properties": {
                          "insert_field": {
                            "$ref": "#/$defs/processors/insert_field"
                          }
                        },
                        "required": [
                          "insert_field"
                        ],
                        "type": "object"
                      },
                      {
                        "properties": {
                          "extract_field": {
                            "$ref": "#/$defs/processors/extract_field"
                          }
                        },
                        "required": [
                          "extract_field"
                        ],
                        "type": "object"
                      },
                      {
                        "properties": {
                          "has_header_filter": {
                            "$ref": "#/$defs/processors/has_header_filter"
                          }
                        },
                        "required": [
                          "has_header_filter"
                        ],
                        "type": "object"
                      },
                      {
                        "properties": {
                          "throttle": {
                            "$ref": "#/$defs/processors/throttle"
                          }
                        },
                        "required": [
                          "throttle"
                        ],
                        "type": "object"
                      }
                    ]
                  },
                  "type": "array"
                }
              },
              "required": [
                "aws_queue_name_or_arn",
                "aws_access_key",
                "aws_secret_key",
                "aws_region",
                "kafka_topic"
              ],
              "type": "object"
            },
            "version": "v1alpha1"
          }
        ],
        "kind": "ConnectorTypeList",
        "page": 1,
        "size": 1,
        "total": 1
      }
      """

  Scenario: Gary searches for connector types on beta channel, ordered by version
    Given I am logged in as "Gary"
    When I GET path "/v1/kafka_connector_types?search=channel=beta&orderBy=version"
    Then the response code should be 200
    And the response should match json:
      """
      {
         "items": [
           {
             "channels": [
               "stable",
               "beta"
             ],
             "description": "AWS SQS Source",
             "featured_rank": 10,
             "href": "/api/connector_mgmt/v1/kafka_connector_types/aws-sqs-source-v1alpha1",
             "icon_href": "TODO",
             "id": "aws-sqs-source-v1alpha1",
             "kind": "ConnectorType",
             "labels": [
               "source",
               "category-streaming-and-messaging",
               "category-amazon",
               "category-featured"
             ],
             "annotations": {
               "cos.bf2.org/pricing-tier": "essentials"
             },
             "capabilities": [
              "processors"
            ],
             "name": "aws-sqs-source",
             "schema": {
               "$defs": {
                 "processors": {
                   "extract_field": {
                     "description": "Extract a field from the body",
                     "properties": {
                       "field": {
                         "description": "The name of the field to be added",
                         "title": "Field",
                         "type": "string"
                       }
                     },
                     "required": [
                       "field"
                     ],
                     "title": "Extract Field Action",
                     "type": "object"
                   },
                   "has_header_filter": {
                     "description": "Filter based on the presence of one header",
                     "properties": {
                       "name": {
                         "description": "The header name to evaluate",
                         "example": "headerName",
                         "title": "Header Name",
                         "type": "string"
                       },
                       "value": {
                         "description": "An optional header value to compare the header to",
                         "example": "headerValue",
                         "title": "Header Value",
                         "type": "string"
                       }
                     },
                     "required": [
                       "name"
                     ],
                     "title": "Has Header Filter Action",
                     "type": "object"
                   },
                   "insert_field": {
                     "description": "Adds a custom field with a constant value to the message in transit.\n\nThis action works with Json Object. So it will expect a Json Array or a Json Object.\n\nIf for example you have an array like '{ \"foo\":\"John\", \"bar\":30 }' and your action has been configured with field as 'element' and value as 'hello', you'll get '{ \"foo\":\"John\", \"bar\":30, \"element\":\"hello\" }'\n\nNo headers mapping supported, only constant values.",
                     "properties": {
                       "field": {
                         "description": "The name of the field to be added",
                         "title": "Field",
                         "type": "string"
                       },
                       "value": {
                         "description": "The value of the field",
                         "title": "Value",
                         "type": "string"
                       }
                     },
                     "required": [
                       "field",
                       "value"
                     ],
                     "title": "Insert Field Action",
                     "type": "object"
                   },
                   "throttle": {
                     "description": "The Throttle action allows to ensure that a specific sink does not get overloaded.",
                     "properties": {
                       "messages": {
                         "description": "The number of messages to send in the time period set",
                         "example": 10,
                         "title": "Messages Number",
                         "type": "integer"
                       },
                       "timePeriod": {
                         "default": "1000",
                         "description": "Sets the time period during which the maximum request count is valid for, in milliseconds",
                         "title": "Time Period",
                         "type": "string"
                       }
                     },
                     "required": [
                       "messages"
                     ],
                     "title": "Throttle Action",
                     "type": "object"
                   }
                 }
               },
               "properties": {
                 "aws_access_key": {
                   "oneOf": [
                     {
                       "description": "The access key obtained from AWS",
                       "format": "password",
                       "title": "Access Key",
                       "type": "string"
                     },
                     {
                       "description": "An opaque reference to the aws_access_key",
                       "properties": {},
                       "type": "object"
                     }
                   ],
                   "title": "Access Key",
                   "x-group": "credentials"
                 },
                 "aws_amazon_a_w_s_host": {
                   "description": "The hostname of the Amazon AWS cloud.",
                   "title": "AWS Host",
                   "type": "string"
                 },
                 "aws_auto_create_queue": {
                   "default": false,
                   "description": "Setting the autocreation of the SQS queue.",
                   "title": "Autocreate Queue",
                   "type": "boolean"
                 },
                 "aws_delete_after_read": {
                   "default": true,
                   "description": "Delete messages after consuming them",
                   "title": "Auto-delete Messages",
                   "type": "boolean"
                 },
                 "aws_protocol": {
                   "default": "https",
                   "description": "The underlying protocol used to communicate with SQS",
                   "example": "http or https",
                   "title": "Protocol",
                   "type": "string"
                 },
                 "aws_queue_name_or_arn": {
                   "description": "The SQS Queue Name or ARN",
                   "title": "Queue Name",
                   "type": "string"
                 },
                 "aws_region": {
                   "description": "The AWS region to connect to",
                   "example": "eu-west-1",
                   "title": "AWS Region",
                   "type": "string"
                 },
                 "aws_secret_key": {
                   "oneOf": [
                     {
                       "description": "The secret key obtained from AWS",
                       "format": "password",
                       "title": "Secret Key",
                       "type": "string"
                     },
                     {
                       "description": "An opaque reference to the aws_secret_key",
                       "properties": {},
                       "type": "object"
                     }
                   ],
                   "title": "Secret Key",
                   "x-group": "credentials"
                 },
                 "kafka_topic": {
                   "description": "Comma separated list of Kafka topic names",
                   "title": "Topic Names",
                   "type": "string"
                 },
                 "processors": {
                   "items": {
                     "oneOf": [
                       {
                         "properties": {
                           "insert_field": {
                             "$ref": "#/$defs/processors/insert_field"
                           }
                         },
                         "required": [
                           "insert_field"
                         ],
                         "type": "object"
                       },
                       {
                         "properties": {
                           "extract_field": {
                             "$ref": "#/$defs/processors/extract_field"
                           }
                         },
                         "required": [
                           "extract_field"
                         ],
                         "type": "object"
                       },
                       {
                         "properties": {
                           "has_header_filter": {
                             "$ref": "#/$defs/processors/has_header_filter"
                           }
                         },
                         "required": [
                           "has_header_filter"
                         ],
                         "type": "object"
                       },
                       {
                         "properties": {
                           "throttle": {
                             "$ref": "#/$defs/processors/throttle"
                           }
                         },
                         "required": [
                           "throttle"
                         ],
                         "type": "object"
                       }
                     ]
                   },
                   "type": "array"
                 }
               },
               "required": [
                 "aws_queue_name_or_arn",
                 "aws_access_key",
                 "aws_secret_key",
                 "aws_region",
                 "kafka_topic"
               ],
               "type": "object"
             },
             "version": "v1alpha1"
           }
         ],
         "kind": "ConnectorTypeList",
         "page": 1,
         "size": 1,
         "total": 1
      }
      """

  Scenario: Gary uses paging to list connector types
    Given I am logged in as "Gary"
    When I GET path "/v1/kafka_connector_types?orderBy=name%20asc&page=2&size=1"
    Then the response code should be 200
    And the response should match json:
      """
      {
         "items": [
           {
             "channels": [
               "stable"
             ],
             "description": "Log Sink",
             "href": "/api/connector_mgmt/v1/kafka_connector_types/log_sink_0.1",
             "icon_href": "TODO",
             "id": "log_sink_0.1",
             "kind": "ConnectorType",
             "labels": [
               "sink"
             ],
             "annotations": {
               "cos.bf2.org/pricing-tier": "free"
             },
             "capabilities": [
              "processors"
            ],
             "name": "Log Sink",
             "schema": {
               "$defs": {
                 "processors": {
                   "extract_field": {
                     "description": "Extract a field from the body",
                     "properties": {
                       "field": {
                         "description": "The name of the field to be added",
                         "title": "Field",
                         "type": "string"
                       }
                     },
                     "required": [
                       "field"
                     ],
                     "title": "Extract Field Action",
                     "type": "object"
                   },
                   "has_header_filter": {
                     "description": "Filter based on the presence of one header",
                     "properties": {
                       "name": {
                         "description": "The header name to evaluate",
                         "example": "headerName",
                         "title": "Header Name",
                         "type": "string"
                       },
                       "value": {
                         "description": "An optional header value to compare the header to",
                         "example": "headerValue",
                         "title": "Header Value",
                         "type": "string"
                       }
                     },
                     "required": [
                       "name"
                     ],
                     "title": "Has Header Filter Action",
                     "type": "object"
                   },
                   "insert_field": {
                     "description": "Adds a custom field with a constant value to the message in transit.\n\nThis action works with Json Object. So it will expect a Json Array or a Json Object.\n\nIf for example you have an array like '{ \"foo\":\"John\", \"bar\":30 }' and your action has been configured with field as 'element' and value as 'hello', you'll get '{ \"foo\":\"John\", \"bar\":30, \"element\":\"hello\" }'\n\nNo headers mapping supported, only constant values.",
                     "properties": {
                       "field": {
                         "description": "The name of the field to be added",
                         "title": "Field",
                         "type": "string"
                       },
                       "value": {
                         "description": "The value of the field",
                         "title": "Value",
                         "type": "string"
                       }
                     },
                     "required": [
                       "field",
                       "value"
                     ],
                     "title": "Insert Field Action",
                     "type": "object"
                   },
                   "throttle": {
                     "description": "The Throttle action allows to ensure that a specific sink does not get overloaded.",
                     "properties": {
                       "messages": {
                         "description": "The number of messages to send in the time period set",
                         "example": 10,
                         "title": "Messages Number",
                         "type": "integer"
                       },
                       "timePeriod": {
                         "default": "1000",
                         "description": "Sets the time period during which the maximum request count is valid for, in milliseconds",
                         "title": "Time Period",
                         "type": "string"
                       }
                     },
                     "required": [
                       "messages"
                     ],
                     "title": "Throttle Action",
                     "type": "object"
                   }
                 }
               },
               "properties": {
                 "kafka_topic": {
                   "description": "Comma separated list of Kafka topic names",
                   "title": "Topic Names",
                   "type": "string"
                 },
                 "log_multi_line": {
                   "default": false,
                   "description": "Multi Line",
                   "title": "Multi Line",
                   "type": "boolean",
                   "x-group": "common"
                 },
                 "log_show_all": {
                   "default": false,
                   "description": "Show All",
                   "title": "Show All",
                   "type": "boolean",
                   "x-group": "common"
                 },
                 "processors": {
                   "items": {
                     "oneOf": [
                       {
                         "properties": {
                           "insert_field": {
                             "$ref": "#/$defs/processors/insert_field"
                           }
                         },
                         "required": [
                           "insert_field"
                         ],
                         "type": "object"
                       },
                       {
                         "properties": {
                           "extract_field": {
                             "$ref": "#/$defs/processors/extract_field"
                           }
                         },
                         "required": [
                           "extract_field"
                         ],
                         "type": "object"
                       },
                       {
                         "properties": {
                           "has_header_filter": {
                             "$ref": "#/$defs/processors/has_header_filter"
                           }
                         },
                         "required": [
                           "has_header_filter"
                         ],
                         "type": "object"
                       },
                       {
                         "properties": {
                           "throttle": {
                             "$ref": "#/$defs/processors/throttle"
                           }
                         },
                         "required": [
                           "throttle"
                         ],
                         "type": "object"
                       }
                     ]
                   },
                   "type": "array"
                 }
               },
               "required": [
                 "kafka_topic"
               ],
               "type": "object"
             },
             "version": "0.1"
           }
         ],
         "kind": "ConnectorTypeList",
         "page": 2,
         "size": 1,
         "total": 3
      }
      """

  Scenario: Gary tries to sql inject connector listing
    Given I am logged in as "Gary"
    When I GET path "/v1/kafka_connectors?orderBy=CAST(CHR(32)||(SELECT+version())+AS+NUMERIC)"
    Then the response code should be 400
    And the response should match json:
      """
      {
        "id":"17",
        "kind":"Error",
        "href":"/api/connector_mgmt/v1/errors/17",
        "code":"CONNECTOR-MGMT-17",
        "reason":"Unable to list connector requests: invalid order by clause 'CAST(CHR(32)||(SELECT version()) AS NUMERIC)'",
        "operation_id": "${response.operation_id}"
      }
      """

    When I GET path "/v1/kafka_connectors?search=CAST(CHR(32)||(SELECT+version())+AS+NUMERIC)"
    Then the response code should be 400
      """
      {
        "id":"23",
        "kind":"Error",
        "href":"/api/connector_mgmt/v1/errors/23",
        "code":"CONNECTOR-MGMT-23",
        "reason":"Unable to list connector type requests: [1] error parsing the filter: invalid column name: 'CAST'",
        "operation_id": "${response.operation_id}"
      }
      """

  Scenario: Gary tries to create a connector with an invalid configuration spec
    Given I am logged in as "Gary"
    When I POST path "/v1/kafka_connectors?async=true" with json body:
      """
      {
        "kind": "Connector",
        "name": "example 1",
        "kafka": {
          "id":"mykafka",
          "url": "kafka.hostname"
        },
        "service_account": {
          "client_secret": "test",
          "client_id": "myclient"
        },
        "connector_type_id": "aws-sqs-source-v1alpha1",
        "connector": {
            "aws_access_key": "test",
            "aws_secret_key": "test",
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
        "reason": "connector spec not conform to the connector type schema. 1 errors encountered.  1st error: (root): aws_queue_name_or_arn is required"
      }
      """

  Scenario: Gary tries to create a connector with an invalid namespace_id
    Given I am logged in as "Gary"
    When I POST path "/v1/kafka_connectors?async=true" with json body:
      """
      {
        "kind": "Connector",
        "name": "example 1",
        "namespace_id": "default",
        "connector_type_id": "aws-sqs-source-v1alpha1",
        "kafka": {
          "id":"mykafka",
          "url": "kafka.hostname"
        },
        "service_account": {
          "client_secret": "test",
          "client_id": "myclient"
        },
        "connector": {
            "aws_queue_name_or_arn": "test",
            "aws_access_key": "test",
            "aws_secret_key": "test",
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
        "reason": "Connector namespace with id='default' not found"
      }
      """

  Scenario: Gary creates lists and deletes a connector verifying that Evil Bob can't access Garys Connectors
  but Coworker Sally can. Gary can't create connectors from unsupported channels
    Given I am logged in as "Gary"
    When I POST path "/v1/kafka_connectors?async=true" with json body:
      """
      {
        "kind": "Connector",
        "name": "example 1",
        "connector_type_id": "aws-sqs-source-v1alpha1",
        "kafka": {
          "id":"mykafka",
          "url": "kafka.hostname"
        },
        "schema_registry": {
          "id":"myregistry",
          "url": "registry.hostname"
        },
        "service_account": {
          "client_secret": "test",
          "client_id": "myclient"
        },
        "connector": {
            "aws_queue_name_or_arn": "test",
            "aws_access_key": "test",
            "aws_secret_key": "test",
            "aws_region": "east",
            "kafka_topic": "test"
        }
      }
      """
    Then the response code should be 202
    And the ".status.state" selection from the response should match "assigning"

    Given I store the ".id" selection from the response as ${connector_id}
    When I GET path "/v1/kafka_connectors?kafka_id=mykafka"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "items": [
          {
            "channel": "stable",
            "kafka": {
              "id": "mykafka",
              "url": "kafka.hostname"
            },
            "schema_registry": {
              "id":"myregistry",
              "url": "registry.hostname"
            },
            "service_account": {
              "client_secret": "",
              "client_id": "myclient"
            },
            "connector": {
              "aws_queue_name_or_arn": "test",
              "aws_access_key": {},
              "aws_secret_key": {},
              "aws_region": "east",
              "kafka_topic": "test"
            },
            "connector_type_id": "aws-sqs-source-v1alpha1",
            "href": "/api/connector_mgmt/v1/kafka_connectors/${connector_id}",
            "id": "${connector_id}",
            "namespace_id": "",
            "kind": "Connector",
            "created_at": "${response.items[0].created_at}",
            "name": "example 1",
            "owner": "${response.items[0].owner}",
            "resource_version": ${response.items[0].resource_version},
            "modified_at": "${response.items[0].modified_at}",
            "desired_state": "ready",
            "annotations": {
              "cos.bf2.org/organisation-id": "13640203",
              "cos.bf2.org/pricing-tier": "essentials"
            },
            "status": {
              "state": "assigning"
            }
          }
        ],
        "kind": "ConnectorList",
        "page": 1,
        "size": 1,
        "total": 1
      }
      """

    When I GET path "/v1/kafka_connectors/${connector_id}"
    Then the response code should be 200
    And the ".status.state" selection from the response should match "assigning"
    And the ".id" selection from the response should match "${connector_id}"
    And the response should match json:
      """
      {
          "id": "${connector_id}",
          "namespace_id": "",
          "kind": "Connector",
          "href": "/api/connector_mgmt/v1/kafka_connectors/${connector_id}",
          "owner": "${response.owner}",
          "name": "example 1",
          "created_at": "${response.created_at}",
          "modified_at": "${response.modified_at}",
          "resource_version": ${response.resource_version},
          "schema_registry": {
            "id": "myregistry",
            "url": "registry.hostname"
          },
          "kafka": {
            "id": "mykafka",
            "url": "kafka.hostname"
          },
          "service_account": {
            "client_secret": "",
            "client_id": "myclient"
          },
          "connector_type_id": "aws-sqs-source-v1alpha1",
          "annotations": {
            "cos.bf2.org/organisation-id": "13640203",
            "cos.bf2.org/pricing-tier": "essentials"
          },
          "channel": "stable",
          "connector": {
              "aws_queue_name_or_arn": "test",
              "aws_access_key": {},
              "aws_secret_key": {},
              "aws_region": "east",
              "kafka_topic": "test"
          },

          "desired_state": "ready",
          "status": {
            "state": "assigning"
          }
      }
      """

    Given I set the "Content-Type" header to "application/merge-patch+json"
    When I PATCH path "/v1/kafka_connectors/${connector_id}" with json body:
      """
      { "connector_type_id": "new-connector-type-id" }
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
        "reason": "An attempt was made to modify one or more immutable field(s): connector_type_id"
      }
      """

    Given I set the "Content-Type" header to "application/merge-patch+json"
    When I PATCH path "/v1/kafka_connectors/${connector_id}" with json body:
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
    When I PATCH path "/v1/kafka_connectors/${connector_id}" with json body:
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
    When I PATCH path "/v1/kafka_connectors/${connector_id}" with json body:
      """
      {
        "connector_type_id": "new-connector-type-id",
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
        "reason": "An attempt was made to modify one or more immutable field(s): connector_type_id, namespace_id, channel"
      }
      """

    # Check update using content type application/json-patch+json
    Given I set the "Content-Type" header to "application/json-patch+json"
    When I PATCH path "/v1/kafka_connectors/${connector_id}" with json body:
      """
      [
        { "op": "replace", "path": "/name", "value": "my-new-name" }
      ]
      """
    Then the response code should be 202
    And the ".name" selection from the response should match "my-new-name"

    # Check annotations update using content type application/merge-patch+json
    Given I set the "Content-Type" header to "application/merge-patch+json"
    When I PATCH path "/v1/kafka_connectors/${connector_id}" with json body:
      """
      {
          "annotations": {
            "cos.bf2.org/organisation-id": "13640203",
            "cos.bf2.org/pricing-tier": "essentials",
            "custom/my-key": "my-value"
          }
      }
      """
    Then the response code should be 202
    And the ".annotations" selection from the response should match json:
      """
      {
          "cos.bf2.org/organisation-id": "13640203",
          "cos.bf2.org/pricing-tier": "essentials",
          "custom/my-key": "my-value"
      }
      """

    # Check invalid annotations update that tries to change pricing-tier
    Given I set the "Content-Type" header to "application/merge-patch+json"
    When I PATCH path "/v1/kafka_connectors/${connector_id}" with json body:
      """
      {
          "annotations": {
            "cos.bf2.org/organisation-id": "13640203",
            "cos.bf2.org/pricing-tier": "free",
            "custom/my-key": "my-value"
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

    # Check invalid annotations update that tries to change organisation-id
    Given I set the "Content-Type" header to "application/merge-patch+json"
    When I PATCH path "/v1/kafka_connectors/${connector_id}" with json body:
      """
      {
          "annotations": {
            "cos.bf2.org/organisation-id": "666",
            "cos.bf2.org/pricing-tier": "essentials",
            "custom/my-key": "my-value"
          }
      }
      """
    Then the response code should be 400
    And the ".reason" selection from the response should match "cannot override reserved annotation cos.bf2.org/organisation-id"

    # Check annotations update to remove an annotation
    Given I set the "Content-Type" header to "application/merge-patch+json"
    When I PATCH path "/v1/kafka_connectors/${connector_id}" with json body:
      """
      {
          "annotations": {
            "cos.bf2.org/organisation-id": "13640203",
            "cos.bf2.org/pricing-tier": "essentials",
            "custom/my-key": null
          }
      }
      """
    Then the response code should be 202
    And the ".annotations" selection from the response should match json:
      """
      {
          "cos.bf2.org/organisation-id": "13640203",
          "cos.bf2.org/pricing-tier": "essentials"
      }
      """

    # Check that not allowed secrets update of a connector are rejected with json-patch
    Given I set the "Content-Type" header to "application/json-patch+json"
    When I PATCH path "/v1/kafka_connectors/${connector_id}" with json body:
      """
      [
        { "op": "add", "path": "/connector/aws_secret_key/ref", "value": "hack" }
      ]
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
        "reason": "invalid patch: attempting to change opaque connector secret"
      }
      """

    # Check that not allowed secrets update of a connector are rejected
    Given I set the "Content-Type" header to "application/merge-patch+json"
    When I PATCH path "/v1/kafka_connectors/${connector_id}" with json body:
      """
      {
          "connector": {
              "aws_secret_key": {
                "ref": "hack"
              }
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
        "reason": "invalid patch: attempting to change opaque connector secret"
      }
      """

    Given I set the "Content-Type" header to "application/merge-patch+json"
    When I PATCH path "/v1/kafka_connectors/${connector_id}" with json body:
      """
      {
          "desired_state": "stopped",
          "connector": {
              "aws_secret_key": {
                "ref": "hack"
              }
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
        "reason": "invalid patch: attempting to change opaque connector secret"
      }
      """

    Given I set the "Content-Type" header to "application/merge-patch+json"
    When I PATCH path "/v1/kafka_connectors/${connector_id}" with json body:
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

    # Check that we can update secrets of a connector
    Given LOCK--------------------------------------------------------------
      Given I reset the vault counters
      Given I set the "Content-Type" header to "application/json"
      When I PATCH path "/v1/kafka_connectors/${connector_id}" with json body:
        """
        {
            "service_account": {
              "client_secret": "patched_secret 1"
            },
            "connector": {
                "aws_secret_key": "patched_secret 2"
            }
        }
        """
      Then the response code should be 202
      And the vault delete counter should be 2
    And UNLOCK---------------------------------------------------------------


    # Before deleting the connector, lets make sure the access control work as expected for other users beside Gary
    Given I am logged in as "Coworker Sally"
    When I GET path "/v1/kafka_connectors/${connector_id}"
    Then the response code should be 200

    Given I am logged in as "Evil Bob"
    When I GET path "/v1/kafka_connectors/${connector_id}"
    Then the response code should be 404

    # We are going to delete the connector...
    Given LOCK--------------------------------------------------------------
      Given I reset the vault counters
      Given I am logged in as "Gary"
      When I DELETE path "/v1/kafka_connectors/${connector_id}"
      Then the response code should be 204
      And the response should match ""

      # The delete occurs async in a worker, so we have to wait a little for the counters to update.
      Given I sleep for 10 seconds
      Then the vault delete counter should be 3
    Given UNLOCK--------------------------------------------------------------

    Given I wait up to "10" seconds for a GET on path "/v1/kafka_connectors/${connector_id}" response code to match "410"
    When I GET path "/v1/kafka_connectors/${connector_id}"
    Then the response code should be 410

    Given I am logged in as "Gary"
    When I POST path "/v1/kafka_connectors?async=true" with json body:
      """
      {
        "name": "wrong-postgresql_sink",
        "kind": "Connector",
        "channel": "alpha",
        "connector_type_id": "postgresql_sink_0.1",
        "desired_state": "ready",
        "kafka": {
          "id": "cfbniubdqprbi2va0r6g",
          "url": "wrong-cfbniubdqprbi-va-r-g.bf2.kafka.rhcloud.com"
        },
        "service_account": {
          "client_id": "client",
          "client_secret": "secret"
        },
        "connector": {
          "data_shape": {
            "consumes": {
              "format": "application/json"
            }
          },
          "kafka_topic": "test",
          "db_server_name": "",
          "db_database_name": "",
          "db_username": "",
          "db_password": "",
          "db_query": "",
          "error_handler": {
            "stop": {}
          }
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
        "reason": "channel is not valid. Must be one of: stable, beta"
      }
      """

  Scenario: Jim creates a connector but later that connector type is removed from the system.  He should
    still be able to list and get the connector, but it's status should note that it has a bad connector
    type and the connector_spec section should be empty since we have lost the json schema for the field.
    The user should still be able to delete the connector.

    Given I am logged in as "Jim"
    When I POST path "/v1/kafka_connectors?async=true" with json body:
      """
      {
        "kind": "Connector",
        "name": "example 1",
        "connector_type_id": "aws-sqs-source-v1alpha1",
        "kafka": {
          "id":"mykafka",
          "url": "kafka.hostname"
        },
        "schema_registry": {
          "id": "myregistry",
          "url": "registry.hostname"
        },
        "service_account": {
          "client_id": "myclient",
          "client_secret": "test"
        },
        "connector": {
            "aws_queue_name_or_arn": "test",
            "aws_access_key": "test",
            "aws_secret_key": "test",
            "aws_region": "east",
            "kafka_topic": "test"
        }
      }
      """
    Then the response code should be 202
    And the ".status.state" selection from the response should match "assigning"
    And I store the ".id" selection from the response as ${connector_id}

    When I run SQL "UPDATE connectors SET connector_type_id='foo' WHERE id = '${connector_id}';" expect 1 row to be affected.

    When I GET path "/v1/kafka_connectors?kafka_id=mykafka"
    Then the response code should be 200
    And the response should match json:
      """
      {
        "items": [
          {
            "channel": "stable",
            "kafka": {
              "id": "mykafka",
              "url": "kafka.hostname"
            },
            "service_account": {
              "client_secret": "",
              "client_id": "myclient"
            },
            "connector": {},
            "connector_type_id": "foo",
            "href": "/api/connector_mgmt/v1/kafka_connectors/${connector_id}",
            "id": "${connector_id}",
            "kind": "Connector",
            "namespace_id": "",
            "created_at": "${response.items[0].created_at}",
            "name": "example 1",
            "owner": "${response.items[0].owner}",
            "resource_version": ${response.items[0].resource_version},
            "modified_at": "${response.items[0].modified_at}",
            "desired_state": "ready",
            "schema_registry": {
              "id": "myregistry",
              "url": "registry.hostname"
            },
            "annotations": {
              "cos.bf2.org/organisation-id": "${jim_org_id}",
              "cos.bf2.org/pricing-tier": "essentials"
            },
            "status": {
              "state": "bad-connector-type"
            }
          }
        ],
        "kind": "ConnectorList",
        "page": 1,
        "size": 1,
        "total": 1
      }
      """

    When I GET path "/v1/kafka_connectors/${connector_id}"
    Then the response code should be 200
    And the response should match json:
      """
      {
          "id": "${connector_id}",
          "kind": "Connector",
          "namespace_id": "",
          "href": "/api/connector_mgmt/v1/kafka_connectors/${connector_id}",
          "owner": "${response.owner}",
          "name": "example 1",
          "created_at": "${response.created_at}",
          "modified_at": "${response.modified_at}",
          "resource_version": ${response.resource_version},
          "kafka": {
            "id": "mykafka",
            "url": "kafka.hostname"
          },
          "service_account": {
            "client_secret": "",
            "client_id": "myclient"
          },
          "connector": {},
          "connector_type_id": "foo",
          "channel": "stable",
          "desired_state": "ready",
          "schema_registry": {
            "id": "myregistry",
            "url": "registry.hostname"
          },
          "annotations": {
            "cos.bf2.org/organisation-id": "${jim_org_id}",
            "cos.bf2.org/pricing-tier": "essentials"
          },
          "status": {
            "state": "bad-connector-type"
          }
      }
      """

    Given LOCK--------------------------------------------------------------
      Given I reset the vault counters

      When I DELETE path "/v1/kafka_connectors/${connector_id}"
      Then the response code should be 204
      And the response should match ""

      # The delete occurs async in a worker, so we have to wait a little for the counters to update.
      Given I sleep for 10 seconds
      # Notice that only 1 key is the deleted.. since we don't have the schema,
      # we can't tell which connector fields are secrets, so we can't clean those up.
      And the vault delete counter should be 1
    And UNLOCK--------------------------------------------------------------

  Scenario: Tommy can delete a connector that was not deployed, but is waiting to be deleted
    Given I am logged in as "Tommy"
    When I POST path "/v1/kafka_connectors?async=true" with json body:
      """
      {
        "kind": "Connector",
        "name": "Tommy's deleting connector",
        "connector_type_id": "aws-sqs-source-v1alpha1",
        "kafka": {
          "id":"mykafka",
          "url": "kafka.hostname"
        },
        "schema_registry": {
          "id":"myregistry",
          "url": "registry.hostname"
        },
        "service_account": {
          "client_secret": "test",
          "client_id": "myclient"
        },
        "connector": {
            "aws_queue_name_or_arn": "test",
            "aws_access_key": "test",
            "aws_secret_key": "test",
            "aws_region": "east",
            "kafka_topic": "test"
        }
      }
      """
    Then the response code should be 202
    And the ".status.state" selection from the response should match "assigning"
    And I store the ".id" selection from the response as ${connector_id}

    Given I run SQL "update connector_statuses SET phase = 'deleting' WHERE id = '${connector_id}'" expect 1 row to be affected.
    And I run SQL "update connectors SET desired_state = 'deleted' WHERE id = '${connector_id}'" expect 1 row to be affected.
    When I wait up to "10" seconds for a GET on path "/v1/kafka_connectors/${connector_id}" response code to match "410"
    Then I GET path "/v1/kafka_connectors/${connector_id}"
    And the response code should be 410
