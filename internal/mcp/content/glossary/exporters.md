# Exporters (Schema Linking)

## Overview

Exporters enable **schema linking** -- replicating schemas from one registry context (or the entire registry) to a destination. They are used for disaster recovery, cross-datacenter replication, and environment promotion (e.g., staging to production).

## Exporter Data Model

| Field | Type | Description |
|-------|------|-------------|
| **name** | string | Unique identifier for this exporter |
| **contextType** | string | AUTO, CUSTOM, or NONE |
| **context** | string | Target context name (used with CUSTOM) |
| **subjects** | []string | Subjects to export (empty = all) |
| **subjectRenameFormat** | string | Optional rename pattern for exported subjects |
| **config** | map[string]string | Destination registry connection details |

## Context Types

| Type | Behavior |
|------|----------|
| **AUTO** | Exports all subjects automatically. New subjects are picked up without configuration changes. |
| **CUSTOM** | Exports only specified subjects. Optional rename format controls how subject names appear at the destination. |
| **NONE** | No context prefix on exported subjects. Subjects appear at the destination with their original names. |

## Exporter Lifecycle

Exporters have a state machine with these states:

    STARTING --> RUNNING --> PAUSED
                    |          |
                    v          v
                  ERROR    RUNNING (resume)

| State | Description |
|-------|-------------|
| **STARTING** | Exporter is initializing |
| **RUNNING** | Actively exporting schemas |
| **PAUSED** | Temporarily stopped (can be resumed) |
| **ERROR** | Failed; check status for error details |

## Lifecycle Operations

- **pause_exporter** -- pause a running exporter
- **resume_exporter** -- resume a paused exporter
- **reset_exporter** -- reset exporter state (clears offsets, restarts from beginning)

## Exporter Configuration

The config map contains destination connection details:

| Property | Description |
|----------|-------------|
| schema.registry.url | Destination registry URL |
| basic.auth.credentials.source | Auth method for destination |
| basic.auth.user.info | username:password for destination |

## MCP Tools

- **create_exporter / get_exporter / list_exporters / update_exporter / delete_exporter** -- manage exporters
- **get_exporter_status** -- check exporter state and progress
- **get_exporter_config / update_exporter_config** -- manage exporter configuration
- **pause_exporter / resume_exporter / reset_exporter** -- lifecycle control

## MCP Resources

- schema://exporters -- list all exporter names
- schema://exporters/{name} -- exporter details by name
