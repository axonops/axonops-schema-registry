@functional
Feature: Compatibility Groups
  Compatibility groups allow multiple incompatible schema lineages within the
  same subject. The compatibilityGroup config property names a metadata property
  key; only schemas with the same value for that property are checked for
  compatibility against each other.

  Background:
    Given the schema registry is running

  # ==========================================================================
  # SET COMPATIBILITY GROUP CONFIG
  # ==========================================================================

  Scenario: Set compatibilityGroup via config
    When I PUT "/config/cg-subject" with body:
      """
      {"compatibility": "BACKWARD", "compatibilityGroup": "major_version"}
      """
    Then the response status should be 200
    And the response body should contain "major_version"
    When I GET "/config/cg-subject"
    Then the response status should be 200
    And the response field "compatibilityGroup" should be "major_version"
    And the audit log should contain an event:
      | event_type           | config_update                            |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | config                                   |
      | target_id            | cg-subject                               |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          |                                          |
      | before_hash          | *                                        |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | PUT                                      |
      | path                 | /config/cg-subject                       |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  # ==========================================================================
  # COMPATIBILITY GROUP FILTERING
  # ==========================================================================

  Scenario: Schemas in different compatibility groups bypass compatibility checks
    # Configure compatibility group
    When I PUT "/config/cg-bypass" with body:
      """
      {"compatibility": "BACKWARD", "compatibilityGroup": "major_version"}
      """
    Then the response status should be 200
    # Register v1 schema with major_version=1
    When I POST "/subjects/cg-bypass/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CgBypass\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}",
        "metadata": {"properties": {"major_version": "1"}}
      }
      """
    Then the response status should be 200
    # Register incompatible schema with major_version=2 — should succeed
    # (removing a field without default is normally backward-incompatible)
    When I POST "/subjects/cg-bypass/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CgBypass\",\"fields\":[{\"name\":\"x\",\"type\":\"int\"}]}",
        "metadata": {"properties": {"major_version": "2"}}
      }
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | cg-bypass                                |
      | schema_id            | *                                        |
      | version              |                                          |
      | schema_type          | AVRO                                     |
      | before_hash          |                                          |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/cg-bypass/versions             |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: Schemas in same compatibility group are checked for compatibility
    # Configure compatibility group
    When I PUT "/config/cg-same" with body:
      """
      {"compatibility": "BACKWARD", "compatibilityGroup": "major_version"}
      """
    Then the response status should be 200
    # Register v1 schema with major_version=1
    When I POST "/subjects/cg-same/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CgSame\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}",
        "metadata": {"properties": {"major_version": "1"}}
      }
      """
    Then the response status should be 200
    # Register incompatible schema with same major_version=1 — should fail
    When I POST "/subjects/cg-same/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CgSame\",\"fields\":[{\"name\":\"x\",\"type\":\"int\"}]}",
        "metadata": {"properties": {"major_version": "1"}}
      }
      """
    Then the response status should be 409

  # ==========================================================================
  # NO COMPATIBILITY GROUP — DEFAULT BEHAVIOR
  # ==========================================================================

  Scenario: Without compatibilityGroup all schemas are compared
    # Set BACKWARD compat without group
    When I PUT "/config/cg-default" with body:
      """
      {"compatibility": "BACKWARD"}
      """
    Then the response status should be 200
    # Register first schema
    When I POST "/subjects/cg-default/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CgDefault\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}",
        "metadata": {"properties": {"major_version": "1"}}
      }
      """
    Then the response status should be 200
    # Register incompatible schema — should fail even with different metadata
    When I POST "/subjects/cg-default/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CgDefault\",\"fields\":[{\"name\":\"x\",\"type\":\"int\"}]}",
        "metadata": {"properties": {"major_version": "2"}}
      }
      """
    Then the response status should be 409
