"""
JSON Schema Serializer/Deserializer Compatibility Tests for AxonOps Schema Registry.

Tests verify that the Confluent Python JSON Schema serializers produce the same
schema fingerprints and wire format as expected by AxonOps Schema Registry.
"""
import time
import struct
import json
import requests
from concurrent.futures import ThreadPoolExecutor, as_completed
from threading import Barrier

import pytest
from confluent_kafka.schema_registry import SchemaRegistryClient, Schema
from confluent_kafka.schema_registry.json_schema import JSONSerializer, JSONDeserializer
from confluent_kafka.serialization import SerializationContext, MessageField


# JSON Schema definitions
USER_JSON_SCHEMA = json.dumps({
    "$schema": "http://json-schema.org/draft-07/schema#",
    "title": "User",
    "type": "object",
    "properties": {
        "id": {"type": "integer"},
        "name": {"type": "string"},
        "email": {"type": "string", "format": "email"}
    },
    "required": ["id", "name"]
})

ORDER_JSON_SCHEMA = json.dumps({
    "$schema": "http://json-schema.org/draft-07/schema#",
    "title": "Order",
    "type": "object",
    "properties": {
        "orderId": {"type": "string"},
        "customerId": {"type": "string"},
        "amount": {"type": "number"},
        "items": {
            "type": "array",
            "items": {
                "type": "object",
                "properties": {
                    "productId": {"type": "string"},
                    "quantity": {"type": "integer"}
                }
            }
        }
    },
    "required": ["orderId", "customerId", "amount"]
})

# Valid Confluent compatibility levels
VALID_COMPATIBILITY_LEVELS = {
    "NONE", "BACKWARD", "FORWARD", "FULL",
    "BACKWARD_TRANSITIVE", "FORWARD_TRANSITIVE", "FULL_TRANSITIVE"
}


class TestJsonSchemaRegistration:
    """Test JSON Schema registration produces consistent fingerprints."""

    def test_register_user_jsonschema(self, schema_registry_url, confluent_version):
        """Test that User JSON schema registration produces consistent schema ID."""
        client = SchemaRegistryClient({"url": schema_registry_url})

        schema = Schema(USER_JSON_SCHEMA, "JSON")

        subject = f"python-json-user-{confluent_version}-value"
        schema_id = client.register_schema(subject, schema)

        assert schema_id > 0, "Schema ID should be positive"

        retrieved = client.get_schema(schema_id)
        assert retrieved is not None
        assert retrieved.schema_type == "JSON"

        print(f"Confluent Python {confluent_version}: JSON User schema registered with ID {schema_id}")

    def test_register_order_jsonschema(self, schema_registry_url, confluent_version):
        """Test that Order JSON schema registration produces consistent schema ID."""
        client = SchemaRegistryClient({"url": schema_registry_url})

        schema = Schema(ORDER_JSON_SCHEMA, "JSON")

        subject = f"python-json-order-{confluent_version}-value"
        schema_id = client.register_schema(subject, schema)

        assert schema_id > 0
        print(f"Confluent Python {confluent_version}: JSON Order schema registered with ID {schema_id}")

    def test_jsonschema_deduplication(self, schema_registry_url, confluent_version):
        """Test that registering the same JSON schema twice returns the same ID."""
        client = SchemaRegistryClient({"url": schema_registry_url})

        schema = Schema(USER_JSON_SCHEMA, "JSON")

        subject = f"python-json-dedup-{confluent_version}-value"

        id1 = client.register_schema(subject, schema)
        id2 = client.register_schema(subject, schema)

        assert id1 == id2, f"Same JSON schema should return same ID: {id1} vs {id2}"
        print(f"JSON schema deduplication verified: ID {id1}")


