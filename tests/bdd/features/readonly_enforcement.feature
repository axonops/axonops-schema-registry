@functional
Feature: READONLY Enforcement and Permanent Delete Restrictions
  READONLY and READONLY_OVERRIDE modes block data mutations (schema registration,
  subject deletion, version deletion) AND config changes (set/delete config).
  Mode changes are always allowed (otherwise you'd get stuck in READONLY).
  This matches Confluent Schema Registry behavior.

  Additionally, permanent delete of version "latest" should not be allowed —
  only explicit numeric version numbers can be permanently deleted.

  Background:
    Given the schema registry is running

  # ==========================================================================
  # READONLY MODE — BLOCKS CONFIG CHANGES (MATCHES CONFLUENT)
  # ==========================================================================

  Scenario: READONLY mode blocks setting subject config
    Given subject "ro-cfg-blocked" has schema:
      """
      {"type":"record","name":"ROCfg","fields":[{"name":"a","type":"string"}]}
      """
    When I set the global mode to "READONLY"
    And I PUT "/config/ro-cfg-blocked" with body:
      """
      {"compatibility": "NONE"}
      """
    Then the response status should be 422
    And the response should have error code 42205
    When I set the global mode to "READWRITE"

  Scenario: READONLY mode blocks deleting subject config
    Given subject "ro-cfgdel-blocked" has schema:
      """
      {"type":"record","name":"ROCfgDel","fields":[{"name":"a","type":"string"}]}
      """
    And I set the config for subject "ro-cfgdel-blocked" to "NONE"
    When I set the global mode to "READONLY"
    And I DELETE "/config/ro-cfgdel-blocked"
    Then the response status should be 422
    And the response should have error code 42205
    When I set the global mode to "READWRITE"

  Scenario: READONLY mode allows setting mode
    When I set the global mode to "READONLY"
    # Mode changes are always allowed (otherwise you'd be stuck)
    When I set the global mode to "READWRITE"
    Then the response status should be 200
    When I get the global mode
    Then the response field "mode" should be "READWRITE"
    And the audit log should contain an event:
      | event_type           | mode_update                    |
      | outcome              | success                        |
      | actor_id             |                                |
      | actor_type           | anonymous                      |
      | auth_method          |                                |
      | role                 |                                |
      | target_type          | mode                           |
      | target_id            | _global                        |
      | schema_id            |                                |
      | version              |                                |
      | schema_type          |                                |
      | before_hash          | *                              |
      | after_hash           | sha256:*                       |
      | context              | .                              |
      | transport_security   | tls                            |
      | source_ip            | *                              |
      | user_agent           | *                              |
      | method               | PUT                            |
      | path                 | /mode                          |
      | status_code          | 200                            |
      | reason               |                                |
      | error                |                                |
      | request_body         |                                |
      | metadata             |                                |
      | timestamp            | *                              |
      | duration_ms          | *                              |
      | request_id           | *                              |

  Scenario: READONLY mode allows deleting subject mode
    When I set the mode for subject "ro-modedel-allowed" to "IMPORT"
    And I set the global mode to "READONLY"
    And I delete the mode for subject "ro-modedel-allowed"
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    And the audit log should contain an event:
      | event_type           | mode_update                    |
      | outcome              | success                        |
      | actor_id             |                                |
      | actor_type           | anonymous                      |
      | auth_method          |                                |
      | role                 |                                |
      | target_type          | mode                           |
      | target_id            | _global                        |
      | schema_id            |                                |
      | version              |                                |
      | schema_type          |                                |
      | before_hash          | *                              |
      | after_hash           | sha256:*                       |
      | context              | .                              |
      | transport_security   | tls                            |
      | source_ip            | *                              |
      | user_agent           | *                              |
      | method               | PUT                            |
      | path                 | /mode                          |
      | status_code          | 200                            |
      | reason               |                                |
      | error                |                                |
      | request_body         |                                |
      | metadata             |                                |
      | timestamp            | *                              |
      | duration_ms          | *                              |
      | request_id           | *                              |

  # ==========================================================================
  # READONLY_OVERRIDE — ALSO BLOCKS CONFIG CHANGES
  # ==========================================================================

  Scenario: READONLY_OVERRIDE blocks setting subject config
    Given subject "override-cfg-blocked" has schema:
      """
      {"type":"record","name":"OverrideCfg","fields":[{"name":"a","type":"string"}]}
      """
    When I set the global mode to "READONLY_OVERRIDE"
    And I PUT "/config/override-cfg-blocked" with body:
      """
      {"compatibility": "NONE"}
      """
    Then the response status should be 422
    And the response should have error code 42205
    When I set the global mode to "READWRITE"

  Scenario: READONLY_OVERRIDE allows changing mode back to READWRITE
    When I set the global mode to "READONLY_OVERRIDE"
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    Then the response status should be 200
    When I get the global mode
    Then the response field "mode" should be "READWRITE"
    And the audit log should contain an event:
      | event_type           | mode_update                    |
      | outcome              | success                        |
      | actor_id             |                                |
      | actor_type           | anonymous                      |
      | auth_method          |                                |
      | role                 |                                |
      | target_type          | mode                           |
      | target_id            | _global                        |
      | schema_id            |                                |
      | version              |                                |
      | schema_type          |                                |
      | before_hash          | *                              |
      | after_hash           | sha256:*                       |
      | context              | .                              |
      | transport_security   | tls                            |
      | source_ip            | *                              |
      | user_agent           | *                              |
      | method               | PUT                            |
      | path                 | /mode                          |
      | status_code          | 200                            |
      | reason               |                                |
      | error                |                                |
      | request_body         |                                |
      | metadata             |                                |
      | timestamp            | *                              |
      | duration_ms          | *                              |
      | request_id           | *                              |

  # ==========================================================================
  # PERMANENT DELETE OF "LATEST" RESOLVES AND PROCEEDS
  # ==========================================================================

  @axonops-only
  Scenario: Permanent delete of version "latest" resolves to actual version
    Given subject "perm-del-latest" has schema:
      """
      {"type":"record","name":"PermDelLatest","fields":[{"name":"a","type":"string"}]}
      """
    # Soft-delete first (required before permanent delete)
    When I DELETE "/subjects/perm-del-latest/versions/1"
    Then the response status should be 200
    # Permanent delete of "latest" — Confluent resolves to actual version and proceeds
    When I DELETE "/subjects/perm-del-latest/versions/latest?permanent=true"
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_delete_permanent                  |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | perm-del-latest                          |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | AVRO                                     |
      | before_hash          | sha256:*                                 |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | DELETE                                   |
      | path                 | /subjects/perm-del-latest/versions       |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: Permanent delete with explicit version number works
    Given subject "perm-del-explicit" has schema:
      """
      {"type":"record","name":"PermDelExplicit","fields":[{"name":"a","type":"string"}]}
      """
    # Soft-delete first
    When I DELETE "/subjects/perm-del-explicit/versions/1"
    Then the response status should be 200
    # Permanent delete with explicit version — should succeed
    When I DELETE "/subjects/perm-del-explicit/versions/1?permanent=true"
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_delete_permanent                      |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | perm-del-explicit                            |
      | schema_id            | *                                            |
      | version              | *                                            |
      | schema_type          | AVRO                                         |
      | before_hash          | sha256:*                                     |
      | after_hash           |                                              |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | DELETE                                       |
      | path                 | /subjects/perm-del-explicit/versions/1       |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |
