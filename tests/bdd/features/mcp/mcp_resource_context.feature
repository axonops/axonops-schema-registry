@mcp
Feature: MCP Context-Scoped Resources and Prompts
  MCP resources and prompts support multi-tenant context isolation.
  Resources use context-prefixed URIs (schema://contexts/{context}/...) to
  access data in specific contexts. Prompts accept an optional context argument.
  Without a context prefix or argument, the default context is used.

  # ==========================================================================
  # 1. CONTEXT-SCOPED SUBJECT RESOURCES
  # ==========================================================================

  Scenario: Context-scoped subjects list shows only context subjects
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "resctx-default-subj",
        "schema": "{\"type\":\"string\"}"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "resctx-team-subj",
        "schema": "{\"type\":\"record\",\"name\":\"Team\",\"fields\":[{\"name\":\"team\",\"type\":\"string\"}]}",
        "schema_type": "AVRO",
        "context": ".teamres"
      }
      """
    Then the MCP result should not be an error
    # Default subjects resource should not include context subjects
    When I read MCP resource "schema://subjects"
    Then the MCP resource result should contain "resctx-default-subj"
    And the MCP resource result should not contain "resctx-team-subj"
    # Context-scoped subjects resource should only include context subjects
    When I read MCP resource "schema://contexts/.teamres/subjects"
    Then the MCP resource result should contain "resctx-team-subj"
    And the MCP resource result should not contain "resctx-default-subj"

  Scenario: Context-scoped subject detail resource
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "resctx-detail-subj",
        "schema": "{\"type\":\"record\",\"name\":\"Detail\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"}]}",
        "schema_type": "AVRO",
        "context": ".detailctx"
      }
      """
    Then the MCP result should not be an error
    When I read MCP resource "schema://contexts/.detailctx/subjects/resctx-detail-subj"
    Then the MCP resource result should contain "resctx-detail-subj"
    And the MCP resource result should contain "latest"

  Scenario: Context-scoped subject versions resource
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "resctx-vers-subj",
        "schema": "{\"type\":\"string\"}",
        "context": ".versctx"
      }
      """
    Then the MCP result should not be an error
    When I read MCP resource "schema://contexts/.versctx/subjects/resctx-vers-subj/versions"
    Then the MCP resource result should contain "1"

  Scenario: Context-scoped subject version detail resource
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "resctx-vd-subj",
        "schema": "{\"type\":\"string\"}",
        "context": ".vdctx"
      }
      """
    Then the MCP result should not be an error
    When I read MCP resource "schema://contexts/.vdctx/subjects/resctx-vd-subj/versions/1"
    Then the MCP resource result should contain "resctx-vd-subj"
    And the MCP resource result should contain "AVRO"

  Scenario: Context-scoped subject config resource
    When I call MCP tool "set_config" with JSON input:
      """
      {"subject": "resctx-cfg-subj", "compatibility_level": "FULL", "context": ".cfgctx"}
      """
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "resctx-cfg-subj",
        "schema": "{\"type\":\"string\"}",
        "context": ".cfgctx"
      }
      """
    Then the MCP result should not be an error
    When I read MCP resource "schema://contexts/.cfgctx/subjects/resctx-cfg-subj/config"
    Then the MCP resource result should contain "FULL"

  Scenario: Context-scoped subject mode resource
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "resctx-mode-subj",
        "schema": "{\"type\":\"string\"}",
        "context": ".modectx"
      }
      """
    Then the MCP result should not be an error
    When I read MCP resource "schema://contexts/.modectx/subjects/resctx-mode-subj/mode"
    Then the MCP resource result should contain "mode"

  # ==========================================================================
  # 2. CONTEXT-SCOPED SCHEMA RESOURCES
  # ==========================================================================

  Scenario: Context-scoped schema by ID resource
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "resctx-sid-subj",
        "schema": "{\"type\":\"string\"}",
        "context": ".sidctx"
      }
      """
    Then the MCP result should not be an error
    And I store the MCP result field "id" as "ctx_schema_id"
    When I read MCP resource "schema://contexts/.sidctx/schemas/$ctx_schema_id"
    Then the MCP resource result should contain "AVRO"

  Scenario: Context-scoped schema subjects resource
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "resctx-ssub-a",
        "schema": "{\"type\":\"string\"}",
        "context": ".ssubctx"
      }
      """
    Then the MCP result should not be an error
    And I store the MCP result field "id" as "ctx_ssub_id"
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "resctx-ssub-b",
        "schema": "{\"type\":\"string\"}",
        "context": ".ssubctx"
      }
      """
    Then the MCP result should not be an error
    When I read MCP resource "schema://contexts/.ssubctx/schemas/$ctx_ssub_id/subjects"
    Then the MCP resource result should contain "resctx-ssub-a"
    And the MCP resource result should contain "resctx-ssub-b"

  Scenario: Context-scoped schema versions resource
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "resctx-sver-subj",
        "schema": "{\"type\":\"string\"}",
        "context": ".sverctx"
      }
      """
    Then the MCP result should not be an error
    And I store the MCP result field "id" as "ctx_sver_id"
    When I read MCP resource "schema://contexts/.sverctx/schemas/$ctx_sver_id/versions"
    Then the MCP resource result should contain "resctx-sver-subj"

  # ==========================================================================
  # 3. CONTEXT-SCOPED CONFIG AND MODE RESOURCES
  # ==========================================================================

  Scenario: Context-scoped global config resource
    When I read MCP resource "schema://contexts/./config"
    Then the MCP resource result should contain "compatibility"
    And the MCP resource result should contain "mode"

  Scenario: Context-scoped global mode resource
    When I read MCP resource "schema://contexts/./mode"
    Then the MCP resource result should contain "mode"

  # ==========================================================================
  # 4. CONTEXT-SCOPED PROMPTS
  # ==========================================================================

  Scenario: Evolve-schema prompt with context enriches from correct context
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "pctx-evolve-subj",
        "schema": "{\"type\":\"record\",\"name\":\"Evolve\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"}]}",
        "schema_type": "AVRO",
        "context": ".promptctx"
      }
      """
    Then the MCP result should not be an error
    When I get MCP prompt "evolve-schema" with arguments:
      | subject | pctx-evolve-subj |
      | context | .promptctx       |
    Then the MCP prompt result should contain "pctx-evolve-subj"
    And the MCP prompt result should contain "version: 1"

  Scenario: Check-compatibility prompt with context enriches from correct context
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "pctx-compat-subj",
        "schema": "{\"type\":\"string\"}",
        "context": ".compatctx"
      }
      """
    Then the MCP result should not be an error
    When I get MCP prompt "check-compatibility" with arguments:
      | subject | pctx-compat-subj |
      | context | .compatctx       |
    Then the MCP prompt result should contain "pctx-compat-subj"
    And the MCP prompt result should contain "compatibility"

  Scenario: Review-schema-quality prompt with context enriches from correct context
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "pctx-quality-subj",
        "schema": "{\"type\":\"string\"}",
        "context": ".qualityctx"
      }
      """
    Then the MCP result should not be an error
    When I get MCP prompt "review-schema-quality" with arguments:
      | subject | pctx-quality-subj |
      | context | .qualityctx       |
    Then the MCP prompt result should contain "pctx-quality-subj"
    And the MCP prompt result should contain "version: 1"

  Scenario: Audit-subject-history prompt with context enriches from correct context
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "pctx-audit-subj",
        "schema": "{\"type\":\"string\"}",
        "context": ".auditctx"
      }
      """
    Then the MCP result should not be an error
    When I get MCP prompt "audit-subject-history" with arguments:
      | subject | pctx-audit-subj |
      | context | .auditctx       |
    Then the MCP prompt result should contain "pctx-audit-subj"
    And the MCP prompt result should contain "[1]"

  Scenario: Impact-analysis prompt with context enriches from correct context
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "pctx-impact-subj",
        "schema": "{\"type\":\"string\"}",
        "context": ".impactctx"
      }
      """
    Then the MCP result should not be an error
    When I get MCP prompt "schema-impact-analysis" with arguments:
      | subject | pctx-impact-subj |
      | context | .impactctx       |
    Then the MCP prompt result should contain "pctx-impact-subj"
    And the MCP prompt result should contain "version: 1"

  # ==========================================================================
  # 5. BACKWARD COMPATIBILITY — DEFAULT CONTEXT
  # ==========================================================================

  Scenario: Non-context resource URIs still work (backward compat)
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "resctx-compat-subj",
        "schema": "{\"type\":\"string\"}"
      }
      """
    Then the MCP result should not be an error
    When I read MCP resource "schema://subjects"
    Then the MCP resource result should contain "resctx-compat-subj"
    When I read MCP resource "schema://subjects/resctx-compat-subj"
    Then the MCP resource result should contain "resctx-compat-subj"
    When I read MCP resource "schema://server/config"
    Then the MCP resource result should contain "compatibility"

  Scenario: Prompts without context argument still use default context
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "pctx-default-subj",
        "schema": "{\"type\":\"string\"}"
      }
      """
    Then the MCP result should not be an error
    When I get MCP prompt "evolve-schema" with arguments:
      | subject | pctx-default-subj |
    Then the MCP prompt result should contain "pctx-default-subj"
    And the MCP prompt result should contain "version: 1"
