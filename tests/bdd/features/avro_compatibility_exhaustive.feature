@functional
Feature: Avro Compatibility — Exhaustive (Confluent v8.1.1 Compatibility)
  Comprehensive Avro compatibility tests from the Confluent Schema Registry v8.1.1
  test suite covering backward, forward, full, and transitive modes.

  # ==========================================================================
  # BACKWARD COMPATIBILITY (Section 22)
  # ==========================================================================

  Scenario: Backward — adding field with default is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-back-add-def" has compatibility level "BACKWARD"
    And subject "avro-ex-back-add-def" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    When I register a schema under subject "avro-ex-back-add-def":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"foo"}]}
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
      | target_id            | avro-ex-back-add-def                         |
      | schema_id            | *                                            |
      | version              | *                                            |
      | schema_type          | AVRO                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/avro-ex-back-add-def/versions      |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  Scenario: Backward — adding field without default is incompatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-back-add-nodef" has compatibility level "BACKWARD"
    And subject "avro-ex-back-add-nodef" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    When I register a schema under subject "avro-ex-back-add-nodef":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string"}]}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                                |
      | outcome              | failure                                        |
      | actor_id             |                                                |
      | actor_type           | anonymous                                      |
      | auth_method          |                                                |
      | role                 |                                                |
      | target_type          | subject                                        |
      | target_id            | avro-ex-back-add-nodef                         |
      | schema_id            |                                                |
      | version              | *                                              |
      | schema_type          | AVRO                                           |
      | before_hash          |                                                |
      | after_hash           |                                                |
      | context              | .                                              |
      | transport_security   | tls                                            |
      | source_ip            | *                                              |
      | user_agent           | *                                              |
      | method               | POST                                           |
      | path                 | /subjects/avro-ex-back-add-nodef/versions      |
      | status_code          | 409                                            |
      | reason               | already_exists                                 |
      | error                |                                                |
      | request_body         |                                                |
      | metadata             |                                                |
      | timestamp            | *                                              |
      | duration_ms          | *                                              |
      | request_id           | *                                              |

  Scenario: Backward — removing field is compatible (old reader ignores extra)
    Given the global compatibility level is "NONE"
    And subject "avro-ex-back-remove" has compatibility level "BACKWARD"
    And subject "avro-ex-back-remove" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"x"}]}
      """
    When I register a schema under subject "avro-ex-back-remove":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                             |
      | outcome              | success                                     |
      | actor_id             |                                             |
      | actor_type           | anonymous                                   |
      | auth_method          |                                             |
      | role                 |                                             |
      | target_type          | subject                                     |
      | target_id            | avro-ex-back-remove                         |
      | schema_id            | *                                           |
      | version              | *                                           |
      | schema_type          | AVRO                                        |
      | before_hash          |                                             |
      | after_hash           | sha256:*                                    |
      | context              | .                                           |
      | transport_security   | tls                                         |
      | source_ip            | *                                           |
      | user_agent           | *                                           |
      | method               | POST                                        |
      | path                 | /subjects/avro-ex-back-remove/versions      |
      | status_code          | 200                                         |
      | reason               |                                             |
      | error                |                                             |
      | request_body         |                                             |
      | metadata             |                                             |
      | timestamp            | *                                           |
      | duration_ms          | *                                           |
      | request_id           | *                                           |

  Scenario: Backward — changing field name with alias is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-back-alias" has compatibility level "BACKWARD"
    And subject "avro-ex-back-alias" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    When I register a schema under subject "avro-ex-back-alias":
      """
      {"type":"record","name":"R","fields":[{"name":"f1_new","type":"string","aliases":["f1"]}]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                            |
      | outcome              | success                                    |
      | actor_id             |                                            |
      | actor_type           | anonymous                                  |
      | auth_method          |                                            |
      | role                 |                                            |
      | target_type          | subject                                    |
      | target_id            | avro-ex-back-alias                         |
      | schema_id            | *                                          |
      | version              | *                                          |
      | schema_type          | AVRO                                       |
      | before_hash          |                                            |
      | after_hash           | sha256:*                                   |
      | context              | .                                          |
      | transport_security   | tls                                        |
      | source_ip            | *                                          |
      | user_agent           | *                                          |
      | method               | POST                                       |
      | path                 | /subjects/avro-ex-back-alias/versions      |
      | status_code          | 200                                        |
      | reason               |                                            |
      | error                |                                            |
      | request_body         |                                            |
      | metadata             |                                            |
      | timestamp            | *                                          |
      | duration_ms          | *                                          |
      | request_id           | *                                          |

  Scenario: Backward — evolving field type to union is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-back-to-union" has compatibility level "BACKWARD"
    And subject "avro-ex-back-to-union" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    When I register a schema under subject "avro-ex-back-to-union":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":["null","string"]}]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                               |
      | outcome              | success                                       |
      | actor_id             |                                               |
      | actor_type           | anonymous                                     |
      | auth_method          |                                               |
      | role                 |                                               |
      | target_type          | subject                                       |
      | target_id            | avro-ex-back-to-union                         |
      | schema_id            | *                                             |
      | version              | *                                             |
      | schema_type          | AVRO                                          |
      | before_hash          |                                               |
      | after_hash           | sha256:*                                      |
      | context              | .                                             |
      | transport_security   | tls                                           |
      | source_ip            | *                                             |
      | user_agent           | *                                             |
      | method               | POST                                          |
      | path                 | /subjects/avro-ex-back-to-union/versions      |
      | status_code          | 200                                           |
      | reason               |                                               |
      | error                |                                               |
      | request_body         |                                               |
      | metadata             |                                               |
      | timestamp            | *                                             |
      | duration_ms          | *                                             |
      | request_id           | *                                             |

  Scenario: Backward — removing type from union is incompatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-back-narrow-union" has compatibility level "BACKWARD"
    And subject "avro-ex-back-narrow-union" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":["null","string"]}]}
      """
    When I register a schema under subject "avro-ex-back-narrow-union":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                                   |
      | outcome              | failure                                           |
      | actor_id             |                                                   |
      | actor_type           | anonymous                                         |
      | auth_method          |                                                   |
      | role                 |                                                   |
      | target_type          | subject                                           |
      | target_id            | avro-ex-back-narrow-union                         |
      | schema_id            |                                                   |
      | version              | *                                                 |
      | schema_type          | AVRO                                              |
      | before_hash          |                                                   |
      | after_hash           |                                                   |
      | context              | .                                                 |
      | transport_security   | tls                                               |
      | source_ip            | *                                                 |
      | user_agent           | *                                                 |
      | method               | POST                                              |
      | path                 | /subjects/avro-ex-back-narrow-union/versions      |
      | status_code          | 409                                               |
      | reason               | already_exists                                    |
      | error                |                                                   |
      | request_body         |                                                   |
      | metadata             |                                                   |
      | timestamp            | *                                                 |
      | duration_ms          | *                                                 |
      | request_id           | *                                                 |

  Scenario: Backward — adding type to union is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-back-widen-union" has compatibility level "BACKWARD"
    And subject "avro-ex-back-widen-union" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":["null","string"]}]}
      """
    When I register a schema under subject "avro-ex-back-widen-union":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":["null","string","int"]}]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                                  |
      | outcome              | success                                          |
      | actor_id             |                                                  |
      | actor_type           | anonymous                                        |
      | auth_method          |                                                  |
      | role                 |                                                  |
      | target_type          | subject                                          |
      | target_id            | avro-ex-back-widen-union                         |
      | schema_id            | *                                                |
      | version              | *                                                |
      | schema_type          | AVRO                                             |
      | before_hash          |                                                  |
      | after_hash           | sha256:*                                         |
      | context              | .                                                |
      | transport_security   | tls                                              |
      | source_ip            | *                                                |
      | user_agent           | *                                                |
      | method               | POST                                             |
      | path                 | /subjects/avro-ex-back-widen-union/versions      |
      | status_code          | 200                                              |
      | reason               |                                                  |
      | error                |                                                  |
      | request_body         |                                                  |
      | metadata             |                                                  |
      | timestamp            | *                                                |
      | duration_ms          | *                                                |
      | request_id           | *                                                |

  Scenario: Backward — int to long promotion is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-back-int-long" has compatibility level "BACKWARD"
    And subject "avro-ex-back-int-long" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"int"}]}
      """
    When I register a schema under subject "avro-ex-back-int-long":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"long"}]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                               |
      | outcome              | success                                       |
      | actor_id             |                                               |
      | actor_type           | anonymous                                     |
      | auth_method          |                                               |
      | role                 |                                               |
      | target_type          | subject                                       |
      | target_id            | avro-ex-back-int-long                         |
      | schema_id            | *                                             |
      | version              | *                                             |
      | schema_type          | AVRO                                          |
      | before_hash          |                                               |
      | after_hash           | sha256:*                                      |
      | context              | .                                             |
      | transport_security   | tls                                           |
      | source_ip            | *                                             |
      | user_agent           | *                                             |
      | method               | POST                                          |
      | path                 | /subjects/avro-ex-back-int-long/versions      |
      | status_code          | 200                                           |
      | reason               |                                               |
      | error                |                                               |
      | request_body         |                                               |
      | metadata             |                                               |
      | timestamp            | *                                             |
      | duration_ms          | *                                             |
      | request_id           | *                                             |

  Scenario: Backward — int to float promotion is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-back-int-float" has compatibility level "BACKWARD"
    And subject "avro-ex-back-int-float" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"int"}]}
      """
    When I register a schema under subject "avro-ex-back-int-float":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"float"}]}
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
      | target_id            | avro-ex-back-int-float                         |
      | schema_id            | *                                              |
      | version              | *                                              |
      | schema_type          | AVRO                                           |
      | before_hash          |                                                |
      | after_hash           | sha256:*                                       |
      | context              | .                                              |
      | transport_security   | tls                                            |
      | source_ip            | *                                              |
      | user_agent           | *                                              |
      | method               | POST                                           |
      | path                 | /subjects/avro-ex-back-int-float/versions      |
      | status_code          | 200                                            |
      | reason               |                                                |
      | error                |                                                |
      | request_body         |                                                |
      | metadata             |                                                |
      | timestamp            | *                                              |
      | duration_ms          | *                                              |
      | request_id           | *                                              |

  Scenario: Backward — int to double promotion is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-back-int-double" has compatibility level "BACKWARD"
    And subject "avro-ex-back-int-double" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"int"}]}
      """
    When I register a schema under subject "avro-ex-back-int-double":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"double"}]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                                 |
      | outcome              | success                                         |
      | actor_id             |                                                 |
      | actor_type           | anonymous                                       |
      | auth_method          |                                                 |
      | role                 |                                                 |
      | target_type          | subject                                         |
      | target_id            | avro-ex-back-int-double                         |
      | schema_id            | *                                               |
      | version              | *                                               |
      | schema_type          | AVRO                                            |
      | before_hash          |                                                 |
      | after_hash           | sha256:*                                        |
      | context              | .                                               |
      | transport_security   | tls                                             |
      | source_ip            | *                                               |
      | user_agent           | *                                               |
      | method               | POST                                            |
      | path                 | /subjects/avro-ex-back-int-double/versions      |
      | status_code          | 200                                             |
      | reason               |                                                 |
      | error                |                                                 |
      | request_body         |                                                 |
      | metadata             |                                                 |
      | timestamp            | *                                               |
      | duration_ms          | *                                               |
      | request_id           | *                                               |

  Scenario: Backward — long to float promotion is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-back-long-float" has compatibility level "BACKWARD"
    And subject "avro-ex-back-long-float" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"long"}]}
      """
    When I register a schema under subject "avro-ex-back-long-float":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"float"}]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                                 |
      | outcome              | success                                         |
      | actor_id             |                                                 |
      | actor_type           | anonymous                                       |
      | auth_method          |                                                 |
      | role                 |                                                 |
      | target_type          | subject                                         |
      | target_id            | avro-ex-back-long-float                         |
      | schema_id            | *                                               |
      | version              | *                                               |
      | schema_type          | AVRO                                            |
      | before_hash          |                                                 |
      | after_hash           | sha256:*                                        |
      | context              | .                                               |
      | transport_security   | tls                                             |
      | source_ip            | *                                               |
      | user_agent           | *                                               |
      | method               | POST                                            |
      | path                 | /subjects/avro-ex-back-long-float/versions      |
      | status_code          | 200                                             |
      | reason               |                                                 |
      | error                |                                                 |
      | request_body         |                                                 |
      | metadata             |                                                 |
      | timestamp            | *                                               |
      | duration_ms          | *                                               |
      | request_id           | *                                               |

  Scenario: Backward — long to double promotion is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-back-long-double" has compatibility level "BACKWARD"
    And subject "avro-ex-back-long-double" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"long"}]}
      """
    When I register a schema under subject "avro-ex-back-long-double":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"double"}]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                                  |
      | outcome              | success                                          |
      | actor_id             |                                                  |
      | actor_type           | anonymous                                        |
      | auth_method          |                                                  |
      | role                 |                                                  |
      | target_type          | subject                                          |
      | target_id            | avro-ex-back-long-double                         |
      | schema_id            | *                                                |
      | version              | *                                                |
      | schema_type          | AVRO                                             |
      | before_hash          |                                                  |
      | after_hash           | sha256:*                                         |
      | context              | .                                                |
      | transport_security   | tls                                              |
      | source_ip            | *                                                |
      | user_agent           | *                                                |
      | method               | POST                                             |
      | path                 | /subjects/avro-ex-back-long-double/versions      |
      | status_code          | 200                                              |
      | reason               |                                                  |
      | error                |                                                  |
      | request_body         |                                                  |
      | metadata             |                                                  |
      | timestamp            | *                                                |
      | duration_ms          | *                                                |
      | request_id           | *                                                |

  Scenario: Backward — float to double promotion is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-back-float-double" has compatibility level "BACKWARD"
    And subject "avro-ex-back-float-double" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"float"}]}
      """
    When I register a schema under subject "avro-ex-back-float-double":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"double"}]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                                   |
      | outcome              | success                                           |
      | actor_id             |                                                   |
      | actor_type           | anonymous                                         |
      | auth_method          |                                                   |
      | role                 |                                                   |
      | target_type          | subject                                           |
      | target_id            | avro-ex-back-float-double                         |
      | schema_id            | *                                                 |
      | version              | *                                                 |
      | schema_type          | AVRO                                              |
      | before_hash          |                                                   |
      | after_hash           | sha256:*                                          |
      | context              | .                                                 |
      | transport_security   | tls                                               |
      | source_ip            | *                                                 |
      | user_agent           | *                                                 |
      | method               | POST                                              |
      | path                 | /subjects/avro-ex-back-float-double/versions      |
      | status_code          | 200                                               |
      | reason               |                                                   |
      | error                |                                                   |
      | request_body         |                                                   |
      | metadata             |                                                   |
      | timestamp            | *                                                 |
      | duration_ms          | *                                                 |
      | request_id           | *                                                 |

  Scenario: Backward — string to bytes promotion is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-back-str-bytes" has compatibility level "BACKWARD"
    And subject "avro-ex-back-str-bytes" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    When I register a schema under subject "avro-ex-back-str-bytes":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"bytes"}]}
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
      | target_id            | avro-ex-back-str-bytes                         |
      | schema_id            | *                                              |
      | version              | *                                              |
      | schema_type          | AVRO                                           |
      | before_hash          |                                                |
      | after_hash           | sha256:*                                       |
      | context              | .                                              |
      | transport_security   | tls                                            |
      | source_ip            | *                                              |
      | user_agent           | *                                              |
      | method               | POST                                           |
      | path                 | /subjects/avro-ex-back-str-bytes/versions      |
      | status_code          | 200                                            |
      | reason               |                                                |
      | error                |                                                |
      | request_body         |                                                |
      | metadata             |                                                |
      | timestamp            | *                                              |
      | duration_ms          | *                                              |
      | request_id           | *                                              |

  Scenario: Backward — bytes to string promotion is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-back-bytes-str" has compatibility level "BACKWARD"
    And subject "avro-ex-back-bytes-str" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"bytes"}]}
      """
    When I register a schema under subject "avro-ex-back-bytes-str":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
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
      | target_id            | avro-ex-back-bytes-str                         |
      | schema_id            | *                                              |
      | version              | *                                              |
      | schema_type          | AVRO                                           |
      | before_hash          |                                                |
      | after_hash           | sha256:*                                       |
      | context              | .                                              |
      | transport_security   | tls                                            |
      | source_ip            | *                                              |
      | user_agent           | *                                              |
      | method               | POST                                           |
      | path                 | /subjects/avro-ex-back-bytes-str/versions      |
      | status_code          | 200                                            |
      | reason               |                                                |
      | error                |                                                |
      | request_body         |                                                |
      | metadata             |                                                |
      | timestamp            | *                                              |
      | duration_ms          | *                                              |
      | request_id           | *                                              |

  Scenario: Backward — changing field type incompatibly is rejected
    Given the global compatibility level is "NONE"
    And subject "avro-ex-back-type-change" has compatibility level "BACKWARD"
    And subject "avro-ex-back-type-change" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    When I register a schema under subject "avro-ex-back-type-change":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"int"}]}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                                   |
      | outcome              | failure                                           |
      | actor_id             |                                                   |
      | actor_type           | anonymous                                         |
      | auth_method          |                                                   |
      | role                 |                                                   |
      | target_type          | subject                                           |
      | target_id            | avro-ex-back-type-change                          |
      | schema_id            |                                                   |
      | version              | *                                                 |
      | schema_type          | AVRO                                              |
      | before_hash          |                                                   |
      | after_hash           |                                                   |
      | context              | .                                                 |
      | transport_security   | tls                                               |
      | source_ip            | *                                                 |
      | user_agent           | *                                                 |
      | method               | POST                                              |
      | path                 | /subjects/avro-ex-back-type-change/versions       |
      | status_code          | 409                                               |
      | reason               | already_exists                                    |
      | error                |                                                   |
      | request_body         |                                                   |
      | metadata             |                                                   |
      | timestamp            | *                                                 |
      | duration_ms          | *                                                 |
      | request_id           | *                                                 |

  Scenario: Backward — changing record name is incompatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-back-rename" has compatibility level "BACKWARD"
    And subject "avro-ex-back-rename" has schema:
      """
      {"type":"record","name":"Original","fields":[{"name":"f1","type":"string"}]}
      """
    When I register a schema under subject "avro-ex-back-rename":
      """
      {"type":"record","name":"Renamed","fields":[{"name":"f1","type":"string"}]}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | failure                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | avro-ex-back-rename                          |
      | schema_id            |                                              |
      | version              | *                                            |
      | schema_type          | AVRO                                         |
      | before_hash          |                                              |
      | after_hash           |                                              |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/avro-ex-back-rename/versions       |
      | status_code          | 409                                          |
      | reason               | already_exists                               |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  Scenario: Backward — adding enum symbol is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-back-enum-add" has compatibility level "BACKWARD"
    And subject "avro-ex-back-enum-add" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"e","type":{"type":"enum","name":"E","symbols":["A","B"]}}]}
      """
    When I register a schema under subject "avro-ex-back-enum-add":
      """
      {"type":"record","name":"R","fields":[{"name":"e","type":{"type":"enum","name":"E","symbols":["A","B","C"]}}]}
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
      | target_id            | avro-ex-back-enum-add                          |
      | schema_id            | *                                              |
      | version              | *                                              |
      | schema_type          | AVRO                                           |
      | before_hash          |                                                |
      | after_hash           | sha256:*                                       |
      | context              | .                                              |
      | transport_security   | tls                                            |
      | source_ip            | *                                              |
      | user_agent           | *                                              |
      | method               | POST                                           |
      | path                 | /subjects/avro-ex-back-enum-add/versions       |
      | status_code          | 200                                            |
      | reason               |                                                |
      | error                |                                                |
      | request_body         |                                                |
      | metadata             |                                                |
      | timestamp            | *                                              |
      | duration_ms          | *                                              |
      | request_id           | *                                              |

  Scenario: Backward — removing enum symbol is incompatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-back-enum-remove" has compatibility level "BACKWARD"
    And subject "avro-ex-back-enum-remove" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"e","type":{"type":"enum","name":"E","symbols":["A","B","C"]}}]}
      """
    When I register a schema under subject "avro-ex-back-enum-remove":
      """
      {"type":"record","name":"R","fields":[{"name":"e","type":{"type":"enum","name":"E","symbols":["A","B"]}}]}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                                   |
      | outcome              | failure                                           |
      | actor_id             |                                                   |
      | actor_type           | anonymous                                         |
      | auth_method          |                                                   |
      | role                 |                                                   |
      | target_type          | subject                                           |
      | target_id            | avro-ex-back-enum-remove                          |
      | schema_id            |                                                   |
      | version              | *                                                 |
      | schema_type          | AVRO                                              |
      | before_hash          |                                                   |
      | after_hash           |                                                   |
      | context              | .                                                 |
      | transport_security   | tls                                               |
      | source_ip            | *                                                 |
      | user_agent           | *                                                 |
      | method               | POST                                              |
      | path                 | /subjects/avro-ex-back-enum-remove/versions       |
      | status_code          | 409                                               |
      | reason               | already_exists                                    |
      | error                |                                                   |
      | request_body         |                                                   |
      | metadata             |                                                   |
      | timestamp            | *                                                 |
      | duration_ms          | *                                                 |
      | request_id           | *                                                 |

  # ==========================================================================
  # FORWARD COMPATIBILITY (Section 23)
  # ==========================================================================

  Scenario: Forward — adding field with default is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-fwd-add-def" has compatibility level "FORWARD"
    And subject "avro-ex-fwd-add-def" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    When I register a schema under subject "avro-ex-fwd-add-def":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"x"}]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                             |
      | outcome              | success                                     |
      | actor_id             |                                             |
      | actor_type           | anonymous                                   |
      | auth_method          |                                             |
      | role                 |                                             |
      | target_type          | subject                                     |
      | target_id            | avro-ex-fwd-add-def                         |
      | schema_id            | *                                           |
      | version              | *                                           |
      | schema_type          | AVRO                                        |
      | before_hash          |                                             |
      | after_hash           | sha256:*                                    |
      | context              | .                                           |
      | transport_security   | tls                                         |
      | source_ip            | *                                           |
      | user_agent           | *                                           |
      | method               | POST                                        |
      | path                 | /subjects/avro-ex-fwd-add-def/versions      |
      | status_code          | 200                                         |
      | reason               |                                             |
      | error                |                                             |
      | request_body         |                                             |
      | metadata             |                                             |
      | timestamp            | *                                           |
      | duration_ms          | *                                           |
      | request_id           | *                                           |

  Scenario: Forward — adding field without default is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-fwd-add-nodef" has compatibility level "FORWARD"
    And subject "avro-ex-fwd-add-nodef" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    When I register a schema under subject "avro-ex-fwd-add-nodef":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string"}]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                               |
      | outcome              | success                                       |
      | actor_id             |                                               |
      | actor_type           | anonymous                                     |
      | auth_method          |                                               |
      | role                 |                                               |
      | target_type          | subject                                       |
      | target_id            | avro-ex-fwd-add-nodef                         |
      | schema_id            | *                                             |
      | version              | *                                             |
      | schema_type          | AVRO                                          |
      | before_hash          |                                               |
      | after_hash           | sha256:*                                      |
      | context              | .                                             |
      | transport_security   | tls                                           |
      | source_ip            | *                                             |
      | user_agent           | *                                             |
      | method               | POST                                          |
      | path                 | /subjects/avro-ex-fwd-add-nodef/versions      |
      | status_code          | 200                                           |
      | reason               |                                               |
      | error                |                                               |
      | request_body         |                                               |
      | metadata             |                                               |
      | timestamp            | *                                             |
      | duration_ms          | *                                             |
      | request_id           | *                                             |

  Scenario: Forward — removing field with default is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-fwd-remove-def" has compatibility level "FORWARD"
    And subject "avro-ex-fwd-remove-def" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"x"}]}
      """
    When I register a schema under subject "avro-ex-fwd-remove-def":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                                 |
      | outcome              | success                                         |
      | actor_id             |                                                 |
      | actor_type           | anonymous                                       |
      | auth_method          |                                                 |
      | role                 |                                                 |
      | target_type          | subject                                         |
      | target_id            | avro-ex-fwd-remove-def                          |
      | schema_id            | *                                               |
      | version              | *                                               |
      | schema_type          | AVRO                                            |
      | before_hash          |                                                 |
      | after_hash           | sha256:*                                        |
      | context              | .                                               |
      | transport_security   | tls                                             |
      | source_ip            | *                                               |
      | user_agent           | *                                               |
      | method               | POST                                            |
      | path                 | /subjects/avro-ex-fwd-remove-def/versions       |
      | status_code          | 200                                             |
      | reason               |                                                 |
      | error                |                                                 |
      | request_body         |                                                 |
      | metadata             |                                                 |
      | timestamp            | *                                               |
      | duration_ms          | *                                               |
      | request_id           | *                                               |

  Scenario: Forward — removing field without default is incompatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-fwd-remove-nodef" has compatibility level "FORWARD"
    And subject "avro-ex-fwd-remove-nodef" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string"}]}
      """
    When I register a schema under subject "avro-ex-fwd-remove-nodef":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                                    |
      | outcome              | failure                                            |
      | actor_id             |                                                    |
      | actor_type           | anonymous                                          |
      | auth_method          |                                                    |
      | role                 |                                                    |
      | target_type          | subject                                            |
      | target_id            | avro-ex-fwd-remove-nodef                           |
      | schema_id            |                                                    |
      | version              | *                                                  |
      | schema_type          | AVRO                                               |
      | before_hash          |                                                    |
      | after_hash           |                                                    |
      | context              | .                                                  |
      | transport_security   | tls                                                |
      | source_ip            | *                                                  |
      | user_agent           | *                                                  |
      | method               | POST                                               |
      | path                 | /subjects/avro-ex-fwd-remove-nodef/versions        |
      | status_code          | 409                                                |
      | reason               | already_exists                                     |
      | error                |                                                    |
      | request_body         |                                                    |
      | metadata             |                                                    |
      | timestamp            | *                                                  |
      | duration_ms          | *                                                  |
      | request_id           | *                                                  |

  Scenario: Forward — int to long is incompatible (old reader can't read long)
    Given the global compatibility level is "NONE"
    And subject "avro-ex-fwd-int-long" has compatibility level "FORWARD"
    And subject "avro-ex-fwd-int-long" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"int"}]}
      """
    When I register a schema under subject "avro-ex-fwd-int-long":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"long"}]}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                               |
      | outcome              | failure                                       |
      | actor_id             |                                               |
      | actor_type           | anonymous                                     |
      | auth_method          |                                               |
      | role                 |                                               |
      | target_type          | subject                                       |
      | target_id            | avro-ex-fwd-int-long                          |
      | schema_id            |                                               |
      | version              | *                                             |
      | schema_type          | AVRO                                          |
      | before_hash          |                                               |
      | after_hash           |                                               |
      | context              | .                                             |
      | transport_security   | tls                                           |
      | source_ip            | *                                             |
      | user_agent           | *                                             |
      | method               | POST                                          |
      | path                 | /subjects/avro-ex-fwd-int-long/versions       |
      | status_code          | 409                                           |
      | reason               | already_exists                                |
      | error                |                                               |
      | request_body         |                                               |
      | metadata             |                                               |
      | timestamp            | *                                             |
      | duration_ms          | *                                             |
      | request_id           | *                                             |

  Scenario: Forward — long to int is compatible (old reader promotes int to long)
    Given the global compatibility level is "NONE"
    And subject "avro-ex-fwd-long-int" has compatibility level "FORWARD"
    And subject "avro-ex-fwd-long-int" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"long"}]}
      """
    When I register a schema under subject "avro-ex-fwd-long-int":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"int"}]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                               |
      | outcome              | success                                       |
      | actor_id             |                                               |
      | actor_type           | anonymous                                     |
      | auth_method          |                                               |
      | role                 |                                               |
      | target_type          | subject                                       |
      | target_id            | avro-ex-fwd-long-int                          |
      | schema_id            | *                                             |
      | version              | *                                             |
      | schema_type          | AVRO                                          |
      | before_hash          |                                               |
      | after_hash           | sha256:*                                      |
      | context              | .                                             |
      | transport_security   | tls                                           |
      | source_ip            | *                                             |
      | user_agent           | *                                             |
      | method               | POST                                          |
      | path                 | /subjects/avro-ex-fwd-long-int/versions       |
      | status_code          | 200                                           |
      | reason               |                                               |
      | error                |                                               |
      | request_body         |                                               |
      | metadata             |                                               |
      | timestamp            | *                                             |
      | duration_ms          | *                                             |
      | request_id           | *                                             |

  # ==========================================================================
  # FULL COMPATIBILITY (Section 24)
  # ==========================================================================

  Scenario: Full — adding field with default is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-full-add-def" has compatibility level "FULL"
    And subject "avro-ex-full-add-def" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    When I register a schema under subject "avro-ex-full-add-def":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"x"}]}
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
      | target_id            | avro-ex-full-add-def                         |
      | schema_id            | *                                            |
      | version              | *                                            |
      | schema_type          | AVRO                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/avro-ex-full-add-def/versions      |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  Scenario: Full — adding field without default is incompatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-full-add-nodef" has compatibility level "FULL"
    And subject "avro-ex-full-add-nodef" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    When I register a schema under subject "avro-ex-full-add-nodef":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string"}]}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                                |
      | outcome              | failure                                        |
      | actor_id             |                                                |
      | actor_type           | anonymous                                      |
      | auth_method          |                                                |
      | role                 |                                                |
      | target_type          | subject                                        |
      | target_id            | avro-ex-full-add-nodef                         |
      | schema_id            |                                                |
      | version              | *                                              |
      | schema_type          | AVRO                                           |
      | before_hash          |                                                |
      | after_hash           |                                                |
      | context              | .                                              |
      | transport_security   | tls                                            |
      | source_ip            | *                                              |
      | user_agent           | *                                              |
      | method               | POST                                           |
      | path                 | /subjects/avro-ex-full-add-nodef/versions      |
      | status_code          | 409                                            |
      | reason               | already_exists                                 |
      | error                |                                                |
      | request_body         |                                                |
      | metadata             |                                                |
      | timestamp            | *                                              |
      | duration_ms          | *                                              |
      | request_id           | *                                              |

  Scenario: Full — removing field with default is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-full-remove-def" has compatibility level "FULL"
    And subject "avro-ex-full-remove-def" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"x"}]}
      """
    When I register a schema under subject "avro-ex-full-remove-def":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                                 |
      | outcome              | success                                         |
      | actor_id             |                                                 |
      | actor_type           | anonymous                                       |
      | auth_method          |                                                 |
      | role                 |                                                 |
      | target_type          | subject                                         |
      | target_id            | avro-ex-full-remove-def                         |
      | schema_id            | *                                               |
      | version              | *                                               |
      | schema_type          | AVRO                                            |
      | before_hash          |                                                 |
      | after_hash           | sha256:*                                        |
      | context              | .                                               |
      | transport_security   | tls                                             |
      | source_ip            | *                                               |
      | user_agent           | *                                               |
      | method               | POST                                            |
      | path                 | /subjects/avro-ex-full-remove-def/versions      |
      | status_code          | 200                                             |
      | reason               |                                                 |
      | error                |                                                 |
      | request_body         |                                                 |
      | metadata             |                                                 |
      | timestamp            | *                                               |
      | duration_ms          | *                                               |
      | request_id           | *                                               |

  Scenario: Full — removing field without default is incompatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-full-remove-nodef" has compatibility level "FULL"
    And subject "avro-ex-full-remove-nodef" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string"}]}
      """
    When I register a schema under subject "avro-ex-full-remove-nodef":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                                    |
      | outcome              | failure                                            |
      | actor_id             |                                                    |
      | actor_type           | anonymous                                          |
      | auth_method          |                                                    |
      | role                 |                                                    |
      | target_type          | subject                                            |
      | target_id            | avro-ex-full-remove-nodef                          |
      | schema_id            |                                                    |
      | version              | *                                                  |
      | schema_type          | AVRO                                               |
      | before_hash          |                                                    |
      | after_hash           |                                                    |
      | context              | .                                                  |
      | transport_security   | tls                                                |
      | source_ip            | *                                                  |
      | user_agent           | *                                                  |
      | method               | POST                                               |
      | path                 | /subjects/avro-ex-full-remove-nodef/versions       |
      | status_code          | 409                                                |
      | reason               | already_exists                                     |
      | error                |                                                    |
      | request_body         |                                                    |
      | metadata             |                                                    |
      | timestamp            | *                                                  |
      | duration_ms          | *                                                  |
      | request_id           | *                                                  |

  Scenario: Full — string/bytes bidirectional promotion is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-full-str-bytes" has compatibility level "FULL"
    And subject "avro-ex-full-str-bytes" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    When I register a schema under subject "avro-ex-full-str-bytes":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"bytes"}]}
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
      | target_id            | avro-ex-full-str-bytes                         |
      | schema_id            | *                                              |
      | version              | *                                              |
      | schema_type          | AVRO                                           |
      | before_hash          |                                                |
      | after_hash           | sha256:*                                       |
      | context              | .                                              |
      | transport_security   | tls                                            |
      | source_ip            | *                                              |
      | user_agent           | *                                              |
      | method               | POST                                           |
      | path                 | /subjects/avro-ex-full-str-bytes/versions      |
      | status_code          | 200                                            |
      | reason               |                                                |
      | error                |                                                |
      | request_body         |                                                |
      | metadata             |                                                |
      | timestamp            | *                                              |
      | duration_ms          | *                                              |
      | request_id           | *                                              |

  Scenario: Full — int to long is incompatible (only forward-compatible)
    Given the global compatibility level is "NONE"
    And subject "avro-ex-full-int-long" has compatibility level "FULL"
    And subject "avro-ex-full-int-long" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"int"}]}
      """
    When I register a schema under subject "avro-ex-full-int-long":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"long"}]}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                               |
      | outcome              | failure                                       |
      | actor_id             |                                               |
      | actor_type           | anonymous                                     |
      | auth_method          |                                               |
      | role                 |                                               |
      | target_type          | subject                                       |
      | target_id            | avro-ex-full-int-long                         |
      | schema_id            |                                               |
      | version              | *                                             |
      | schema_type          | AVRO                                          |
      | before_hash          |                                               |
      | after_hash           |                                               |
      | context              | .                                             |
      | transport_security   | tls                                           |
      | source_ip            | *                                             |
      | user_agent           | *                                             |
      | method               | POST                                          |
      | path                 | /subjects/avro-ex-full-int-long/versions      |
      | status_code          | 409                                           |
      | reason               | already_exists                                |
      | error                |                                               |
      | request_body         |                                               |
      | metadata             |                                               |
      | timestamp            | *                                             |
      | duration_ms          | *                                             |
      | request_id           | *                                             |

  # ==========================================================================
  # TRANSITIVE COMPATIBILITY (Section 25)
  # ==========================================================================

  Scenario: BACKWARD_TRANSITIVE — progressive field addition is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-bt-add" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    And subject "avro-ex-bt-add" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"a"}]}
      """
    When I set the config for subject "avro-ex-bt-add" to "BACKWARD_TRANSITIVE"
    And I register a schema under subject "avro-ex-bt-add":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"a"},{"name":"f3","type":"string","default":"b"}]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                         |
      | outcome              | success                                 |
      | actor_id             |                                         |
      | actor_type           | anonymous                               |
      | auth_method          |                                         |
      | role                 |                                         |
      | target_type          | subject                                 |
      | target_id            | avro-ex-bt-add                          |
      | schema_id            | *                                       |
      | version              | *                                       |
      | schema_type          | AVRO                                    |
      | before_hash          |                                         |
      | after_hash           | sha256:*                                |
      | context              | .                                       |
      | transport_security   | tls                                     |
      | source_ip            | *                                       |
      | user_agent           | *                                       |
      | method               | POST                                    |
      | path                 | /subjects/avro-ex-bt-add/versions       |
      | status_code          | 200                                     |
      | reason               |                                         |
      | error                |                                         |
      | request_body         |                                         |
      | metadata             |                                         |
      | timestamp            | *                                       |
      | duration_ms          | *                                       |
      | request_id           | *                                       |

  Scenario: BACKWARD_TRANSITIVE — removing default transitively is incompatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-bt-nodef" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    And subject "avro-ex-bt-nodef" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"a"}]}
      """
    When I set the config for subject "avro-ex-bt-nodef" to "BACKWARD_TRANSITIVE"
    And I register a schema under subject "avro-ex-bt-nodef":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string"}]}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                           |
      | outcome              | failure                                   |
      | actor_id             |                                           |
      | actor_type           | anonymous                                 |
      | auth_method          |                                           |
      | role                 |                                           |
      | target_type          | subject                                   |
      | target_id            | avro-ex-bt-nodef                          |
      | schema_id            |                                           |
      | version              | *                                         |
      | schema_type          | AVRO                                      |
      | before_hash          |                                           |
      | after_hash           |                                           |
      | context              | .                                         |
      | transport_security   | tls                                       |
      | source_ip            | *                                         |
      | user_agent           | *                                         |
      | method               | POST                                      |
      | path                 | /subjects/avro-ex-bt-nodef/versions       |
      | status_code          | 409                                       |
      | reason               | already_exists                            |
      | error                |                                           |
      | request_body         |                                           |
      | metadata             |                                           |
      | timestamp            | *                                         |
      | duration_ms          | *                                         |
      | request_id           | *                                         |

  Scenario: FORWARD_TRANSITIVE — progressive field removal is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-ft-remove" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"a"},{"name":"f3","type":"string","default":"b"}]}
      """
    And subject "avro-ex-ft-remove" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"a"}]}
      """
    When I set the config for subject "avro-ex-ft-remove" to "FORWARD_TRANSITIVE"
    And I register a schema under subject "avro-ex-ft-remove":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                            |
      | outcome              | success                                    |
      | actor_id             |                                            |
      | actor_type           | anonymous                                  |
      | auth_method          |                                            |
      | role                 |                                            |
      | target_type          | subject                                    |
      | target_id            | avro-ex-ft-remove                          |
      | schema_id            | *                                          |
      | version              | *                                          |
      | schema_type          | AVRO                                       |
      | before_hash          |                                            |
      | after_hash           | sha256:*                                   |
      | context              | .                                          |
      | transport_security   | tls                                        |
      | source_ip            | *                                          |
      | user_agent           | *                                          |
      | method               | POST                                       |
      | path                 | /subjects/avro-ex-ft-remove/versions       |
      | status_code          | 200                                        |
      | reason               |                                            |
      | error                |                                            |
      | request_body         |                                            |
      | metadata             |                                            |
      | timestamp            | *                                          |
      | duration_ms          | *                                          |
      | request_id           | *                                          |

  Scenario: FORWARD_TRANSITIVE — adding field without default is compatible (old readers ignore new fields)
    Given the global compatibility level is "NONE"
    And subject "avro-ex-ft-add-nodef" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    And subject "avro-ex-ft-add-nodef" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"a"}]}
      """
    When I set the config for subject "avro-ex-ft-add-nodef" to "FORWARD_TRANSITIVE"
    And I register a schema under subject "avro-ex-ft-add-nodef":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f3","type":"string"}]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                               |
      | outcome              | success                                       |
      | actor_id             |                                               |
      | actor_type           | anonymous                                     |
      | auth_method          |                                               |
      | role                 |                                               |
      | target_type          | subject                                       |
      | target_id            | avro-ex-ft-add-nodef                          |
      | schema_id            | *                                             |
      | version              | *                                             |
      | schema_type          | AVRO                                          |
      | before_hash          |                                               |
      | after_hash           | sha256:*                                      |
      | context              | .                                             |
      | transport_security   | tls                                           |
      | source_ip            | *                                             |
      | user_agent           | *                                             |
      | method               | POST                                          |
      | path                 | /subjects/avro-ex-ft-add-nodef/versions       |
      | status_code          | 200                                           |
      | reason               |                                               |
      | error                |                                               |
      | request_body         |                                               |
      | metadata             |                                               |
      | timestamp            | *                                             |
      | duration_ms          | *                                             |
      | request_id           | *                                             |

  Scenario: FULL_TRANSITIVE — safe evolution with defaults is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-fullt-safe" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    And subject "avro-ex-fullt-safe" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"a"}]}
      """
    When I set the config for subject "avro-ex-fullt-safe" to "FULL_TRANSITIVE"
    And I register a schema under subject "avro-ex-fullt-safe":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"a"},{"name":"f3","type":"string","default":"b"}]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                             |
      | outcome              | success                                     |
      | actor_id             |                                             |
      | actor_type           | anonymous                                   |
      | auth_method          |                                             |
      | role                 |                                             |
      | target_type          | subject                                     |
      | target_id            | avro-ex-fullt-safe                          |
      | schema_id            | *                                           |
      | version              | *                                           |
      | schema_type          | AVRO                                        |
      | before_hash          |                                             |
      | after_hash           | sha256:*                                    |
      | context              | .                                           |
      | transport_security   | tls                                         |
      | source_ip            | *                                           |
      | user_agent           | *                                           |
      | method               | POST                                        |
      | path                 | /subjects/avro-ex-fullt-safe/versions       |
      | status_code          | 200                                         |
      | reason               |                                             |
      | error                |                                             |
      | request_body         |                                             |
      | metadata             |                                             |
      | timestamp            | *                                           |
      | duration_ms          | *                                           |
      | request_id           | *                                           |

  Scenario: FULL_TRANSITIVE — field without default transitively is incompatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-fullt-nodef" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    And subject "avro-ex-fullt-nodef" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"a"}]}
      """
    When I set the config for subject "avro-ex-fullt-nodef" to "FULL_TRANSITIVE"
    And I register a schema under subject "avro-ex-fullt-nodef":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string"}]}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | failure                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | avro-ex-fullt-nodef                          |
      | schema_id            |                                              |
      | version              | *                                            |
      | schema_type          | AVRO                                         |
      | before_hash          |                                              |
      | after_hash           |                                              |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/avro-ex-fullt-nodef/versions       |
      | status_code          | 409                                          |
      | reason               | already_exists                               |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ==========================================================================
  # COMPATIBILITY CHECK ENDPOINT (REST API)
  # ==========================================================================

  Scenario: Compatibility check endpoint — compatible returns is_compatible true
    Given the global compatibility level is "NONE"
    And subject "avro-ex-check-compat" has compatibility level "BACKWARD"
    And subject "avro-ex-check-compat" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    When I check compatibility of schema against subject "avro-ex-check-compat":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"x"}]}
      """
    Then the compatibility check should be compatible

  Scenario: Compatibility check endpoint — incompatible returns is_compatible false
    Given the global compatibility level is "NONE"
    And subject "avro-ex-check-incompat" has compatibility level "BACKWARD"
    And subject "avro-ex-check-incompat" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    When I check compatibility of schema against subject "avro-ex-check-incompat":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string"}]}
      """
    Then the compatibility check should be incompatible

  Scenario: Compatibility check against specific version
    Given the global compatibility level is "NONE"
    And subject "avro-ex-check-ver" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    And subject "avro-ex-check-ver" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"x"}]}
      """
    When I set the config for subject "avro-ex-check-ver" to "BACKWARD"
    And I check compatibility of schema against subject "avro-ex-check-ver" version 1:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"x"},{"name":"f3","type":"string","default":"y"}]}
      """
    Then the compatibility check should be compatible

  Scenario: Compatibility check against all versions
    Given the global compatibility level is "NONE"
    And subject "avro-ex-check-all" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    And subject "avro-ex-check-all" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"x"}]}
      """
    When I set the config for subject "avro-ex-check-all" to "BACKWARD"
    And I check compatibility of schema against all versions of subject "avro-ex-check-all":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"x"},{"name":"f3","type":"string","default":"y"}]}
      """
    Then the compatibility check should be compatible

  # ==========================================================================
  # NESTED RECORD COMPATIBILITY
  # ==========================================================================

  Scenario: Backward — nested record field addition with default is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-nested-add" has compatibility level "BACKWARD"
    And subject "avro-ex-nested-add" has schema:
      """
      {"type":"record","name":"Outer","fields":[{"name":"inner","type":{"type":"record","name":"Inner","fields":[{"name":"a","type":"string"}]}}]}
      """
    When I register a schema under subject "avro-ex-nested-add":
      """
      {"type":"record","name":"Outer","fields":[{"name":"inner","type":{"type":"record","name":"Inner","fields":[{"name":"a","type":"string"},{"name":"b","type":"string","default":"x"}]}}]}
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
      | target_id            | avro-ex-nested-add                           |
      | schema_id            | *                                            |
      | version              | *                                            |
      | schema_type          | AVRO                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/avro-ex-nested-add/versions        |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  Scenario: Backward — nested record type change is incompatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-nested-type" has compatibility level "BACKWARD"
    And subject "avro-ex-nested-type" has schema:
      """
      {"type":"record","name":"Outer","fields":[{"name":"inner","type":{"type":"record","name":"Inner","fields":[{"name":"a","type":"string"}]}}]}
      """
    When I register a schema under subject "avro-ex-nested-type":
      """
      {"type":"record","name":"Outer","fields":[{"name":"inner","type":{"type":"record","name":"Inner","fields":[{"name":"a","type":"int"}]}}]}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                               |
      | outcome              | failure                                       |
      | actor_id             |                                               |
      | actor_type           | anonymous                                     |
      | auth_method          |                                               |
      | role                 |                                               |
      | target_type          | subject                                       |
      | target_id            | avro-ex-nested-type                           |
      | schema_id            |                                               |
      | version              | *                                             |
      | schema_type          | AVRO                                          |
      | before_hash          |                                               |
      | after_hash           |                                               |
      | context              | .                                             |
      | transport_security   | tls                                           |
      | source_ip            | *                                             |
      | user_agent           | *                                             |
      | method               | POST                                          |
      | path                 | /subjects/avro-ex-nested-type/versions        |
      | status_code          | 409                                           |
      | reason               | already_exists                                |
      | error                |                                               |
      | request_body         |                                               |
      | metadata             |                                               |
      | timestamp            | *                                             |
      | duration_ms          | *                                             |
      | request_id           | *                                             |

  # ==========================================================================
  # MAP AND ARRAY COMPATIBILITY
  # ==========================================================================

  Scenario: Backward — map value type change is incompatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-map-type" has compatibility level "BACKWARD"
    And subject "avro-ex-map-type" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"m","type":{"type":"map","values":"string"}}]}
      """
    When I register a schema under subject "avro-ex-map-type":
      """
      {"type":"record","name":"R","fields":[{"name":"m","type":{"type":"map","values":"int"}}]}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                            |
      | outcome              | failure                                    |
      | actor_id             |                                            |
      | actor_type           | anonymous                                  |
      | auth_method          |                                            |
      | role                 |                                            |
      | target_type          | subject                                    |
      | target_id            | avro-ex-map-type                           |
      | schema_id            |                                            |
      | version              | *                                          |
      | schema_type          | AVRO                                       |
      | before_hash          |                                            |
      | after_hash           |                                            |
      | context              | .                                          |
      | transport_security   | tls                                        |
      | source_ip            | *                                          |
      | user_agent           | *                                          |
      | method               | POST                                       |
      | path                 | /subjects/avro-ex-map-type/versions        |
      | status_code          | 409                                        |
      | reason               | already_exists                             |
      | error                |                                            |
      | request_body         |                                            |
      | metadata             |                                            |
      | timestamp            | *                                          |
      | duration_ms          | *                                          |
      | request_id           | *                                          |

  Scenario: Backward — array item type change is incompatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-array-type" has compatibility level "BACKWARD"
    And subject "avro-ex-array-type" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"a","type":{"type":"array","items":"string"}}]}
      """
    When I register a schema under subject "avro-ex-array-type":
      """
      {"type":"record","name":"R","fields":[{"name":"a","type":{"type":"array","items":"int"}}]}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | failure                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | avro-ex-array-type                           |
      | schema_id            |                                              |
      | version              | *                                            |
      | schema_type          | AVRO                                         |
      | before_hash          |                                              |
      | after_hash           |                                              |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/avro-ex-array-type/versions        |
      | status_code          | 409                                          |
      | reason               | already_exists                               |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  Scenario: Backward — map value promotion (int to long) is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-map-promo" has compatibility level "BACKWARD"
    And subject "avro-ex-map-promo" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"m","type":{"type":"map","values":"int"}}]}
      """
    When I register a schema under subject "avro-ex-map-promo":
      """
      {"type":"record","name":"R","fields":[{"name":"m","type":{"type":"map","values":"long"}}]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                            |
      | outcome              | success                                    |
      | actor_id             |                                            |
      | actor_type           | anonymous                                  |
      | auth_method          |                                            |
      | role                 |                                            |
      | target_type          | subject                                    |
      | target_id            | avro-ex-map-promo                          |
      | schema_id            | *                                          |
      | version              | *                                          |
      | schema_type          | AVRO                                       |
      | before_hash          |                                            |
      | after_hash           | sha256:*                                   |
      | context              | .                                          |
      | transport_security   | tls                                        |
      | source_ip            | *                                          |
      | user_agent           | *                                          |
      | method               | POST                                       |
      | path                 | /subjects/avro-ex-map-promo/versions       |
      | status_code          | 200                                        |
      | reason               |                                            |
      | error                |                                            |
      | request_body         |                                            |
      | metadata             |                                            |
      | timestamp            | *                                          |
      | duration_ms          | *                                          |
      | request_id           | *                                          |

  Scenario: Backward — array item promotion (int to long) is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-array-promo" has compatibility level "BACKWARD"
    And subject "avro-ex-array-promo" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"a","type":{"type":"array","items":"int"}}]}
      """
    When I register a schema under subject "avro-ex-array-promo":
      """
      {"type":"record","name":"R","fields":[{"name":"a","type":{"type":"array","items":"long"}}]}
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
      | target_id            | avro-ex-array-promo                          |
      | schema_id            | *                                            |
      | version              | *                                            |
      | schema_type          | AVRO                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/avro-ex-array-promo/versions       |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |
