# Cassandra Storage Layer — Critical Analysis

**Date:** 2026-02-15
**Scope:** `internal/storage/cassandra/store.go` + `migrations.go`
**Target:** Cassandra 5.0+ (no backwards compatibility required)
**Status:** All 4 phases implemented, CI verified (23/23 jobs green)

---

## Executive Summary

The Cassandra storage layer works correctly but was built with an RDBMS mindset. It suffers from:

1. **Excessive LWT usage** — up to 4 LWT operations per schema registration (each is a Paxos round = ~10ms minimum)
2. **N+1 query patterns** — several operations scan all subjects then query each one individually
3. **Missing batch statements** — multi-table writes without atomicity guarantees
4. **Redundant denormalized tables** — tables that Cassandra 5 SAI indexes can eliminate entirely
5. **Fire-and-forget error handling** — silent failures on data-integrity writes

The good news: the dataset is small (schema registries have hundreds-to-thousands of schemas, not millions), so these problems affect latency more than they affect correctness at current scale. But several have real data-integrity risks that should be fixed regardless.

---

## Current Table Inventory (16 tables)

| # | Table | PK | Purpose |
|---|-------|----|---------|
| 1 | `schemas_by_id` | schema_id (int) | Global schema content by ID |
| 2 | `schemas_by_fingerprint` | fingerprint (text) | Global dedup by content hash |
| 3 | `subject_versions` | (subject), version ASC | Versions within a subject |
| 4 | `subject_latest` | subject (text) | Latest version tracker |
| 5 | `subjects` | (bucket), subject | Bucketed subject listing |
| 6 | `schema_references` | (schema_id), name | Schema dependencies |
| 7 | `references_by_target` | (ref_subject, ref_version), ... | Reverse reference lookup |
| 8 | `subject_configs` | subject (text) | Per-subject config |
| 9 | `global_config` | key (text) | Global config |
| 10 | `modes` | key (text) | Mode settings |
| 11 | `id_alloc` | name (text) | Sequential ID allocation |
| 12-16 | `users_*`, `api_keys_*` | various | Auth tables |

---

## Operation-by-Operation Analysis

### 1. `NextID` — Sequential ID Allocation (LWT)

**Location:** `store.go:170-212`

**Current approach:** Read `id_alloc.next_id`, then CAS update `SET next_id = current + 1 IF next_id = current`. Retries up to 50 times on contention.

**Risk: HIGH — This is the single biggest bottleneck**

- Every `CreateSchema` calls `NextID` → every registration hits an LWT
- LWT = Paxos consensus = minimum ~10-15ms per operation (even on a single node)
- Under concurrent registrations, contention causes retries, each retry is another LWT
- Maximum theoretical throughput: ~50-100 ID allocations/sec on a single partition

**Recommendation: Block-based ID allocation**

Allocate IDs in blocks (e.g., 100 at a time). Each application instance reserves a block via a single LWT, then hands out IDs locally from the block.

```go
type IDAllocator struct {
    mu      sync.Mutex
    current int64
    ceiling int64  // IDs available up to (exclusive)
    block   int64  // Block size (e.g., 100)
}

func (a *IDAllocator) Next(ctx context.Context, store *Store) (int64, error) {
    a.mu.Lock()
    defer a.mu.Unlock()
    if a.current >= a.ceiling {
        // Reserve next block via single LWT
        newCeiling, err := store.reserveIDBlock(ctx, a.block)
        if err != nil {
            return 0, err
        }
        a.current = newCeiling - a.block
        a.ceiling = newCeiling
    }
    id := a.current
    a.current++
    return id, nil
}
```

