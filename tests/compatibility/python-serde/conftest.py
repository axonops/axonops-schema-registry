"""
Pytest fixtures and configuration for Python data contract and CSFLE tests.

These tests mirror the Go serde tests at tests/compatibility/go-serde/ and
validate that the AxonOps Schema Registry correctly handles data contract
rules (CEL, JSONata migration) and CSFLE encryption through the Confluent
Python client library.

Prerequisites:
  - Schema registry running at SCHEMA_REGISTRY_URL (default: http://localhost:8081)
  - For CSFLE tests: Vault running at VAULT_URL (default: http://localhost:18200)
    with Transit engine enabled and test-key created
"""

import os
import time

import pytest
import requests


# =============================================================================
# Custom Markers
# =============================================================================

def pytest_configure(config):
    """Register custom markers for selective test execution."""
    config.addinivalue_line(
        "markers", "data_contracts: tests for CEL/JSONata data contract rules"
    )
    config.addinivalue_line(
        "markers", "csfle: tests for Client-Side Field Level Encryption"
    )


# =============================================================================
# Fixtures
# =============================================================================

@pytest.fixture(scope="session")
def schema_registry_url():
    """Return the Schema Registry URL from environment or default."""
    return os.environ.get("SCHEMA_REGISTRY_URL", "http://localhost:8081")


@pytest.fixture(scope="session")
def vault_url():
    """Return the Vault URL from environment or default."""
    return os.environ.get("VAULT_URL", "http://localhost:18200")


@pytest.fixture(scope="session")
def vault_token():
    """Return the Vault token from environment or default."""
    return os.environ.get("VAULT_TOKEN", "test-root-token")


@pytest.fixture(scope="session")
def vault_base_url(vault_url):
    """Return the Vault base URL with trailing slash stripped."""
    return vault_url.rstrip("/")


@pytest.fixture(scope="session")
def registry_healthy(schema_registry_url):
    """Verify the schema registry is reachable. Skip all tests if not."""
    try:
        resp = requests.get(schema_registry_url, timeout=5)
        if resp.status_code != 200:
            pytest.skip(
                f"Schema registry not healthy at {schema_registry_url}: "
                f"HTTP {resp.status_code}"
            )
    except requests.ConnectionError:
        pytest.skip(
            f"Schema registry not reachable at {schema_registry_url}"
        )


@pytest.fixture(scope="session")
def vault_healthy(vault_url):
    """Verify Vault is reachable. Skip CSFLE tests if not."""
    try:
        resp = requests.get(f"{vault_url}/v1/sys/health", timeout=5)
        if resp.status_code not in (200, 429, 472):
            pytest.skip(
                f"Vault not healthy at {vault_url}: HTTP {resp.status_code}"
            )
    except requests.ConnectionError:
        pytest.skip(f"Vault not reachable at {vault_url}")


# =============================================================================
# Helpers
# =============================================================================

def unique_subject(prefix):
    """Generate a unique subject name using a timestamp suffix.

    Matches the Go helper: fmt.Sprintf("%s-%d-value", prefix, time.Now().UnixMilli())
    """
    return f"{prefix}-{int(time.time() * 1000)}-value"


def topic_from_subject(subject):
    """Derive the topic name from a subject by stripping the '-value' suffix.

    Matches the Go helper: strings.TrimSuffix(subject, "-value")
    """
    if subject.endswith("-value"):
        return subject[: -len("-value")]
    return subject