class TestJsonSchemaSerialization:
    """Test JSON Schema serialization produces correct wire format."""

    def test_wire_format_structure(self, schema_registry_url, confluent_version):
        """Test that serialized JSON data follows Confluent wire format."""
        client = SchemaRegistryClient({"url": schema_registry_url})

        def user_to_dict(user, ctx):
            return user

        serializer = JSONSerializer(USER_JSON_SCHEMA, client, user_to_dict)

        user = {"id": 1, "name": "Test User", "email": "test@example.com"}

        ctx = SerializationContext(f"python-json-wire-{confluent_version}", MessageField.VALUE)
        serialized = serializer(user, ctx)

        assert serialized is not None
        assert len(serialized) >= 5, "Serialized data must have at least 5 bytes"

        # Check wire format
        magic_byte = serialized[0]
        schema_id = struct.unpack(">I", serialized[1:5])[0]

        assert magic_byte == 0, f"Magic byte should be 0, got {magic_byte}"
        assert schema_id > 0, f"Schema ID should be positive, got {schema_id}"

        print(f"JSON wire format: magic=0x{magic_byte:02x}, schema_id={schema_id}, len={len(serialized)}")

    def test_serialization_roundtrip(self, schema_registry_url, confluent_version):
        """Test that JSON data can be serialized and deserialized correctly."""
        client = SchemaRegistryClient({"url": schema_registry_url})

        def user_to_dict(user, ctx):
            return user

        def dict_to_user(data, ctx):
            return data

        serializer = JSONSerializer(USER_JSON_SCHEMA, client, user_to_dict)
        deserializer = JSONDeserializer(USER_JSON_SCHEMA, dict_to_user)

        original = {"id": 42, "name": "Jane Doe", "email": "jane@example.com"}

        ctx = SerializationContext(f"python-json-roundtrip-{confluent_version}", MessageField.VALUE)

        serialized = serializer(original, ctx)
        deserialized = deserializer(serialized, ctx)

        assert deserialized["id"] == original["id"]
        assert deserialized["name"] == original["name"]
        assert deserialized["email"] == original["email"]

        print(f"JSON roundtrip verified: {original['name']}")

    def test_complex_json_schema(self, schema_registry_url, confluent_version):
        """Test serialization of complex JSON schema with nested objects."""
        client = SchemaRegistryClient({"url": schema_registry_url})

        def order_to_dict(order, ctx):
            return order

        def dict_to_order(data, ctx):
            return data

        serializer = JSONSerializer(ORDER_JSON_SCHEMA, client, order_to_dict)
        deserializer = JSONDeserializer(ORDER_JSON_SCHEMA, dict_to_order)

        order = {
            "orderId": "ORD-001",
            "customerId": "CUST-123",
            "amount": 199.99,
            "items": [
                {"productId": "PROD-A", "quantity": 2},
                {"productId": "PROD-B", "quantity": 1}
            ]
        }

        ctx = SerializationContext(f"python-json-order-{confluent_version}", MessageField.VALUE)

        serialized = serializer(order, ctx)
        deserialized = deserializer(serialized, ctx)

        assert deserialized["orderId"] == order["orderId"]
        assert deserialized["amount"] == order["amount"]
        assert len(deserialized["items"]) == 2

        print(f"Complex JSON schema roundtrip verified: {order['orderId']}")


class TestJsonSchemaEvolution:
    """Test JSON Schema evolution compatibility."""

    def test_add_optional_property(self, schema_registry_url, confluent_version):
        """Test adding an optional property (backward compatible)."""
        client = SchemaRegistryClient({"url": schema_registry_url})

        v1_schema = json.dumps({
            "$schema": "http://json-schema.org/draft-07/schema#",
            "title": "Config",
            "type": "object",
            "properties": {
                "name": {"type": "string"},
                "value": {"type": "string"}
            },
            "required": ["name", "value"]
        })

        v2_schema = json.dumps({
            "$schema": "http://json-schema.org/draft-07/schema#",
            "title": "Config",
            "type": "object",
            "properties": {
                "name": {"type": "string"},
                "value": {"type": "string"},
                "description": {"type": "string"}
            },
            "required": ["name", "value"]
        })

        subject = f"python-json-evolution-{confluent_version}-value"

        schema1 = Schema(v1_schema, "JSON")
        id1 = client.register_schema(subject, schema1)

        schema2 = Schema(v2_schema, "JSON")
        id2 = client.register_schema(subject, schema2)

        assert id1 != id2, "Different schemas should have different IDs"
        print(f"JSON schema evolution: v1 ID={id1}, v2 ID={id2}")


