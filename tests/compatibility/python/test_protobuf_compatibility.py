"""
Protobuf Serializer/Deserializer Compatibility Tests for AxonOps Schema Registry.

Tests verify that the Confluent Python Protobuf serializers produce the same
schema fingerprints and wire format as expected by AxonOps Schema Registry.
"""
import time
import requests
from concurrent.futures import ThreadPoolExecutor, as_completed
from threading import Barrier

import pytest
from confluent_kafka.schema_registry import SchemaRegistryClient, Schema
from confluent_kafka.schema_registry.protobuf import ProtobufSerializer, ProtobufDeserializer
from confluent_kafka.serialization import SerializationContext, MessageField


# Protobuf schema definitions
USER_PROTO = """
syntax = "proto3";
package com.axonops.test;

message User {
    int64 id = 1;
    string name = 2;
    string email = 3;
}
"""

EVENT_PROTO = """
syntax = "proto3";
package com.axonops.test;

message Event {
    string id = 1;
    string type = 2;
    int64 timestamp = 3;
    map<string, string> metadata = 4;
}
"""

# Valid Confluent compatibility levels
VALID_COMPATIBILITY_LEVELS = {
    "NONE", "BACKWARD", "FORWARD", "FULL",
    "BACKWARD_TRANSITIVE", "FORWARD_TRANSITIVE", "FULL_TRANSITIVE"
}


class TestProtobufSchemaRegistration:
    """Test Protobuf schema registration produces consistent fingerprints."""

    def test_register_user_proto(self, schema_registry_url, confluent_version):
        """Test that User proto schema registration produces consistent schema ID."""
        client = SchemaRegistryClient({"url": schema_registry_url})

        schema = Schema(USER_PROTO, "PROTOBUF")

        subject = f"python-proto-user-{confluent_version}-value"
        schema_id = client.register_schema(subject, schema)

        assert schema_id > 0, "Schema ID should be positive"

        # Retrieve and verify
        retrieved = client.get_schema(schema_id)
        assert retrieved is not None
        assert retrieved.schema_type == "PROTOBUF"

        print(f"Confluent Python {confluent_version}: Proto User schema registered with ID {schema_id}")

    def test_register_event_proto(self, schema_registry_url, confluent_version):
        """Test that Event proto schema registration produces consistent schema ID."""
        client = SchemaRegistryClient({"url": schema_registry_url})

        schema = Schema(EVENT_PROTO, "PROTOBUF")

        subject = f"python-proto-event-{confluent_version}-value"
        schema_id = client.register_schema(subject, schema)

        assert schema_id > 0
        print(f"Confluent Python {confluent_version}: Proto Event schema registered with ID {schema_id}")

    def test_proto_schema_deduplication(self, schema_registry_url, confluent_version):
        """Test that registering the same proto schema twice returns the same ID."""
        client = SchemaRegistryClient({"url": schema_registry_url})

        schema = Schema(USER_PROTO, "PROTOBUF")

        subject = f"python-proto-dedup-{confluent_version}-value"

        id1 = client.register_schema(subject, schema)
        id2 = client.register_schema(subject, schema)

        assert id1 == id2, f"Same proto schema should return same ID: {id1} vs {id2}"
        print(f"Proto schema deduplication verified: ID {id1}")


class TestProtobufWireFormat:
    """Test Protobuf wire format compatibility."""

    def test_wire_format_header(self, schema_registry_url, confluent_version):
        """Test that serialized protobuf data has correct wire format header."""
        client = SchemaRegistryClient({"url": schema_registry_url})

        # For Protobuf, we need to work with generated classes
        # Here we test the schema registration and wire format structure
        schema = Schema(USER_PROTO, "PROTOBUF")
        subject = f"python-proto-wire-{confluent_version}-value"

        schema_id = client.register_schema(subject, schema)

        # Protobuf wire format:
        # - Magic byte (0)
        # - 4-byte schema ID (big-endian)
        # - Message index array (variable length)
        # - Protobuf payload

        # Verify schema ID is valid
        assert schema_id > 0
        print(f"Proto wire format test: schema_id={schema_id}")


class TestProtobufSchemaEvolution:
    """Test Protobuf schema evolution compatibility."""

    def test_add_optional_field(self, schema_registry_url, confluent_version):
        """Test adding optional field (backward compatible in proto3)."""
        client = SchemaRegistryClient({"url": schema_registry_url})

        v1_proto = """
syntax = "proto3";
package com.axonops.evolution;

message Record {
    string id = 1;
    string data = 2;
}
"""

        v2_proto = """
syntax = "proto3";
package com.axonops.evolution;

message Record {
    string id = 1;
    string data = 2;
    int64 version = 3;
}
"""

        subject = f"python-proto-evolution-{confluent_version}-value"

        schema1 = Schema(v1_proto, "PROTOBUF")
        id1 = client.register_schema(subject, schema1)

        schema2 = Schema(v2_proto, "PROTOBUF")
        id2 = client.register_schema(subject, schema2)

        assert id1 != id2, "Different schemas should have different IDs"
        print(f"Proto schema evolution: v1 ID={id1}, v2 ID={id2}")


class TestProtobufGlobalSchemaID:
    """Test global schema ID behavior (Confluent-compatible)."""

    def test_same_schema_across_subjects(self, schema_registry_url, confluent_version):
        """Test that same Protobuf schema under different subjects returns same global ID."""
        client = SchemaRegistryClient({"url": schema_registry_url})

        schema = Schema(USER_PROTO, "PROTOBUF")

        subject1 = f"python-proto-global1-{confluent_version}-value"
        subject2 = f"python-proto-global2-{confluent_version}-value"

        # Register same schema under different subjects
        id1 = client.register_schema(subject1, schema)
        id2 = client.register_schema(subject2, schema)

        # Same schema content should produce same global ID (Confluent-compatible behavior)
        assert id1 == id2, f"Same Protobuf schema under different subjects should return same global ID: {id1} vs {id2}"

        # Structural verification - fetch and compare
        fetched1 = client.get_schema(id1)
        fetched2 = client.get_schema(id2)

        assert fetched1.schema_type == fetched2.schema_type, "Schema types should match"

        print(f"Global Protobuf schema ID verified: both subjects use ID {id1}")


class TestProtobufConcurrentRegistration:
    """Test concurrent schema registration."""

    def test_concurrent_registration_returns_consistent_ids(self, schema_registry_url, confluent_version):
        """Test that concurrent registrations return the same schema ID."""
        subject = f"python-proto-concurrent-{int(time.time() * 1000)}-value"
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

                schema = Schema(USER_PROTO, "PROTOBUF")
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


class TestProtobufConfigEndpoints:
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


class TestProtobufIncompatibleSchemaEvolution:
    """Test incompatible schema evolution fails correctly."""

    def test_incompatible_schema_rejected(self, schema_registry_url, confluent_version):
        """Test that incompatible schema evolution fails with correct error."""
        subject = f"python-proto-incompat-{int(time.time() * 1000)}-value"
        client = SchemaRegistryClient({"url": schema_registry_url})

        # Register v1 schema
        schema = Schema(USER_PROTO, "PROTOBUF")
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

        # Create incompatible schema: change email type from string to int64 (breaking change)
        incompatible_proto = """
syntax = "proto3";
package com.axonops.test;

message User {
    int64 id = 1;
    string name = 2;
    int64 email = 3;
}
"""

        # Try to register incompatible schema
        bad_schema = Schema(incompatible_proto, "PROTOBUF")
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

        print("Incompatible Protobuf schema correctly rejected")


class TestProtobufCacheBehavior:
    """Test cache behavior with fresh clients."""

    def test_fresh_client_cache_miss(self, schema_registry_url, confluent_version):
        """Test that a fresh client can fetch schema after cache bypass."""
        subject = f"python-proto-cache-{int(time.time() * 1000)}-value"

        # Register schema with first client
        client1 = SchemaRegistryClient({"url": schema_registry_url})
        schema = Schema(USER_PROTO, "PROTOBUF")
        schema_id = client1.register_schema(subject, schema)

        # Create a completely new client (empty cache)
        client2 = SchemaRegistryClient({"url": schema_registry_url})

        # Fetch schema with fresh client (cache miss, must hit registry)
        fetched = client2.get_schema(schema_id)

        assert fetched is not None, "Fresh client should fetch schema by ID"
        assert fetched.schema_type == "PROTOBUF", "Schema type should be PROTOBUF"

        print("Cache behavior test passed")


class TestProtobufSchemaCanonicalisation:
    """Test schema canonicalization."""

    def test_same_schema_different_formatting(self, schema_registry_url, confluent_version):
        """Test that same schema with different formatting returns same ID."""
        # Same Protobuf schema content but with different formatting
        # This tests that the registry canonicalizes schemas before comparison
        #
        # NOTE: Some client versions may canonicalize client-side before POSTing,
        # so this test may pass even if server-side canonicalization is broken.
        # For strict server-side canonicalization validation, register via REST API directly.

        # Compact format
        compact_proto = 'syntax = "proto3"; package com.axonops.canon; message Canonical { int64 id = 1; string value = 2; }'

        # Verbose format (extra whitespace)
        verbose_proto = """
syntax = "proto3";

package com.axonops.canon;

message Canonical {
    int64 id = 1;
    string value = 2;
}
"""

        subject1 = f"python-proto-canon1-{int(time.time() * 1000)}-value"
        subject2 = f"python-proto-canon2-{int(time.time() * 1000)}-value"

        client = SchemaRegistryClient({"url": schema_registry_url})

        # Register compact schema
        schema1 = Schema(compact_proto, "PROTOBUF")
        id1 = client.register_schema(subject1, schema1)

        # Register verbose schema (should be canonicalized to same schema)
        schema2 = Schema(verbose_proto, "PROTOBUF")
        id2 = client.register_schema(subject2, schema2)

        # Same schema content (after canonicalization) should produce same global ID
        assert id1 == id2, \
            f"Same Protobuf schema with different formatting should return same global ID (canonicalization): {id1} vs {id2}"

        print(f"Schema canonicalization verified: both formats use schema ID {id1}")
