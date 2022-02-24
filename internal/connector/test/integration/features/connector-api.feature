Feature: create a connector
  In order to use connectors api
  As an API user
  I need to be able to manage connectors

  Background:
    Given the path prefix is "/api/connector_mgmt"
    # Greg and Coworker Sally will end up in the same org
    Given a user named "Greg" in organization "13640203"
    Given a user named "Coworker Sally" in organization "13640203"
    Given a user named "Evil Bob"
    Given a user named "Jim"

  Scenario: Greg lists all connector types
    Given I am logged in as "Greg"
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
            "href": "/api/connector_mgmt/v1/kafka_connector_types/aws-sqs-source-v1alpha1",
            "icon_href": "TODO",
            "id": "aws-sqs-source-v1alpha1",
            "kind": "ConnectorType",
            "labels": [
              "source"
            ],
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
        "size": 2,
        "total": 2
      }
      """

  Scenario: Greg searches for sink connector types
    Given I am logged in as "Greg"
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

  Scenario: Greg searches for connector types on beta channel, ordered by version
    Given I am logged in as "Greg"
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
             "href": "/api/connector_mgmt/v1/kafka_connector_types/aws-sqs-source-v1alpha1",
             "icon_href": "TODO",
             "id": "aws-sqs-source-v1alpha1",
             "kind": "ConnectorType",
             "labels": [
               "source"
             ],
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

  Scenario: Greg uses paging to list connector types
    Given I am logged in as "Greg"
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
         "total": 2
      }
      """

  Scenario: Greg tries to create a connector with an invalid configuration spec
    Given I am logged in as "Greg"
    When I POST path "/v1/kafka_connectors?async=true" with json body:
      """
      {
        "kind": "Connector",
        "name": "example 1",
        "deployment_location": {
        },
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

  Scenario: Greg tries to create a connector with an invalid namespace_id
    Given I am logged in as "Greg"
    When I POST path "/v1/kafka_connectors?async=true" with json body:
      """
      {
        "kind": "Connector",
        "name": "example 1",
        "deployment_location": {
          "namespace_id": "default"
        },
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
        "reason": "deployment_location.namespace_id is not valid: KAFKAS-MGMT-9: failed to get connector namespace: record not found"
      }
      """

  Scenario: Greg creates lists and deletes a connector verifying that Evil Bob can't access Gregs Connectors
  but Coworker Sally can.
    Given I am logged in as "Greg"
    When I POST path "/v1/kafka_connectors?async=true" with json body:
      """
      {
        "kind": "Connector",
        "name": "example 1",
        "deployment_location": {
        },
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
            "deployment_location": {
            },
            "href": "/api/connector_mgmt/v1/kafka_connectors/${connector_id}",
            "id": "${connector_id}",
            "kind": "Connector",
            "created_at": "${response.items[0].created_at}",
            "name": "example 1",
            "owner": "${response.items[0].owner}",
            "resource_version": ${response.items[0].resource_version},
            "modified_at": "${response.items[0].modified_at}",
            "desired_state": "ready",
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
          "deployment_location": {
          },
          "connector_type_id": "aws-sqs-source-v1alpha1",
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
    When I PATCH path "/v1/kafka-connectors/${connector_id}" with json body:
      """
      {
          "connector_spec": {
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

    # Check that we can update secrets of a connector
    Given LOCK--------------------------------------------------------------
      Given I reset the vault counters
      Given I set the "Content-Type" header to "application/merge-patch+json"
      When I PATCH path "/v1/kafka-connectors/${connector_id}" with json body:
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


    # Before deleting the connector, lets make sure the access control work as expected for other users beside Greg
    Given I am logged in as "Coworker Sally"
    When I GET path "/v1/kafka_connectors/${connector_id}"
    Then the response code should be 200

    Given I am logged in as "Evil Bob"
    When I GET path "/v1/kafka_connectors/${connector_id}"
    Then the response code should be 404

    # We are going to delete the connector...
    Given LOCK--------------------------------------------------------------
      Given I reset the vault counters
      Given I am logged in as "Greg"
      When I DELETE path "/v1/kafka_connectors/${connector_id}"
      Then the response code should be 204
      And the response should match ""

      # The delete occurs async in a worker, so we have to wait a little for the counters to update.
      Given I sleep for 2 seconds
      Then the vault delete counter should be 3
    Given UNLOCK--------------------------------------------------------------

    Given I wait up to "5" seconds for a GET on path "/v1/kafka_connectors/${connector_id}" response code to match "404"
    When I GET path "/v1/kafka_connectors/${connector_id}"
    Then the response code should be 404

  Scenario: Greg can discover the API endpoints
    Given I am logged in as "Greg"
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
        "href": "/api/connector_mgmt",
        "id": "connector_mgmt",
        "kind": "API",
        "versions": [
          {
            "kind": "APIVersion",
            "id": "v1",
            "href": "/api/connector_mgmt/v1",
            "collections": null
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

  Scenario: Greg can inspect errors codes
    Given I am logged in as "Greg"
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
            "code": "CONNECTOR-MGMT-35",
            "href": "/api/connector_mgmt/v1/errors/35",
            "id": "35",
            "kind": "Error",
            "reason": "Only multiAZ Kafkas are supported, use multi_az=true"
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
            "reason": "Too Many requests"
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
        "size": 40,
        "total": 40
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
        "deployment_location": {
        },
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
            "deployment_location": {
            },
            "href": "/api/connector_mgmt/v1/kafka_connectors/${connector_id}",
            "id": "${connector_id}",
            "kind": "Connector",
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
          "deployment_location": {
          },
          "connector": {},
          "connector_type_id": "foo",
          "channel": "stable",
          "desired_state": "ready",
          "schema_registry": {
            "id": "myregistry",
            "url": "registry.hostname"
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
      Given I sleep for 2 seconds
      # Notice that only 1 key is the deleted.. since we don't have the schema,
      # we can't tell which connector fields are secrets, so we can't clean those up.
      And the vault delete counter should be 1
    And UNLOCK--------------------------------------------------------------
