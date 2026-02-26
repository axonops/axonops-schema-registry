@functional @contexts
Feature: Contexts — Schema Evolution Workflows
  Real-world schema evolution workflows within named contexts.
  All scenarios use the qualified subject format :.contextname:subject
  which is the standard Confluent Schema Registry context API.
  Each scenario uses a unique context name to avoid cross-scenario pollution.

  Background:
    Given the schema registry is running

  # ==========================================================================
  # MULTI-VERSION EVOLUTION
  # ==========================================================================

  Scenario: Multi-version Avro schema evolution in a context
    # Register v1 with just a name field
    When I POST "/subjects/:.evo1:User/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"User\",\"namespace\":\"com.example.evo1\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "evo1_v1_id"
    # Set BACKWARD compatibility for the subject
    When I PUT "/config/:.evo1:User" with body:
      """
      {"compatibility": "BACKWARD"}
      """
    Then the response status should be 200
    # Register v2: add optional "age" field (backward compatible — new reader has default)
    When I POST "/subjects/:.evo1:User/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"User\",\"namespace\":\"com.example.evo1\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"age\",\"type\":[\"null\",\"int\"],\"default\":null}]}"}
      """
    Then the response status should be 200
    # Register v3: add optional "email" field
    When I POST "/subjects/:.evo1:User/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"User\",\"namespace\":\"com.example.evo1\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"age\",\"type\":[\"null\",\"int\"],\"default\":null},{\"name\":\"email\",\"type\":[\"null\",\"string\"],\"default\":null}]}"}
      """
    Then the response status should be 200
    # Verify 3 versions exist
    When I GET "/subjects/:.evo1:User/versions"
    Then the response status should be 200
    And the response should be an array of length 3
    # Verify latest is version 3 with all 3 fields
    When I GET "/subjects/:.evo1:User/versions/latest"
    Then the response status should be 200
    And the response field "version" should be 3
    And the response body should contain "email"
    And the response body should contain "age"
    And the response body should contain "name"
    # Verify version 1 only has the name field
    When I GET "/subjects/:.evo1:User/versions/1"
    Then the response status should be 200
    And the response field "version" should be 1
    And the response body should contain "name"
    And the response body should not contain "age"
    And the response body should not contain "email"

  # ==========================================================================
  # BACKWARD_TRANSITIVE ENFORCEMENT
  # ==========================================================================

  Scenario: BACKWARD_TRANSITIVE enforcement rejects schema incompatible with earlier version
    # Register v1 with name and code fields
    When I POST "/subjects/:.evo2:Event/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Event\",\"namespace\":\"com.example.evo2\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"},{\"name\":\"code\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Set BACKWARD_TRANSITIVE
    When I PUT "/config/:.evo2:Event" with body:
      """
      {"compatibility": "BACKWARD_TRANSITIVE"}
      """
    Then the response status should be 200
    # Register v2: remove code field (backward compatible — new reader v2 ignores code)
    When I POST "/subjects/:.evo2:Event/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Event\",\"namespace\":\"com.example.evo2\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Try v3: re-add code as int type — compatible with v2 (new field with default)
    # but INCOMPATIBLE with v1 (code was string, now int — type mismatch)
    # BACKWARD_TRANSITIVE checks against ALL versions, so this must be rejected
    When I POST "/subjects/:.evo2:Event/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Event\",\"namespace\":\"com.example.evo2\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"},{\"name\":\"code\",\"type\":\"int\",\"default\":0}]}"}
      """
    Then the response status should be 409

  # ==========================================================================
  # FORWARD COMPATIBILITY
  # ==========================================================================

  Scenario: FORWARD compatibility allows adding fields in a context
    # FORWARD means: data written with the NEW schema can be read by readers using OLD schema.
    # Adding optional fields to new schema: old reader ignores them — forward compatible.
    When I POST "/subjects/:.evo3:Record/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Record\",\"namespace\":\"com.example.evo3\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Set FORWARD compatibility
    When I PUT "/config/:.evo3:Record" with body:
      """
      {"compatibility": "FORWARD"}
      """
    Then the response status should be 200
    # Register v2: add an optional field — old reader (v1) ignores it — forward compatible
    When I POST "/subjects/:.evo3:Record/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Record\",\"namespace\":\"com.example.evo3\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"age\",\"type\":[\"null\",\"int\"],\"default\":null}]}"}
      """
    Then the response status should be 200
    # Verify 2 versions
    When I GET "/subjects/:.evo3:Record/versions"
    Then the response status should be 200
    And the response should be an array of length 2

  # ==========================================================================
  # FULL COMPATIBILITY
  # ==========================================================================

  Scenario: FULL compatibility requires both backward and forward in a context
    # FULL = BACKWARD + FORWARD.
    # Adding a nullable field with null default is both backward and forward compatible.
    When I POST "/subjects/:.evo4:Profile/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Profile\",\"namespace\":\"com.example.evo4\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Set FULL compatibility
    When I PUT "/config/:.evo4:Profile" with body:
      """
      {"compatibility": "FULL"}
      """
    Then the response status should be 200
    # Register v2: add nullable field with default — both backward and forward compatible
    When I POST "/subjects/:.evo4:Profile/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Profile\",\"namespace\":\"com.example.evo4\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"nickname\",\"type\":[\"null\",\"string\"],\"default\":null}]}"}
      """
    Then the response status should be 200
    # Check compat of a schema that changes field type — should be incompatible
    When I POST "/compatibility/subjects/:.evo4:Profile/versions/latest" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Profile\",\"namespace\":\"com.example.evo4\",\"fields\":[{\"name\":\"name\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    And the response field "is_compatible" should be false

  # ==========================================================================
  # VERSION DELETION
  # ==========================================================================

  Scenario: Version deletion preserves latest tracking in a context
    When I POST "/subjects/:.evo5:Data/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Data\",\"namespace\":\"com.example.evo5\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.evo5:Data/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Data\",\"namespace\":\"com.example.evo5\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\",\"default\":\"\"}]}"}
      """
    Then the response status should be 200
    # Soft-delete version 1
    When I DELETE "/subjects/:.evo5:Data/versions/1"
    Then the response status should be 200
    # Latest should return version 2
    When I GET "/subjects/:.evo5:Data/versions/latest"
    Then the response status should be 200
    And the response field "version" should be 2
    # Version list should only contain version 2
    When I GET "/subjects/:.evo5:Data/versions"
    Then the response status should be 200
    And the response should be an array of length 1
    # Version 2 still accessible
    When I GET "/subjects/:.evo5:Data/versions/2"
    Then the response status should be 200
    And the response body should contain "Data"

  # ==========================================================================
  # PERMANENT VERSION DELETION
  # ==========================================================================

  Scenario: Permanent version deletion in a context
    When I POST "/subjects/:.evo6:Entry/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Entry\",\"namespace\":\"com.example.evo6\",\"fields\":[{\"name\":\"x\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.evo6:Entry/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Entry\",\"namespace\":\"com.example.evo6\",\"fields\":[{\"name\":\"x\",\"type\":\"string\"},{\"name\":\"y\",\"type\":\"string\",\"default\":\"\"}]}"}
      """
    Then the response status should be 200
    # Soft-delete version 1
    When I DELETE "/subjects/:.evo6:Entry/versions/1"
    Then the response status should be 200
    # Permanently delete version 1
    When I DELETE "/subjects/:.evo6:Entry/versions/1?permanent=true"
    Then the response status should be 200
    # Version 1 is gone
    When I GET "/subjects/:.evo6:Entry/versions/1"
    Then the response status should be 404
    # Version 2 still works with correct schema
    When I GET "/subjects/:.evo6:Entry/versions/2"
    Then the response status should be 200
    And the response body should contain "Entry"

  # ==========================================================================
  # INDEPENDENT EVOLUTION ACROSS CONTEXTS
  # ==========================================================================

  Scenario: Independent evolution paths across two contexts
    # Context .evo7a: evolve User with "age" field
    When I POST "/subjects/:.evo7a:User/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"User\",\"namespace\":\"com.example.evo7a\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.evo7a:User/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"User\",\"namespace\":\"com.example.evo7a\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"age\",\"type\":[\"null\",\"int\"],\"default\":null}]}"}
      """
    Then the response status should be 200
    # Context .evo7b: evolve User with "email" field instead
    When I POST "/subjects/:.evo7b:User/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"User\",\"namespace\":\"com.example.evo7b\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.evo7b:User/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"User\",\"namespace\":\"com.example.evo7b\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"email\",\"type\":[\"null\",\"string\"],\"default\":null}]}"}
      """
    Then the response status should be 200
    # Verify .evo7a latest has "age"
    When I GET "/subjects/:.evo7a:User/versions/latest"
    Then the response status should be 200
    And the response body should contain "age"
    And the response body should not contain "email"
    # Verify .evo7b latest has "email"
    When I GET "/subjects/:.evo7b:User/versions/latest"
    Then the response status should be 200
    And the response body should contain "email"
    And the response body should not contain "age"
    # Versions in each context are independent
    When I GET "/subjects/:.evo7a:User/versions"
    Then the response status should be 200
    And the response should be an array of length 2
    When I GET "/subjects/:.evo7b:User/versions"
    Then the response status should be 200
    And the response should be an array of length 2

  # ==========================================================================
  # IDEMPOTENT RE-REGISTRATION
  # ==========================================================================

  Scenario: Schema re-registration is idempotent within a context
    When I POST "/subjects/:.evo8:Metric/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Metric\",\"namespace\":\"com.example.evo8\",\"fields\":[{\"name\":\"value\",\"type\":\"double\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "evo8_first_id"
    # Register the exact same schema again
    When I POST "/subjects/:.evo8:Metric/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Metric\",\"namespace\":\"com.example.evo8\",\"fields\":[{\"name\":\"value\",\"type\":\"double\"}]}"}
      """
    Then the response status should be 200
    And the response field "id" should equal stored "evo8_first_id"
    # Only 1 version should exist
    When I GET "/subjects/:.evo8:Metric/versions"
    Then the response status should be 200
    And the response should be an array of length 1

  # ==========================================================================
  # COMPATIBILITY CHECK AGAINST SPECIFIC VERSION
  # ==========================================================================

  Scenario: Compatibility check against specific version in a context
    When I POST "/subjects/:.evo9:Order/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Order\",\"namespace\":\"com.example.evo9\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.evo9:Order/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Order\",\"namespace\":\"com.example.evo9\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"total\",\"type\":[\"null\",\"double\"],\"default\":null}]}"}
      """
    Then the response status should be 200
    # Set BACKWARD compat
    When I PUT "/config/:.evo9:Order" with body:
      """
      {"compatibility": "BACKWARD"}
      """
    Then the response status should be 200
    # Check compat against version 1 specifically — adding optional field is backward compatible
    When I POST "/compatibility/subjects/:.evo9:Order/versions/1" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Order\",\"namespace\":\"com.example.evo9\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"total\",\"type\":[\"null\",\"double\"],\"default\":null},{\"name\":\"currency\",\"type\":[\"null\",\"string\"],\"default\":null}]}"}
      """
    Then the response status should be 200
    And the response field "is_compatible" should be true
    # Check compat against "latest" — should also work
    When I POST "/compatibility/subjects/:.evo9:Order/versions/latest" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Order\",\"namespace\":\"com.example.evo9\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"total\",\"type\":[\"null\",\"double\"],\"default\":null},{\"name\":\"currency\",\"type\":[\"null\",\"string\"],\"default\":null}]}"}
      """
    Then the response status should be 200
    And the response field "is_compatible" should be true

  # ==========================================================================
  # COMPATIBILITY CHECK AGAINST ALL VERSIONS
  # ==========================================================================

  Scenario: Compatibility check against all versions in a context
    When I POST "/subjects/:.evo10:Item/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Item\",\"namespace\":\"com.example.evo10\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    # Set BACKWARD_TRANSITIVE before registering v2
    When I PUT "/config/:.evo10:Item" with body:
      """
      {"compatibility": "BACKWARD_TRANSITIVE"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.evo10:Item/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Item\",\"namespace\":\"com.example.evo10\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"label\",\"type\":[\"null\",\"string\"],\"default\":null}]}"}
      """
    Then the response status should be 200
    # Check compat against all versions (POST to /compatibility/subjects/{subject}/versions)
    When I POST "/compatibility/subjects/:.evo10:Item/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Item\",\"namespace\":\"com.example.evo10\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"label\",\"type\":[\"null\",\"string\"],\"default\":null},{\"name\":\"desc\",\"type\":[\"null\",\"string\"],\"default\":null}]}"}
      """
    Then the response status should be 200
    And the response field "is_compatible" should be true

  # ==========================================================================
  # FULL_TRANSITIVE REJECTION
  # ==========================================================================

  Scenario: FULL_TRANSITIVE rejects schema incompatible with earlier version
    # FULL_TRANSITIVE = BACKWARD + FORWARD checked against ALL versions.
    # Adding nullable fields with null default is safe for both directions.
    # Changing a field type breaks compatibility.
    When I POST "/subjects/:.evo11:Sensor/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Sensor\",\"namespace\":\"com.example.evo11\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    # Set FULL_TRANSITIVE
    When I PUT "/config/:.evo11:Sensor" with body:
      """
      {"compatibility": "FULL_TRANSITIVE"}
      """
    Then the response status should be 200
    # Register v2: add nullable field — both backward and forward compatible
    When I POST "/subjects/:.evo11:Sensor/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Sensor\",\"namespace\":\"com.example.evo11\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"reading\",\"type\":[\"null\",\"double\"],\"default\":null}]}"}
      """
    Then the response status should be 200
    # Try v3 that changes id type from int to string — incompatible with v1 and v2
    When I POST "/subjects/:.evo11:Sensor/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Sensor\",\"namespace\":\"com.example.evo11\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"},{\"name\":\"reading\",\"type\":[\"null\",\"double\"],\"default\":null}]}"}
      """
    Then the response status should be 409
