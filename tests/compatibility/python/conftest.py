"""Pytest configuration and fixtures for Schema Registry compatibility tests."""
import os
import pytest


def pytest_configure(config):
    """Configure pytest with custom markers."""
    config.addinivalue_line(
        "markers", "avro: mark test as Avro serialization test"
    )
    config.addinivalue_line(
        "markers", "protobuf: mark test as Protobuf serialization test"
    )
    config.addinivalue_line(
        "markers", "jsonschema: mark test as JSON Schema serialization test"
    )


@pytest.fixture
def schema_registry_url():
    """Get Schema Registry URL from environment or use default."""
    return os.environ.get("SCHEMA_REGISTRY_URL", "http://localhost:8081")


@pytest.fixture
def confluent_version():
    """Get the Confluent Kafka Python version being tested."""
    try:
        import confluent_kafka
        return confluent_kafka.version()[0]
    except Exception:
        return "unknown"
