"""
CSFLE (Client-Side Field Level Encryption) Tests — Python SerDe

Mirrors the Go serde tests at tests/compatibility/go-serde/csfle_vault_test.go
and csfle_extras_test.go.

These tests validate that:
  - PII-tagged fields are encrypted in the wire format (no plaintext)
  - Encrypted data round-trips correctly through serialize/deserialize
  - Multiple PII fields are all encrypted
  - DEKs and KEKs are auto-created upon first serialization
  - DEK caching allows decryption without explicit Vault token

Prerequisites:
  - Schema registry running at SCHEMA_REGISTRY_URL (default: http://localhost:8081)
  - Vault running at VAULT_URL (default: http://localhost:18200)
    with Transit engine enabled and test-key created
  - confluent-kafka[avro,schemaregistry] >= 2.6.0 with encryption support

Usage:
  pytest test_csfle.py -v
  pytest test_csfle.py -v -m csfle
"""

import json
import os
import time

import pytest
import requests
from confluent_kafka.schema_registry import SchemaRegistryClient
from confluent_kafka.schema_registry.avro import AvroSerializer, AvroDeserializer
from confluent_kafka.serialization import SerializationContext, MessageField

# ---------------------------------------------------------------------------
# Rule executor registration for encryption
# ---------------------------------------------------------------------------
try:
    from confluent_kafka.schema_registry.rules.cel.cel_executor import CelExecutor
    from confluent_kafka.schema_registry.rules.cel.cel_field_executor import CelFieldExecutor
    from confluent_kafka.schema_registry.serde import RuleRegistry

    # CEL executors must be registered even for CSFLE (the ENCRYPT executor
    # depends on the rule infrastructure).
    RuleRegistry.register_rule_executor(CelExecutor())
    RuleRegistry.register_rule_executor(CelFieldExecutor())
    HAS_RULE_EXECUTORS = True
except ImportError as e:
    import sys
    print(f"WARNING: Rule executors not available: {e}", file=sys.stderr)
    HAS_RULE_EXECUTORS = False

try:
    from confluent_kafka.schema_registry.rules.encryption.encrypt_executor import (
        FieldEncryptionExecutor,
    )
    from confluent_kafka.schema_registry.rules.encryption.hcvault.hcvault_driver import (
        HcVaultKmsDriver,
    )
    from confluent_kafka.schema_registry.serde import RuleRegistry as _RR

    _RR.register_rule_executor(FieldEncryptionExecutor())
    HcVaultKmsDriver.register()
    HAS_ENCRYPTION = True
except ImportError as e:
    import sys
    print(f"WARNING: Encryption executors not available: {e}", file=sys.stderr)
    HAS_ENCRYPTION = False


# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------
REGISTRY_URL = os.environ.get("SCHEMA_REGISTRY_URL", "http://localhost:8081")
VAULT_URL = os.environ.get("VAULT_URL", "http://localhost:18200")
VAULT_TOKEN = os.environ.get("VAULT_TOKEN", "test-root-token")

CONTENT_TYPE = "application/vnd.schemaregistry.v1+json"
HEADERS = {
    "Content-Type": CONTENT_TYPE,
    "Accept": CONTENT_TYPE,
}


# ===========================================================================
# Avro Schemas (with PII tags for CSFLE)
# ===========================================================================

CUSTOMER_SCHEMA = json.dumps({
    "type": "record",
    "name": "Customer",
    "namespace": "com.axonops.test.csfle",
    "fields": [
        {"name": "customerId", "type": "string"},
        {"name": "name", "type": "string"},
        {"name": "ssn", "type": "string", "confluent:tags": ["PII"]},
    ],
})

USER_PROFILE_SCHEMA = json.dumps({
    "type": "record",
    "name": "UserProfile",
    "namespace": "com.axonops.test.csfle",
    "fields": [
        {"name": "userId", "type": "string"},
        {"name": "ssn", "type": "string", "confluent:tags": ["PII"]},
        {"name": "email", "type": "string", "confluent:tags": ["PII"]},
        {"name": "creditCard", "type": "string", "confluent:tags": ["PII"]},
    ],
})

PAYMENT_EVENT_SCHEMA = json.dumps({
    "type": "record",
    "name": "PaymentEvent",
    "namespace": "com.axonops.test.csfle",
    "fields": [
        {"name": "customerId", "type": "string"},
        {"name": "creditCardNumber", "type": "string", "confluent:tags": ["PII"]},
        {"name": "amount", "type": "double"},
        {"name": "merchantName", "type": "string"},
    ],
})


# ===========================================================================
# HTTP / Subject Helpers
# ===========================================================================

