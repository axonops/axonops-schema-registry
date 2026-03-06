Set up client-side field encryption (CSFLE) with {kms_type}.

## Step 1: Create a KEK (Key Encryption Key)

Use the **create_kek** tool:
- `name`: descriptive name (e.g., "production-kek")
- `kms_type`: {kms_type}
- `kms_key_id`: your KMS key identifier (see provider-specific guidance below)
- `kms_props`: provider-specific connection properties (see below)
- `shared`: false (recommended -- each KEK is used by one application)

## Step 2: Test KEK Connectivity

Use the **test_kek** tool immediately after creation:
- `name`: the KEK name from Step 1
- This verifies the registry can reach your KMS and the key is usable
- If this fails, check your kms_props and KMS permissions

## Step 3: Create a DEK (Data Encryption Key)

Use the **create_dek** tool:
- `kek_name`: name of the KEK created in Step 1
- `subject`: schema subject whose data will be encrypted
- `algorithm`: `AES256_GCM` (recommended) or `AES256_SIV` (deterministic, supports search)

The DEK is automatically generated and wrapped (encrypted) by the KEK via your KMS.

## Available Tools

- **create_kek** / **get_kek** / **update_kek** / **delete_kek** / **undelete_kek** / **list_keks** -- KEK management
- **test_kek** -- verify KMS connectivity
- **create_dek** / **get_dek** / **list_deks** / **list_dek_versions** -- DEK management
- **delete_dek** / **undelete_dek** / **rewrap_dek** -- DEK lifecycle

---

## Provider: HashiCorp Vault (hcvault)

**Prerequisites:**
1. Transit secrets engine enabled: `vault secrets enable transit`
2. Encryption key created: `vault write transit/keys/my-key type=aes256-gcm96`
3. Policy granting encrypt/decrypt on the key path

**Configuration:**
- `kms_key_id`: Transit engine key name (e.g., `my-encryption-key`)
- `kms_props`:
  - `vault.address`: Vault server URL (e.g., `http://vault:8200`)
  - `vault.token`: Vault authentication token
  - OR for AppRole auth: `vault.role.id` + `vault.secret.id`
  - For Vault Enterprise: add `vault.namespace`

**Example:**
```json
{
  "name": "prod-kek",
  "kms_type": "hcvault",
  "kms_key_id": "my-encryption-key",
  "kms_props": {
    "vault.address": "http://vault:8200",
    "vault.token": "s.xxxxxxxxxxxx"
  },
  "shared": false
}
```

---

## Provider: OpenBao (openbao)

OpenBao is an open-source fork of HashiCorp Vault with an identical Transit API.

**Configuration:**
- `kms_key_id`: Transit engine key name
- `kms_props`:
  - `openbao.address`: OpenBao server URL
  - `openbao.token`: authentication token

---

## Provider: AWS KMS (aws-kms)

**Prerequisites:**
- KMS key created in your AWS account
- IAM identity has `kms:Encrypt` and `kms:Decrypt` permissions on the key

**Configuration:**
- `kms_key_id`: full ARN (e.g., `arn:aws:kms:us-east-1:123456789:key/uuid`) or alias (e.g., `alias/my-key`)
- `kms_props`:
  - `aws.region`: AWS region (required, e.g., `us-east-1`)
  - `aws.access.key.id` + `aws.secret.access.key`: explicit credentials (optional if using IAM role)

---

## Provider: Azure Key Vault (azure-kms)

**Prerequisites:**
- Key Vault created with a key
- Service principal has `wrap` and `unwrap` permissions on the key

**Configuration:**
- `kms_key_id`: Key Vault key URL (e.g., `https://myvault.vault.azure.net/keys/my-key`)
- `kms_props`:
  - `azure.tenant.id`: Azure AD tenant ID
  - `azure.client.id`: service principal client ID
  - `azure.client.secret`: service principal secret

---

## Provider: GCP KMS (gcp-kms)

**Prerequisites:**
- Key ring and crypto key created in GCP
- Service account has `cloudkms.cryptoKeyVersions.useToEncrypt` and `useToDecrypt` roles

**Configuration:**
- `kms_key_id`: full resource name (e.g., `projects/my-project/locations/global/keyRings/my-ring/cryptoKeys/my-key`)
- `kms_props`:
  - `gcp.project.id`: GCP project ID
  - `gcp.credentials.json`: path to credentials file (optional if using application default credentials)

---

## Common Best Practices

- Use **separate KEKs** per environment (dev, staging, production)
- Set `shared: false` unless multiple applications need the same DEK
- Always run **test_kek** immediately after **create_kek** to verify connectivity
- Use **AES256_GCM** for general encryption; use **AES256_SIV** only if you need deterministic encryption for searchable fields
- Rotate DEKs periodically using **rewrap_dek** (re-encrypts the DEK with a new KMS key version without changing the DEK itself)
- See the **full-encryption-lifecycle** prompt for the complete key rotation and cleanup workflow
