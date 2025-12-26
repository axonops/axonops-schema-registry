"""
JSON Schema Serializer/Deserializer Compatibility Tests for AxonOps Schema Registry.

Tests verify that the Confluent Python JSON Schema serializers produce the same
schema fingerprints and wire format as expected by AxonOps Schema Registry.
"""
import pytest
import struct
import json
from confluent_kafka.schema_registry import SchemaRegistryClient
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


class TestJsonSchemaRegistration:
    """Test JSON Schema registration produces consistent fingerprints."""

    def test_register_user_jsonschema(self, schema_registry_url, confluent_version):
        """Test that User JSON schema registration produces consistent schema ID."""
        client = SchemaRegistryClient({"url": schema_registry_url})

        from confluent_kafka.schema_registry import Schema
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

        from confluent_kafka.schema_registry import Schema
        schema = Schema(ORDER_JSON_SCHEMA, "JSON")

        subject = f"python-json-order-{confluent_version}-value"
        schema_id = client.register_schema(subject, schema)

        assert schema_id > 0
        print(f"Confluent Python {confluent_version}: JSON Order schema registered with ID {schema_id}")

    def test_jsonschema_deduplication(self, schema_registry_url, confluent_version):
        """Test that registering the same JSON schema twice returns the same ID."""
        client = SchemaRegistryClient({"url": schema_registry_url})

        from confluent_kafka.schema_registry import Schema
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

        from confluent_kafka.schema_registry import Schema

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
