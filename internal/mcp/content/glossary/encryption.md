# Client-Side Field Level Encryption (CSFLE)

## Overview

CSFLE allows producers to encrypt specific fields in a message before sending it to Kafka. Instead of encrypting entire topics, CSFLE targets individual fields -- an email, SSN, or credit card number -- leaving the rest in plaintext for indexing, filtering, and debugging.

The schema registry acts as the **key metadata store**. It does not perform encryption itself. It stores key metadata that serializer/deserializer clients use to encrypt and decrypt fields.

## The Envelope Encryption Pattern

CSFLE uses a two-tier key hierarchy:

| Key Type | Purpose | Where the Actual Key Lives |
|----------|---------|---------------------------|
| **KEK** (Key Encryption Key) | References an external KMS key; used to wrap (encrypt) DEKs | External KMS (Vault, AWS KMS, Azure Key Vault, GCP KMS) |
| **DEK** (Data Encryption Key) | The actual encryption key for field values | Stored in registry as encrypted bytes (wrapped by KEK) |

**How it works:**
1. A DEK is generated (by client or registry).
2. The DEK is encrypted ("wrapped") using the KEK via the external KMS.
3. The wrapped DEK is stored in the registry.
4. At encryption time, the client fetches the wrapped DEK, unwraps it via the KMS, and uses the plaintext DEK to encrypt field values.
5. The plaintext DEK never leaves the client. The registry only stores the wrapped form.

## KEK Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| **name** | string | Yes | Unique identifier |
| **kmsType** | string | Yes | KMS provider: aws-kms, azure-kms, gcp-kms, hcvault, openbao |
| **kmsKeyId** | string | Yes | KMS-specific key identifier (ARN, resource name, transit path) |
| **kmsProps** | map | No | Provider-specific properties (region, endpoint, auth) |
| **doc** | string | No | Human-readable description |
| **shared** | boolean | No | If true, all DEKs share the same key material. Default: false |

The **shared** flag controls DEK creation behavior:
- shared=false (default): client generates its own key material, sends only encrypted form.
- shared=true: registry generates key material via KMS, returns plaintext on creation for client caching.

## DEK Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| **kekName** | string | Yes | Parent KEK name |
| **subject** | string | Yes | Schema subject this DEK encrypts for |
| **version** | integer | No | Auto-assigned sequential version |
| **algorithm** | string | No | AES256_GCM (default), AES128_GCM, or AES256_SIV |
| **encryptedKeyMaterial** | string | No | Wrapped DEK bytes (base64) |

DEKs are versioned per subject under a KEK, enabling key rotation.

## Supported KMS Types

| KMS Type | Value | Status |
|----------|-------|--------|
| HashiCorp Vault | hcvault | Production |
| OpenBao | openbao | Production |
| AWS KMS | aws-kms | Coming Soon |
| Azure Key Vault | azure-kms | Coming Soon |
| GCP KMS | gcp-kms | Coming Soon |

## Supported Algorithms

| Algorithm | Key Size | Mode | Use Case |
|-----------|----------|------|----------|
| **AES256_GCM** | 256-bit | Galois/Counter | Default. Authenticated encryption. Recommended for most use cases. |
| **AES128_GCM** | 128-bit | Galois/Counter | Authenticated encryption with lower key size. |
| **AES256_SIV** | 256-bit | Synthetic IV | Deterministic encryption. Same plaintext produces same ciphertext. Enables equality searches on encrypted fields but leaks value equality information. |

## Key Rotation

To rotate encryption keys:
1. Create a new DEK version for the subject under the same KEK.
2. New messages are encrypted with the latest DEK version.
3. Old messages can still be decrypted with previous DEK versions.

## Rewrapping

When the underlying KMS key is rotated, use the **rewrap** operation:
- The DEK plaintext key material stays the same.
- The encryptedKeyMaterial is re-encrypted with the new KMS key version.
- Existing encrypted data remains readable without re-encryption.

## Soft-Delete Model

Both KEKs and DEKs support three-level deletion:

| Operation | Effect | Reversible? |
|-----------|--------|-------------|
| DELETE | Sets deleted=true. Hidden from listings. | Yes, via undelete |
| DELETE ?permanent=true | Permanently removed from storage. | No |
| POST .../undelete | Clears deleted flag. | N/A |

## MCP Tools

- **create_kek / get_kek / list_keks / update_kek / delete_kek** -- manage KEKs
- **create_dek / get_dek / list_deks / delete_dek** -- manage DEKs
- **test_kek** -- test KMS connectivity for a KEK
- **get_dek_versions** -- list DEK version numbers
