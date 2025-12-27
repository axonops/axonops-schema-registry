"""
Avro Serializer/Deserializer Compatibility Tests for AxonOps Schema Registry.

Tests verify that the Confluent Python Avro serializers produce the same
schema fingerprints and wire format as expected by AxonOps Schema Registry.
"""
import time
import struct
import requests
from concurrent.futures import ThreadPoolExecutor, as_completed
from threading import Barrier

import pytest
from confluent_kafka.schema_registry import SchemaRegistryClient, Schema
from confluent_kafka.schema_registry.avro import AvroSerializer, AvroDeserializer
from confluent_kafka.serialization import SerializationContext, MessageField


# Test Avro schemas
USER_SCHEMA = """{
    "type": "record",
    "name": "User",
    "namespace": "com.axonops.test",
    "fields": [
        {"name": "id", "type": "long"},
        {"name": "name", "type": "string"},
        {"name": "email", "type": ["null", "string"], "default": null}
    ]
}"""

SIMPLE_SCHEMA = """{
    "type": "record",
    "name": "SimpleRecord",
    "fields": [
        {"name": "value", "type": "string"}
    ]
}"""

PAYMENT_SCHEMA = """{
    "type": "record",
    "name": "Payment",
    "namespace": "com.axonops.test",
    "fields": [
        {"name": "id", "type": "string"},
        {"name": "amount", "type": "double"},
        {"name": "currency", "type": {"type": "enum", "name": "Currency", "symbols": ["USD", "EUR", "GBP"]}}
    ]
}"""

# Valid Confluent compatibility levels
VALID_COMPATIBILITY_LEVELS = {
    "NONE", "BACKWARD", "FORWARD", "FULL",
    "BACKWARD_TRANSITIVE", "FORWARD_TRANSITIVE", "FULL_TRANSITIVE"
}


class TestAvroSchemaRegistration:
    """Test schema registration produces consistent fingerprints."""

    def test_register_user_schema(self, schema_registry_url, confluent_version):
        """Test that User schema registration produces consistent schema ID."""
        client = SchemaRegistryClient({"url": schema_registry_url})

        schema = Schema(USER_SCHEMA, "AVRO")

        # Register schema
        subject = f"python-avro-user-{confluent_version}-value"
        schema_id = client.register_schema(subject, schema)

        assert schema_id > 0, "Schema ID should be positive"

        # Retrieve and verify schema
        retrieved = client.get_schema(schema_id)
        assert retrieved is not None
        assert retrieved.schema_type == "AVRO"

        print(f"Confluent Python {confluent_version}: User schema registered with ID {schema_id}")

    def test_register_simple_schema(self, schema_registry_url, confluent_version):
        """Test that SimpleRecord schema registration produces consistent schema ID."""
        client = SchemaRegistryClient({"url": schema_registry_url})

        schema = Schema(SIMPLE_SCHEMA, "AVRO")

        subject = f"python-avro-simple-{confluent_version}-value"
        schema_id = client.register_schema(subject, schema)

        assert schema_id > 0
        print(f"Confluent Python {confluent_version}: Simple schema registered with ID {schema_id}")

    def test_schema_deduplication(self, schema_registry_url, confluent_version):
        """Test that registering the same schema twice returns the same ID."""
        client = SchemaRegistryClient({"url": schema_registry_url})

        schema = Schema(USER_SCHEMA, "AVRO")

        subject = f"python-avro-dedup-{confluent_version}-value"

        # Register twice
        id1 = client.register_schema(subject, schema)
        id2 = client.register_schema(subject, schema)

        assert id1 == id2, f"Same schema should return same ID: {id1} vs {id2}"
        print(f"Schema deduplication verified: ID {id1}")


