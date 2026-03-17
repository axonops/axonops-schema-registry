# Storage Backends

## Overview

The schema registry supports 4 storage backends. The architecture is stateless -- no leader election or distributed coordination is needed. The database handles concurrency via its native mechanisms.

## Backends

### Memory

- **Use case:** Development, testing, single-instance deployments
- **Persistence:** None (data lost on restart)
- **Concurrency:** Go mutexes
- **ID allocation:** Simple atomic counter
- **Config:** `storage.type: memory`

### PostgreSQL

- **Use case:** Production deployments, existing PostgreSQL infrastructure
- **Concurrency:** `INSERT ... ON CONFLICT` for deduplication, `SELECT ... FOR UPDATE` for ID allocation
- **ID allocation:** Single-row counter table with row locking
- **Migrations:** Auto-applied on startup
- **Config:** `storage.type: postgresql` with host, port, database, user, password, ssl_mode
- **Connection pool:** Configurable via `max_open_conns`, `max_idle_conns`, `conn_max_lifetime`

### MySQL

- **Use case:** Production deployments, existing MySQL infrastructure
- **Concurrency:** `INSERT IGNORE` for deduplication, `FOR UPDATE` under REPEATABLE READ isolation
- **ID allocation:** Single-row counter table with `FOR UPDATE` to bypass REPEATABLE READ snapshot isolation
- **Migrations:** Auto-applied on startup
- **Config:** `storage.type: mysql` with host, port, database, user, password, tls
- **Connection pool:** Configurable via `max_open_conns`, `max_idle_conns`, `conn_max_lifetime`

### Cassandra

- **Use case:** Multi-datacenter deployments, high availability, large-scale deployments
- **Concurrency:** Lightweight transactions (LWT / compare-and-set) for writes
- **ID allocation:** Block-based allocation via LWT (reserves blocks of IDs to reduce round trips)
- **Consistency:** Configurable read/write/serial consistency levels
- **Migrations:** Auto-applied when `migrate: true`
- **Config:** `storage.type: cassandra` with hosts, port, keyspace, consistency, local_dc
- **Tuning:** `id_block_size` (default 20), `max_retries` for CAS operations

## Choosing a Backend

| Scenario | Recommended Backend |
|----------|-------------------|
| Development / testing | Memory |
| Simple production deployment | PostgreSQL |
| Existing MySQL infrastructure | MySQL |
| Multi-datacenter / high availability | Cassandra |
| Lowest operational overhead | PostgreSQL |

## Key Characteristics

| Feature | PostgreSQL | MySQL | Cassandra |
|---------|-----------|-------|-----------|
| Deduplication | INSERT ON CONFLICT | INSERT IGNORE | LWT IF NOT EXISTS |
| ID allocation | Row lock (FOR UPDATE) | Row lock (FOR UPDATE) | Block allocation (LWT) |
| Consistency | Strong (ACID) | Strong (ACID) | Tunable (eventual to strong) |
| Multi-DC | Requires external replication | Requires external replication | Native multi-DC |
| Schema migrations | Auto on startup | Auto on startup | Auto on startup |

## Stateless Architecture

Multiple registry instances can connect to the same database without coordination. The database handles all concurrency:

- **Schema deduplication:** Content-addressed via SHA-256 fingerprint. Duplicate registrations return the existing ID.
- **ID uniqueness:** Guaranteed by the database's concurrency mechanisms (row locks, LWT).
- **No leader election:** Any instance can serve any request.
- **Horizontal scaling:** Add more instances behind a load balancer.

## MCP Tools

- **health_check** -- verify storage backend connectivity
- **get_server_info** -- check server version and capabilities
- **get_registry_statistics** -- registry-wide statistics