class TestJsonSchemaGlobalSchemaID:
    """Test global schema ID behavior (Confluent-compatible)."""

    def test_same_schema_across_subjects(self, schema_registry_url, confluent_version):
        """Test that same JSON schema under different subjects returns same global ID."""
        client = SchemaRegistryClient({"url": schema_registry_url})

        schema = Schema(USER_JSON_SCHEMA, "JSON")

        subject1 = f"python-json-global1-{confluent_version}-value"
        subject2 = f"python-json-global2-{confluent_version}-value"

        # Register same schema under different subjects
        id1 = client.register_schema(subject1, schema)
        id2 = client.register_schema(subject2, schema)

        # Same schema content should produce same global ID (Confluent-compatible behavior)
        assert id1 == id2, f"Same JSON schema under different subjects should return same global ID: {id1} vs {id2}"

        # Structural verification - fetch and compare
        fetched1 = client.get_schema(id1)
        fetched2 = client.get_schema(id2)

        assert fetched1.schema_type == fetched2.schema_type, "Schema types should match"

        print(f"Global JSON schema ID verified: both subjects use ID {id1}")


class TestJsonSchemaConcurrentRegistration:
    """Test concurrent schema registration."""

    def test_concurrent_registration_returns_consistent_ids(self, schema_registry_url, confluent_version):
        """Test that concurrent registrations return the same schema ID."""
        subject = f"python-json-concurrent-{int(time.time() * 1000)}-value"
        num_threads = 10

        # Use a barrier to synchronize thread start
        barrier = Barrier(num_threads)
        results = []
        errors = []

        def register_schema(thread_id):
            try:
                # Each thread gets its own client
                thread_client = SchemaRegistryClient({"url": schema_registry_url})

                # Wait for all threads to be ready
                barrier.wait()

                schema = Schema(USER_JSON_SCHEMA, "JSON")
                schema_id = thread_client.register_schema(subject, schema)
                return schema_id
            except Exception as e:
                return e

        with ThreadPoolExecutor(max_workers=num_threads) as executor:
            futures = [executor.submit(register_schema, i) for i in range(num_threads)]
            for future in as_completed(futures):
                result = future.result()
                if isinstance(result, Exception):
                    errors.append(result)
                else:
                    results.append(result)

        assert len(errors) == 0, f"Concurrent registration errors: {errors}"

        # All concurrent registrations should return the same ID
        unique_ids = set(results)
        assert len(unique_ids) == 1, \
            f"All concurrent registrations should return the same schema ID, got: {unique_ids}"

        # Verify only one version was created
        client = SchemaRegistryClient({"url": schema_registry_url})
        versions = client.get_versions(subject)
        assert len(versions) == 1, \
            f"Only one version should exist after concurrent registration, got: {len(versions)}"

        print(f"Concurrent registration test passed: {num_threads} threads all got schema ID {results[0]}")


class TestJsonSchemaConfigEndpoints:
    """Test config endpoints."""

    def test_get_global_compatibility(self, schema_registry_url):
        """Test that global compatibility returns a valid Confluent level."""
        response = requests.get(f"{schema_registry_url}/config")
        assert response.status_code == 200

        config = response.json()
        compat_level = config.get("compatibilityLevel")

        assert compat_level in VALID_COMPATIBILITY_LEVELS, \
            f"Global compatibility should be a valid Confluent level, got: {compat_level}"

        print(f"Global compatibility: {compat_level}")