- **Reduces LWT frequency by 100x** (1 LWT per 100 registrations instead of 1 per registration)
- IDs are still globally unique and monotonically increasing
- Trade-off: gaps in IDs if an instance crashes mid-block (acceptable — Confluent doesn't guarantee gap-free IDs)
- The block size should be configurable (default 100, higher for high-throughput deployments)

---

### 2. `CreateSchema` — Main Write Path

**Location:** `store.go:417-557`

**Current flow:**

```
1. canonicalize + fingerprint
2. ensureGlobalSchema()          → 1-2 LWTs (fingerprint IF NOT EXISTS + schemas_by_id IF NOT EXISTS)
3. findSchemaInSubject()         → Full partition scan of subject_versions
4. CAS loop (up to 50 retries):
   a. getSubjectLatest()         → Read subject_latest
   b. INSERT subject_versions    → LWT (IF NOT EXISTS)
   c. INSERT/UPDATE subject_latest → LWT (IF latest_version = ?)
5. INSERT subjects               → Fire-and-forget, no error check
6. INSERT schema_references      → Fire-and-forget, individual writes, no batch
7. INSERT references_by_target   → Fire-and-forget, individual writes, no batch
```

**Issues identified:**

| Issue | Severity | Description |
|-------|----------|-------------|
| **4 LWTs per registration** | HIGH | NextID + fingerprint + subject_versions + subject_latest. At ~10-15ms each = 40-60ms minimum per registration |
| **Step 5 fire-and-forget** | MEDIUM | If `subjects` INSERT fails, the subject won't appear in `ListSubjects`. No retry, no error propagation |
| **Steps 6-7 fire-and-forget** | HIGH | If reference writes fail, `GetReferencedBy` returns incomplete data. This silently corrupts reference tracking |
| **No batch for steps 5-7** | MEDIUM | These are independent-partition writes that should be in a logged batch for atomicity |
| **Step 3 full scan** | LOW | `findSchemaInSubject` scans all versions looking for a matching schema_id. With SAI on `subject_versions.schema_id`, this becomes a point query |

**Recommendation:**

1. **Reduce LWTs**: Use block-based ID allocation (eliminates 1 LWT). Consider whether `schemas_by_fingerprint` CAS is needed — see SAI table elimination below.
2. **Logged batch for subjects + references**: Steps 5-7 should be a logged batch. These are different partitions, so a logged batch provides atomicity.
3. **Error propagation**: At minimum, steps 5-7 must return errors. Currently a failed reference write means `GetReferencedBy` silently returns wrong data.

```go
// Proposed: batch subjects + references writes
batch := s.session.NewBatch(gocql.LoggedBatch).WithContext(ctx)
batch.Query(`INSERT INTO subjects (bucket, subject) VALUES (?, ?)`, bucket, subject)
for _, ref := range record.References {
    batch.Query(`INSERT INTO schema_references (schema_id, name, ref_subject, ref_version) VALUES (?, ?, ?, ?)`,
        schemaID, ref.Name, ref.Subject, ref.Version)
    batch.Query(`INSERT INTO references_by_target (ref_subject, ref_version, schema_subject, schema_version) VALUES (?, ?, ?, ?)`,
        ref.Subject, ref.Version, subject, newVersion)
}
if err := s.session.ExecuteBatch(batch); err != nil {
    return fmt.Errorf("failed to write subjects/references: %w", err)
}
```

---

### 3. `ensureGlobalSchema` — Dual-Table Global Dedup

**Location:** `store.go:559-620`

**Current approach:** CAS INSERT into `schemas_by_fingerprint` (IF NOT EXISTS), then CAS INSERT into `schemas_by_id` (IF NOT EXISTS). Two separate LWTs.

**Issue: Dual-table consistency gap**

If `schemas_by_fingerprint` succeeds but `schemas_by_id` fails (e.g., timeout), the schema is "half-created". The code handles this with `ensureSchemaByIDExists` on subsequent reads, which is a defensive idempotent repair — but it's a band-aid.

**Recommendation: Eliminate `schemas_by_fingerprint` with SAI**

With Cassandra 5 SAI, we can add an index on `schemas_by_id.fingerprint`:

```sql
CREATE INDEX ON schemas_by_id (fingerprint) USING 'StorageAttachedIndex';
```

Then fingerprint lookups become:
```sql
SELECT schema_id, created_at FROM schemas_by_id WHERE fingerprint = ?
```

**Benefits:**
- **Eliminates 1 table** (schemas_by_fingerprint)
- **Eliminates 1 LWT** per registration (no CAS on fingerprint table)
- **Eliminates dual-table consistency gap** entirely
- **Simplifies cleanup** (no need to delete from both tables on permanent delete)

**Trade-off:**
- SAI lookup is slightly slower than partition key lookup (~1-5ms vs sub-ms)
- For a schema registry with hundreds-to-thousands of schemas, this is negligible
- The fingerprint column is high-cardinality (essentially unique per schema), which is ideal for SAI

**Impact on `ensureGlobalSchema`:**
```go
func (s *Store) ensureGlobalSchema(ctx context.Context, schemaType, schemaText, canonical, fp string) (int64, time.Time, error) {
    // SAI lookup by fingerprint — single query, no LWT
    var existingID int
    var createdUUID gocql.UUID
    err := s.readQuery(
        `SELECT schema_id, created_at FROM schemas_by_id WHERE fingerprint = ?`, fp,
    ).WithContext(ctx).Scan(&existingID, &createdUUID)
    if err == nil {
        return int64(existingID), createdUUID.Time(), nil
    }
    if !errors.Is(err, gocql.ErrNotFound) {
        return 0, time.Time{}, err
    }

    // New schema — allocate ID and insert (still needs IF NOT EXISTS for race safety)
    newID, err := s.NextID(ctx)
    if err != nil {
        return 0, time.Time{}, err
    }
    createdUUID = gocql.TimeUUID()

    applied, err := casApplied(s.session.Query(
        `INSERT INTO schemas_by_id (schema_id, schema_type, fingerprint, schema_text, canonical_text, created_at)
         VALUES (?, ?, ?, ?, ?, ?) IF NOT EXISTS`,
        int(newID), schemaType, fp, schemaText, canonical, createdUUID,
    ).WithContext(ctx))
    if err != nil {
        return 0, time.Time{}, err
    }
    if !applied {
        // Race: someone inserted with this ID. Re-check by fingerprint.
        // (Block-based IDs make this extremely unlikely)
    }
    return newID, createdUUID.Time(), nil
}
```

---

### 4. `GetSchemasBySubject` — N+1 Query Problem

**Location:** `store.go:773-829`

**Current approach:** Read all versions from `subject_versions`, then for **each version** call `GetSchemaByID()`. Each `GetSchemaByID` also reads `schema_references`.

**For a subject with N versions:**
- 1 query to `subject_versions`
- N queries to `schemas_by_id`
- N queries to `schema_references`
- = **2N + 1 queries total**

**Risk: MEDIUM** — With 10 versions, that's 21 queries. With 100 versions, 201 queries.

**Recommendation: IN-clause batch reads**

For small N (typical for schema registries):
```go
// Collect all schema IDs first
schemaIDs := make([]int, 0, len(entries))
for _, e := range entries {
    schemaIDs = append(schemaIDs, e.schemaID)
}

// Single IN query for all schema content
// Note: IN on partition key works well for small sets (<100)
iter := s.readQuery(
    fmt.Sprintf(`SELECT schema_id, schema_type, schema_text, created_at FROM %s.schemas_by_id WHERE schema_id IN ?`, qident(s.cfg.Keyspace)),
    schemaIDs,
).WithContext(ctx).Iter()
```

**Reduces 2N+1 → 3 queries** (versions + schemas IN + references IN).

**Alternative: Denormalize schema_text into subject_versions**

Store `schema_type` and `schema_text` directly in `subject_versions`. This eliminates the join entirely at the cost of duplicated storage. For a schema registry where schema text is typically 1-10KB, this is a reasonable trade-off.

```sql
ALTER TABLE subject_versions ADD schema_type text;
ALTER TABLE subject_versions ADD schema_text text;
ALTER TABLE subject_versions ADD fingerprint text;
```

Then `GetSchemasBySubject` becomes a single partition read.

---

### 5. `ListSubjects` — Full Table Scan + N+1 Filtering

**Location:** `store.go:1083-1125`

**Current approach:**
1. Scan all 16 subject buckets
2. For each subject, read all its versions from `subject_versions` to check if any are non-deleted
3. If `includeDeleted=false`, filter out fully-deleted subjects

**For S subjects:**
- 16 bucket reads + S version scans = **16 + S queries**

**Risk: MEDIUM** — With 500 subjects, that's 516 queries for a single `ListSubjects` call.

**Recommendation: Eliminate `subjects` bucketed table, use SAI instead**

**Option A: SAI on `subject_latest`**

`subject_latest` already has one row per subject. We can scan it directly instead of bucketing:

```sql
CREATE INDEX ON subject_latest (subject) USING 'StorageAttachedIndex';
-- Or simply: SELECT * FROM subject_latest (full table scan, but it's tiny)
```

Actually, `subject_latest` already has `subject` as PK, so a full table scan is just `SELECT subject FROM subject_latest`. The issue is Cassandra full table scans require token-range scanning. But at registry scale (hundreds of subjects), this is fine.

**Better Option B: Add `active_count` to `subject_latest`**

```sql
ALTER TABLE subject_latest ADD active_count int;
```

Maintain `active_count` in `CreateSchema` (+1), `DeleteSchema` soft-delete (-1), undelete (+1). Then:

```sql
-- Without SAI: still need to scan subject_latest, but no per-subject version scan
-- With SAI on active_count:
CREATE INDEX ON subject_latest (active_count) USING 'StorageAttachedIndex';
-- Non-deleted subjects: WHERE active_count > 0
```

This eliminates the N+1 version-scanning entirely. `ListSubjects` becomes 1 query.

**Option C: Keep it simple — token-range scan of `subject_latest` + SAI on `subject_versions.deleted`**

```sql
CREATE INDEX ON subject_versions (deleted) USING 'StorageAttachedIndex';
```

Then `ListSubjects` with `includeDeleted=false`:
```sql
-- Get all subjects from subject_latest (small table)
SELECT subject FROM subject_latest;
-- Then for filtering: use SAI to check existence of non-deleted versions
SELECT subject FROM subject_versions WHERE subject = ? AND deleted = false LIMIT 1
```

But this is still N+1. The best approach is **Option B** with `active_count`.

**Recommendation: Option B** — maintain `active_count` in `subject_latest`.

---

### 6. `GetSubjectsBySchemaID` / `GetVersionsBySchemaID` — O(S×V) Full Scan

**Location:** `store.go:1270-1351`

**Current approach:** Lists ALL subjects via `ListSubjects(true)`, then for EACH subject scans all versions looking for matching `schema_id`.

**This is the worst query pattern in the entire codebase.**

For S subjects with average V versions each: **16 + S + S×V queries** (ListSubjects + 1 version scan per subject).

**Risk: HIGH** — 500 subjects × 10 versions = 5,016 queries for a single API call.

**Fix: SAI on `subject_versions.schema_id`**

```sql
CREATE INDEX ON subject_versions (schema_id) USING 'StorageAttachedIndex';
```

Then:
```go
// GetSubjectsBySchemaID — single query instead of O(S×V)
iter := s.readQuery(
    `SELECT DISTINCT subject FROM subject_versions WHERE schema_id = ?`, schemaID,
).WithContext(ctx).Iter()

// GetVersionsBySchemaID — single query
iter := s.readQuery(
    `SELECT subject, version FROM subject_versions WHERE schema_id = ? AND deleted = ?`, schemaID, false,
).WithContext(ctx).Iter()
```

**Reduces O(S×V) → O(1).** This is the single highest-impact improvement.

---

### 7. `cleanupOrphanedSchema` — Same O(S×V) Problem

**Location:** `store.go:1016-1057`

**Current approach:** Lists ALL subjects, scans EACH subject's versions looking for any remaining reference to `schema_id`. This runs during permanent delete.

**With SAI on `subject_versions.schema_id`:**
```go
func (s *Store) cleanupOrphanedSchema(ctx context.Context, schemaID int) {
    // Single SAI query: does any subject_version still reference this schema?
    var dummy string
    err := s.readQuery(
        `SELECT subject FROM subject_versions WHERE schema_id = ? LIMIT 1`, schemaID,
    ).WithContext(ctx).Scan(&dummy)
    if err == nil {
        return // Still referenced, don't clean up
    }
    // Not referenced — delete from schemas_by_id (and schemas_by_fingerprint if kept)
    // ...
}
```

**Reduces O(S×V) → O(1).**

---

### 8. `GetReferencedBy` — N+1 Soft-Delete Check

**Location:** `store.go:1244-1267`

**Current approach:** Reads `references_by_target`, then for EACH referrer queries `subject_versions` to check if soft-deleted.

**Risk: LOW** (references are typically few) but **architecturally problematic.**

**Option A: Add `deleted` to `references_by_target`**

Maintain deletion state in the references table itself. When soft-deleting a version, also update `references_by_target`. This eliminates the per-referrer lookup.

**Option B: Accept the N+1 for now**

At registry scale (typically 0-5 referrers per schema), this is 1-6 queries. Not worth the added write complexity unless we're optimizing aggressively.

**Recommendation: Option B for now.** The current approach is correct and the N is small.

---

### 9. `ImportSchema` — Multi-Table Write Without Batch

**Location:** `store.go:229-362`

**Current approach:** 6 sequential individual writes to different tables:
1. `schemas_by_id` (if new ID)
2. `schemas_by_fingerprint` (if new ID)
3. `subject_versions`
4. `subject_latest` (conditional)
5. `subjects`
6. `schema_references` + `references_by_target` (per reference)

**All writes except #1-2 silently discard errors.**

**Risk: HIGH** — If any write fails, data is partially written with no cleanup. Import is typically a batch operation (many schemas at once), so a failure mid-import leaves the system in an inconsistent state.

**Recommendation:**

Group writes into logical batches:
```go
// Batch 1: Global schema content (if new ID)
// schemas_by_id [+ schemas_by_fingerprint if kept]

// Batch 2: Subject-version data (logged batch for atomicity)
batch := s.session.NewBatch(gocql.LoggedBatch)
batch.Query(`INSERT INTO subject_versions ...`)
batch.Query(`INSERT INTO subject_latest ...`)
batch.Query(`INSERT INTO subjects ...`)
for _, ref := range record.References {
    batch.Query(`INSERT INTO schema_references ...`)
    batch.Query(`INSERT INTO references_by_target ...`)
}
if err := s.session.ExecuteBatch(batch); err != nil {
    return fmt.Errorf("failed to import schema: %w", err)
}
```

**Error handling:** All writes must propagate errors. Remove `_ =` patterns.

---

### 10. `DeleteSubject` — Loop Calling `DeleteSchema` Individually

**Location:** `store.go:1128-1204`

**Current approach:** For each version, calls `DeleteSchema` individually. For a soft-delete of N versions, that's N separate UPDATE statements.

**Recommendation: Batch soft-deletes**

```go
// For soft-delete: batch all version updates
batch := s.session.NewBatch(gocql.LoggedBatch).WithContext(ctx)
for _, v := range deletedVersions {
    batch.Query(
        `UPDATE subject_versions SET deleted = true WHERE subject = ? AND version = ?`,
        subject, v,
    )
}
if err := s.session.ExecuteBatch(batch); err != nil {
    return nil, err
}
```

This is all within the same partition (`subject` is the partition key), so even an **unlogged batch** would be atomic (single-partition batches are atomic by default in Cassandra).

For permanent deletes, batching is harder due to the cross-table cleanup logic, but the version deletions themselves can still be batched.

---

### 11. `GetLatestSchema` — Unnecessary Full Partition Scan

**Location:** `store.go:917-945`

**Current approach:** Checks `subject_latest` exists, then scans ALL versions in `subject_versions` to find the highest non-deleted version.

**Issue:** `subject_latest` stores the latest version, but we ignore it and re-scan because soft-deleted versions make it unreliable. The latest version in `subject_latest` might be deleted.

**Recommendation: Use `subject_versions` clustering order + SAI**

```sql
-- subject_versions already has `WITH CLUSTERING ORDER BY (version ASC)`
-- We need the latest non-deleted version:
SELECT version FROM subject_versions WHERE subject = ? AND deleted = false ORDER BY version DESC LIMIT 1
```

With SAI on `subject_versions.deleted`, this becomes an efficient single-partition query with reverse clustering order.

Without SAI, the current approach works but scans the entire partition. For subjects with many versions (e.g., 100+), this is wasteful.

---

### 12. `SubjectExists` — Same Unnecessary Scan

**Location:** `store.go:1207-1228`

**Current approach:** Checks `subject_latest`, then scans ALL versions to see if any are non-deleted.

**Same fix as #11:** SAI on `deleted` + LIMIT 1.

```go
var version int
err := s.readQuery(
    `SELECT version FROM subject_versions WHERE subject = ? AND deleted = false LIMIT 1`, subject,
).WithContext(ctx).Scan(&version)
if err == nil {
    return true, nil
}
```

---

### 13. `GetSchemaByFingerprint` — Full Partition Scan

**Location:** `store.go:832-888`

**Current approach:** Gets schema by global fingerprint, then scans ALL versions of the subject looking for matching schema_id.

**With SAI on `subject_versions.schema_id`:**
```go
// Single query instead of full partition scan
var version int
var deleted bool
err := s.readQuery(
    `SELECT version, deleted FROM subject_versions WHERE subject = ? AND schema_id = ?`,
    subject, int(globalRec.ID),
).WithContext(ctx).Scan(&version, &deleted)
```

---

### 14. `findSchemaInSubject` — Full Partition Scan

**Location:** `store.go:638-658`

**Current approach:** Scans all versions of a subject looking for a matching schema_id (non-deleted).

**Same fix as #13:** SAI on `subject_versions.schema_id` + filter on `deleted`.

---

### 15. Fire-and-Forget Error Handling — Comprehensive List

Every `_ =` or `_ = s.writeQuery(...)` in the codebase where the error is silently discarded:

| Location | Operation | Severity |
|----------|-----------|----------|
| `store.go:339-342` | `ImportSchema` → subjects INSERT | MEDIUM |
| `store.go:347-349` | `ImportSchema` → schema_references INSERT | HIGH |
| `store.go:351-354` | `ImportSchema` → references_by_target INSERT | HIGH |
| `store.go:528-531` | `CreateSchema` → subjects INSERT | MEDIUM |
| `store.go:536-539` | `CreateSchema` → schema_references INSERT | HIGH |
| `store.go:540-543` | `CreateSchema` → references_by_target INSERT | HIGH |
| `store.go:1043-1046` | `cleanupOrphanedSchema` → schemas_by_fingerprint DELETE | LOW (cleanup) |
| `store.go:1048-1051` | `cleanupOrphanedSchema` → schemas_by_id DELETE | LOW (cleanup) |
| `store.go:1053-1056` | `cleanupOrphanedSchema` → schema_references DELETE | LOW (cleanup) |
| `store.go:1072-1075` | `cleanupReferencesByTarget` → references_by_target DELETE | LOW (cleanup) |
| `store.go:1190-1193` | `DeleteSubject` → subject_latest DELETE | MEDIUM |
| `store.go:1196-1199` | `DeleteSubject` → subjects DELETE | MEDIUM |

**Recommendation:** All HIGH-severity fire-and-forget writes should propagate errors. Cleanup operations (LOW) can remain best-effort but should at minimum log warnings.

---

## SAI Index Recommendations

### Indexes to Create

```sql
-- #1: Eliminates O(S×V) full scans in GetSubjectsBySchemaID, GetVersionsBySchemaID,
-- cleanupOrphanedSchema, findSchemaInSubject, GetSchemaByFingerprint
CREATE INDEX ON subject_versions (schema_id) USING 'StorageAttachedIndex';

-- #2: Enables efficient filtering of deleted versions in ListSubjects,
-- SubjectExists, GetLatestSchema
CREATE INDEX ON subject_versions (deleted) USING 'StorageAttachedIndex';

-- #3: Eliminates schemas_by_fingerprint table entirely
CREATE INDEX ON schemas_by_id (fingerprint) USING 'StorageAttachedIndex';
```

### Tables That Can Be Eliminated

| Table | Replaced By | Savings |
|-------|-------------|---------|
| `schemas_by_fingerprint` | SAI on `schemas_by_id.fingerprint` | Eliminates 1 table, 1 LWT per registration, dual-table consistency issue |
| `subjects` (bucketed) | Token-range scan of `subject_latest` + `active_count` column | Eliminates 1 table, removes bucket management code |

### Tables to Keep

| Table | Reason |
|-------|--------|
| `schemas_by_id` | Primary lookup by ID — must be partition key for performance |
| `subject_versions` | Core partition design — subject as PK, version as CK |
| `subject_latest` | Avoids scanning subject_versions for latest — essential for write path CAS |
| `schema_references` | Small table, efficient partition access by schema_id |
| `references_by_target` | Reverse lookup for reference checking — no efficient SAI alternative |
| `subject_configs` | Simple key-value, fine as-is |
| `global_config` | Simple key-value, fine as-is |
| `modes` | Simple key-value, fine as-is |
| `id_alloc` | Needed for LWT-based ID allocation |
| Auth tables (5) | Already correctly using logged batches |

---

## Batch Statement Opportunities

### Writes That Should Use Logged Batches

| Operation | Tables Involved | Current | Proposed |
|-----------|----------------|---------|----------|
| `CreateSchema` steps 5-7 | subjects + schema_references + references_by_target | Individual fire-and-forget | Logged batch with error propagation |
| `ImportSchema` steps 3-6 | subject_versions + subject_latest + subjects + references | Individual fire-and-forget | Logged batch with error propagation |
| `DeleteSubject` soft-delete | N × subject_versions UPDATE | Individual writes in loop | **Unlogged** batch (same partition = atomic) |
| `DeleteSubject` permanent | subject_latest + subjects DELETE | Individual fire-and-forget | Logged batch |

### Writes Already Correctly Batched

| Operation | Tables | Batch Type |
|-----------|--------|------------|
| `CreateUser` | users_by_id + users_by_email | Logged |
| `UpdateUser` | users_by_id + users_by_email | Logged |
| `DeleteUser` | users_by_id + users_by_email | Logged |
| `CreateAPIKey` | api_keys_by_id + by_user + by_hash | Logged |
| `UpdateAPIKey` | api_keys_by_id + by_user + by_hash | Logged |
| `DeleteAPIKey` | api_keys_by_id + by_user + by_hash | Logged |
| `UpdateAPIKeyLastUsed` | api_keys_by_id + by_user + by_hash | Logged |

The auth operations are a good model for how the schema operations should work.

### Unlogged Batch Opportunities (Same Partition)

Single-partition operations are inherently atomic in Cassandra. Using an unlogged batch for same-partition writes provides atomicity without the coordination overhead of logged batches:

```go
// Soft-delete all versions of a subject — same partition
batch := s.session.NewBatch(gocql.UnloggedBatch).WithContext(ctx)
for _, v := range versions {
    batch.Query(`UPDATE subject_versions SET deleted = true WHERE subject = ? AND version = ?`, subject, v)
}
s.session.ExecuteBatch(batch) // Atomic — single partition
```

---

## LWT Usage Audit

### Current LWT Usage

| Method | LWT Operation | Justified? |
|--------|--------------|------------|
| `NextID` | `UPDATE id_alloc SET next_id = ? IF next_id = ?` | **Yes, but reduce with block allocation** |
| `NextID` (init) | `INSERT id_alloc IF NOT EXISTS` | Yes |
| `SetNextID` | `UPDATE id_alloc SET next_id = ? IF next_id = ?` | Yes (infrequent, import-only) |
| `ensureGlobalSchema` | `INSERT schemas_by_fingerprint IF NOT EXISTS` | **No — eliminate with SAI** |
| `ensureSchemaByIDExists` | `INSERT schemas_by_id IF NOT EXISTS` | Partly — idempotent repair, could simplify |
| `CreateSchema` step 4a | `INSERT subject_versions IF NOT EXISTS` | **Yes — essential for version uniqueness** |
| `CreateSchema` step 4b | `INSERT/UPDATE subject_latest IF (NOT EXISTS / latest_version = ?)` | **Yes — essential for version ordering** |

### LWT Reduction Summary

| Change | LWTs Removed | Risk |
|--------|-------------|------|
| Block-based ID allocation | ~99% of NextID LWTs | None (IDs still unique, gaps acceptable) |
| SAI on fingerprint (eliminate schemas_by_fingerprint) | 1 LWT per registration | SAI lookup slightly slower but adequate |
| **Net reduction** | **From 4 LWTs/registration to 2** | Low |

The remaining 2 LWTs (`subject_versions IF NOT EXISTS` + `subject_latest IF/CAS`) are genuinely necessary for correctness — they ensure version uniqueness and ordering under concurrent writes.

---

## Performance Improvement Summary

| Improvement | Operations Affected | Impact |
|-------------|-------------------|--------|
| SAI on `subject_versions.schema_id` | GetSubjectsBySchemaID, GetVersionsBySchemaID, cleanupOrphanedSchema, findSchemaInSubject | **O(S×V) → O(1)** |
| SAI on `subject_versions.deleted` | ListSubjects, SubjectExists, GetLatestSchema | **O(S) → O(1)** for filtering |
| SAI on `schemas_by_id.fingerprint` | ensureGlobalSchema, all fingerprint lookups | Eliminates 1 table + 1 LWT |
| Block-based ID allocation | CreateSchema | **~100x fewer LWTs** |
| Logged batches for references | CreateSchema, ImportSchema | Data integrity (was fire-and-forget) |
| IN-clause reads for GetSchemasBySubject | GetSchemasBySubject | **2N+1 → 3 queries** |
| Unlogged batch for soft-delete | DeleteSubject | **N → 1 query** (same partition) |

---

## Recommended Implementation Order

### Phase 1: SAI Indexes + Table Elimination (Highest Impact, Lowest Risk)
1. Add SAI indexes in migrations
2. Replace `schemas_by_fingerprint` with SAI on `schemas_by_id.fingerprint`
3. Rewrite `GetSubjectsBySchemaID` / `GetVersionsBySchemaID` / `cleanupOrphanedSchema` using SAI
4. Simplify `ListSubjects` using SAI on `deleted`

### Phase 2: Batch Statements + Error Handling (Data Integrity)
1. Add logged batches for reference writes in `CreateSchema` and `ImportSchema`
2. Add unlogged batch for `DeleteSubject` soft-delete
3. Fix all fire-and-forget writes to propagate errors
4. Add logged batch for permanent delete cleanup

### Phase 3: LWT Reduction (Performance)
1. Implement block-based ID allocation
2. Simplify `ensureGlobalSchema` to use SAI instead of LWT on fingerprint
3. Review and simplify the CreateSchema CAS loop

### Phase 4: N+1 Query Optimization (Read Performance)
1. Use IN-clause for `GetSchemasBySubject` schema content reads
2. Optimize `GetLatestSchema` with SAI-aware reverse scan
3. Optimize `findSchemaInSubject` with SAI point query

---

## Risk Assessment

| Risk | Mitigation |
|------|-----------|
| SAI index creation on existing data | SAI builds index in background from existing SSTables. Non-disruptive. |
| Removing `schemas_by_fingerprint` table | Migration can drop after verifying SAI index is built. Include verification step. |
| Block-based IDs create gaps | Confluent doesn't guarantee gap-free IDs. BDD tests don't depend on sequential IDs. |
| Logged batch size limits | Cassandra warns at 5KB default batch size. Schema reference batches should be small (few references per schema). Monitor. |
| SAI query performance on large datasets | Schema registries are small (100s-1000s of schemas). SAI is more than adequate. |

---

## Appendix: Current Query Count Per API Operation

| API Operation | Current Queries | With SAI | Improvement |
|---------------|----------------|----------|-------------|
| Register schema | ~8-12 (4 LWT + reads) | ~6-8 (2 LWT + reads + batch) | ~30% fewer |
| Get schema by ID | 3 (schema + refs + subject lookup) | 2-3 (same, but subject lookup via SAI) | Marginal |
| Get schema by subject/version | 4 (version + schema + refs + maybe subject) | 3-4 (same) | Marginal |
| List versions of subject | 2N+1 (versions + N×schema + N×refs) | 3 (versions + IN schema + IN refs) | **~90% fewer** |
| List all subjects | 16 + S (buckets + version scans) | 1-2 (subject_latest scan + SAI filter) | **~99% fewer** |
| Get subjects by schema ID | 16 + S + S×V | 1 (SAI query) | **~99.9% fewer** |
| Delete subject (soft, N versions) | N+1 (read + N updates) | 2 (read + batch) | **~50-90% fewer** |
| Import schema | ~8-10 individual writes | ~4-5 (batches) | Data integrity fix |

---

## Implementation Status

**All 4 phases implemented and verified.** CI run `22052167266` — 23/23 jobs green.

### Phase 1: SAI Indexes + Table Elimination — DONE
- Added 3 SAI indexes: `idx_schemas_fingerprint`, `idx_sv_schema_id`, `idx_sv_deleted`
- Eliminated `schemas_by_fingerprint` and `subjects` tables (DROP TABLE in migration)
- Rewrote `ensureGlobalSchema`, `findSchemaInSubject`, `GetSchemaByFingerprint`, `GetSchemaByGlobalFingerprint`, `GetLatestSchema`, `SubjectExists`, `GetSubjectsBySchemaID`, `GetVersionsBySchemaID`, `cleanupOrphanedSchema`, `ListSubjects` to use SAI queries
- Removed `subjectBucket` helper, `SubjectBuckets` config, `ensureSchemaByIDExists` helper
- Reduced table count from 16 to 14

### Phase 2: Batch Statements + Error Handling — DONE
- Logged batches for reference writes in `CreateSchema` and `ImportSchema`
- Unlogged batch for soft-deletes in `DeleteSubject`
- `slog.Warn` for cleanup errors (best-effort operations)

### Phase 3: Block-Based ID Allocation — DONE
- Added `idAllocator` struct with `reserveIDBlock` method
- Configurable `IDBlockSize` (default 100)
- ~100x fewer LWTs for ID allocation

### Phase 4: N+1 Query Optimization — DONE
- `GetSchemasBySubject` uses IN-clause batch reads (2N+1 → 3 queries)
- `GetSchemaByID` uses SAI instead of `GetVersionsBySchemaID`

### CI Fixes Applied
- Upgraded Cassandra from 4.1 to 5.0 (SAI requires 5.0+)
- Fixed concurrent idempotency race in `CreateSchema` CAS retry loop
- Removed dropped tables from BDD/conformance truncation lists
- Re-seed `id_alloc` after BDD truncation for `GetMaxSchemaID`
- Reuse long-lived Cassandra session across BDD scenario cleanups

### Commits
- `92a4255` — refactor(cassandra): optimize storage layer with SAI indexes, batch writes, and block-based IDs
- `1faf68a` — ci: upgrade Cassandra from 4.1 to 5.0 for SAI index support
- `7d6be2b` — fix(cassandra): resolve concurrent idempotency race and remove dropped tables from cleanup
- `631e77c` — fix(cassandra): re-seed id_alloc after BDD cleanup for GetMaxSchemaID
- `52735ea` — perf: reuse Cassandra session across BDD scenario cleanups
