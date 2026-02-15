@functional @import @axonops-only
Feature: Advanced Schema Import
  As an operator migrating from another schema registry, I want the import API
  to handle edge cases correctly including ID conflicts, partial failures, and references

  # --------------------------------------------------------------------------
  # CONFLICT HANDLING
  # --------------------------------------------------------------------------

  Scenario: Import with conflicting schema ID fails
    When I import a schema with ID 100 under subject "imp-conflict-a" version 1:
      """
      {"type":"record","name":"First","fields":[{"name":"id","type":"string"}]}
      """
    Then the response status should be 200
    And the import should have 1 imported and 0 errors
    When I import a schema with ID 100 under subject "imp-conflict-b" version 1:
      """
      {"type":"record","name":"Second","fields":[{"name":"name","type":"string"}]}
      """
    Then the response status should be 200
    And the import should have 0 imported and 1 errors

  Scenario: Import with conflicting subject and version fails
    When I import a schema with ID 200 under subject "imp-sv-conflict" version 1:
      """
      {"type":"record","name":"Original","fields":[{"name":"id","type":"string"}]}
      """
    Then the response status should be 200
    And the import should have 1 imported and 0 errors
    When I import a schema with ID 201 under subject "imp-sv-conflict" version 1:
      """
      {"type":"record","name":"Replacement","fields":[{"name":"name","type":"string"}]}
      """
    Then the response status should be 200
    And the import should have 0 imported and 1 errors

  # --------------------------------------------------------------------------
  # PARTIAL IMPORT
  # --------------------------------------------------------------------------

  Scenario: Partial import success with invalid schema in batch
    When I import schemas:
      """
      {"schemas":[
        {"id":300,"subject":"imp-partial-a","version":1,"schema":"{\"type\":\"record\",\"name\":\"Good1\",\"fields\":[{\"name\":\"f\",\"type\":\"string\"}]}"},
        {"id":301,"subject":"imp-partial-b","version":1,"schema":"{invalid json not a schema"},
        {"id":302,"subject":"imp-partial-c","version":1,"schema":"{\"type\":\"record\",\"name\":\"Good2\",\"fields\":[{\"name\":\"f\",\"type\":\"int\"}]}"}
      ]}
      """
    Then the response status should be 200
    And the import should have 2 imported and 1 errors
    When I get schema by ID 300
    Then the response status should be 200
    And the response should contain "Good1"
    When I get schema by ID 302
    Then the response status should be 200
    And the response should contain "Good2"

  # --------------------------------------------------------------------------
  # ID SEQUENCING
  # --------------------------------------------------------------------------

  Scenario: Import then register continues IDs above imported
    When I import a schema with ID 500 under subject "imp-seq-imported" version 1:
      """
      {"type":"record","name":"Imported","fields":[{"name":"id","type":"string"}]}
      """
    Then the response status should be 200
    And the import should have 1 imported and 0 errors
    When I register a schema under subject "imp-seq-new":
      """
      {"type":"record","name":"AutoAssigned","fields":[{"name":"name","type":"string"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "new_id"
    And the response should have field "id"

  # --------------------------------------------------------------------------
  # IMPORT WITH REFERENCES
  # --------------------------------------------------------------------------

  Scenario: Import schema with references to another subject
    When I import a schema with ID 600 under subject "imp-ref-base" version 1:
      """
      {"type":"record","name":"Address","namespace":"com.imp","fields":[{"name":"street","type":"string"},{"name":"city","type":"string"}]}
      """
    Then the response status should be 200
    And the import should have 1 imported and 0 errors
    When I import schemas:
      """
      {"schemas":[
        {"id":601,"subject":"imp-ref-person","version":1,"schema":"{\"type\":\"record\",\"name\":\"Person\",\"namespace\":\"com.imp\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"address\",\"type\":\"com.imp.Address\"}]}","references":[{"name":"com.imp.Address","subject":"imp-ref-base","version":1}]}
      ]}
      """
    Then the response status should be 200
    And the import should have 1 imported and 0 errors
    When I get schema by ID 601
    Then the response status should be 200
    And the response should contain "Person"

  # --------------------------------------------------------------------------
  # SCHEMA TYPE PRESERVATION
  # --------------------------------------------------------------------------

  Scenario: Import preserves schema type across retrieval
    When I import a "PROTOBUF" schema with ID 700 under subject "imp-type-proto" version 1:
      """
      syntax = "proto3";
      message TypedImport {
        string name = 1;
        int32 value = 2;
      }
      """
    Then the response status should be 200
    And the import should have 1 imported and 0 errors
    When I get schema by ID 700
    Then the response status should be 200
    And the response field "schemaType" should be "PROTOBUF"
    And the response should contain "TypedImport"
    When I get version 1 of subject "imp-type-proto"
    Then the response status should be 200
    And the response field "subject" should be "imp-type-proto"
    And the response field "version" should be 1

  # --------------------------------------------------------------------------
  # RETRIEVAL BY SUBJECT AND VERSION
  # --------------------------------------------------------------------------

  Scenario: Imported schema retrievable by subject and version
    When I import a schema with ID 800 under subject "imp-retrieve" version 1:
      """
      {"type":"record","name":"Retrievable","fields":[{"name":"key","type":"string"},{"name":"value","type":"long"}]}
      """
    Then the response status should be 200
    And the import should have 1 imported and 0 errors
    When I get version 1 of subject "imp-retrieve"
    Then the response status should be 200
    And the response field "subject" should be "imp-retrieve"
    And the response field "version" should be 1
    And the response should contain "Retrievable"
    When I get schema by ID 800
    Then the response status should be 200
    And the response should contain "Retrievable"

  # --------------------------------------------------------------------------
  # MULTIPLE VERSIONS OF SAME SUBJECT
  # --------------------------------------------------------------------------

  Scenario: Import multiple versions of same subject
    Given the global compatibility level is "NONE"
    When I import schemas:
      """
      {"schemas":[
        {"id":900,"subject":"imp-multi-ver","version":1,"schema":"{\"type\":\"record\",\"name\":\"Evolving\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"}]}"},
        {"id":901,"subject":"imp-multi-ver","version":2,"schema":"{\"type\":\"record\",\"name\":\"Evolving\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"},{\"name\":\"name\",\"type\":\"string\"}]}"},
        {"id":902,"subject":"imp-multi-ver","version":3,"schema":"{\"type\":\"record\",\"name\":\"Evolving\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"},{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"email\",\"type\":\"string\"}]}"}
      ]}
      """
    Then the response status should be 200
    And the import should have 3 imported and 0 errors
    When I list versions of subject "imp-multi-ver"
    Then the response status should be 200
    And the response should be an array of length 3
    When I get version 1 of subject "imp-multi-ver"
    Then the response status should be 200
    And the response field "version" should be 1
    When I get version 3 of subject "imp-multi-ver"
    Then the response status should be 200
    And the response field "version" should be 3
    And the response should contain "email"