class TestJsonSchemaIncompatibleSchemaEvolution:
    """Test incompatible schema evolution fails correctly."""

    def test_incompatible_schema_rejected(self, schema_registry_url, confluent_version):
        """Test that incompatible schema evolution fails with correct error."""
        subject = f"python-json-incompat-{int(time.time() * 1000)}-value"
        client = SchemaRegistryClient({"url": schema_registry_url})

        # Register v1 schema
        schema = Schema(USER_JSON_SCHEMA, "JSON")
        client.register_schema(subject, schema)

        # Set subject compatibility to BACKWARD
        response = requests.put(
            f"{schema_registry_url}/config/{subject}",
            json={"compatibility": "BACKWARD"},
            headers={"Content-Type": "application/json"}
        )
        assert response.status_code == 200

        # Verify compatibility was set
        response = requests.get(f"{schema_registry_url}/config/{subject}")
        assert response.status_code == 200
        assert response.json().get("compatibilityLevel") == "BACKWARD"

        # Create incompatible schema: change email type from string to integer (breaking change)
        incompatible_schema = json.dumps({
            "$schema": "http://json-schema.org/draft-07/schema#",
            "title": "User",
            "type": "object",
            "properties": {
                "id": {"type": "integer"},
                "name": {"type": "string"},
                "email": {"type": "integer"}
            },
            "required": ["id", "name"]
        })

        # Try to register incompatible schema
        bad_schema = Schema(incompatible_schema, "JSON")
        with pytest.raises(Exception) as exc_info:
            client.register_schema(subject, bad_schema)

        error_msg = str(exc_info.value).lower()
        is_incompat_error = (
            "incompatible" in error_msg or
            "compatibility" in error_msg or
            "409" in error_msg or
            "422" in error_msg
        )
        assert is_incompat_error, f"Expected incompatibility error, got: {exc_info.value}"

        print("Incompatible JSON schema correctly rejected")


class TestJsonSchemaCacheBehavior:
    """Test cache behavior with fresh clients."""

    def test_fresh_client_cache_miss(self, schema_registry_url, confluent_version):
        """Test that a fresh client can fetch schema after cache bypass."""
        subject = f"python-json-cache-{int(time.time() * 1000)}-value"

        # Register schema with first client
        client1 = SchemaRegistryClient({"url": schema_registry_url})
        schema = Schema(USER_JSON_SCHEMA, "JSON")
        schema_id = client1.register_schema(subject, schema)

        # Create a completely new client (empty cache)
        client2 = SchemaRegistryClient({"url": schema_registry_url})

        # Fetch schema with fresh client (cache miss, must hit registry)
        fetched = client2.get_schema(schema_id)

        assert fetched is not None, "Fresh client should fetch schema by ID"
        assert fetched.schema_type == "JSON", "Schema type should be JSON"

        print("Cache behavior test passed")


class TestJsonSchemaCanonicalisation:
    """Test schema canonicalization."""

    def test_same_schema_different_formatting(self, schema_registry_url, confluent_version):
        """Test that same schema with different formatting returns same ID."""
        # Same JSON Schema content but with different formatting
        # This tests that the registry canonicalizes schemas before comparison
        #
        # NOTE: Some client versions may canonicalize client-side before POSTing,
        # so this test may pass even if server-side canonicalization is broken.
        # For strict server-side canonicalization validation, register via REST API directly.

        # Compact format (minimal whitespace)
        compact_schema = '{"$schema":"http://json-schema.org/draft-07/schema#","title":"Canonical","type":"object","properties":{"id":{"type":"integer"},"value":{"type":"string"}},"required":["id","value"]}'

        # Verbose format (extra whitespace) - using dict to ensure same structure
        verbose_schema = json.dumps({
            "$schema": "http://json-schema.org/draft-07/schema#",
            "title": "Canonical",
            "type": "object",
            "properties": {
                "id": {"type": "integer"},
                "value": {"type": "string"}
            },
            "required": ["id", "value"]
        }, indent=4)

        subject1 = f"python-json-canon1-{int(time.time() * 1000)}-value"
        subject2 = f"python-json-canon2-{int(time.time() * 1000)}-value"

        client = SchemaRegistryClient({"url": schema_registry_url})

        # Register compact schema
        schema1 = Schema(compact_schema, "JSON")
        id1 = client.register_schema(subject1, schema1)

        # Register verbose schema (should be canonicalized to same schema)
        schema2 = Schema(verbose_schema, "JSON")
        id2 = client.register_schema(subject2, schema2)

        # Same schema content (after canonicalization) should produce same global ID
        assert id1 == id2, \
            f"Same JSON schema with different formatting should return same global ID (canonicalization): {id1} vs {id2}"

        print(f"Schema canonicalization verified: both formats use schema ID {id1}")
