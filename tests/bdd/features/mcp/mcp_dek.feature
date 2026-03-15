@mcp
Feature: MCP KEK & DEK Tools
  MCP tools for managing Key Encryption Keys (KEK) and Data Encryption Keys (DEK)
  for client-side field encryption (CSFLE).

  Scenario: Create and get KEK
    When I call MCP tool "create_kek" with input:
      | name       | mcp-test-kek                                      |
      | kms_type   | aws-kms                                           |
      | kms_key_id | arn:aws:kms:us-east-1:123456789:key/mcp-test-kek  |
    Then the MCP result should contain "mcp-test-kek"
    Then the MCP result should contain "aws-kms"
    When I call MCP tool "get_kek" with input:
      | name | mcp-test-kek |
    Then the MCP result should contain "mcp-test-kek"
    Then the MCP result should contain "aws-kms"
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
      | path                 | get_kek                |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  Scenario: List KEKs
    When I call MCP tool "create_kek" with input:
      | name       | mcp-list-kek-a                                      |
      | kms_type   | aws-kms                                             |
      | kms_key_id | arn:aws:kms:us-east-1:123456789:key/mcp-list-kek-a  |
    When I call MCP tool "create_kek" with input:
      | name       | mcp-list-kek-b                                      |
      | kms_type   | aws-kms                                             |
      | kms_key_id | arn:aws:kms:us-east-1:123456789:key/mcp-list-kek-b  |
    When I call MCP tool "list_keks"
    Then the MCP result should contain "mcp-list-kek-a"
    Then the MCP result should contain "mcp-list-kek-b"
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
      | path                 | list_keks              |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  Scenario: Delete and undelete KEK
    When I call MCP tool "create_kek" with input:
      | name       | mcp-del-kek                                      |
      | kms_type   | aws-kms                                           |
      | kms_key_id | arn:aws:kms:us-east-1:123456789:key/mcp-del-kek   |
    When I call MCP tool "delete_kek" with input:
      | name | mcp-del-kek |
    Then the MCP result should contain "true"
    When I call MCP tool "undelete_kek" with input:
      | name | mcp-del-kek |
    Then the MCP result should contain "true"
    When I call MCP tool "get_kek" with input:
      | name | mcp-del-kek |
    Then the MCP result should contain "mcp-del-kek"
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
      | path                 | get_kek                |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  Scenario: Create and get DEK
    When I call MCP tool "create_kek" with input:
      | name       | mcp-dek-parent                                      |
      | kms_type   | aws-kms                                              |
      | kms_key_id | arn:aws:kms:us-east-1:123456789:key/mcp-dek-parent   |
    When I call MCP tool "create_dek" with input:
      | kek_name  | mcp-dek-parent   |
      | subject   | mcp-dek-subject  |
      | algorithm | AES256_GCM       |
    Then the MCP result should contain "mcp-dek-parent"
    Then the MCP result should contain "mcp-dek-subject"
    When I call MCP tool "get_dek" with input:
      | kek_name  | mcp-dek-parent   |
      | subject   | mcp-dek-subject  |
      | algorithm | AES256_GCM       |
    Then the MCP result should contain "mcp-dek-parent"
    Then the MCP result should contain "mcp-dek-subject"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | mcp-dek-subject        |
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
      | path                 | get_dek                |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  Scenario: List DEKs
    When I call MCP tool "create_kek" with input:
      | name       | mcp-list-dek-kek                                      |
      | kms_type   | aws-kms                                                |
      | kms_key_id | arn:aws:kms:us-east-1:123456789:key/mcp-list-dek-kek  |
    When I call MCP tool "create_dek" with input:
      | kek_name | mcp-list-dek-kek |
      | subject  | mcp-dek-subj-a   |
    When I call MCP tool "create_dek" with input:
      | kek_name | mcp-list-dek-kek |
      | subject  | mcp-dek-subj-b   |
    When I call MCP tool "list_deks" with input:
      | kek_name | mcp-list-dek-kek |
    Then the MCP result should contain "mcp-dek-subj-a"
    Then the MCP result should contain "mcp-dek-subj-b"
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
      | path                 | list_deks              |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  Scenario: Delete DEK
    When I call MCP tool "create_kek" with input:
      | name       | mcp-del-dek-kek                                      |
      | kms_type   | aws-kms                                               |
      | kms_key_id | arn:aws:kms:us-east-1:123456789:key/mcp-del-dek-kek  |
    When I call MCP tool "create_dek" with input:
      | kek_name | mcp-del-dek-kek  |
      | subject  | mcp-del-dek-subj |
    When I call MCP tool "delete_dek" with input:
      | kek_name | mcp-del-dek-kek  |
      | subject  | mcp-del-dek-subj |
      | version  | 1                |
    Then the MCP result should contain "true"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | mcp-del-dek-subj       |
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
      | path                 | delete_dek             |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |
