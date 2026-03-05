End-to-end CSFLE (Client-Side Field Level Encryption) lifecycle.

## Phase 1: Create a KEK
A KEK references an external KMS key. It wraps (encrypts) the DEKs.

Use **create_kek**:
    name: "prod-kek"
    kms_type: "hcvault" (or aws-kms, azure-kms, gcp-kms, openbao)
    kms_key_id: transit key name or ARN
    kms_props: provider-specific connection details
    shared: false (recommended for client-managed keys)

Verify with **test_kek** to confirm KMS connectivity.

## Phase 2: Create DEKs
A DEK is the actual encryption key, scoped to a schema subject.

Use **create_dek**:
    kek_name: "prod-kek"
    subject: "orders-value"
    algorithm: "AES256_GCM" (or AES128_GCM, AES256_SIV)

The DEK is automatically wrapped by the KEK. The plaintext key material stays on the client.

## Phase 3: Key Rotation
Create a new DEK version for the same subject:

Use **create_dek** again with the same kek_name and subject. A new version is auto-assigned.
- New messages are encrypted with the latest DEK version.
- Old messages remain decryptable with previous DEK versions.

## Phase 4: KMS Key Rotation (Rewrap)
When the underlying KMS key is rotated:

Rewrap existing DEKs so they are encrypted with the new KMS key version. The DEK plaintext stays the same -- only the wrapper changes. No re-encryption of data is needed.

## Phase 5: Cleanup
Soft-delete old DEK versions that are no longer needed:
Use **delete_dek** -- sets deleted=true. Can be undone with undelete_dek.

Permanent delete (irreversible):
Use **delete_dek** with permanent: true.

Delete a KEK only after ALL its DEKs are permanently deleted.

## Algorithm Choice
- **AES256_GCM** (default): strongest confidentiality, non-deterministic
- **AES128_GCM**: lower key size, still authenticated
- **AES256_SIV**: deterministic -- enables equality searches but leaks value equality

## MCP Tools
create_kek, get_kek, list_keks, update_kek, delete_kek, test_kek,
create_dek, get_dek, list_deks, delete_dek, get_dek_versions

For domain knowledge, read: schema://glossary/encryption