def register_schema(subject, body):
    """Register a schema via the REST API and return the global schema ID."""
    url = f"{REGISTRY_URL}/subjects/{subject}/versions"
    resp = requests.post(url, headers=HEADERS, data=body)
    assert resp.status_code == 200, (
        f"Failed to register schema for {subject}: "
        f"HTTP {resp.status_code} - {resp.text}"
    )
    return resp.json()["id"]


def delete_subject(subject):
    """Permanently delete a subject (soft then hard). Errors are ignored."""
    url = f"{REGISTRY_URL}/subjects/{subject}"
    try:
        requests.delete(url, headers={"Accept": CONTENT_TYPE})
    except Exception:
        pass
    try:
        requests.delete(f"{url}?permanent=true", headers={"Accept": CONTENT_TYPE})
    except Exception:
        pass


def unique_subject(prefix):
    return f"{prefix}-{int(time.time() * 1000)}-value"


def topic_from_subject(subject):
    if subject.endswith("-value"):
        return subject[: -len("-value")]
    return subject


def get_kek(kek_name):
    """Fetch a KEK from the DEK Registry. Returns empty string if not found."""
    url = f"{REGISTRY_URL}/dek-registry/v1/keks/{kek_name}"
    try:
        resp = requests.get(url, headers={"Accept": CONTENT_TYPE})
        if resp.status_code != 200:
            return ""
        return resp.text
    except Exception:
        return ""


def get_dek(kek_name, subject):
    """Fetch a DEK from the DEK Registry. Returns empty string if not found."""
    url = f"{REGISTRY_URL}/dek-registry/v1/keks/{kek_name}/deks/{subject}"
    try:
        resp = requests.get(url, headers={"Accept": CONTENT_TYPE})
        if resp.status_code != 200:
            return ""
        return resp.text
    except Exception:
        return ""


# ===========================================================================
# CSFLE Schema Builder (mirrors Go's buildSchemaWithEncryptRule)
# ===========================================================================

def build_schema_with_encrypt_rule(avro_schema, kek_name):
    """Build JSON body for registering a schema with an ENCRYPT rule."""
    vault_base = VAULT_URL.rstrip("/")
    return json.dumps({
        "schemaType": "AVRO",
        "schema": avro_schema,
        "ruleSet": {
            "domainRules": [{
                "name": "encrypt-pii",
                "kind": "TRANSFORM",
                "type": "ENCRYPT",
                "mode": "WRITEREAD",
                "tags": ["PII"],
                "params": {
                    "encrypt.kek.name": kek_name,
                    "encrypt.kms.type": "hcvault",
                    "encrypt.kms.key.id": f"{vault_base}/transit/keys/test-key",
                },
                "onFailure": "ERROR,NONE",
            }],
        },
    })


# ===========================================================================
# SerDe Factories
# ===========================================================================

def new_client():
    """Create a SchemaRegistryClient."""
    return SchemaRegistryClient({"url": REGISTRY_URL})


def new_csfle_serializer(client, schema_str):
    """Create an AvroSerializer configured for CSFLE."""
    os.environ["VAULT_TOKEN"] = VAULT_TOKEN
    return AvroSerializer(
        client,
        schema_str,
        to_dict=lambda obj, ctx: obj,
        conf={
            "auto.register.schemas": False,
            "use.latest.version": True,
            "rules.secret.access.key": VAULT_TOKEN,
        },
    )


def new_csfle_deserializer(client, schema_str=None):
    """Create an AvroDeserializer configured for CSFLE."""
    os.environ["VAULT_TOKEN"] = VAULT_TOKEN
    return AvroDeserializer(
        client,
        schema_str,
        from_dict=lambda obj, ctx: obj,
        conf={
            "use.latest.version": True,
            "rules.secret.access.key": VAULT_TOKEN,
        },
    )


def new_rule_deserializer(client, schema_str=None):
    """Create an AvroDeserializer without explicit CSFLE Vault config.

    Used for testing DEK caching (decryption without explicit token).
    """
    return AvroDeserializer(
        client,
        schema_str,
        from_dict=lambda obj, ctx: obj,
        conf={"use.latest.version": True},
    )


# ===========================================================================
# Vault health check
# ===========================================================================

def vault_is_healthy():
    """Return True if Vault is reachable and healthy."""
    try:
        resp = requests.get(f"{VAULT_URL}/v1/sys/health", timeout=5)
        return resp.status_code in (200, 429, 472)
    except Exception:
        return False


requires_vault = pytest.mark.skipif(
    not vault_is_healthy(),
    reason=f"Vault not accessible at {VAULT_URL}",
)

requires_encryption = pytest.mark.skipif(
    not HAS_ENCRYPTION,
    reason="confluent-kafka encryption executors not available",
)


# ###########################################################################
#
# CSFLE Tests
#
# ###########################################################################

