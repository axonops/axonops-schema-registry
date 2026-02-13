@functional @import
Feature: Schema Import
  As an operator migrating from another schema registry, I want to import schemas with specific IDs

  Scenario: Import single schema with specific ID
    When I import a schema with ID 100 under subject "imported-value" version 1:
      """
      {"type":"record","name":"Imported","fields":[{"name":"name","type":"string"}]}
      """
    Then the response status should be 200
    And the import should have 1 imported and 0 errors
    When I get schema by ID 100
    Then the response status should be 200
    And the response should contain "Imported"

  Scenario: Import schema and retrieve by subject/version
    When I import a schema with ID 200 under subject "import-subj" version 1:
      """
      {"type":"record","name":"ImportSubj","fields":[{"name":"id","type":"string"}]}
      """
    Then the response status should be 200
    When I get version 1 of subject "import-subj"
    Then the response status should be 200
    And the response field "subject" should be "import-subj"
    And the response field "version" should be 1

  Scenario: Import multiple schemas in one request
    When I import schemas:
      """
      {"schemas":[
        {"id":300,"subject":"bulk-a","version":1,"schema":"{\"type\":\"record\",\"name\":\"A\",\"fields\":[{\"name\":\"f\",\"type\":\"string\"}]}"},
        {"id":301,"subject":"bulk-b","version":1,"schema":"{\"type\":\"record\",\"name\":\"B\",\"fields\":[{\"name\":\"f\",\"type\":\"string\"}]}"},
        {"id":302,"subject":"bulk-c","version":1,"schema":"{\"type\":\"record\",\"name\":\"C\",\"fields\":[{\"name\":\"f\",\"type\":\"string\"}]}"}
      ]}
      """
    Then the response status should be 200
    And the import should have 3 imported and 0 errors
    When I get schema by ID 300
    Then the response status should be 200
    When I get schema by ID 301
    Then the response status should be 200
    When I get schema by ID 302
    Then the response status should be 200

  Scenario: IDs after import continue above highest imported
    When I import a schema with ID 500 under subject "import-seq" version 1:
      """
      {"type":"record","name":"Seq","fields":[{"name":"f","type":"string"}]}
      """
    Then the response status should be 200
    When I register a schema under subject "new-after-import":
      """
      {"type":"record","name":"AfterImport","fields":[{"name":"f","type":"string"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "new_id"

  Scenario: Import Protobuf schema with specific ID
    When I import a "PROTOBUF" schema with ID 600 under subject "import-proto" version 1:
      """
      syntax = "proto3";
      message ImportedProto {
        string name = 1;
      }
      """
    Then the response status should be 200
    When I get schema by ID 600
    Then the response status should be 200
    And the response field "schemaType" should be "PROTOBUF"

  Scenario: Import JSON Schema with specific ID
    When I import a "JSON" schema with ID 700 under subject "import-json" version 1:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
      """
    Then the response status should be 200
    When I get schema by ID 700
    Then the response status should be 200
    And the response field "schemaType" should be "JSON"
