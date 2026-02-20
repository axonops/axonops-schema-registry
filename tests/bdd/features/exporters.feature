@functional
Feature: Exporters API
  The Exporters API provides Confluent Schema Linking compatible functionality
  for exporting schemas to remote schema registries. Exporters can be created,
  configured, paused, resumed, reset, and deleted.

  Background:
    Given the schema registry is running

  # List Operations
  Scenario: List exporters when none exist
    When I GET "/exporters"
    Then the response status should be 200
    And the response should be valid JSON
    And the response should be an array of length 0

  Scenario: List exporters after creating several
    When I POST "/exporters" with body:
      """
      {
        "name": "exporter-1",
        "contextType": "AUTO",
        "subjects": ["test-1"]
      }
      """
    And I POST "/exporters" with body:
      """
      {
        "name": "exporter-2",
        "contextType": "NONE",
        "subjects": ["test-2"]
      }
      """
    And I POST "/exporters" with body:
      """
      {
        "name": "exporter-3",
        "contextType": "AUTO",
        "subjects": ["test-3"]
      }
      """
    And I GET "/exporters"
    Then the response status should be 200
    And the response should be valid JSON
    And the response should be an array of length 3
    And the response array should contain "exporter-1"
    And the response array should contain "exporter-2"
    And the response array should contain "exporter-3"

  # Create Operations
  Scenario: Create exporter with all fields
    When I POST "/exporters" with body:
      """
      {
        "name": "full-exporter",
        "contextType": "CUSTOM",
        "context": "prod",
        "subjects": ["orders-value", "users-value"],
        "subjectRenameFormat": "staging.${subject}",
        "config": {
          "schema.registry.url": "http://remote:8081",
          "basic.auth.credentials.source": "USER_INFO",
          "basic.auth.user.info": "admin:secret"
        }
      }
      """
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "name" should be "full-exporter"

  Scenario: Create exporter with minimal fields
    When I POST "/exporters" with body:
      """
      {
        "name": "minimal-exporter"
      }
      """
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "name" should be "minimal-exporter"

  Scenario: Create exporter with contextType AUTO
    When I POST "/exporters" with body:
      """
      {
        "name": "auto-exporter",
        "contextType": "AUTO",
        "subjects": ["test-value"]
      }
      """
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "name" should be "auto-exporter"

  Scenario: Create exporter with contextType CUSTOM and context
    When I POST "/exporters" with body:
      """
      {
        "name": "custom-exporter",
        "contextType": "CUSTOM",
        "context": "production",
        "subjects": ["test-value"]
      }
      """
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "name" should be "custom-exporter"

  Scenario: Create exporter with contextType NONE
    When I POST "/exporters" with body:
      """
      {
        "name": "none-exporter",
        "contextType": "NONE",
        "subjects": ["test-value"]
      }
      """
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "name" should be "none-exporter"

  Scenario: Create exporter with subject filters
    When I POST "/exporters" with body:
      """
      {
        "name": "filtered-exporter",
        "contextType": "AUTO",
        "subjects": ["orders-*", "users-*", "events-*"]
      }
      """
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "name" should be "filtered-exporter"

  Scenario: Create exporter with multiple subjects
    When I POST "/exporters" with body:
      """
      {
        "name": "multi-subject-exporter",
        "contextType": "AUTO",
        "subjects": ["subject-1", "subject-2", "subject-3", "subject-4", "subject-5"]
      }
      """
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "name" should be "multi-subject-exporter"

  Scenario: Create exporter with empty config
    When I POST "/exporters" with body:
      """
      {
        "name": "empty-config-exporter",
        "contextType": "AUTO",
        "subjects": ["test-value"],
        "config": {}
      }
      """
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "name" should be "empty-config-exporter"

  Scenario: Create exporter returns name in response
    When I POST "/exporters" with body:
      """
      {
        "name": "response-check-exporter",
        "contextType": "AUTO",
        "subjects": ["test-value"]
      }
      """
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "name" should be "response-check-exporter"

  Scenario: Create duplicate exporter
    When I POST "/exporters" with body:
      """
      {
        "name": "duplicate-exporter",
        "contextType": "AUTO",
        "subjects": ["test-value"]
      }
      """
    And I POST "/exporters" with body:
      """
      {
        "name": "duplicate-exporter",
        "contextType": "AUTO",
        "subjects": ["test-value"]
      }
      """
    Then the response status should be 409
    And the response should be valid JSON
    And the response should have error code 40950

  Scenario: Create exporter with invalid contextType
    When I POST "/exporters" with body:
      """
      {
        "name": "invalid-context-exporter",
        "contextType": "INVALID",
        "subjects": ["test-value"]
      }
      """
    Then the response status should be 422
    And the response should be valid JSON

  Scenario: Create exporter with empty name
    When I POST "/exporters" with body:
      """
      {
        "name": "",
        "contextType": "AUTO",
        "subjects": ["test-value"]
      }
      """
    Then the response status should be 422
    And the response should be valid JSON

  # Get Operations
  Scenario: Get exporter by name
    When I POST "/exporters" with body:
      """
      {
        "name": "get-test-exporter",
        "contextType": "AUTO",
        "context": "",
        "subjects": ["test-value"],
        "subjectRenameFormat": "${subject}",
        "config": {
          "schema.registry.url": "http://remote:8081"
        }
      }
      """
    And I GET "/exporters/get-test-exporter"
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "name" should be "get-test-exporter"
    And the response field "contextType" should be "AUTO"

  Scenario: Get non-existent exporter
    When I GET "/exporters/non-existent-exporter"
    Then the response status should be 404
    And the response should be valid JSON
    And the response should have error code 40450

  # Update Operations
  Scenario: Update exporter
    When I POST "/exporters" with body:
      """
      {
        "name": "update-test-exporter",
        "contextType": "AUTO",
        "subjects": ["test-value"]
      }
      """
    And I PUT "/exporters/update-test-exporter" with body:
      """
      {
        "contextType": "CUSTOM",
        "context": "updated",
        "subjects": ["new-test-value"],
        "subjectRenameFormat": "updated.${subject}"
      }
      """
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "name" should be "update-test-exporter"

  Scenario: Update exporter subjects
    When I POST "/exporters" with body:
      """
      {
        "name": "subject-update-exporter",
        "contextType": "AUTO",
        "subjects": ["old-subject"]
      }
      """
    And I PUT "/exporters/subject-update-exporter" with body:
      """
      {
        "subjects": ["new-subject-1", "new-subject-2"]
      }
      """
    Then the response status should be 200
    And the response should be valid JSON

  Scenario: Update exporter without changing all fields
    When I POST "/exporters" with body:
      """
      {
        "name": "partial-update-exporter",
        "contextType": "AUTO",
        "subjects": ["test-value"],
        "subjectRenameFormat": "${subject}"
      }
      """
    And I PUT "/exporters/partial-update-exporter" with body:
      """
      {
        "contextType": "NONE"
      }
      """
    Then the response status should be 200
    And the response should be valid JSON

  # Delete Operations
  Scenario: Delete exporter
    When I POST "/exporters" with body:
      """
      {
        "name": "delete-test-exporter",
        "contextType": "AUTO",
        "subjects": ["test-value"]
      }
      """
    And I DELETE "/exporters/delete-test-exporter"
    Then the response status should be 200

  Scenario: Delete exporter returns name in response
    When I POST "/exporters" with body:
      """
      {
        "name": "delete-response-exporter",
        "contextType": "AUTO",
        "subjects": ["test-value"]
      }
      """
    And I DELETE "/exporters/delete-response-exporter"
    Then the response status should be 200

  Scenario: Delete non-existent exporter
    When I DELETE "/exporters/non-existent-delete-exporter"
    Then the response status should be 404
    And the response should be valid JSON
    And the response should have error code 40450

  # Pause/Resume/Reset Operations
  Scenario: Pause exporter
    When I POST "/exporters" with body:
      """
      {
        "name": "pause-test-exporter",
        "contextType": "AUTO",
        "subjects": ["test-value"]
      }
      """
    And I PUT "/exporters/pause-test-exporter/pause" with body:
      """
      {}
      """
    Then the response status should be 200

  Scenario: Resume exporter
    When I POST "/exporters" with body:
      """
      {
        "name": "resume-test-exporter",
        "contextType": "AUTO",
        "subjects": ["test-value"]
      }
      """
    And I PUT "/exporters/resume-test-exporter/pause" with body:
      """
      {}
      """
    And I PUT "/exporters/resume-test-exporter/resume" with body:
      """
      {}
      """
    Then the response status should be 200

  Scenario: Reset exporter
    When I POST "/exporters" with body:
      """
      {
        "name": "reset-test-exporter",
        "contextType": "AUTO",
        "subjects": ["test-value"]
      }
      """
    And I PUT "/exporters/reset-test-exporter/reset" with body:
      """
      {}
      """
    Then the response status should be 200

  Scenario: Pause non-existent exporter
    When I PUT "/exporters/non-existent-pause/pause" with body:
      """
      {}
      """
    Then the response status should be 404
    And the response should be valid JSON
    And the response should have error code 40450

  Scenario: Resume non-existent exporter
    When I PUT "/exporters/non-existent-resume/resume" with body:
      """
      {}
      """
    Then the response status should be 404
    And the response should be valid JSON
    And the response should have error code 40450

  Scenario: Reset non-existent exporter
    When I PUT "/exporters/non-existent-reset/reset" with body:
      """
      {}
      """
    Then the response status should be 404
    And the response should be valid JSON
    And the response should have error code 40450

  # Status Operations
  Scenario: Get exporter status
    When I POST "/exporters" with body:
      """
      {
        "name": "status-test-exporter",
        "contextType": "AUTO",
        "subjects": ["test-value"]
      }
      """
    And I GET "/exporters/status-test-exporter/status"
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "name" should be "status-test-exporter"

  Scenario: Get status of non-existent exporter
    When I GET "/exporters/non-existent-status/status"
    Then the response status should be 404
    And the response should be valid JSON
    And the response should have error code 40450

  # Config Operations
  Scenario: Get exporter config
    When I POST "/exporters" with body:
      """
      {
        "name": "config-test-exporter",
        "contextType": "AUTO",
        "subjects": ["test-value"],
        "config": {
          "schema.registry.url": "http://remote:8081",
          "timeout.ms": "30000"
        }
      }
      """
    And I GET "/exporters/config-test-exporter/config"
    Then the response status should be 200
    And the response should be valid JSON

  Scenario: Update exporter config
    When I POST "/exporters" with body:
      """
      {
        "name": "config-update-exporter",
        "contextType": "AUTO",
        "subjects": ["test-value"],
        "config": {
          "schema.registry.url": "http://old:8081"
        }
      }
      """
    And I PUT "/exporters/config-update-exporter/config" with body:
      """
      {
        "config": {
          "schema.registry.url": "http://new:8081",
          "timeout.ms": "60000"
        }
      }
      """
    Then the response status should be 200
    And the response should be valid JSON

  Scenario: Get config of non-existent exporter
    When I GET "/exporters/non-existent-config/config"
    Then the response status should be 404
    And the response should be valid JSON
    And the response should have error code 40450

  Scenario: Update config of non-existent exporter
    When I PUT "/exporters/non-existent-config-update/config" with body:
      """
      {
        "config": {
          "schema.registry.url": "http://remote:8081"
        }
      }
      """
    Then the response status should be 404
    And the response should be valid JSON
    And the response should have error code 40450

  # Lifecycle Scenario
  Scenario: Exporter lifecycle - create, pause, resume, reset, delete
    When I POST "/exporters" with body:
      """
      {
        "name": "lifecycle-exporter",
        "contextType": "AUTO",
        "subjects": ["test-value"],
        "config": {
          "schema.registry.url": "http://remote:8081"
        }
      }
      """
    Then the response status should be 200
    And the response field "name" should be "lifecycle-exporter"

    When I GET "/exporters/lifecycle-exporter/status"
    Then the response status should be 200
    And the response field "name" should be "lifecycle-exporter"

    When I PUT "/exporters/lifecycle-exporter/pause" with body:
      """
      {}
      """
    Then the response status should be 200

    When I PUT "/exporters/lifecycle-exporter/resume" with body:
      """
      {}
      """
    Then the response status should be 200

    When I PUT "/exporters/lifecycle-exporter/reset" with body:
      """
      {}
      """
    Then the response status should be 200

    When I DELETE "/exporters/lifecycle-exporter"
    Then the response status should be 200

    When I GET "/exporters/lifecycle-exporter"
    Then the response status should be 404
    And the response should have error code 40450