class TestAvroSerialization:
    """Test serialization produces correct wire format."""

    def test_wire_format_structure(self, schema_registry_url, confluent_version):
        """Test that serialized data follows Confluent wire format."""
        client = SchemaRegistryClient({"url": schema_registry_url})

        def user_to_dict(user, ctx):
            return user

        serializer = AvroSerializer(
            client,
            USER_SCHEMA,
            user_to_dict
        )

        # Create test data
        # For union types like ["null", "string"], fastavro expects the raw value, not a tagged dict
        user = {"id": 1, "name": "Test User", "email": "test@example.com"}

        # Serialize
        ctx = SerializationContext(f"python-wire-{confluent_version}", MessageField.VALUE)
        serialized = serializer(user, ctx)

        assert serialized is not None
        assert len(serialized) >= 5, "Serialized data must have at least 5 bytes (magic + schema ID)"

        # Check wire format: magic byte (0) + 4-byte schema ID (big-endian)
        magic_byte = serialized[0]
        schema_id = struct.unpack(">I", serialized[1:5])[0]

        assert magic_byte == 0, f"Magic byte should be 0, got {magic_byte}"
        assert schema_id > 0, f"Schema ID should be positive, got {schema_id}"

        print(f"Wire format verified: magic=0x{magic_byte:02x}, schema_id={schema_id}, total_len={len(serialized)}")

    def test_serialization_roundtrip(self, schema_registry_url, confluent_version):
        """Test that data can be serialized and deserialized correctly."""
        client = SchemaRegistryClient({"url": schema_registry_url})

        def user_to_dict(user, ctx):
            return user

        def dict_to_user(data, ctx):
            return data

        serializer = AvroSerializer(client, USER_SCHEMA, user_to_dict)
        deserializer = AvroDeserializer(client, USER_SCHEMA, dict_to_user)

        # Test data - for union types, fastavro expects the raw value
        original = {"id": 42, "name": "Jane Doe", "email": "jane@example.com"}

        ctx = SerializationContext(f"python-roundtrip-{confluent_version}", MessageField.VALUE)

        # Serialize
        serialized = serializer(original, ctx)

        # Deserialize
        deserialized = deserializer(serialized, ctx)

        assert deserialized["id"] == original["id"]
        assert deserialized["name"] == original["name"]
        assert deserialized["email"] == original["email"]

        print(f"Roundtrip verified for user: {original['name']}")

    def test_null_handling(self, schema_registry_url, confluent_version):
        """Test that null values are handled correctly."""
        client = SchemaRegistryClient({"url": schema_registry_url})

        def user_to_dict(user, ctx):
            return user

        def dict_to_user(data, ctx):
            return data

        serializer = AvroSerializer(client, USER_SCHEMA, user_to_dict)
        deserializer = AvroDeserializer(client, USER_SCHEMA, dict_to_user)

        # User with null email
        original = {"id": 100, "name": "No Email User", "email": None}

        ctx = SerializationContext(f"python-null-{confluent_version}", MessageField.VALUE)

        serialized = serializer(original, ctx)
        deserialized = deserializer(serialized, ctx)

        assert deserialized["email"] is None
        print("Null handling verified")


class TestAvroSchemaEvolution:
    """Test schema evolution compatibility."""

    def test_backward_compatible_schema(self, schema_registry_url, confluent_version):
        """Test registering a backward-compatible schema evolution."""
        client = SchemaRegistryClient({"url": schema_registry_url})

        # Original schema
        v1_schema = """{
            "type": "record",
            "name": "Event",
            "namespace": "com.axonops.evolution",
            "fields": [
                {"name": "id", "type": "long"},
                {"name": "type", "type": "string"}
            ]
        }"""

        # Backward compatible: add field with default
        v2_schema = """{
            "type": "record",
            "name": "Event",
            "namespace": "com.axonops.evolution",
            "fields": [
                {"name": "id", "type": "long"},
                {"name": "type", "type": "string"},
                {"name": "timestamp", "type": "long", "default": 0}
            ]
        }"""

        subject = f"python-evolution-{confluent_version}-value"

        # Register v1
        schema1 = Schema(v1_schema, "AVRO")
        id1 = client.register_schema(subject, schema1)

        # Register v2
        schema2 = Schema(v2_schema, "AVRO")
        id2 = client.register_schema(subject, schema2)

        assert id1 != id2, "Different schemas should have different IDs"
        assert id2 > id1, "Newer schema should have higher ID"

        print(f"Schema evolution: v1 ID={id1}, v2 ID={id2}")


class TestAvroPaymentSchema:
    """Test complex schema with enum type."""

    def test_payment_serialization(self, schema_registry_url, confluent_version):
        """Test serialization of schema with enum type."""
        client = SchemaRegistryClient({"url": schema_registry_url})

        def payment_to_dict(payment, ctx):
            return payment

        def dict_to_payment(data, ctx):
            return data

        serializer = AvroSerializer(client, PAYMENT_SCHEMA, payment_to_dict)
        deserializer = AvroDeserializer(client, PAYMENT_SCHEMA, dict_to_payment)

        payment = {"id": "pay-001", "amount": 99.99, "currency": "USD"}

        ctx = SerializationContext(f"python-payment-{confluent_version}", MessageField.VALUE)

        serialized = serializer(payment, ctx)
        deserialized = deserializer(serialized, ctx)

        assert deserialized["id"] == payment["id"]
        assert deserialized["amount"] == payment["amount"]
        assert deserialized["currency"] == payment["currency"]

        print(f"Payment serialization verified: {payment['id']}")