@pytest.mark.csfle
@requires_vault
@requires_encryption
class TestCsfle:
    """CSFLE encryption tests using HashiCorp Vault as KMS backend."""

    # -----------------------------------------------------------------
    # 1. Encrypt/decrypt round trip
    # -----------------------------------------------------------------
    def test_csfle_encrypt_decrypt_round_trip(self):
        subject = unique_subject("py-csfle-roundtrip")
        kek_name = f"kek-roundtrip-{subject}"
        try:
            body = build_schema_with_encrypt_rule(CUSTOMER_SCHEMA, kek_name)
            register_schema(subject, body)

            client = new_client()
            ser = new_csfle_serializer(client, CUSTOMER_SCHEMA)
            deser = new_csfle_deserializer(client, CUSTOMER_SCHEMA)

            original = {
                "customerId": "CUST-001",
                "name": "Jane Doe",
                "ssn": "123-45-6789",
            }

            ctx = SerializationContext(topic_from_subject(subject), MessageField.VALUE)
            data = ser(original, ctx)
            assert data is not None and len(data) > 0

            result = deser(data, ctx)
            assert result["customerId"] == "CUST-001"
            assert result["name"] == "Jane Doe"
            assert result["ssn"] == "123-45-6789"
        finally:
            delete_subject(subject)

    # -----------------------------------------------------------------
    # 2. Raw bytes contain no plaintext PII
    # -----------------------------------------------------------------
    def test_csfle_raw_bytes_no_plaintext(self):
        subject = unique_subject("py-csfle-noplain")
        kek_name = f"kek-noplain-{subject}"
        try:
            body = build_schema_with_encrypt_rule(CUSTOMER_SCHEMA, kek_name)
            register_schema(subject, body)

            client = new_client()
            ser = new_csfle_serializer(client, CUSTOMER_SCHEMA)

            original = {
                "customerId": "CUST-002",
                "name": "John Smith",
                "ssn": "123-45-6789",
            }

            ctx = SerializationContext(topic_from_subject(subject), MessageField.VALUE)
            data = ser(original, ctx)
            assert data is not None and len(data) > 0

            # The raw bytes must not contain the plaintext SSN.
            raw_str = data.decode("latin-1")  # Use latin-1 to preserve all bytes
            assert "123-45-6789" not in raw_str, (
                "raw bytes must not contain plaintext SSN"
            )
        finally:
            delete_subject(subject)

    # -----------------------------------------------------------------
    # 3. Multiple PII fields encrypted and round-trip
    # -----------------------------------------------------------------
    def test_csfle_multiple_pii_fields(self):
        subject = unique_subject("py-csfle-multipii")
        kek_name = f"kek-multipii-{subject}"
        try:
            body = build_schema_with_encrypt_rule(USER_PROFILE_SCHEMA, kek_name)
            register_schema(subject, body)

            client = new_client()
            ser = new_csfle_serializer(client, USER_PROFILE_SCHEMA)
            deser = new_csfle_deserializer(client, USER_PROFILE_SCHEMA)

            original = {
                "userId": "USER-100",
                "ssn": "987-65-4321",
                "email": "secret@example.com",
                "creditCard": "4111-1111-1111-1111",
            }

            ctx = SerializationContext(topic_from_subject(subject), MessageField.VALUE)
            data = ser(original, ctx)
            assert data is not None and len(data) > 0

            # Verify no plaintext PII in raw bytes.
            raw_str = data.decode("latin-1")
            assert "987-65-4321" not in raw_str, (
                "raw bytes must not contain plaintext SSN"
            )
            assert "secret@example.com" not in raw_str, (
                "raw bytes must not contain plaintext email"
            )
            assert "4111-1111-1111-1111" not in raw_str, (
                "raw bytes must not contain plaintext credit card"
            )

            # Round-trip decryption.
            result = deser(data, ctx)
            assert result["userId"] == "USER-100"
            assert result["ssn"] == "987-65-4321"
            assert result["email"] == "secret@example.com"
            assert result["creditCard"] == "4111-1111-1111-1111"
        finally:
            delete_subject(subject)

    # -----------------------------------------------------------------
    # 4. Credit card protection
    # -----------------------------------------------------------------
    def test_csfle_credit_card_protection(self):
        subject = unique_subject("py-csfle-cc")
        kek_name = f"kek-cc-{subject}"
        try:
            body = build_schema_with_encrypt_rule(PAYMENT_EVENT_SCHEMA, kek_name)
            register_schema(subject, body)

            client = new_client()
            ser = new_csfle_serializer(client, PAYMENT_EVENT_SCHEMA)
            deser = new_csfle_deserializer(client, PAYMENT_EVENT_SCHEMA)

            original = {
                "customerId": "CUST-PAY-001",
                "creditCardNumber": "4532-0150-1234-5678",
                "amount": 149.99,
                "merchantName": "Coffee Shop",
            }

            ctx = SerializationContext(topic_from_subject(subject), MessageField.VALUE)
            data = ser(original, ctx)
            assert data is not None and len(data) > 0

            # Credit card number must not appear as plaintext.
            raw_str = data.decode("latin-1")
            assert "4532-0150-1234-5678" not in raw_str, (
                "raw bytes must not contain plaintext credit card number"
            )

            # Round-trip decryption.
            result = deser(data, ctx)
            assert result["creditCardNumber"] == "4532-0150-1234-5678"
            assert result["customerId"] == "CUST-PAY-001"
            assert abs(result["amount"] - 149.99) < 0.001
            assert result["merchantName"] == "Coffee Shop"
        finally:
            delete_subject(subject)

    # -----------------------------------------------------------------
    # 5. DEK caching — second deserializer without explicit Vault token
    # -----------------------------------------------------------------
    def test_csfle_dek_caching(self):
        subject = unique_subject("py-csfle-dekcache")
        kek_name = f"kek-dekcache-{subject}"
        try:
            body = build_schema_with_encrypt_rule(CUSTOMER_SCHEMA, kek_name)
            register_schema(subject, body)

            client = new_client()
            ser = new_csfle_serializer(client, CUSTOMER_SCHEMA)

            original = {
                "customerId": "CUST-CACHE",
                "name": "Cache Test",
                "ssn": "555-66-7777",
            }

            ctx = SerializationContext(topic_from_subject(subject), MessageField.VALUE)
            data = ser(original, ctx)
            assert data is not None and len(data) > 0

            raw_str = data.decode("latin-1")
            assert "555-66-7777" not in raw_str, (
                "raw bytes must not contain plaintext SSN"
            )

            # Use a rule deserializer without explicit Vault token -- should
            # still work because the DEK is cached in the client-side library.
            deser2 = new_rule_deserializer(client, CUSTOMER_SCHEMA)
            result = deser2(data, ctx)

            assert result["customerId"] == "CUST-CACHE"
            assert result["name"] == "Cache Test"
            assert result["ssn"] == "555-66-7777"
        finally:
            delete_subject(subject)

    # -----------------------------------------------------------------
    # 6. DEK auto-created on first serialization
    # -----------------------------------------------------------------
    def test_csfle_dek_auto_created(self):
        subject = unique_subject("py-csfle-dekauto")
        kek_name = f"kek-dekauto-{subject}"
        try:
            body = build_schema_with_encrypt_rule(CUSTOMER_SCHEMA, kek_name)
            register_schema(subject, body)

            # Before serialization, no DEK should exist.
            dek_before = get_dek(kek_name, subject)
            assert dek_before == "", "DEK should not exist before first serialization"

            client = new_client()
            ser = new_csfle_serializer(client, CUSTOMER_SCHEMA)

            original = {
                "customerId": "CUST-DEKAUTO",
                "name": "DEK Auto",
                "ssn": "111-22-3333",
            }

            ctx = SerializationContext(topic_from_subject(subject), MessageField.VALUE)
            data = ser(original, ctx)
            assert data is not None and len(data) > 0

            # After serialization, the DEK should have been auto-created.
            dek_after = get_dek(kek_name, subject)
            assert dek_after != "", "DEK should exist after first serialization"
            assert "encryptedKeyMaterial" in dek_after, (
                "DEK response should contain encryptedKeyMaterial"
            )
        finally:
            delete_subject(subject)

    # -----------------------------------------------------------------
    # 7. KEK auto-created on first serialization
    # -----------------------------------------------------------------
    def test_csfle_kek_auto_created(self):
        subject = unique_subject("py-csfle-kekauto")
        kek_name = f"kek-kekauto-{subject}"

        # Before schema registration, no KEK should exist.
        kek_before = get_kek(kek_name)
        assert kek_before == "", "KEK should not exist before schema registration"

        try:
            body = build_schema_with_encrypt_rule(CUSTOMER_SCHEMA, kek_name)
            register_schema(subject, body)

            client = new_client()
            ser = new_csfle_serializer(client, CUSTOMER_SCHEMA)

            original = {
                "customerId": "CUST-KEKAUTO",
                "name": "KEK Auto",
                "ssn": "444-55-6666",
            }

            ctx = SerializationContext(topic_from_subject(subject), MessageField.VALUE)
            data = ser(original, ctx)
            assert data is not None and len(data) > 0

            # After serialization, the KEK should have been auto-created.
            kek_after = get_kek(kek_name)
            assert kek_after != "", "KEK should exist after first serialization"
            assert "hcvault" in kek_after, (
                "KEK response should reference hcvault as the KMS type"
            )
        finally:
            delete_subject(subject)
