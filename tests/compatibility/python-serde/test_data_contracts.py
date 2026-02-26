"""
Data Contract Tests — Python SerDe

Mirrors the Go serde tests at tests/compatibility/go-serde/ for:
  - CEL CONDITION rules (WRITE, READ, WRITEREAD modes)
  - CEL_FIELD TRANSFORM and CONDITION rules (PII masking, normalization)
  - Migration rules (JSONata UPGRADE / DOWNGRADE)
  - Global policies (defaultRuleSet, overrideRuleSet, rule inheritance)
  - JSON Schema and Protobuf rule storage verification

Prerequisites:
  - Schema registry running at SCHEMA_REGISTRY_URL (default: http://localhost:8081)
  - confluent-kafka[avro,schemaregistry] >= 2.6.0 for rule executor support

Usage:
  pytest test_data_contracts.py -v
  pytest test_data_contracts.py -v -m data_contracts
  pytest test_data_contracts.py -v -k "cel"
  pytest test_data_contracts.py -v -k "migration"
  pytest test_data_contracts.py -v -k "policy"
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
# Rule executor registration (optional — available since confluent-kafka 2.6.0+)
# ---------------------------------------------------------------------------
try:
    from confluent_kafka.schema_registry.rules.cel.cel_executor import CelExecutor
    from confluent_kafka.schema_registry.rules.cel.cel_field_executor import CelFieldExecutor
    from confluent_kafka.schema_registry.rules.jsonata.jsonata_executor import JsonataExecutor
    from confluent_kafka.schema_registry.serde import RuleRegistry

    RuleRegistry.register_rule_executor(CelExecutor())
    RuleRegistry.register_rule_executor(CelFieldExecutor())
    RuleRegistry.register_rule_executor(JsonataExecutor())
    HAS_RULE_EXECUTORS = True
except ImportError:
    HAS_RULE_EXECUTORS = False

# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------
REGISTRY_URL = os.environ.get("SCHEMA_REGISTRY_URL", "http://localhost:8081")

CONTENT_TYPE = "application/vnd.schemaregistry.v1+json"
HEADERS = {
    "Content-Type": CONTENT_TYPE,
    "Accept": CONTENT_TYPE,
}


# ===========================================================================
# HTTP Helpers (mirrors Go testhelper_test.go)
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


def get_schema_version(subject, version):
    """Fetch a schema version and return the raw JSON response string."""
    url = f"{REGISTRY_URL}/subjects/{subject}/versions/{version}"
    resp = requests.get(url, headers={"Accept": CONTENT_TYPE})
    assert resp.status_code == 200, (
        f"Failed to get {subject} v{version}: "
        f"HTTP {resp.status_code} - {resp.text}"
    )
    return resp.text


def delete_subject(subject):
    """Permanently delete a subject (soft then hard). Errors are ignored."""
    url = f"{REGISTRY_URL}/subjects/{subject}"
    try:
        requests.delete(url, headers={"Accept": CONTENT_TYPE})
    except Exception:
        pass
    try:
        requests.delete(
            f"{url}?permanent=true", headers={"Accept": CONTENT_TYPE}
        )
    except Exception:
        pass


def set_subject_config(subject, body):
    """Set subject-level config (compatibility, defaultRuleSet, etc.)."""
    url = f"{REGISTRY_URL}/config/{subject}"
    resp = requests.put(url, headers=HEADERS, data=body)
    assert resp.status_code == 200, (
        f"Failed to set config for {subject}: "
        f"HTTP {resp.status_code} - {resp.text}"
    )


# ===========================================================================
# Unique subject / topic helpers
# ===========================================================================

def unique_subject(prefix):
    return f"{prefix}-{int(time.time() * 1000)}-value"


def topic_from_subject(subject):
    if subject.endswith("-value"):
        return subject[: -len("-value")]
    return subject


# ===========================================================================
# JSON escape helper (mirrors Go's escapeJSON)
# ===========================================================================

def escape_json(s):
    """Escape a raw string for embedding inside a JSON string value."""
    s = s.replace("\\", "\\\\")
    s = s.replace('"', '\\"')
    s = s.replace("\n", "\\n")
    s = s.replace("\r", "\\r")
    s = s.replace("\t", "\\t")
    return s


# ===========================================================================
# Schema Registry Client & SerDe Factories
# ===========================================================================

def new_client():
    """Create a SchemaRegistryClient."""
    return SchemaRegistryClient({"url": REGISTRY_URL})


def new_rule_serializer(client, schema_str):
    """Create an AvroSerializer with auto.register=False, use.latest=True.

    Uses to_dict=lambda obj, ctx: obj so we can pass plain dicts.
    """
    return AvroSerializer(
        client,
        schema_str,
        to_dict=lambda obj, ctx: obj,
        conf={
            "auto.register.schemas": False,
            "use.latest.version": True,
        },
    )


def new_rule_deserializer(client, schema_str=None):
    """Create an AvroDeserializer with use.latest=True.

    Uses from_dict=lambda obj, ctx: obj so we get plain dicts back.
    """
    return AvroDeserializer(
        client,
        schema_str,
        from_dict=lambda obj, ctx: obj,
        conf={"use.latest.version": True},
    )


def new_metadata_pinned_deserializer(client, schema_str, metadata):
    """Create a deserializer pinned to a specific schema version via metadata.

    Used for DOWNGRADE migration testing: the deserializer targets an older
    schema version (identified by its metadata), causing the migration engine
    to execute DOWNGRADE rules when it encounters newer-versioned data.
    """
    return AvroDeserializer(
        client,
        schema_str,
        from_dict=lambda obj, ctx: obj,
        conf={"use.latest.with.metadata": metadata},
    )


# ===========================================================================
# Rule error detection (mirrors Go's isRuleError)
# ===========================================================================

def is_rule_error(exc):
    """Check if an exception is from a data contract rule violation."""
    if exc is None:
        return False
    msg = str(exc).lower()
    return any(
        kw in msg
        for kw in ["rule", "condition", "expr failed", "expr_failed"]
    )


# ===========================================================================
# Avro Schema Definitions
# ===========================================================================

ORDER_SCHEMA = json.dumps({
    "type": "record",
    "name": "Order",
    "namespace": "com.axonops.test.cel",
    "fields": [
        {"name": "orderId", "type": "string"},
        {"name": "amount", "type": "double"},
        {"name": "currency", "type": "string"},
    ],
})

ORDER_STATUS_SCHEMA = json.dumps({
    "type": "record",
    "name": "OrderStatus",
    "namespace": "com.axonops.test.cel",
    "fields": [
        {"name": "orderId", "type": "string"},
        {"name": "status", "type": "string"},
    ],
})

USER_SCHEMA = json.dumps({
    "type": "record",
    "name": "User",
    "namespace": "com.axonops.test.cel",
    "fields": [
        {"name": "name", "type": "string"},
        {"name": "ssn", "type": "string", "confluent:tags": ["PII"]},
    ],
})

ADDRESS_SCHEMA = json.dumps({
    "type": "record",
    "name": "Address",
    "namespace": "com.axonops.test.cel",
    "fields": [
        {"name": "street", "type": "string"},
        {"name": "country", "type": "string"},
    ],
})

ORDER_POLICY_SCHEMA = json.dumps({
    "type": "record",
    "name": "Order",
    "namespace": "com.axonops.test.policy",
    "fields": [
        {"name": "orderId", "type": "string"},
        {"name": "amount", "type": "double"},
    ],
})

ORDER_POLICY_V2_SCHEMA = json.dumps({
    "type": "record",
    "name": "Order",
    "namespace": "com.axonops.test.policy",
    "fields": [
        {"name": "orderId", "type": "string"},
        {"name": "amount", "type": "double"},
        {"name": "notes", "type": ["null", "string"], "default": None},
    ],
})

CONTACT_SCHEMA = json.dumps({
    "type": "record",
    "name": "Contact",
    "namespace": "com.axonops.test.policy",
    "fields": [
        {"name": "name", "type": "string"},
        {"name": "email", "type": "string", "confluent:tags": ["PII"]},
    ],
})


# ###########################################################################
#
# CEL RULES (Avro - full SerDe execution)
#
# ###########################################################################

requires_executors = pytest.mark.skipif(
    not HAS_RULE_EXECUTORS,
    reason="confluent-kafka rule executors not available",
)


@pytest.mark.data_contracts
@requires_executors
class TestCelRulesAvro:
    """CEL data contract rules exercised through the Confluent Python SerDe."""

    # -----------------------------------------------------------------
    # 1. CEL CONDITION — valid data passes
    # -----------------------------------------------------------------
    def test_cel_condition_valid_data_passes(self):
        subject = unique_subject("py-cel-cond-valid")
        try:
            body = json.dumps({
                "schemaType": "AVRO",
                "schema": ORDER_SCHEMA,
                "ruleSet": {
                    "domainRules": [{
                        "name": "amount-positive",
                        "kind": "CONDITION",
                        "type": "CEL",
                        "mode": "WRITE",
                        "expr": "message.Amount > 0.0",
                        "onFailure": "ERROR",
                    }],
                },
            })
            register_schema(subject, body)

            client = new_client()
            ser = new_rule_serializer(client, ORDER_SCHEMA)
            deser = new_rule_deserializer(client, ORDER_SCHEMA)

            order = {"orderId": "ORD-001", "amount": 100.50, "currency": "USD"}
            ctx = SerializationContext(topic_from_subject(subject), MessageField.VALUE)
            data = ser(order, ctx)
            assert data is not None and len(data) > 0

            result = deser(data, ctx)
            assert result["orderId"] == "ORD-001"
            assert result["amount"] == 100.50
            assert result["currency"] == "USD"
        finally:
            delete_subject(subject)

    # -----------------------------------------------------------------
    # 2. CEL CONDITION — invalid data rejected
    # -----------------------------------------------------------------
    def test_cel_condition_invalid_data_rejected(self):
        subject = unique_subject("py-cel-cond-invalid")
        try:
            body = json.dumps({
                "schemaType": "AVRO",
                "schema": ORDER_SCHEMA,
                "ruleSet": {
                    "domainRules": [{
                        "name": "amount-positive",
                        "kind": "CONDITION",
                        "type": "CEL",
                        "mode": "WRITE",
                        "expr": "message.Amount > 0.0",
                        "onFailure": "ERROR",
                    }],
                },
            })
            register_schema(subject, body)

            client = new_client()
            ser = new_rule_serializer(client, ORDER_SCHEMA)

            order = {"orderId": "ORD-BAD", "amount": -5.0, "currency": "USD"}
            ctx = SerializationContext(topic_from_subject(subject), MessageField.VALUE)
            with pytest.raises(Exception) as exc_info:
                ser(order, ctx)
            assert is_rule_error(exc_info.value), (
                f"Expected rule error, got: {exc_info.value}"
            )
        finally:
            delete_subject(subject)

    # -----------------------------------------------------------------
    # 3. CEL CONDITION READ — rejects at deserialization
    # -----------------------------------------------------------------
    def test_cel_condition_read_rejects_at_deserialization(self):
        subject = unique_subject("py-cel-cond-read")
        try:
            body = json.dumps({
                "schemaType": "AVRO",
                "schema": ORDER_STATUS_SCHEMA,
                "ruleSet": {
                    "domainRules": [{
                        "name": "no-cancelled-on-read",
                        "kind": "CONDITION",
                        "type": "CEL",
                        "mode": "READ",
                        "expr": "message.Status != 'CANCELLED'",
                        "onFailure": "ERROR",
                    }],
                },
            })
            register_schema(subject, body)

            client = new_client()
            ser = new_rule_serializer(client, ORDER_STATUS_SCHEMA)
            deser = new_rule_deserializer(client, ORDER_STATUS_SCHEMA)

            data_dict = {"orderId": "ORD-CANCEL", "status": "CANCELLED"}
            ctx = SerializationContext(topic_from_subject(subject), MessageField.VALUE)

            # Serialization should succeed (rule is READ mode only).
            data = ser(data_dict, ctx)
            assert data is not None

            # Deserialization should fail (rule fires on READ).
            with pytest.raises(Exception) as exc_info:
                deser(data, ctx)
            assert is_rule_error(exc_info.value), (
                f"Expected rule error on read, got: {exc_info.value}"
            )
        finally:
            delete_subject(subject)

    # -----------------------------------------------------------------
    # 4. Multiple CEL CONDITION rules — all must pass
    # -----------------------------------------------------------------
    def test_multiple_cel_conditions_chained(self):
        subject = unique_subject("py-cel-cond-multi")
        try:
            body = json.dumps({
                "schemaType": "AVRO",
                "schema": ORDER_SCHEMA,
                "ruleSet": {
                    "domainRules": [
                        {
                            "name": "amount-positive",
                            "kind": "CONDITION",
                            "type": "CEL",
                            "mode": "WRITE",
                            "expr": "message.Amount > 0.0",
                            "onFailure": "ERROR",
                        },
                        {
                            "name": "currency-three-chars",
                            "kind": "CONDITION",
                            "type": "CEL",
                            "mode": "WRITE",
                            "expr": "size(message.Currency) == 3",
                            "onFailure": "ERROR",
                        },
                    ],
                },
            })
            register_schema(subject, body)

            client = new_client()
            ser = new_rule_serializer(client, ORDER_SCHEMA)
            deser = new_rule_deserializer(client, ORDER_SCHEMA)
            ctx = SerializationContext(topic_from_subject(subject), MessageField.VALUE)

            # Case A: Valid amount but currency is only 2 chars -> should fail.
            bad_order = {"orderId": "ORD-SHORT", "amount": 100.0, "currency": "US"}
            with pytest.raises(Exception) as exc_info:
                ser(bad_order, ctx)
            assert is_rule_error(exc_info.value), (
                f"Expected rule error for 2-char currency, got: {exc_info.value}"
            )

            # Case B: Both conditions satisfied -> should succeed.
            good_order = {"orderId": "ORD-GOOD", "amount": 100.0, "currency": "USD"}
            data = ser(good_order, ctx)
            assert data is not None

            result = deser(data, ctx)
            assert result["orderId"] == "ORD-GOOD"
            assert result["amount"] == 100.0
            assert result["currency"] == "USD"
        finally:
            delete_subject(subject)

    # -----------------------------------------------------------------
    # 5. CEL_FIELD TRANSFORM — mask SSN on READ
    # -----------------------------------------------------------------
    def test_cel_field_transform_masks_ssn_on_read(self):
        subject = unique_subject("py-cel-field-mask")
        try:
            body = json.dumps({
                "schemaType": "AVRO",
                "schema": USER_SCHEMA,
                "ruleSet": {
                    "domainRules": [{
                        "name": "mask-pii",
                        "kind": "TRANSFORM",
                        "type": "CEL_FIELD",
                        "mode": "READ",
                        "tags": ["PII"],
                        "expr": "typeName == 'STRING' ; 'XXX-XX-' + value.substring(7, 11)",
                        "onFailure": "ERROR",
                    }],
                },
            })
            register_schema(subject, body)

            client = new_client()
            ser = new_rule_serializer(client, USER_SCHEMA)
            deser = new_rule_deserializer(client, USER_SCHEMA)
            ctx = SerializationContext(topic_from_subject(subject), MessageField.VALUE)

            user = {"name": "Jane Doe", "ssn": "123-45-6789"}
            data = ser(user, ctx)
            assert data is not None

            result = deser(data, ctx)
            assert result["name"] == "Jane Doe", "name should be unchanged"
            assert result["ssn"] == "XXX-XX-6789", "SSN should be masked on read"
        finally:
            delete_subject(subject)

    # -----------------------------------------------------------------
    # 6. CEL_FIELD CONDITION — reject empty PII
    # -----------------------------------------------------------------
    def test_cel_field_condition_rejects_empty_pii(self):
        subject = unique_subject("py-cel-field-cond")
        try:
            body = json.dumps({
                "schemaType": "AVRO",
                "schema": USER_SCHEMA,
                "ruleSet": {
                    "domainRules": [{
                        "name": "pii-not-empty",
                        "kind": "CONDITION",
                        "type": "CEL_FIELD",
                        "mode": "WRITE",
                        "tags": ["PII"],
                        "expr": "typeName == 'STRING' ; value != ''",
                        "onFailure": "ERROR",
                    }],
                },
            })
            register_schema(subject, body)

            client = new_client()
            ser = new_rule_serializer(client, USER_SCHEMA)
            ctx = SerializationContext(topic_from_subject(subject), MessageField.VALUE)

            # Case A: Empty SSN -> should fail.
            with pytest.raises(Exception) as exc_info:
                ser({"name": "Empty SSN", "ssn": ""}, ctx)
            assert is_rule_error(exc_info.value), (
                f"Expected rule error for empty PII, got: {exc_info.value}"
            )

            # Case B: Valid SSN -> should succeed.
            data = ser({"name": "Valid SSN", "ssn": "123-45-6789"}, ctx)
            assert data is not None
        finally:
            delete_subject(subject)

    # -----------------------------------------------------------------
    # 7. CEL_FIELD TRANSFORM — normalize country to uppercase on WRITE
    # -----------------------------------------------------------------
    def test_cel_field_transform_normalizes_country(self):
        subject = unique_subject("py-cel-field-upper")
        try:
            body = json.dumps({
                "schemaType": "AVRO",
                "schema": ADDRESS_SCHEMA,
                "ruleSet": {
                    "domainRules": [{
                        "name": "normalize-country",
                        "kind": "TRANSFORM",
                        "type": "CEL_FIELD",
                        "mode": "WRITE",
                        "expr": "name == 'country' ; value.upperAscii()",
                        "onFailure": "ERROR",
                    }],
                },
            })
            register_schema(subject, body)

            client = new_client()
            ser = new_rule_serializer(client, ADDRESS_SCHEMA)
            deser = new_rule_deserializer(client, ADDRESS_SCHEMA)
            ctx = SerializationContext(topic_from_subject(subject), MessageField.VALUE)

            addr = {"street": "123 Main St", "country": "us"}
            data = ser(addr, ctx)
            assert data is not None

            result = deser(data, ctx)
            assert result["street"] == "123 Main St", "street should be unchanged"
            assert result["country"] == "US", (
                "country should be uppercased by WRITE transform"
            )
        finally:
            delete_subject(subject)

    # -----------------------------------------------------------------
    # 8. Disabled rule — should be skipped
    # -----------------------------------------------------------------
    def test_disabled_rule_skipped(self):
        subject = unique_subject("py-cel-disabled")
        try:
            body = json.dumps({
                "schemaType": "AVRO",
                "schema": ORDER_SCHEMA,
                "ruleSet": {
                    "domainRules": [{
                        "name": "high-amount-only",
                        "kind": "CONDITION",
                        "type": "CEL",
                        "mode": "WRITE",
                        "expr": "message.Amount > 1000.0",
                        "onFailure": "ERROR",
                        "disabled": True,
                    }],
                },
            })
            register_schema(subject, body)

            client = new_client()
            ser = new_rule_serializer(client, ORDER_SCHEMA)
            deser = new_rule_deserializer(client, ORDER_SCHEMA)
            ctx = SerializationContext(topic_from_subject(subject), MessageField.VALUE)

            # Amount is 5.0 which violates the rule, but the rule is disabled.
            order = {"orderId": "ORD-LOW", "amount": 5.0, "currency": "USD"}
            data = ser(order, ctx)
            assert data is not None, "serialization should succeed (rule is disabled)"

            result = deser(data, ctx)
            assert result["orderId"] == "ORD-LOW"
            assert result["amount"] == 5.0
            assert result["currency"] == "USD"
        finally:
            delete_subject(subject)


# ###########################################################################
#
# CEL RULES — JSON Schema and Protobuf (API-level storage verification)
#
# ###########################################################################

@pytest.mark.data_contracts
class TestCelRulesApiLevel:
    """Verify rules are correctly stored and returned for JSON Schema and
    Protobuf subjects via the REST API.

    The Python confluent-kafka library (like Go) only provides Avro
    serializer/deserializer with rule execution. JSON Schema and Protobuf
    SerDe with rules are only available in the Java client. These tests
    validate storage/retrieval of rules via the REST API.
    """

    # -----------------------------------------------------------------
    # 9. Rules stored with JSON Schema
    # -----------------------------------------------------------------
    def test_rules_stored_with_json_schema(self):
        subject = unique_subject("py-cel-jsonschema")
        try:
            json_schema_str = json.dumps({
                "$schema": "http://json-schema.org/draft-07/schema#",
                "title": "Product",
                "type": "object",
                "properties": {
                    "name": {"type": "string"},
                    "price": {"type": "number"},
                    "sku": {"type": "string"},
                },
                "required": ["name", "price", "sku"],
            })

            body = json.dumps({
                "schemaType": "JSON",
                "schema": json_schema_str,
                "ruleSet": {
                    "domainRules": [
                        {
                            "name": "nameNotEmpty",
                            "kind": "CONDITION",
                            "type": "CEL",
                            "mode": "WRITE",
                            "expr": "message.name != ''",
                            "onFailure": "ERROR",
                        },
                        {
                            "name": "pricePositive",
                            "kind": "CONDITION",
                            "type": "CEL",
                            "mode": "WRITE",
                            "expr": "message.price > 0",
                            "onFailure": "ERROR",
                        },
                    ],
                },
            })

            schema_id = register_schema(subject, body)
            assert schema_id > 0, "schema should be registered with a positive ID"

            # Fetch version response and verify rules are present.
            version_resp = get_schema_version(subject, 1)
            assert "ruleSet" in version_resp
            assert "nameNotEmpty" in version_resp
            assert "pricePositive" in version_resp
            assert "CEL" in version_resp
            assert "CONDITION" in version_resp

            # Parse and structurally verify the ruleSet.
            parsed = json.loads(version_resp)
            assert parsed["schemaType"] == "JSON"

            rule_set = parsed["ruleSet"]
            domain_rules = rule_set["domainRules"]
            assert len(domain_rules) == 2

            assert domain_rules[0]["name"] == "nameNotEmpty"
            assert domain_rules[0]["type"] == "CEL"
            assert domain_rules[0]["kind"] == "CONDITION"
            assert domain_rules[0]["mode"] == "WRITE"

            assert domain_rules[1]["name"] == "pricePositive"
            assert domain_rules[1]["type"] == "CEL"
        finally:
            delete_subject(subject)

    # -----------------------------------------------------------------
    # 10. Rules stored with Protobuf
    # -----------------------------------------------------------------
    def test_rules_stored_with_protobuf(self):
        subject = unique_subject("py-cel-protobuf")
        try:
            proto_schema = (
                'syntax = "proto3";\n'
                "package com.axonops.test.cel;\n\n"
                "message Product {\n"
                "    string name = 1;\n"
                "    double price = 2;\n"
                "    string sku = 3;\n"
                "}\n"
            )

            body = json.dumps({
                "schemaType": "PROTOBUF",
                "schema": proto_schema,
                "ruleSet": {
                    "domainRules": [
                        {
                            "name": "nameNotEmpty",
                            "kind": "CONDITION",
                            "type": "CEL",
                            "mode": "WRITE",
                            "expr": "message.name != ''",
                            "onFailure": "ERROR",
                        },
                        {
                            "name": "pricePositive",
                            "kind": "CONDITION",
                            "type": "CEL",
                            "mode": "WRITE",
                            "expr": "message.price > 0.0",
                            "onFailure": "ERROR",
                        },
                    ],
                },
            })

            schema_id = register_schema(subject, body)
            assert schema_id > 0, "schema should be registered with a positive ID"

            # Fetch version response and verify rules are present.
            version_resp = get_schema_version(subject, 1)
            assert "ruleSet" in version_resp
            assert "nameNotEmpty" in version_resp
            assert "pricePositive" in version_resp
            assert "CEL" in version_resp
            assert "CONDITION" in version_resp

            # Parse and structurally verify the ruleSet.
            parsed = json.loads(version_resp)
            assert parsed["schemaType"] == "PROTOBUF"

            rule_set = parsed["ruleSet"]
            domain_rules = rule_set["domainRules"]
            assert len(domain_rules) == 2

            assert domain_rules[0]["name"] == "nameNotEmpty"
            assert domain_rules[0]["type"] == "CEL"
            assert domain_rules[0]["kind"] == "CONDITION"
            assert domain_rules[0]["mode"] == "WRITE"

            assert domain_rules[1]["name"] == "pricePositive"
            assert domain_rules[1]["type"] == "CEL"
        finally:
            delete_subject(subject)


# ###########################################################################
#
# MIGRATION RULES (JSONata UPGRADE / DOWNGRADE)
#
# Pattern (mirrors Go):
#   1. Set compatibility to NONE (so v2 can break v1 freely).
#   2. Register v1 schema, serialize data with v1 as latest.
#   3. Register v2 schema with a JSONata migration rule.
#   4. Create a FRESH client+deserializer (avoids cached v1 metadata).
#   5. Deserialize the v1-encoded bytes -- the migration rule transforms
#      the payload into the v2 shape automatically.
#
# ###########################################################################

# -- Migration Avro Schemas --

ORDER_V1_SCHEMA = json.dumps({
    "type": "record",
    "name": "OrderV1",
    "namespace": "com.example",
    "fields": [
        {"name": "orderId", "type": "string"},
        {"name": "state", "type": "string"},
    ],
})

ORDER_V2_SCHEMA = json.dumps({
    "type": "record",
    "name": "OrderV2",
    "namespace": "com.example",
    "fields": [
        {"name": "orderId", "type": "string"},
        {"name": "status", "type": "string"},
    ],
})

PAYMENT_V1_SCHEMA = json.dumps({
    "type": "record",
    "name": "PaymentV1",
    "namespace": "com.example",
    "fields": [
        {"name": "id", "type": "string"},
        {"name": "amount", "type": "double"},
    ],
})

PAYMENT_V2_SCHEMA = json.dumps({
    "type": "record",
    "name": "PaymentV2",
    "namespace": "com.example",
    "fields": [
        {"name": "id", "type": "string"},
        {"name": "amount", "type": "double"},
        {"name": "currency", "type": "string", "default": "UNKNOWN"},
    ],
})

PERSON_V1_SCHEMA = json.dumps({
    "type": "record",
    "name": "PersonV1",
    "namespace": "com.example",
    "fields": [
        {"name": "firstName", "type": "string"},
        {"name": "lastName", "type": "string"},
    ],
})

PERSON_V2_SCHEMA = json.dumps({
    "type": "record",
    "name": "PersonV2",
    "namespace": "com.example",
    "fields": [
        {"name": "fullName", "type": "string"},
    ],
})

# -- Downgrade migration schemas (use same record name for Avro compatibility) --

ORDER_DG_V1_SCHEMA = json.dumps({
    "type": "record",
    "name": "Order",
    "namespace": "com.example",
    "fields": [
        {"name": "orderId", "type": "string"},
        {"name": "state", "type": "string"},
    ],
})

ORDER_DG_V2_SCHEMA = json.dumps({
    "type": "record",
    "name": "Order",
    "namespace": "com.example",
    "fields": [
        {"name": "orderId", "type": "string"},
        {"name": "status", "type": "string"},
    ],
})

SHIPMENT_V1_SCHEMA = json.dumps({
    "type": "record",
    "name": "Shipment",
    "namespace": "com.example",
    "fields": [
        {"name": "shipmentId", "type": "string"},
        {"name": "state", "type": "string"},
        {"name": "location", "type": "string"},
    ],
})

SHIPMENT_V2_SCHEMA = json.dumps({
    "type": "record",
    "name": "Shipment",
    "namespace": "com.example",
    "fields": [
        {"name": "shipmentId", "type": "string"},
        {"name": "status", "type": "string"},
        {"name": "region", "type": "string"},
    ],
})


@pytest.mark.data_contracts
@requires_executors
class TestMigrationRules:
    """JSONata migration rule tests exercised through Confluent Python SerDe."""

    # -----------------------------------------------------------------
    # 11. UPGRADE — field rename (state -> status)
    # -----------------------------------------------------------------
    def test_upgrade_field_rename(self):
        subject = unique_subject("py-migrate-rename")
        try:
            set_subject_config(subject, json.dumps({"compatibility": "NONE"}))

            # Register v1 and serialize.
            register_schema(subject, json.dumps({"schema": ORDER_V1_SCHEMA}))

            client1 = new_client()
            ser = new_rule_serializer(client1, ORDER_V1_SCHEMA)
            ctx = SerializationContext(topic_from_subject(subject), MessageField.VALUE)

            v1_data = {"orderId": "ORD-001", "state": "PENDING"}
            v1_bytes = ser(v1_data, ctx)
            assert v1_bytes is not None

            # Register v2 with UPGRADE migration rule.
            v2_body = json.dumps({
                "schema": ORDER_V2_SCHEMA,
                "schemaType": "AVRO",
                "ruleSet": {
                    "migrationRules": [{
                        "name": "renameStateToStatus",
                        "kind": "TRANSFORM",
                        "type": "JSONATA",
                        "mode": "UPGRADE",
                        "expr": "$merge([$sift($, function($v, $k) {$k != 'state'}), {'status': $.state}])",
                    }],
                },
            })
            register_schema(subject, v2_body)

            # Fresh client + deserializer to pick up v2 metadata.
            client2 = new_client()
            deser = new_rule_deserializer(client2, ORDER_V2_SCHEMA)
            result = deser(v1_bytes, ctx)

            assert result["orderId"] == "ORD-001"
            assert result["status"] == "PENDING", (
                "migration should rename state -> status"
            )
        finally:
            delete_subject(subject)

    # -----------------------------------------------------------------
    # 12. Bidirectional UPGRADE + DOWNGRADE stored
    # -----------------------------------------------------------------
    def test_bidirectional_upgrade_downgrade(self):
        subject = unique_subject("py-migrate-bidir")
        try:
            set_subject_config(subject, json.dumps({"compatibility": "NONE"}))

            # Register v1 and serialize.
            register_schema(subject, json.dumps({"schema": ORDER_V1_SCHEMA}))

            client1 = new_client()
            ser = new_rule_serializer(client1, ORDER_V1_SCHEMA)
            ctx = SerializationContext(topic_from_subject(subject), MessageField.VALUE)

            v1_bytes = ser({"orderId": "ORD-002", "state": "SHIPPED"}, ctx)
            assert v1_bytes is not None

            # Register v2 with BOTH upgrade and downgrade rules.
            v2_body = json.dumps({
                "schema": ORDER_V2_SCHEMA,
                "schemaType": "AVRO",
                "ruleSet": {
                    "migrationRules": [
                        {
                            "name": "upgradeStateToStatus",
                            "kind": "TRANSFORM",
                            "type": "JSONATA",
                            "mode": "UPGRADE",
                            "expr": "$merge([$sift($, function($v, $k) {$k != 'state'}), {'status': $.state}])",
                        },
                        {
                            "name": "downgradeStatusToState",
                            "kind": "TRANSFORM",
                            "type": "JSONATA",
                            "mode": "DOWNGRADE",
                            "expr": "$merge([$sift($, function($v, $k) {$k != 'status'}), {'state': $.status}])",
                        },
                    ],
                },
            })
            register_schema(subject, v2_body)

            # Verify both rules are stored in the version response.
            version_resp = get_schema_version(subject, 2)
            assert "upgradeStateToStatus" in version_resp
            assert "downgradeStatusToState" in version_resp

            # Fresh client + deserializer for upgrade.
            client2 = new_client()
            deser = new_rule_deserializer(client2, ORDER_V2_SCHEMA)
            result = deser(v1_bytes, ctx)

            assert result["orderId"] == "ORD-002"
            assert result["status"] == "SHIPPED", (
                "upgrade migration should rename state -> status"
            )
        finally:
            delete_subject(subject)

    # -----------------------------------------------------------------
    # 13. UPGRADE — field addition with default
    # -----------------------------------------------------------------
    def test_upgrade_field_addition_with_default(self):
        subject = unique_subject("py-migrate-addfield")
        try:
            set_subject_config(subject, json.dumps({"compatibility": "NONE"}))

            # Register v1 and serialize.
            register_schema(subject, json.dumps({"schema": PAYMENT_V1_SCHEMA}))

            client1 = new_client()
            ser = new_rule_serializer(client1, PAYMENT_V1_SCHEMA)
            ctx = SerializationContext(topic_from_subject(subject), MessageField.VALUE)

            v1_bytes = ser({"id": "PAY-001", "amount": 99.99}, ctx)
            assert v1_bytes is not None

            # Register v2 with currency addition migration.
            v2_body = json.dumps({
                "schema": PAYMENT_V2_SCHEMA,
                "schemaType": "AVRO",
                "ruleSet": {
                    "migrationRules": [{
                        "name": "addCurrencyDefault",
                        "kind": "TRANSFORM",
                        "type": "JSONATA",
                        "mode": "UPGRADE",
                        "expr": "$merge([$, {'currency': 'USD'}])",
                    }],
                },
            })
            register_schema(subject, v2_body)

            # Fresh client + deserializer.
            client2 = new_client()
            deser = new_rule_deserializer(client2, PAYMENT_V2_SCHEMA)
            result = deser(v1_bytes, ctx)

            assert result["id"] == "PAY-001"
            assert result["amount"] == 99.99
            assert result["currency"] == "USD", (
                "migration should set currency to USD"
            )
        finally:
            delete_subject(subject)

    # -----------------------------------------------------------------
    # 14. DOWNGRADE — field rename execution
    # -----------------------------------------------------------------
    def test_downgrade_field_rename_execution(self):
        subject = unique_subject("py-migrate-dg-rename")
        try:
            set_subject_config(subject, json.dumps({"compatibility": "NONE"}))

            # Register v1 with metadata major=1.
            v1_body = json.dumps({
                "schema": ORDER_DG_V1_SCHEMA,
                "schemaType": "AVRO",
                "metadata": {"properties": {"major": "1"}},
            })
            register_schema(subject, v1_body)

            # Register v2 with metadata major=2 and UPGRADE + DOWNGRADE rules.
            v2_body = json.dumps({
                "schema": ORDER_DG_V2_SCHEMA,
                "schemaType": "AVRO",
                "metadata": {"properties": {"major": "2"}},
                "ruleSet": {
                    "migrationRules": [
                        {
                            "name": "upgradeStateToStatus",
                            "kind": "TRANSFORM",
                            "type": "JSONATA",
                            "mode": "UPGRADE",
                            "expr": "$merge([$sift($, function($v, $k) {$k != 'state'}), {'status': $.state}])",
                        },
                        {
                            "name": "downgradeStatusToState",
                            "kind": "TRANSFORM",
                            "type": "JSONATA",
                            "mode": "DOWNGRADE",
                            "expr": "$merge([$sift($, function($v, $k) {$k != 'status'}), {'state': $.status}])",
                        },
                    ],
                },
            })
            register_schema(subject, v2_body)

            # Serialize with v2 (the latest version).
            client1 = new_client()
            ser = new_rule_serializer(client1, ORDER_DG_V2_SCHEMA)
            ctx = SerializationContext(topic_from_subject(subject), MessageField.VALUE)

            v2_bytes = ser({"orderId": "ORD-DG-001", "status": "ACTIVE"}, ctx)
            assert v2_bytes is not None

            # Deserialize with reader pinned to v1 via metadata.
            client2 = new_client()
            deser = new_metadata_pinned_deserializer(
                client2, ORDER_DG_V1_SCHEMA, {"major": "1"}
            )
            result = deser(v2_bytes, ctx)

            assert result["orderId"] == "ORD-DG-001"
            assert result["state"] == "ACTIVE", (
                "DOWNGRADE should rename status back to state"
            )
        finally:
            delete_subject(subject)

    # -----------------------------------------------------------------
    # 15. DOWNGRADE — multiple field transforms
    # -----------------------------------------------------------------
    def test_downgrade_multiple_field_transforms(self):
        subject = unique_subject("py-migrate-dg-multi")
        try:
            set_subject_config(subject, json.dumps({"compatibility": "NONE"}))

            # Register v1 with metadata major=1.
            v1_body = json.dumps({
                "schema": SHIPMENT_V1_SCHEMA,
                "schemaType": "AVRO",
                "metadata": {"properties": {"major": "1"}},
            })
            register_schema(subject, v1_body)

            # Register v2 with metadata major=2 and UPGRADE + DOWNGRADE rules.
            v2_body = json.dumps({
                "schema": SHIPMENT_V2_SCHEMA,
                "schemaType": "AVRO",
                "metadata": {"properties": {"major": "2"}},
                "ruleSet": {
                    "migrationRules": [
                        {
                            "name": "upgradeFields",
                            "kind": "TRANSFORM",
                            "type": "JSONATA",
                            "mode": "UPGRADE",
                            "expr": "$merge([$sift($, function($v, $k) {$k != 'state' and $k != 'location'}), {'status': $.state, 'region': $.location}])",
                        },
                        {
                            "name": "downgradeFields",
                            "kind": "TRANSFORM",
                            "type": "JSONATA",
                            "mode": "DOWNGRADE",
                            "expr": "$merge([$sift($, function($v, $k) {$k != 'status' and $k != 'region'}), {'state': $.status, 'location': $.region}])",
                        },
                    ],
                },
            })
            register_schema(subject, v2_body)

            # Serialize with v2.
            client1 = new_client()
            ser = new_rule_serializer(client1, SHIPMENT_V2_SCHEMA)
            ctx = SerializationContext(topic_from_subject(subject), MessageField.VALUE)

            v2_bytes = ser(
                {"shipmentId": "SHIP-001", "status": "IN_TRANSIT", "region": "EU-WEST-1"},
                ctx,
            )
            assert v2_bytes is not None

            # Deserialize with reader pinned to v1 via metadata.
            client2 = new_client()
            deser = new_metadata_pinned_deserializer(
                client2, SHIPMENT_V1_SCHEMA, {"major": "1"}
            )
            result = deser(v2_bytes, ctx)

            assert result["shipmentId"] == "SHIP-001"
            assert result["state"] == "IN_TRANSIT", (
                "DOWNGRADE should rename status back to state"
            )
            assert result["location"] == "EU-WEST-1", (
                "DOWNGRADE should rename region back to location"
            )
        finally:
            delete_subject(subject)

    # -----------------------------------------------------------------
    # 16. Breaking change bridged by migration
    # -----------------------------------------------------------------
    def test_breaking_change_bridged_by_migration(self):
        subject = unique_subject("py-migrate-breaking")
        try:
            set_subject_config(subject, json.dumps({"compatibility": "NONE"}))

            # Register v1 and serialize.
            register_schema(subject, json.dumps({"schema": PERSON_V1_SCHEMA}))

            client1 = new_client()
            ser = new_rule_serializer(client1, PERSON_V1_SCHEMA)
            ctx = SerializationContext(topic_from_subject(subject), MessageField.VALUE)

            v1_bytes = ser({"firstName": "John", "lastName": "Doe"}, ctx)
            assert v1_bytes is not None

            # Register v2: completely breaking change (fullName only) with migration.
            v2_body = json.dumps({
                "schema": PERSON_V2_SCHEMA,
                "schemaType": "AVRO",
                "ruleSet": {
                    "migrationRules": [{
                        "name": "mergeNames",
                        "kind": "TRANSFORM",
                        "type": "JSONATA",
                        "mode": "UPGRADE",
                        "expr": "{'fullName': $.firstName & ' ' & $.lastName}",
                    }],
                },
            })
            register_schema(subject, v2_body)

            # Fresh client + deserializer.
            client2 = new_client()
            deser = new_rule_deserializer(client2, PERSON_V2_SCHEMA)
            result = deser(v1_bytes, ctx)

            assert result["fullName"] == "John Doe", (
                "migration should concatenate firstName+lastName into fullName"
            )
        finally:
            delete_subject(subject)


# ###########################################################################
#
# GLOBAL POLICIES (defaultRuleSet, overrideRuleSet, rule inheritance)
#
# ###########################################################################

@pytest.mark.data_contracts
@requires_executors
class TestGlobalPolicies:
    """Global policy tests for subject-level defaultRuleSet, overrideRuleSet,
    and rule inheritance across schema versions."""

    # -----------------------------------------------------------------
    # 17. defaultRuleSet applied
    # -----------------------------------------------------------------
    def test_default_ruleset_applied(self):
        subject = unique_subject("py-default-rule")
        try:
            # Set subject config with a defaultRuleSet that enforces amount > 0.
            config_body = json.dumps({
                "compatibility": "NONE",
                "defaultRuleSet": {
                    "domainRules": [{
                        "name": "amount-positive",
                        "kind": "CONDITION",
                        "type": "CEL",
                        "mode": "WRITE",
                        "expr": "message.Amount > 0.0",
                        "onFailure": "ERROR",
                    }],
                },
            })
            set_subject_config(subject, config_body)

            # Register the schema WITHOUT any ruleSet -- it should inherit default.
            register_schema(
                subject,
                json.dumps({"schema": ORDER_POLICY_SCHEMA}),
            )

            # Verify the inherited rule appears in the version response.
            version_resp = get_schema_version(subject, 1)
            assert "amount-positive" in version_resp, (
                "expected version response to contain inherited rule 'amount-positive'"
            )

            client = new_client()
            ser = new_rule_serializer(client, ORDER_POLICY_SCHEMA)
            ctx = SerializationContext(topic_from_subject(subject), MessageField.VALUE)

            # Negative amount should be rejected by the inherited default rule.
            with pytest.raises(Exception) as exc_info:
                ser({"orderId": "ORD-BAD", "amount": -1.0}, ctx)
            assert is_rule_error(exc_info.value), (
                f"Expected rule error, got: {exc_info.value}"
            )

            # Positive amount should succeed.
            data = ser({"orderId": "ORD-GOOD", "amount": 100.0}, ctx)
            assert data is not None and len(data) > 0
        finally:
            delete_subject(subject)

    # -----------------------------------------------------------------
    # 18. overrideRuleSet enforced
    # -----------------------------------------------------------------
    def test_override_ruleset_enforced(self):
        subject = unique_subject("py-override-rule")
        try:
            # Set subject config with an overrideRuleSet that enforces a strict range.
            config_body = json.dumps({
                "compatibility": "NONE",
                "overrideRuleSet": {
                    "domainRules": [{
                        "name": "amount-range-override",
                        "kind": "CONDITION",
                        "type": "CEL",
                        "mode": "WRITE",
                        "expr": "message.Amount > 0.0 && message.Amount < 10000.0",
                        "onFailure": "ERROR",
                    }],
                },
            })
            set_subject_config(subject, config_body)

            # Register schema WITH its own permissive rule (orderId non-empty).
            schema_body = json.dumps({
                "schema": ORDER_POLICY_SCHEMA,
                "ruleSet": {
                    "domainRules": [{
                        "name": "orderId-required",
                        "kind": "CONDITION",
                        "type": "CEL",
                        "mode": "WRITE",
                        "expr": "size(message.OrderID) > 0",
                        "onFailure": "ERROR",
                    }],
                },
            })
            register_schema(subject, schema_body)

            client = new_client()
            ser = new_rule_serializer(client, ORDER_POLICY_SCHEMA)
            ctx = SerializationContext(topic_from_subject(subject), MessageField.VALUE)

            # Amount 50000 exceeds the override range -- should fail.
            with pytest.raises(Exception) as exc_info:
                ser({"orderId": "ORD-OVER", "amount": 50000.0}, ctx)
            assert is_rule_error(exc_info.value), (
                f"Expected rule error for override violation, got: {exc_info.value}"
            )

            # Empty orderId violates the schema-level rule -- should fail.
            with pytest.raises(Exception) as exc_info:
                ser({"orderId": "", "amount": 100.0}, ctx)
            assert is_rule_error(exc_info.value), (
                f"Expected rule error for empty orderId, got: {exc_info.value}"
            )

            # Valid order within range and with non-empty orderId -- should succeed.
            data = ser({"orderId": "ORD-VALID", "amount": 100.0}, ctx)
            assert data is not None and len(data) > 0
        finally:
            delete_subject(subject)

    # -----------------------------------------------------------------
    # 19. Rule inheritance from v1 to v2
    # -----------------------------------------------------------------
    def test_rule_inheritance_v1_to_v2(self):
        subject = unique_subject("py-rule-inherit")
        try:
            set_subject_config(subject, json.dumps({"compatibility": "NONE"}))

            # Register v1 with an explicit rule: amount must be positive.
            v1_body = json.dumps({
                "schema": ORDER_POLICY_SCHEMA,
                "ruleSet": {
                    "domainRules": [{
                        "name": "amount-positive",
                        "kind": "CONDITION",
                        "type": "CEL",
                        "mode": "WRITE",
                        "expr": "message.Amount > 0.0",
                        "onFailure": "ERROR",
                    }],
                },
            })
            register_schema(subject, v1_body)

            # Register v2 WITHOUT any ruleSet -- should inherit from v1.
            register_schema(
                subject,
                json.dumps({"schema": ORDER_POLICY_V2_SCHEMA}),
            )

            # Verify the inherited rule appears in the v2 response.
            v2_resp = get_schema_version(subject, 2)
            assert "amount-positive" in v2_resp, (
                "expected v2 response to contain inherited rule 'amount-positive'"
            )

            client = new_client()
            ser = new_rule_serializer(client, ORDER_POLICY_V2_SCHEMA)
            ctx = SerializationContext(topic_from_subject(subject), MessageField.VALUE)

            # Negative amount should be rejected by the inherited rule.
            with pytest.raises(Exception) as exc_info:
                ser({"orderId": "V2-BAD", "amount": -1.0, "notes": None}, ctx)
            assert is_rule_error(exc_info.value), (
                f"Expected rule error, got: {exc_info.value}"
            )

            # Positive amount with notes should succeed.
            data = ser(
                {"orderId": "V2-GOOD", "amount": 50.0, "notes": {"string": "priority order"}},
                ctx,
            )
            assert data is not None and len(data) > 0
        finally:
            delete_subject(subject)

    # -----------------------------------------------------------------
    # 20. PII masking via tag propagation
    # -----------------------------------------------------------------
    def test_pii_masking_via_tag_propagation(self):
        subject = unique_subject("py-pii-mask")
        try:
            set_subject_config(subject, json.dumps({"compatibility": "NONE"}))

            # Register Contact schema with CEL_FIELD rule that masks PII-tagged fields.
            body = json.dumps({
                "schema": CONTACT_SCHEMA,
                "ruleSet": {
                    "domainRules": [{
                        "name": "mask-pii",
                        "kind": "TRANSFORM",
                        "type": "CEL_FIELD",
                        "mode": "READ",
                        "tags": ["PII"],
                        "expr": "typeName == 'STRING' ; 'REDACTED'",
                        "onFailure": "ERROR",
                    }],
                },
            })
            register_schema(subject, body)

            # Verify the rule and PII tag appear in the version response.
            version_resp = get_schema_version(subject, 1)
            assert "mask-pii" in version_resp
            assert "PII" in version_resp

            client = new_client()
            ser = new_rule_serializer(client, CONTACT_SCHEMA)
            deser = new_rule_deserializer(client, CONTACT_SCHEMA)
            ctx = SerializationContext(topic_from_subject(subject), MessageField.VALUE)

            # Serialize a contact with real data.
            data = ser({"name": "Alice Smith", "email": "user@example.com"}, ctx)
            assert data is not None and len(data) > 0

            # Deserialize -- the PII-tagged email field should be redacted.
            result = deser(data, ctx)
            assert result["name"] == "Alice Smith", (
                "name should not be redacted (no PII tag)"
            )
            assert result["email"] == "REDACTED", (
                "email should be redacted via PII masking rule"
            )
        finally:
            delete_subject(subject)