class TestAvroGlobalSchemaID:
    """Test global schema ID behavior (Confluent-compatible)."""

    def test_same_schema_across_subjects(self, schema_registry_url, confluent_version):
        """Test that same schema under different subjects returns same global ID."""
        client = SchemaRegistryClient({"url": schema_registry_url})

        schema = Schema(USER_SCHEMA, "AVRO")

        subject1 = f"python-avro-global1-{confluent_version}-value"
        subject2 = f"python-avro-global2-{confluent_version}-value"

        # Register same schema under different subjects
        id1 = client.register_schema(subject1, schema)
        id2 = client.register_schema(subject2, schema)

        # Same schema content should produce same global ID (Confluent-compatible behavior)
        assert id1 == id2, f"Same Avro schema under different subjects should return same global ID: {id1} vs {id2}"

        # Structural verification - fetch and compare
        fetched1 = client.get_schema(id1)
        fetched2 = client.get_schema(id2)

        assert fetched1.schema_type == fetched2.schema_type, "Schema types should match"

        print(f"Global Avro schema ID verified: both subjects use ID {id1}")


class TestAvroConcurrentRegistration:
    """Test concurrent schema registration."""

    def test_concurrent_registration_returns_consistent_ids(self, schema_registry_url, confluent_version):
        """Test that concurrent registrations return the same schema ID."""
        subject = f"python-avro-concurrent-{int(time.time() * 1000)}-value"
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

                schema = Schema(USER_SCHEMA, "AVRO")
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


class TestAvroConfigEndpoints:
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


class TestAvroIncompatibleSchemaEvolution:
    """Test incompatible schema evolution fails correctly."""

    def test_incompatible_schema_rejected(self, schema_registry_url, confluent_version):
        """Test that incompatible schema evolution fails with correct error."""
        subject = f"python-avro-incompat-{int(time.time() * 1000)}-value"
        client = SchemaRegistryClient({"url": schema_registry_url})

        # Register v1 schema
        schema = Schema(USER_SCHEMA, "AVRO")
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

        # Create incompatible schema: change email type from union to int (breaking change)
        incompatible_schema = """{
            "type": "record",
            "name": "User",
            "namespace": "com.axonops.test",
            "fields": [
                {"name": "id", "type": "long"},
                {"name": "name", "type": "string"},
                {"name": "email", "type": "int"}
            ]
        }"""

        # Try to register incompatible schema
        bad_schema = Schema(incompatible_schema, "AVRO")
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

        print("Incompatible schema correctly rejected")


class TestAvroCacheBehavior:
    """Test cache behavior with fresh clients."""

    def test_fresh_client_cache_miss(self, schema_registry_url, confluent_version):
        """Test that a fresh client can fetch schema after cache bypass."""
        subject = f"python-avro-cache-{int(time.time() * 1000)}-value"

        # Register schema with first client
        client1 = SchemaRegistryClient({"url": schema_registry_url})
        schema = Schema(USER_SCHEMA, "AVRO")
        schema_id = client1.register_schema(subject, schema)

        # Create a completely new client (empty cache)
        client2 = SchemaRegistryClient({"url": schema_registry_url})

        # Fetch schema with fresh client (cache miss, must hit registry)
        fetched = client2.get_schema(schema_id)

        assert fetched is not None, "Fresh client should fetch schema by ID"
        assert fetched.schema_type == "AVRO", "Schema type should be AVRO"

        print("Cache behavior test passed")


class TestAvroSchemaCanonicalisation:
    """Test schema canonicalization."""

    def test_same_schema_different_formatting(self, schema_registry_url, confluent_version):
        """Test that same schema with different formatting returns same ID."""
        # Same Avro schema content but with different formatting
        # This tests that the registry canonicalizes schemas before comparison
        #
        # NOTE: Some client versions may canonicalize client-side before POSTing,
        # so this test may pass even if server-side canonicalization is broken.
        # For strict server-side canonicalization validation, register via REST API directly.

        # Compact format (minimal whitespace)
        compact_schema = '{"type":"record","name":"Canonical","namespace":"com.axonops.canon","fields":[{"name":"id","type":"long"},{"name":"value","type":"string"}]}'

        # Verbose format (extra whitespace)
        verbose_schema = """{
            "type": "record",
            "name": "Canonical",
            "namespace": "com.axonops.canon",
            "fields": [
                {"name": "id", "type": "long"},
                {"name": "value", "type": "string"}
            ]
        }"""

        subject1 = f"python-avro-canon1-{int(time.time() * 1000)}-value"
        subject2 = f"python-avro-canon2-{int(time.time() * 1000)}-value"

        client = SchemaRegistryClient({"url": schema_registry_url})

        # Register compact schema
        schema1 = Schema(compact_schema, "AVRO")
        id1 = client.register_schema(subject1, schema1)

        # Register verbose schema (should be canonicalized to same schema)
        schema2 = Schema(verbose_schema, "AVRO")
        id2 = client.register_schema(subject2, schema2)

        # Same schema content (after canonicalization) should produce same global ID
        assert id1 == id2, \
            f"Same Avro schema with different formatting should return same global ID (canonicalization): {id1} vs {id2}"

        print(f"Schema canonicalization verified: both formats use schema ID {id1}")
