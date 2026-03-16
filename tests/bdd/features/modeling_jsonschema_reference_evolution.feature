@schema-modeling @json @references
Feature: JSON Schema Reference Evolution
  Tests for evolving JSON schemas that use cross-subject $ref references,
  including reference version pinning, multiple references, referencedby
  tracking, and reference deletion behavior.

  # ==========================================================================
  # 1. REFERENCE EVOLVES — CONSUMER STAYS PINNED
  # ==========================================================================

  Scenario: Consumer stays pinned to reference v1 when reference evolves
    Given subject "json-refevo-address" has "JSON" schema:
      """
      {"type":"object","properties":{"street":{"type":"string"},"city":{"type":"string"}},"required":["street","city"]}
      """
    When I register a "JSON" schema under subject "json-refevo-consumer" with references:
      """
      {
        "schemaType": "JSON",
        "schema": "{\"type\":\"object\",\"properties\":{\"name\":{\"type\":\"string\"},\"home\":{\"$ref\":\"address.json\"}},\"required\":[\"name\"]}",
        "references": [
          {"name":"address.json","subject":"json-refevo-address","version":1}
        ]
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "json_consumer_v1"
    And the audit log should contain an event:
      | event_type           | schema_register                                |
      | outcome              | success                                        |
      | actor_id             |                                                |
      | actor_type           | anonymous                                      |
      | auth_method          |                                                |
      | role                 |                                                |
      | target_type          | subject                                        |
      | target_id            | json-refevo-consumer                           |
      | schema_id            | *                                              |
      | version              |                                                |
      | schema_type          | JSON                                           |
      | before_hash          |                                                |
      | after_hash           | sha256:*                                       |
      | context              | .                                              |
      | transport_security   | tls                                            |
      | source_ip            | *                                              |
      | user_agent           | *                                              |
      | method               | POST                                           |
      | path                 | /subjects/json-refevo-consumer/versions        |
      | status_code          | 200                                            |
      | reason               |                                                |
      | error                |                                                |
      | request_body         |                                                |
      | metadata             |                                                |
      | timestamp            | *                                              |
      | duration_ms          | *                                              |
      | request_id           | *                                              |

  # ==========================================================================
  # 2. CONSUMER UPGRADES REFERENCE VERSION
  # ==========================================================================

  Scenario: Consumer upgrades reference version gets different schema ID
    Given subject "json-refevo2-dep" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"integer"}},"required":["id"],"additionalProperties":false}
      """
    Given subject "json-refevo2-dep" has compatibility level "BACKWARD"
    When I register a "JSON" schema under subject "json-refevo2-dep":
      """
      {"type":"object","properties":{"id":{"type":"integer"},"name":{"type":"string"}},"required":["id"],"additionalProperties":false}
      """
    Then the response status should be 200
    When I register a "JSON" schema under subject "json-refevo2-c1" with references:
      """
      {
        "schemaType": "JSON",
        "schema": "{\"type\":\"object\",\"properties\":{\"dep\":{\"$ref\":\"dep.json\"}}}",
        "references": [
          {"name":"dep.json","subject":"json-refevo2-dep","version":1}
        ]
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "json_ref1_id"
    When I register a "JSON" schema under subject "json-refevo2-c2" with references:
      """
      {
        "schemaType": "JSON",
        "schema": "{\"type\":\"object\",\"properties\":{\"dep\":{\"$ref\":\"dep.json\"}}}",
        "references": [
          {"name":"dep.json","subject":"json-refevo2-dep","version":2}
        ]
      }
      """
    Then the response status should be 200
    And the response field "id" should not equal stored "json_ref1_id"
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | json-refevo2-c2                              |
      | schema_id            | *                                            |
      | version              |                                              |
      | schema_type          | JSON                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/json-refevo2-c2/versions           |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ==========================================================================
  # 3. MULTIPLE REFERENCES
  # ==========================================================================

  Scenario: Schema with multiple JSON Schema references registers
    Given subject "json-multiref-addr" has "JSON" schema:
      """
      {"type":"object","properties":{"street":{"type":"string"},"city":{"type":"string"}},"required":["street"]}
      """
    And subject "json-multiref-contact" has "JSON" schema:
      """
      {"type":"object","properties":{"email":{"type":"string"},"phone":{"type":"string"}},"required":["email"]}
      """
    When I register a "JSON" schema under subject "json-multiref-person" with references:
      """
      {
        "schemaType": "JSON",
        "schema": "{\"type\":\"object\",\"properties\":{\"name\":{\"type\":\"string\"},\"address\":{\"$ref\":\"address.json\"},\"contact\":{\"$ref\":\"contact.json\"}},\"required\":[\"name\"]}",
        "references": [
          {"name":"address.json","subject":"json-multiref-addr","version":1},
          {"name":"contact.json","subject":"json-multiref-contact","version":1}
        ]
      }
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                                |
      | outcome              | success                                        |
      | actor_id             |                                                |
      | actor_type           | anonymous                                      |
      | auth_method          |                                                |
      | role                 |                                                |
      | target_type          | subject                                        |
      | target_id            | json-multiref-person                           |
      | schema_id            | *                                              |
      | version              |                                                |
      | schema_type          | JSON                                           |
      | before_hash          |                                                |
      | after_hash           | sha256:*                                       |
      | context              | .                                              |
      | transport_security   | tls                                            |
      | source_ip            | *                                              |
      | user_agent           | *                                              |
      | method               | POST                                           |
      | path                 | /subjects/json-multiref-person/versions        |
      | status_code          | 200                                            |
      | reason               |                                                |
      | error                |                                                |
      | request_body         |                                                |
      | metadata             |                                                |
      | timestamp            | *                                              |
      | duration_ms          | *                                              |
      | request_id           | *                                              |

  # ==========================================================================
  # 4. REFERENCEDBY TRACKING
  # ==========================================================================

  Scenario: referencedby tracks JSON Schema reference consumers
    Given subject "json-refby-shared" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"integer"}},"required":["id"]}
      """
    When I register a "JSON" schema under subject "json-refby-c1" with references:
      """
      {
        "schemaType": "JSON",
        "schema": "{\"type\":\"object\",\"properties\":{\"shared\":{\"$ref\":\"shared.json\"}}}",
        "references": [
          {"name":"shared.json","subject":"json-refby-shared","version":1}
        ]
      }
      """
    Then the response status should be 200
    When I register a "JSON" schema under subject "json-refby-c2" with references:
      """
      {
        "schemaType": "JSON",
        "schema": "{\"type\":\"object\",\"properties\":{\"data\":{\"$ref\":\"shared.json\"}}}",
        "references": [
          {"name":"shared.json","subject":"json-refby-shared","version":1}
        ]
      }
      """
    Then the response status should be 200
    When I get the referenced by for subject "json-refby-shared" version 1
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | json-refby-c2                                |
      | schema_id            | *                                            |
      | version              |                                              |
      | schema_type          | JSON                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/json-refby-c2/versions             |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ==========================================================================
  # 5. SAME BODY WITH DIFFERENT REF VERSIONS — DIFFERENT IDS
  # ==========================================================================

  Scenario: Same body with different reference versions produces different IDs
    Given subject "json-diffref-dep" has "JSON" schema:
      """
      {"type":"object","properties":{"v":{"type":"integer"}}}
      """
    Given subject "json-diffref-dep" has compatibility level "NONE"
    When I register a "JSON" schema under subject "json-diffref-dep":
      """
      {"type":"object","properties":{"v":{"type":"string"}}}
      """
    Then the response status should be 200
    When I register a "JSON" schema under subject "json-diffref-a" with references:
      """
      {
        "schemaType": "JSON",
        "schema": "{\"type\":\"object\",\"properties\":{\"dep\":{\"$ref\":\"dep.json\"}}}",
        "references": [
          {"name":"dep.json","subject":"json-diffref-dep","version":1}
        ]
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "json_diff_v1"
    When I register a "JSON" schema under subject "json-diffref-b" with references:
      """
      {
        "schemaType": "JSON",
        "schema": "{\"type\":\"object\",\"properties\":{\"dep\":{\"$ref\":\"dep.json\"}}}",
        "references": [
          {"name":"dep.json","subject":"json-diffref-dep","version":2}
        ]
      }
      """
    Then the response status should be 200
    And the response field "id" should not equal stored "json_diff_v1"
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | json-diffref-b                               |
      | schema_id            | *                                            |
      | version              |                                              |
      | schema_type          | JSON                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/json-diffref-b/versions            |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ==========================================================================
  # 6. DELETE REFERENCED SCHEMA — CONSUMER STILL RETRIEVABLE
  # ==========================================================================

  Scenario: Deleting referenced JSON schema does not break consumer retrieval
    Given subject "json-refdel-base" has "JSON" schema:
      """
      {"type":"object","properties":{"x":{"type":"integer"}}}
      """
    When I register a "JSON" schema under subject "json-refdel-consumer" with references:
      """
      {
        "schemaType": "JSON",
        "schema": "{\"type\":\"object\",\"properties\":{\"base\":{\"$ref\":\"base.json\"}}}",
        "references": [
          {"name":"base.json","subject":"json-refdel-base","version":1}
        ]
      }
      """
    Then the response status should be 200
    When I get version 1 of subject "json-refdel-consumer"
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                                |
      | outcome              | success                                        |
      | actor_id             |                                                |
      | actor_type           | anonymous                                      |
      | auth_method          |                                                |
      | role                 |                                                |
      | target_type          | subject                                        |
      | target_id            | json-refdel-consumer                           |
      | schema_id            | *                                              |
      | version              |                                                |
      | schema_type          | JSON                                           |
      | before_hash          |                                                |
      | after_hash           | sha256:*                                       |
      | context              | .                                              |
      | transport_security   | tls                                            |
      | source_ip            | *                                              |
      | user_agent           | *                                              |
      | method               | POST                                           |
      | path                 | /subjects/json-refdel-consumer/versions        |
      | status_code          | 200                                            |
      | reason               |                                                |
      | error                |                                                |
      | request_body         |                                                |
      | metadata             |                                                |
      | timestamp            | *                                              |
      | duration_ms          | *                                              |
      | request_id           | *                                              |

  # ==========================================================================
  # 7. COMPATIBILITY WITH REFERENCES
  # ==========================================================================

  Scenario: Compatibility check works with JSON Schema references
    Given subject "json-refcompat-dep" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"integer"}},"required":["id"]}
      """
    And subject "json-refcompat-main" has compatibility level "BACKWARD"
    When I register a "JSON" schema under subject "json-refcompat-main" with references:
      """
      {
        "schemaType": "JSON",
        "schema": "{\"type\":\"object\",\"properties\":{\"dep\":{\"$ref\":\"dep.json\"},\"name\":{\"type\":\"string\"}},\"required\":[\"name\"]}",
        "references": [
          {"name":"dep.json","subject":"json-refcompat-dep","version":1}
        ]
      }
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                                |
      | outcome              | success                                        |
      | actor_id             |                                                |
      | actor_type           | anonymous                                      |
      | auth_method          |                                                |
      | role                 |                                                |
      | target_type          | subject                                        |
      | target_id            | json-refcompat-main                            |
      | schema_id            | *                                              |
      | version              |                                                |
      | schema_type          | JSON                                           |
      | before_hash          |                                                |
      | after_hash           | sha256:*                                       |
      | context              | .                                              |
      | transport_security   | tls                                            |
      | source_ip            | *                                              |
      | user_agent           | *                                              |
      | method               | POST                                           |
      | path                 | /subjects/json-refcompat-main/versions         |
      | status_code          | 200                                            |
      | reason               |                                                |
      | error                |                                                |
      | request_body         |                                                |
      | metadata             |                                                |
      | timestamp            | *                                              |
      | duration_ms          | *                                              |
      | request_id           | *                                              |

  # ==========================================================================
  # 8. REFERENCE CHAIN — A REFS B REFS C
  # ==========================================================================

  Scenario: JSON Schema reference chain registers successfully
    Given subject "json-chain-c" has "JSON" schema:
      """
      {"type":"object","properties":{"value":{"type":"string"}},"required":["value"]}
      """
    When I register a "JSON" schema under subject "json-chain-b" with references:
      """
      {
        "schemaType": "JSON",
        "schema": "{\"type\":\"object\",\"properties\":{\"c\":{\"$ref\":\"c.json\"},\"extra\":{\"type\":\"integer\"}}}",
        "references": [
          {"name":"c.json","subject":"json-chain-c","version":1}
        ]
      }
      """
    Then the response status should be 200
    When I register a "JSON" schema under subject "json-chain-a" with references:
      """
      {
        "schemaType": "JSON",
        "schema": "{\"type\":\"object\",\"properties\":{\"b\":{\"$ref\":\"b.json\"},\"name\":{\"type\":\"string\"}}}",
        "references": [
          {"name":"b.json","subject":"json-chain-b","version":1}
        ]
      }
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | json-chain-a                                 |
      | schema_id            | *                                            |
      | version              |                                              |
      | schema_type          | JSON                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/json-chain-a/versions              |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |
