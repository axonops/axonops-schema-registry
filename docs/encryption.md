# DEK Registry (Client-Side Field Level Encryption)

Client-Side Field Level Encryption (CSFLE) allows producers to encrypt specific fields in a message before sending it to Kafka. Instead of encrypting entire topics or relying on broker-side encryption, CSFLE targets individual fields -- an email address, a social security number, a credit card number -- leaving the rest of the message in plaintext for indexing, filtering, and debugging.

The schema registry acts as the **key metadata store** for CSFLE. It does not perform encryption itself. Instead, it stores two types of key metadata that serializer/deserializer (SerDe) clients use to encrypt and decrypt fields:

| Key Type | Purpose | Where the actual key lives |
|----------|---------|---------------------------|
| **Key Encryption Key (KEK)** | References an external KMS key used to wrap (encrypt) DEKs | External KMS (AWS KMS, Azure Key Vault, GCP KMS, HashiCorp Vault) |
| **Data Encryption Key (DEK)** | The actual encryption key used to encrypt field values | Stored in the registry as encrypted bytes (wrapped by the KEK) |

This two-tier design follows the **envelope encryption** pattern: DEKs encrypt data, KEKs encrypt DEKs. The plaintext DEK never leaves the client -- the registry only stores the KMS-wrapped (encrypted) form.

This feature is **Confluent-compatible**: the DEK Registry API follows the same endpoints, request/response formats, and semantics as Confluent Schema Registry's Enterprise CSFLE feature. In AxonOps Schema Registry, it is available at no additional cost.

## Contents

- [Key Encryption Keys (KEKs)](#key-encryption-keys-keks)
- [Data Encryption Keys (DEKs)](#data-encryption-keys-deks)
- [Creating and Managing KEKs](#creating-and-managing-keks)
- [Creating and Managing DEKs](#creating-and-managing-deks)
- [Soft-Delete and Undelete](#soft-delete-and-undelete)
- [Supported KMS Types](#supported-kms-types)
- [Supported Algorithms](#supported-algorithms)
- [API Reference](#api-reference)

---

## Key Encryption Keys (KEKs)

A KEK is a reference to an external key in a Key Management Service (KMS). The registry does not store the actual KEK material -- it stores a pointer (the `kmsKeyId`) that the client uses to call the KMS for wrap/unwrap operations.

A KEK has the following fields:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Unique identifier for this KEK. Used by DEKs to reference their parent KEK. |
| `kmsType` | string | Yes | KMS provider type: `aws-kms`, `azure-kms`, `gcp-kms`, `hcvault`, or `openbao`. |
| `kmsKeyId` | string | Yes | KMS-specific key identifier (e.g., an AWS KMS ARN, a GCP KMS resource name). |
| `kmsProps` | map | No | Additional KMS-specific properties (e.g., region, endpoint, authentication parameters). |
| `doc` | string | No | Human-readable documentation string describing this KEK's purpose. |
| `shared` | boolean | No | If `true`, all DEKs under this KEK share the same underlying key material. Defaults to `false`. |
| `ts` | integer | Read-only | Last modification timestamp (epoch milliseconds). Set by the registry. |
| `deleted` | boolean | Read-only | Soft-delete flag. Set via the `DELETE` endpoint. |

> The `shared` flag controls whether DEK creation returns plaintext key material. When `shared` is `true`, the plaintext `keyMaterial` is returned on DEK creation so the client can reuse the same key across multiple subjects. When `false`, the client generates its own key material and only sends the encrypted form to the registry.

---

## Data Encryption Keys (DEKs)

A DEK is the actual encryption key used to encrypt and decrypt field values. DEKs are always associated with a parent KEK and are scoped to a specific schema subject. The registry stores the DEK in its encrypted (wrapped) form -- the plaintext key material is never persisted.

A DEK has the following fields:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `kekName` | string | Yes | Name of the parent KEK that wraps this DEK. |
| `subject` | string | Yes | Schema subject this DEK encrypts for. Ties the encryption key to a specific data topic. |
| `version` | integer | No | Version number. Auto-assigned as the next sequential version if not specified. |
| `algorithm` | string | No | Encryption algorithm. One of `AES256_GCM` (default), `AES128_GCM`, or `AES256_SIV`. |
| `encryptedKeyMaterial` | string | No | The DEK bytes encrypted (wrapped) by the parent KEK. Base64-encoded. |
| `keyMaterial` | string | Read-only | Plaintext DEK bytes. Returned only on creation when the parent KEK has `shared=true`. Never persisted by the registry. |
| `ts` | integer | Read-only | Last modification timestamp (epoch milliseconds). Set by the registry. |
| `deleted` | boolean | Read-only | Soft-delete flag. Set via the `DELETE` endpoint. |

DEKs are versioned per subject under a KEK. This allows key rotation: when you create a new DEK version for a subject, new messages are encrypted with the latest version while old messages can still be decrypted with the previous version.

---

## Creating and Managing KEKs

### Create a KEK

Register a new KEK referencing an AWS KMS key:

```bash
curl -X POST http://localhost:8081/dek-registry/v1/keks \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "name": "my-aws-kek",
    "kmsType": "aws-kms",
    "kmsKeyId": "arn:aws:kms:us-east-1:123456789012:key/abcd-1234-efgh-5678",
    "kmsProps": {
      "KeyState": "Enabled"
    },
    "doc": "Production encryption key for PII fields",
    "shared": false
  }'
```

Response:

```json
{
  "name": "my-aws-kek",
  "kmsType": "aws-kms",
  "kmsKeyId": "arn:aws:kms:us-east-1:123456789012:key/abcd-1234-efgh-5678",
  "kmsProps": {
    "KeyState": "Enabled"
  },
  "doc": "Production encryption key for PII fields",
  "shared": false,
  "ts": 1708444800000,
  "deleted": false
}
```

### List KEK Names

```bash
curl http://localhost:8081/dek-registry/v1/keks
```

Response:

```json
["my-aws-kek", "my-gcp-kek"]
```

> To include soft-deleted KEKs in the listing, append `?deleted=true`.

### Get a KEK

```bash
curl http://localhost:8081/dek-registry/v1/keks/my-aws-kek
```

Response:

```json
{
  "name": "my-aws-kek",
  "kmsType": "aws-kms",
  "kmsKeyId": "arn:aws:kms:us-east-1:123456789012:key/abcd-1234-efgh-5678",
  "kmsProps": {
    "KeyState": "Enabled"
  },
  "doc": "Production encryption key for PII fields",
  "shared": false,
  "ts": 1708444800000,
  "deleted": false
}
```

### Update a KEK

Update the documentation and KMS properties of an existing KEK. Only `kmsProps`, `doc`, and `shared` can be updated -- the `name`, `kmsType`, and `kmsKeyId` are immutable after creation.

```bash
curl -X PUT http://localhost:8081/dek-registry/v1/keks/my-aws-kek \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "kmsProps": {
      "KeyState": "Enabled"
    },
    "doc": "Updated: production encryption key for PII fields (rotated 2026-01)"
  }'
```

Response:

```json
{
  "name": "my-aws-kek",
  "kmsType": "aws-kms",
  "kmsKeyId": "arn:aws:kms:us-east-1:123456789012:key/abcd-1234-efgh-5678",
  "kmsProps": {
    "KeyState": "Enabled"
  },
  "doc": "Updated: production encryption key for PII fields (rotated 2026-01)",
  "shared": false,
  "ts": 1708531200000,
  "deleted": false
}
```

---

## Creating and Managing DEKs

### Create a DEK

Create a DEK for the `orders-value` subject under the `my-aws-kek` KEK:

```bash
curl -X POST http://localhost:8081/dek-registry/v1/keks/my-aws-kek/deks \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "subject": "orders-value",
    "algorithm": "AES256_GCM",
    "encryptedKeyMaterial": "base64-encoded-wrapped-key-bytes..."
  }'
```

Response:

```json
{
  "kekName": "my-aws-kek",
  "subject": "orders-value",
  "version": 1,
  "algorithm": "AES256_GCM",
  "encryptedKeyMaterial": "base64-encoded-wrapped-key-bytes...",
  "ts": 1708444800000,
  "deleted": false
}
```

> If the parent KEK has `shared=true`, the response also includes a `keyMaterial` field containing the plaintext DEK for the client to cache locally. This plaintext is never stored by the registry.

### List DEK Subjects

List all subjects that have DEKs under a given KEK:

```bash
curl http://localhost:8081/dek-registry/v1/keks/my-aws-kek/deks
```

Response:

```json
["orders-value", "payments-value"]
```

> To include soft-deleted DEKs in the listing, append `?deleted=true`.

### Get the Latest DEK

Retrieve the latest DEK version for a subject:

```bash
curl http://localhost:8081/dek-registry/v1/keks/my-aws-kek/deks/orders-value
```

Response:

```json
{
  "kekName": "my-aws-kek",
  "subject": "orders-value",
  "version": 2,
  "algorithm": "AES256_GCM",
  "encryptedKeyMaterial": "base64-encoded-wrapped-key-bytes...",
  "ts": 1708531200000,
  "deleted": false
}
```

> You can filter by algorithm with the `?algorithm=AES256_GCM` query parameter.

### List DEK Versions

```bash
curl http://localhost:8081/dek-registry/v1/keks/my-aws-kek/deks/orders-value/versions
```

Response:

```json
[1, 2]
```

### Get a Specific DEK Version

```bash
curl http://localhost:8081/dek-registry/v1/keks/my-aws-kek/deks/orders-value/versions/1
```

Response:

```json
{
  "kekName": "my-aws-kek",
  "subject": "orders-value",
  "version": 1,
  "algorithm": "AES256_GCM",
  "encryptedKeyMaterial": "base64-encoded-wrapped-key-bytes...",
  "ts": 1708444800000,
  "deleted": false
}
```

---

## Soft-Delete and Undelete

Both KEKs and DEKs support a three-level deletion model consistent with the rest of the schema registry:

| Operation | Effect | Reversible? |
|-----------|--------|-------------|
| `DELETE /...` | Sets `deleted=true`. Resource no longer appears in listings (unless `?deleted=true`). | Yes, via undelete. |
| `DELETE /...?permanent=true` | Permanently removes the resource from storage. | No. |
| `PUT /.../undelete` | Clears the `deleted` flag, restoring the resource to active state. | N/A |

### Soft-Delete a KEK

```bash
curl -X DELETE http://localhost:8081/dek-registry/v1/keks/my-aws-kek
```

### Undelete a KEK

```bash
curl -X PUT http://localhost:8081/dek-registry/v1/keks/my-aws-kek/undelete
```

### Permanently Delete a KEK

```bash
curl -X DELETE "http://localhost:8081/dek-registry/v1/keks/my-aws-kek?permanent=true"
```

> Permanently deleting a KEK SHOULD only be done after all DEKs under that KEK have been permanently deleted. If DEKs still reference the KEK, they become unusable since the wrapping key is no longer available for unwrap operations.

### Soft-Delete a DEK

```bash
curl -X DELETE http://localhost:8081/dek-registry/v1/keks/my-aws-kek/deks/orders-value
```

### Undelete a DEK

```bash
curl -X PUT http://localhost:8081/dek-registry/v1/keks/my-aws-kek/deks/orders-value/undelete
```

### Permanently Delete a DEK

```bash
curl -X DELETE "http://localhost:8081/dek-registry/v1/keks/my-aws-kek/deks/orders-value?permanent=true"
```

---

## Supported KMS Types

The `kmsType` field on a KEK identifies which external Key Management Service holds the actual key encryption key.

In **client-side mode** (the default, when `shared=false`), the registry stores the KMS reference so that clients know which KMS to call for wrap/unwrap operations. The registry itself does not contact the KMS.

In **server-side mode** (when `shared=true`), the registry uses its built-in KMS provider integrations to generate and wrap DEKs on behalf of the client. The registry calls the KMS directly using the `kmsProps` configured on the KEK. This mode is useful when clients cannot access the KMS directly or when centralized key management is preferred.

| KMS Type | Value | Key ID Format | Description |
|----------|-------|---------------|-------------|
| AWS KMS | `aws-kms` | ARN (e.g., `arn:aws:kms:us-east-1:123456789012:key/...`) | Amazon Web Services Key Management Service |
| Azure Key Vault | `azure-kms` | Key URL (e.g., `https://myvault.vault.azure.net/keys/mykey/version`) | Microsoft Azure Key Vault |
| GCP KMS | `gcp-kms` | Resource name (e.g., `projects/my-project/locations/global/keyRings/my-ring/cryptoKeys/my-key`) | Google Cloud Key Management Service |
| HashiCorp Vault | `hcvault` | Transit path (e.g., `transit/keys/my-key`) | HashiCorp Vault Transit secrets engine |
| OpenBao | `openbao` | Transit path (e.g., `transit/keys/my-key`) | OpenBao Transit secrets engine (Vault-compatible fork) |

### KMS Properties Reference

Each KMS provider accepts provider-specific properties via the `kmsProps` field on a KEK. These properties configure how the registry (in server-side mode) or the client (in client-side mode) connects to the KMS.

#### HashiCorp Vault (`hcvault`)

| Property | Description | Environment Variable Fallback |
|----------|-------------|-------------------------------|
| `vault.address` | Vault server URL (e.g., `https://vault.example.com:8200`) | `VAULT_ADDR` |
| `vault.token` | Authentication token | `VAULT_TOKEN` |
| `vault.namespace` | Vault Enterprise namespace | `VAULT_NAMESPACE` |
| `vault.transit.mount` | Transit secrets engine mount path (default: `transit`) | -- |

#### OpenBao (`openbao`)

| Property | Description | Environment Variable Fallback |
|----------|-------------|-------------------------------|
| `openbao.address` | OpenBao server URL (e.g., `https://bao.example.com:8200`) | `BAO_ADDR` |
| `openbao.token` | Authentication token | `BAO_TOKEN` |
| `openbao.namespace` | Namespace | `BAO_NAMESPACE` |
| `openbao.transit.mount` | Transit secrets engine mount path (default: `transit`) | -- |

#### AWS KMS (`aws-kms`)

| Property | Description | Environment Variable Fallback |
|----------|-------------|-------------------------------|
| `aws.region` | AWS region (e.g., `us-east-1`) | `AWS_REGION` / `AWS_DEFAULT_REGION` |
| `aws.access.key.id` | AWS access key ID | `AWS_ACCESS_KEY_ID` |
| `aws.secret.access.key` | AWS secret access key | `AWS_SECRET_ACCESS_KEY` |
| `aws.endpoint` | Custom KMS endpoint URL (for LocalStack, VPC endpoints) | -- |

> When `aws.access.key.id` and `aws.secret.access.key` are not set, the provider falls back to the standard AWS credential chain (environment variables, shared credentials file, IAM role).

#### Azure Key Vault (`azure-kms`)

| Property | Description | Environment Variable Fallback |
|----------|-------------|-------------------------------|
| `azure.tenant.id` | Azure AD tenant ID | `AZURE_TENANT_ID` |
| `azure.client.id` | Azure AD application (client) ID | `AZURE_CLIENT_ID` |
| `azure.client.secret` | Azure AD client secret | `AZURE_CLIENT_SECRET` |
| `azure.keyvault.url` | Key Vault URL (e.g., `https://myvault.vault.azure.net`) | -- |
| `azure.key.version` | Specific key version (optional; uses latest if empty) | -- |

#### GCP KMS (`gcp-kms`)

| Property | Description | Environment Variable Fallback |
|----------|-------------|-------------------------------|
| `gcp.project.id` | GCP project ID | `GOOGLE_CLOUD_PROJECT` |
| `gcp.location` | KMS location (e.g., `global`, `us-east1`) | -- |
| `gcp.key.ring` | Key ring name | -- |
| `gcp.credentials.path` | Path to service account JSON key file | `GOOGLE_APPLICATION_CREDENTIALS` |
| `gcp.endpoint` | Custom KMS endpoint URL (for emulators) | -- |

---

## Supported Algorithms

The `algorithm` field on a DEK specifies the symmetric encryption algorithm used to encrypt field values. If not specified, `AES256_GCM` is used as the default.

| Algorithm | Value | Key Size | Mode | Use Case |
|-----------|-------|----------|------|----------|
| AES-256-GCM | `AES256_GCM` | 256-bit | Galois/Counter Mode | Default. Authenticated encryption with high security. Recommended for most use cases. |
| AES-128-GCM | `AES128_GCM` | 128-bit | Galois/Counter Mode | Authenticated encryption with lower key size. Suitable when 128-bit security is sufficient. |
| AES-256-SIV | `AES256_SIV` | 256-bit | Synthetic Initialization Vector | Deterministic encryption. Produces the same ciphertext for the same plaintext, enabling equality searches on encrypted fields. |

> `AES256_SIV` is deterministic: the same plaintext always produces the same ciphertext. This is useful when you need to search or join on encrypted fields, but it leaks information about value equality. Use `AES256_GCM` or `AES128_GCM` when stronger confidentiality is required.

---

## API Reference

### KEK Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/dek-registry/v1/keks` | Create a new KEK |
| `GET` | `/dek-registry/v1/keks` | List all KEK names (add `?deleted=true` to include soft-deleted) |
| `GET` | `/dek-registry/v1/keks/{name}` | Get a KEK by name (add `?deleted=true` to return even if soft-deleted) |
| `PUT` | `/dek-registry/v1/keks/{name}` | Update a KEK (`kmsProps`, `doc`, `shared` only) |
| `DELETE` | `/dek-registry/v1/keks/{name}` | Soft-delete a KEK (add `?permanent=true` to permanently delete) |
| `PUT` | `/dek-registry/v1/keks/{name}/undelete` | Undelete a soft-deleted KEK |

### DEK Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/dek-registry/v1/keks/{name}/deks` | Create a new DEK under a KEK |
| `GET` | `/dek-registry/v1/keks/{name}/deks` | List DEK subjects under a KEK (add `?deleted=true` to include soft-deleted) |
| `GET` | `/dek-registry/v1/keks/{name}/deks/{subject}` | Get the latest DEK for a subject (add `?algorithm=...` to filter, `?deleted=true` to include soft-deleted) |
| `GET` | `/dek-registry/v1/keks/{name}/deks/{subject}/versions` | List DEK version numbers for a subject |
| `GET` | `/dek-registry/v1/keks/{name}/deks/{subject}/versions/{version}` | Get a specific DEK version (add `?algorithm=...` to filter, `?deleted=true` to include soft-deleted) |
| `DELETE` | `/dek-registry/v1/keks/{name}/deks/{subject}` | Soft-delete a DEK (add `?permanent=true` to permanently delete, `?algorithm=...` to target specific algorithm) |
| `PUT` | `/dek-registry/v1/keks/{name}/deks/{subject}/undelete` | Undelete a soft-deleted DEK (add `?algorithm=...` to target specific algorithm) |

---

## Related Documentation

- [Data Contracts](data-contracts.md) -- metadata, rulesets, and encoding rules that reference KEKs for field-level encryption
- [Fundamentals](fundamentals.md) -- core schema registry concepts including schema IDs, subjects, and versions
- [Configuration](configuration.md) -- YAML configuration reference
- [API Reference](api-reference.md) -- complete endpoint documentation
