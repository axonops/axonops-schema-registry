Set up schema linking with a {exporter_type} context exporter.

Steps:
1. Create an exporter using the create_exporter tool:
   - name: descriptive name (e.g. "prod-to-dr")
   - context_type: {exporter_type}
   - subjects: list of subjects to export (empty = all)
   - config: destination registry connection details

2. Monitor the exporter using get_exporter_status
3. Control the exporter: pause_exporter, resume_exporter, reset_exporter

Context types:
- AUTO: exports all subjects automatically
- CUSTOM: exports only specified subjects with optional rename format
- NONE: no context prefix on exported subjects

Config properties:
- schema.registry.url: destination registry URL
- basic.auth.credentials.source: auth method
- basic.auth.user.info: username:password

Available tools: create_exporter, get_exporter, list_exporters, get_exporter_status, pause_exporter, resume_exporter
