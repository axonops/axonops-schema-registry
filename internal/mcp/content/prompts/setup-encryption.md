Set up client-side field encryption (CSFLE) with {kms_type}.

Steps:
1. Create a KEK (Key Encryption Key) using the create_kek tool:
   - name: descriptive name (e.g. "production-kek")
   - kms_type: {kms_type}
   - kms_key_id: your KMS key identifier
   - kms_props: provider-specific properties

2. Create a DEK (Data Encryption Key) using the create_dek tool:
   - kek_name: name of the KEK created above
   - subject: schema subject to encrypt
   - algorithm: AES256_GCM (recommended) or AES256_SIV

3. The DEK is automatically wrapped (encrypted) by the KEK via your KMS

Available tools: create_kek, get_kek, list_keks, create_dek, get_dek, list_deks

KMS provider {kms_type} considerations:
