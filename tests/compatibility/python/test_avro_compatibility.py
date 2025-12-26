"""
Avro Serializer/Deserializer Compatibility Tests for AxonOps Schema Registry.

Tests verify that the Confluent Python Avro serializers produce the same
schema fingerprints and wire format as expected by AxonOps Schema Registry.
"""
import pytest
import struct
from confluent_kafka.schema_registry import SchemaRegistryClient
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


class TestAvroSchemaRegistration:
    """Test schema registration produces consistent fingerprints."""

    def test_register_user_schema(self, schema_registry_url, confluent_version):
        """Test that User schema registration produces consistent schema ID."""
        client = SchemaRegistryClient({"url": schema_registry_url})

        from confluent_kafka.schema_registry import Schema
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

        from confluent_kafka.schema_registry import Schema
        schema = Schema(SIMPLE_SCHEMA, "AVRO")

        subject = f"python-avro-simple-{confluent_version}-value"
        schema_id = client.register_schema(subject, schema)

        assert schema_id > 0
        print(f"Confluent Python {confluent_version}: Simple schema registered with ID {schema_id}")

    def test_schema_deduplication(self, schema_registry_url, confluent_version):
        """Test that registering the same schema twice returns the same ID."""
        client = SchemaRegistryClient({"url": schema_registry_url})

        from confluent_kafka.schema_registry import Schema
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
        user = {"id": 1, "name": "Test User", "email": {"string": "test@example.com"}}

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

        # Test data
        original = {"id": 42, "name": "Jane Doe", "email": {"string": "jane@example.com"}}

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

        from confluent_kafka.schema_registry import Schema

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
