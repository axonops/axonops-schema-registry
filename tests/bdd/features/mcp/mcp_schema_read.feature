@mcp
Feature: MCP Schema Read Tools
  MCP tools for reading schemas and schema metadata.

  Background:
    Given I register an Avro schema for subject "mcp-read-test"
    And I store the response field "id" as "schema_id"

  Scenario: Get schema by ID
    When I call MCP tool "get_schema_by_id" with input:
      | id | $schema_id |
    Then the MCP result should contain "AVRO"
    And the MCP result should contain "string"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          |                        |
      | target_id            |                        |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          |                        |
      | after_hash           |                        |
      | context              |                        |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | get_schema_by_id       |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  Scenario: Get raw schema by ID
    When I call MCP tool "get_raw_schema_by_id" with input:
      | id | $schema_id |
    Then the MCP result should contain "string"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          |                        |
      | target_id            |                        |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          |                        |
      | after_hash           |                        |
      | context              |                        |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | get_raw_schema_by_id   |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  Scenario: Get schema by subject and version
    When I call MCP tool "get_schema_version" with input:
      | subject | mcp-read-test |
      | version | 1             |
    Then the MCP result should contain "mcp-read-test"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | mcp-read-test          |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          |                        |
      | after_hash           |                        |
      | context              |                        |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | get_schema_version     |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  Scenario: Get raw schema by subject and version
    When I call MCP tool "get_raw_schema_version" with input:
      | subject | mcp-read-test |
      | version | 1             |
    Then the MCP result should contain "string"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | mcp-read-test          |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          |                        |
      | after_hash           |                        |
      | context              |                        |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | get_raw_schema_version |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  Scenario: Get latest schema
    When I call MCP tool "get_latest_schema" with input:
      | subject | mcp-read-test |
    Then the MCP result should contain "mcp-read-test"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | mcp-read-test          |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          |                        |
      | after_hash           |                        |
      | context              |                        |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | get_latest_schema      |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  Scenario: List versions
    When I call MCP tool "list_versions" with input:
      | subject | mcp-read-test |
    Then the MCP result should contain "1"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | mcp-read-test          |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          |                        |
      | after_hash           |                        |
      | context              |                        |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | list_versions          |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  Scenario: Get subjects for schema
    When I call MCP tool "get_subjects_for_schema" with input:
      | id | $schema_id |
    Then the MCP result should contain "mcp-read-test"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          |                        |
      | target_id            |                        |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          |                        |
      | after_hash           |                        |
      | context              |                        |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | get_subjects_for_schema |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  Scenario: Get versions for schema
    When I call MCP tool "get_versions_for_schema" with input:
      | id | $schema_id |
    Then the MCP result should contain "mcp-read-test"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          |                        |
      | target_id            |                        |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          |                        |
      | after_hash           |                        |
      | context              |                        |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | get_versions_for_schema |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  Scenario: Lookup schema by content
    When I call MCP tool "lookup_schema" with input:
      | subject | mcp-read-test     |
      | schema  | {"type":"string"} |
    Then the MCP result should contain "mcp-read-test"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | mcp-read-test          |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          |                        |
      | after_hash           |                        |
      | context              |                        |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | lookup_schema          |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  Scenario: Get schema types
    When I call MCP tool "get_schema_types"
    Then the MCP result should contain "AVRO"
    And the MCP result should contain "PROTOBUF"
    And the MCP result should contain "JSON"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          |                        |
      | target_id            |                        |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          |                        |
      | after_hash           |                        |
      | context              |                        |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | get_schema_types       |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  Scenario: List schemas
    When I call MCP tool "list_schemas"
    Then the MCP result should contain "mcp-read-test"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          |                        |
      | target_id            |                        |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          |                        |
      | after_hash           |                        |
      | context              |                        |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | list_schemas           |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  Scenario: Get max schema ID
    When I call MCP tool "get_max_schema_id"
    Then the MCP result should contain "$schema_id"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          |                        |
      | target_id            |                        |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          |                        |
      | after_hash           |                        |
      | context              |                        |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | get_max_schema_id      |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  Scenario: Get schema by ID not found
    When I call MCP tool "get_schema_by_id" with input:
      | id | 99999 |
    Then the MCP result should contain "error"
    And the audit log should contain an event:
      | event_type           | mcp_tool_error         |
      | outcome              | failure                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          |                        |
      | target_id            |                        |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          |                        |
      | after_hash           |                        |
      | context              |                        |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | get_schema_by_id       |
      | status_code          | 0                      |
      | reason               | internal_error         |
      | error                | tool returned error    |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |
