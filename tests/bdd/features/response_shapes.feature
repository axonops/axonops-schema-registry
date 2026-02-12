@functional
Feature: Response Shapes — Confluent Wire Compatibility
  Verify that all API responses match the exact field names and shapes
  expected by Confluent Schema Registry clients.

  # ==========================================================================
  # SCHEMA REGISTRATION RESPONSE
  # ==========================================================================

  Scenario: Registration response contains only id field
    When I register a schema under subject "resp-reg-only-id":
      """
      {"type":"record","name":"RegOnly","fields":[{"name":"x","type":"string"}]}
      """
    Then the response status should be 200
    And the response should have field "id"

  # ==========================================================================
  # SCHEMA-BY-ID RESPONSE — schemaType OMISSION FOR AVRO
  # ==========================================================================

  Scenario: GET schema by ID for Avro omits schemaType field
    Given subject "resp-avro-byid" has schema:
      """
      {"type":"record","name":"AvroById","fields":[{"name":"a","type":"string"}]}
      """
    And I store the response field "id" as "avro_byid"
    When I get schema by ID {{avro_byid}}
    Then the response status should be 200
    And the response should have field "schema"
    And the response should not have field "schemaType"

  Scenario: GET schema by ID for Protobuf includes schemaType PROTOBUF
    Given subject "resp-proto-byid" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message ProtoById {
        string a = 1;
      }
      """
    When I get the latest version of subject "resp-proto-byid"
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response field "schemaType" should be "PROTOBUF"

  Scenario: GET schema by ID for JSON includes schemaType JSON
    Given subject "resp-json-byid" has "JSON" schema:
      """
      {"type":"object","properties":{"a":{"type":"string"}},"required":["a"]}
      """
    When I get the latest version of subject "resp-json-byid"
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response field "schemaType" should be "JSON"

  # ==========================================================================
  # SUBJECT-VERSION RESPONSE
  # ==========================================================================

  Scenario: GET subject/version for Avro has all fields, omits schemaType
    Given subject "resp-avro-ver" has schema:
      """
      {"type":"record","name":"AvroVer","fields":[{"name":"v","type":"string"}]}
      """
    When I get version 1 of subject "resp-avro-ver"
    Then the response status should be 200
    And the response field "subject" should be "resp-avro-ver"
    And the response field "version" should be 1
    And the response should have field "id"
    And the response should have field "schema"
    And the response should not have field "schemaType"

  Scenario: GET subject/version for Protobuf includes schemaType
    Given subject "resp-proto-ver" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message ProtoVer {
        string v = 1;
      }
      """
    When I get version 1 of subject "resp-proto-ver"
    Then the response status should be 200
    And the response field "subject" should be "resp-proto-ver"
    And the response field "version" should be 1
    And the response should have field "id"
    And the response should have field "schema"
    And the response field "schemaType" should be "PROTOBUF"

  Scenario: GET subject/version for JSON includes schemaType
    Given subject "resp-json-ver" has "JSON" schema:
      """
      {"type":"object","properties":{"v":{"type":"string"}},"required":["v"]}
      """
    When I get version 1 of subject "resp-json-ver"
    Then the response status should be 200
    And the response field "subject" should be "resp-json-ver"
    And the response field "version" should be 1
    And the response should have field "id"
    And the response should have field "schema"
    And the response field "schemaType" should be "JSON"

  # ==========================================================================
  # LOOKUP RESPONSE
  # ==========================================================================

  Scenario: Lookup response for Avro has all fields, omits schemaType
    Given subject "resp-avro-lookup" has schema:
      """
      {"type":"record","name":"AvroLookup","fields":[{"name":"l","type":"string"}]}
      """
    When I lookup schema in subject "resp-avro-lookup":
      """
      {"type":"record","name":"AvroLookup","fields":[{"name":"l","type":"string"}]}
      """
    Then the response status should be 200
    And the response field "subject" should be "resp-avro-lookup"
    And the response field "version" should be 1
    And the response should have field "id"
    And the response should have field "schema"
    And the response should not have field "schemaType"

  Scenario: Lookup response for Protobuf includes schemaType
    Given subject "resp-proto-lookup" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message ProtoLookup {
        string l = 1;
      }
      """
    When I lookup a "PROTOBUF" schema in subject "resp-proto-lookup":
      """
      syntax = "proto3";
      message ProtoLookup {
        string l = 1;
      }
      """
    Then the response status should be 200
    And the response field "subject" should be "resp-proto-lookup"
    And the response field "version" should be 1
    And the response should have field "id"
    And the response should have field "schema"
    And the response field "schemaType" should be "PROTOBUF"

  # ==========================================================================
  # CONFIG RESPONSE FIELD NAMES
  # ==========================================================================

  Scenario: GET /config returns compatibilityLevel field
    When I get the global config
    Then the response status should be 200
    And the response should have field "compatibilityLevel"

  Scenario: PUT /config returns compatibility field
    When I set the global config to "FULL"
    Then the response status should be 200
    And the response should have field "compatibility"

  Scenario: GET /config/{subject} returns compatibilityLevel field
    Given subject "resp-cfg-sub" has compatibility level "FORWARD"
    When I get the config for subject "resp-cfg-sub"
    Then the response status should be 200
    And the response should have field "compatibilityLevel"
    And the response field "compatibilityLevel" should be "FORWARD"

  Scenario: PUT /config/{subject} returns compatibility field
    When I set the config for subject "resp-cfg-sub2" to "FULL"
    Then the response status should be 200
    And the response should have field "compatibility"
    And the response field "compatibility" should be "FULL"

  # ==========================================================================
  # MODE RESPONSE
  # ==========================================================================

  Scenario: GET /mode returns mode field
    When I get the global mode
    Then the response status should be 200
    And the response should have field "mode"
    And the response field "mode" should be "READWRITE"

  Scenario: PUT /mode returns mode field
    When I set the global mode to "READWRITE"
    Then the response status should be 200
    And the response should have field "mode"
    And the response field "mode" should be "READWRITE"

  # ==========================================================================
  # DELETE SUBJECT RESPONSE BODY
  # ==========================================================================

  Scenario: DELETE subject with 1 version returns array with version number
    Given subject "resp-del-sub-1" has schema:
      """
      {"type":"record","name":"DelSub1","fields":[{"name":"x","type":"string"}]}
      """
    When I delete subject "resp-del-sub-1"
    Then the response status should be 200
    And the response should be an array of length 1
    And the response array should contain integer 1

  Scenario: DELETE subject with 3 versions returns array with all version numbers
    Given the global compatibility level is "NONE"
    And subject "resp-del-sub-3" has schema:
      """
      {"type":"record","name":"V1","fields":[{"name":"a","type":"string"}]}
      """
    And subject "resp-del-sub-3" has schema:
      """
      {"type":"record","name":"V2","fields":[{"name":"b","type":"string"}]}
      """
    And subject "resp-del-sub-3" has schema:
      """
      {"type":"record","name":"V3","fields":[{"name":"c","type":"string"}]}
      """
    When I delete subject "resp-del-sub-3"
    Then the response status should be 200
    And the response should be an array of length 3
    And the response array should contain integer 1
    And the response array should contain integer 2
    And the response array should contain integer 3

  # ==========================================================================
  # DELETE VERSION RESPONSE BODY
  # ==========================================================================

  Scenario: DELETE version returns the version number as integer
    Given the global compatibility level is "NONE"
    And subject "resp-del-ver" has schema:
      """
      {"type":"record","name":"V1","fields":[{"name":"a","type":"string"}]}
      """
    And subject "resp-del-ver" has schema:
      """
      {"type":"record","name":"V2","fields":[{"name":"b","type":"string"}]}
      """
    When I delete version 1 of subject "resp-del-ver"
    Then the response status should be 200
    And the response should be an integer with value 1

  Scenario: DELETE version 2 returns integer 2
    Given the global compatibility level is "NONE"
    And subject "resp-del-ver2" has schema:
      """
      {"type":"record","name":"V1","fields":[{"name":"a","type":"string"}]}
      """
    And subject "resp-del-ver2" has schema:
      """
      {"type":"record","name":"V2","fields":[{"name":"b","type":"string"}]}
      """
    When I delete version 2 of subject "resp-del-ver2"
    Then the response status should be 200
    And the response should be an integer with value 2

  # ==========================================================================
  # DELETE CONFIG RESPONSE
  # ==========================================================================

  Scenario: DELETE /config/{subject} returns the removed compatibility level
    Given subject "resp-del-cfg" has compatibility level "FULL"
    When I delete the config for subject "resp-del-cfg"
    Then the response status should be 200
    And the response should have field "compatibilityLevel"
    And the response field "compatibilityLevel" should be "FULL"
