#!/usr/bin/env bash
# demo-seed.sh — Populate Schema Registry with rich demo data for UI showcase.
# Usage: ./scripts/demo-seed.sh [SR_URL]
set -euo pipefail

SR="${1:-http://localhost:8081}"
CT="Content-Type: application/vnd.schemaregistry.v1+json"

echo "=== Seeding Schema Registry at $SR ==="

# Helper: register schema
register() {
  local subject="$1" body="$2"
  local resp
  resp=$(curl -sf -X POST "$SR/subjects/$subject/versions" -H "$CT" -d "$body" 2>&1) || {
    echo "  WARN: $subject registration returned non-2xx"
    return 0
  }
  local id
  id=$(echo "$resp" | grep -o '"id":[0-9]*' | cut -d: -f2)
  echo "  $subject → schema ID $id"
}

# ---------- 1. Avro Schemas ----------
echo ""
echo "--- Avro Schemas ---"

# user-events v1
register "user-events-value" '{
  "schema": "{\"type\":\"record\",\"name\":\"UserEvent\",\"namespace\":\"com.example.events\",\"fields\":[{\"name\":\"user_id\",\"type\":\"string\",\"doc\":\"Unique user identifier\"},{\"name\":\"event_type\",\"type\":\"string\"},{\"name\":\"timestamp\",\"type\":\"long\",\"logicalType\":\"timestamp-millis\"},{\"name\":\"email\",\"type\":\"string\",\"doc\":\"PII: user email address\"}]}",
  "schemaType": "AVRO"
}'

# user-events v2 (evolved: added optional source, ip_address PII field)
register "user-events-value" '{
  "schema": "{\"type\":\"record\",\"name\":\"UserEvent\",\"namespace\":\"com.example.events\",\"fields\":[{\"name\":\"user_id\",\"type\":\"string\",\"doc\":\"Unique user identifier\"},{\"name\":\"event_type\",\"type\":\"string\"},{\"name\":\"timestamp\",\"type\":\"long\",\"logicalType\":\"timestamp-millis\"},{\"name\":\"email\",\"type\":\"string\",\"doc\":\"PII: user email address\"},{\"name\":\"source\",\"type\":[\"null\",\"string\"],\"default\":null},{\"name\":\"ip_address\",\"type\":[\"null\",\"string\"],\"default\":null,\"doc\":\"PII: client IP address\"}]}",
  "schemaType": "AVRO"
}'

# order-events v1
register "order-events-value" '{
  "schema": "{\"type\":\"record\",\"name\":\"OrderEvent\",\"namespace\":\"com.example.orders\",\"fields\":[{\"name\":\"order_id\",\"type\":\"string\"},{\"name\":\"customer_id\",\"type\":\"string\",\"doc\":\"PII: links to customer\"},{\"name\":\"amount\",\"type\":{\"type\":\"bytes\",\"logicalType\":\"decimal\",\"precision\":10,\"scale\":2}},{\"name\":\"currency\",\"type\":\"string\"},{\"name\":\"status\",\"type\":{\"type\":\"enum\",\"name\":\"OrderStatus\",\"symbols\":[\"CREATED\",\"PAID\",\"SHIPPED\",\"DELIVERED\",\"CANCELLED\"]}},{\"name\":\"created_at\",\"type\":\"long\",\"logicalType\":\"timestamp-millis\"}]}",
  "schemaType": "AVRO"
}'

# payment-transactions v1
register "payment-transactions-value" '{
  "schema": "{\"type\":\"record\",\"name\":\"PaymentTransaction\",\"namespace\":\"com.example.payments\",\"fields\":[{\"name\":\"payment_id\",\"type\":\"string\"},{\"name\":\"order_id\",\"type\":\"string\"},{\"name\":\"card_last_four\",\"type\":\"string\",\"doc\":\"PII: last 4 digits of card\"},{\"name\":\"card_brand\",\"type\":\"string\"},{\"name\":\"amount\",\"type\":\"double\"},{\"name\":\"currency\",\"type\":\"string\"},{\"name\":\"status\",\"type\":{\"type\":\"enum\",\"name\":\"PaymentStatus\",\"symbols\":[\"PENDING\",\"AUTHORIZED\",\"CAPTURED\",\"FAILED\",\"REFUNDED\"]}},{\"name\":\"processed_at\",\"type\":\"long\",\"logicalType\":\"timestamp-millis\"}]}",
  "schemaType": "AVRO"
}'

# customer-profiles (PII-heavy)
register "customer-profiles-value" '{
  "schema": "{\"type\":\"record\",\"name\":\"CustomerProfile\",\"namespace\":\"com.example.customers\",\"fields\":[{\"name\":\"customer_id\",\"type\":\"string\"},{\"name\":\"first_name\",\"type\":\"string\",\"doc\":\"PII: first name\"},{\"name\":\"last_name\",\"type\":\"string\",\"doc\":\"PII: last name\"},{\"name\":\"email\",\"type\":\"string\",\"doc\":\"PII: email address\"},{\"name\":\"phone\",\"type\":[\"null\",\"string\"],\"default\":null,\"doc\":\"PII: phone number\"},{\"name\":\"address\",\"type\":{\"type\":\"record\",\"name\":\"Address\",\"fields\":[{\"name\":\"street\",\"type\":\"string\",\"doc\":\"PII: street address\"},{\"name\":\"city\",\"type\":\"string\"},{\"name\":\"state\",\"type\":\"string\"},{\"name\":\"zip\",\"type\":\"string\",\"doc\":\"PII: postal code\"},{\"name\":\"country\",\"type\":\"string\"}]},\"doc\":\"PII: full address\"},{\"name\":\"date_of_birth\",\"type\":[\"null\",\"string\"],\"default\":null,\"doc\":\"PII: date of birth\"},{\"name\":\"created_at\",\"type\":\"long\",\"logicalType\":\"timestamp-millis\"}]}",
  "schemaType": "AVRO"
}'

# inventory-updates
register "inventory-updates-value" '{
  "schema": "{\"type\":\"record\",\"name\":\"InventoryUpdate\",\"namespace\":\"com.example.inventory\",\"fields\":[{\"name\":\"product_id\",\"type\":\"string\"},{\"name\":\"warehouse_id\",\"type\":\"string\"},{\"name\":\"quantity\",\"type\":\"int\"},{\"name\":\"operation\",\"type\":{\"type\":\"enum\",\"name\":\"InventoryOp\",\"symbols\":[\"RESTOCK\",\"SALE\",\"RETURN\",\"ADJUSTMENT\"]}},{\"name\":\"timestamp\",\"type\":\"long\",\"logicalType\":\"timestamp-millis\"}]}",
  "schemaType": "AVRO"
}'

# product-catalog
register "product-catalog-value" '{
  "schema": "{\"type\":\"record\",\"name\":\"Product\",\"namespace\":\"com.example.catalog\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"},{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"description\",\"type\":[\"null\",\"string\"],\"default\":null},{\"name\":\"price\",\"type\":\"double\"},{\"name\":\"category\",\"type\":\"string\"},{\"name\":\"tags\",\"type\":{\"type\":\"array\",\"items\":\"string\"}},{\"name\":\"active\",\"type\":\"boolean\",\"default\":true}]}",
  "schemaType": "AVRO"
}'

# ---------- 2. JSON Schemas ----------
echo ""
echo "--- JSON Schemas ---"

# notification-settings (JSON Schema)
register "notification-settings-value" '{
  "schema": "{\"$schema\":\"http://json-schema.org/draft-07/schema#\",\"title\":\"NotificationSettings\",\"type\":\"object\",\"properties\":{\"user_id\":{\"type\":\"string\"},\"email_enabled\":{\"type\":\"boolean\",\"default\":true},\"sms_enabled\":{\"type\":\"boolean\",\"default\":false},\"push_enabled\":{\"type\":\"boolean\",\"default\":true},\"preferences\":{\"type\":\"object\",\"properties\":{\"marketing\":{\"type\":\"boolean\",\"default\":false},\"transactional\":{\"type\":\"boolean\",\"default\":true},\"security_alerts\":{\"type\":\"boolean\",\"default\":true}}}},\"required\":[\"user_id\"]}",
  "schemaType": "JSON"
}'

# audit-log (JSON Schema)
register "audit-log-value" '{
  "schema": "{\"$schema\":\"http://json-schema.org/draft-07/schema#\",\"title\":\"AuditLogEntry\",\"type\":\"object\",\"properties\":{\"event_id\":{\"type\":\"string\",\"format\":\"uuid\"},\"actor\":{\"type\":\"string\",\"description\":\"PII: username or service account\"},\"action\":{\"type\":\"string\",\"enum\":[\"CREATE\",\"READ\",\"UPDATE\",\"DELETE\",\"LOGIN\",\"LOGOUT\",\"EXPORT\"]},\"resource_type\":{\"type\":\"string\"},\"resource_id\":{\"type\":\"string\"},\"ip_address\":{\"type\":\"string\",\"description\":\"PII: source IP\"},\"user_agent\":{\"type\":\"string\"},\"timestamp\":{\"type\":\"string\",\"format\":\"date-time\"},\"details\":{\"type\":\"object\",\"additionalProperties\":true}},\"required\":[\"event_id\",\"actor\",\"action\",\"resource_type\",\"timestamp\"]}",
  "schemaType": "JSON"
}'

# feature-flags (JSON Schema)
register "feature-flags-value" '{
  "schema": "{\"$schema\":\"http://json-schema.org/draft-07/schema#\",\"title\":\"FeatureFlag\",\"type\":\"object\",\"properties\":{\"flag_name\":{\"type\":\"string\"},\"enabled\":{\"type\":\"boolean\"},\"rollout_percentage\":{\"type\":\"number\",\"minimum\":0,\"maximum\":100},\"allowed_users\":{\"type\":\"array\",\"items\":{\"type\":\"string\"}},\"metadata\":{\"type\":\"object\",\"properties\":{\"owner\":{\"type\":\"string\"},\"description\":{\"type\":\"string\"},\"created_at\":{\"type\":\"string\",\"format\":\"date-time\"}}}},\"required\":[\"flag_name\",\"enabled\"]}",
  "schemaType": "JSON"
}'

# ---------- 3. Protobuf Schemas ----------
echo ""
echo "--- Protobuf Schemas ---"

# shipping-events (Protobuf)
register "shipping-events-value" '{
  "schema": "syntax = \"proto3\";\npackage com.example.shipping;\n\nmessage ShippingEvent {\n  string shipment_id = 1;\n  string order_id = 2;\n  string carrier = 3;\n  string tracking_number = 4;\n  ShipmentStatus status = 5;\n  string recipient_name = 6; // PII\n  string recipient_address = 7; // PII\n  int64 updated_at = 8;\n\n  enum ShipmentStatus {\n    UNKNOWN = 0;\n    LABEL_CREATED = 1;\n    PICKED_UP = 2;\n    IN_TRANSIT = 3;\n    OUT_FOR_DELIVERY = 4;\n    DELIVERED = 5;\n    RETURNED = 6;\n  }\n}\n",
  "schemaType": "PROTOBUF"
}'

# analytics-events (Protobuf)
register "analytics-events-value" '{
  "schema": "syntax = \"proto3\";\npackage com.example.analytics;\n\nmessage AnalyticsEvent {\n  string event_id = 1;\n  string session_id = 2;\n  string user_id = 3; // PII\n  string event_name = 4;\n  map<string, string> properties = 5;\n  int64 timestamp = 6;\n  DeviceInfo device = 7;\n\n  message DeviceInfo {\n    string platform = 1;\n    string os_version = 2;\n    string app_version = 3;\n    string device_id = 4; // PII\n  }\n}\n",
  "schemaType": "PROTOBUF"
}'

# ---------- 4. Schemas with RuleSets (Data Governance) ----------
echo ""
echo "--- Schemas with Data Rules ---"

# Register a schema with PII encryption rules
register "sensitive-user-data-value" '{
  "schema": "{\"type\":\"record\",\"name\":\"SensitiveUserData\",\"namespace\":\"com.example.pii\",\"fields\":[{\"name\":\"user_id\",\"type\":\"string\"},{\"name\":\"ssn\",\"type\":\"string\",\"doc\":\"PII: Social Security Number — MUST be encrypted\"},{\"name\":\"tax_id\",\"type\":[\"null\",\"string\"],\"default\":null,\"doc\":\"PII: Tax identification number\"},{\"name\":\"bank_account\",\"type\":[\"null\",\"string\"],\"default\":null,\"doc\":\"PII: Bank account number\"},{\"name\":\"salary\",\"type\":[\"null\",\"double\"],\"default\":null,\"doc\":\"PII: Annual salary\"},{\"name\":\"updated_at\",\"type\":\"long\",\"logicalType\":\"timestamp-millis\"}]}",
  "schemaType": "AVRO",
  "ruleSet": {
    "domainRules": [
      {
        "name": "encrypt-ssn",
        "doc": "Encrypt SSN field using field-level encryption",
        "kind": "TRANSFORM",
        "mode": "WRITEREAD",
        "type": "ENCRYPT",
        "tags": ["PII", "SSN"],
        "params": {"encrypt.kek.name": "pii-encryption-key", "encrypt.kms.type": "aws-kms"},
        "onFailure": "ERROR"
      },
      {
        "name": "encrypt-bank-account",
        "doc": "Encrypt bank account number",
        "kind": "TRANSFORM",
        "mode": "WRITEREAD",
        "type": "ENCRYPT",
        "tags": ["PII", "FINANCIAL"],
        "params": {"encrypt.kek.name": "pii-encryption-key", "encrypt.kms.type": "aws-kms"},
        "onFailure": "ERROR"
      },
      {
        "name": "validate-ssn-format",
        "doc": "Validate SSN matches expected format before write",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "expr": "message.ssn.matches(\"^[0-9]{3}-[0-9]{2}-[0-9]{4}$\")",
        "onFailure": "DLQ",
        "onSuccess": "NONE"
      }
    ]
  }
}'

# Register schema with data quality rules
register "data-quality-metrics-value" '{
  "schema": "{\"type\":\"record\",\"name\":\"DataQualityMetric\",\"namespace\":\"com.example.quality\",\"fields\":[{\"name\":\"metric_id\",\"type\":\"string\"},{\"name\":\"dataset\",\"type\":\"string\"},{\"name\":\"completeness_score\",\"type\":\"double\"},{\"name\":\"accuracy_score\",\"type\":\"double\"},{\"name\":\"freshness_hours\",\"type\":\"int\"},{\"name\":\"measured_at\",\"type\":\"long\",\"logicalType\":\"timestamp-millis\"}]}",
  "schemaType": "AVRO",
  "ruleSet": {
    "domainRules": [
      {
        "name": "score-range-check",
        "doc": "Ensure quality scores are between 0 and 1",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "expr": "message.completeness_score >= 0.0 && message.completeness_score <= 1.0 && message.accuracy_score >= 0.0 && message.accuracy_score <= 1.0",
        "onFailure": "ERROR",
        "onSuccess": "NONE"
      }
    ]
  }
}'

# ---------- 5. Per-Subject Compatibility Overrides ----------
echo ""
echo "--- Compatibility Configs ---"

# Global is already BACKWARD (default). Set overrides.
curl -sf -X PUT "$SR/config/customer-profiles-value" -H "$CT" \
  -d '{"compatibility":"FULL_TRANSITIVE"}' > /dev/null
echo "  customer-profiles-value → FULL_TRANSITIVE"

curl -sf -X PUT "$SR/config/audit-log-value" -H "$CT" \
  -d '{"compatibility":"NONE"}' > /dev/null
echo "  audit-log-value → NONE"

curl -sf -X PUT "$SR/config/sensitive-user-data-value" -H "$CT" \
  -d '{"compatibility":"FULL"}' > /dev/null
echo "  sensitive-user-data-value → FULL"

curl -sf -X PUT "$SR/config/feature-flags-value" -H "$CT" \
  -d '{"compatibility":"FORWARD"}' > /dev/null
echo "  feature-flags-value → FORWARD"

# ---------- 6. KEKs (Encryption Keys) ----------
echo ""
echo "--- Encryption Keys (KEKs) ---"

curl -sf -X POST "$SR/dek-registry/v1/keks" -H "$CT" \
  -d '{
    "name": "pii-encryption-key",
    "kmsType": "aws-kms",
    "kmsKeyId": "arn:aws:kms:us-east-1:123456789012:key/abcd1234-5678-90ab-cdef-example11111",
    "kmsProps": {"region": "us-east-1"},
    "doc": "Master encryption key for PII data fields (SSN, bank accounts, etc.)",
    "shared": false
  }' > /dev/null 2>&1 && echo "  pii-encryption-key (aws-kms) created" || echo "  pii-encryption-key already exists"

curl -sf -X POST "$SR/dek-registry/v1/keks" -H "$CT" \
  -d '{
    "name": "financial-data-key",
    "kmsType": "aws-kms",
    "kmsKeyId": "arn:aws:kms:eu-west-1:123456789012:key/abcd1234-5678-90ab-cdef-example22222",
    "kmsProps": {"region": "eu-west-1"},
    "doc": "Encryption key for financial transaction data — EU region for GDPR compliance",
    "shared": false
  }' > /dev/null 2>&1 && echo "  financial-data-key (aws-kms, eu-west-1) created" || echo "  financial-data-key already exists"

curl -sf -X POST "$SR/dek-registry/v1/keks" -H "$CT" \
  -d '{
    "name": "analytics-shared-key",
    "kmsType": "gcp-kms",
    "kmsKeyId": "projects/my-project/locations/global/keyRings/analytics/cryptoKeys/shared-key",
    "kmsProps": {},
    "doc": "Shared encryption key for analytics data across all subjects",
    "shared": true
  }' > /dev/null 2>&1 && echo "  analytics-shared-key (gcp-kms, shared) created" || echo "  analytics-shared-key already exists"

# ---------- 7. DEKs (Data Encryption Keys) ----------
echo ""
echo "--- Data Encryption Keys (DEKs) ---"

curl -sf -X POST "$SR/dek-registry/v1/keks/pii-encryption-key/deks" -H "$CT" \
  -d '{
    "subject": "sensitive-user-data-value",
    "algorithm": "AES256_GCM"
  }' > /dev/null 2>&1 && echo "  DEK: sensitive-user-data-value (AES256_GCM)" || echo "  DEK already exists"

curl -sf -X POST "$SR/dek-registry/v1/keks/pii-encryption-key/deks" -H "$CT" \
  -d '{
    "subject": "customer-profiles-value",
    "algorithm": "AES256_GCM"
  }' > /dev/null 2>&1 && echo "  DEK: customer-profiles-value (AES256_GCM)" || echo "  DEK already exists"

curl -sf -X POST "$SR/dek-registry/v1/keks/financial-data-key/deks" -H "$CT" \
  -d '{
    "subject": "payment-transactions-value",
    "algorithm": "AES256_SIV"
  }' > /dev/null 2>&1 && echo "  DEK: payment-transactions-value (AES256_SIV)" || echo "  DEK already exists"

# ---------- 8. Exporters ----------
echo ""
echo "--- Exporters ---"

curl -sf -X POST "$SR/exporters" -H "$CT" \
  -d '{
    "name": "dr-replica-us-west",
    "contextType": "AUTO",
    "context": ".dr-us-west",
    "subjects": ["order-events-value", "payment-transactions-value", "customer-profiles-value"],
    "config": {
      "schema.registry.url": "http://sr-dr-us-west.internal:8081"
    }
  }' > /dev/null 2>&1 && echo "  dr-replica-us-west (3 subjects)" || echo "  dr-replica-us-west already exists"

curl -sf -X POST "$SR/exporters" -H "$CT" \
  -d '{
    "name": "analytics-mirror",
    "contextType": "AUTO",
    "context": ".analytics",
    "subjects": ["analytics-events-value", "user-events-value", "audit-log-value"],
    "subjectRenameFormat": "analytics-${subject}",
    "config": {
      "schema.registry.url": "http://sr-analytics.internal:8081"
    }
  }' > /dev/null 2>&1 && echo "  analytics-mirror (3 subjects, with rename)" || echo "  analytics-mirror already exists"

# ---------- Summary ----------
echo ""
echo "=== Seed Complete ==="
SUBJECTS=$(curl -sf "$SR/subjects" | grep -o '"' | wc -l)
SUBJECTS=$((SUBJECTS / 2))
echo "  Subjects: $SUBJECTS"
echo "  Schema types: Avro, JSON Schema, Protobuf"
echo "  Compatibility overrides: 4 subjects"
echo "  KEKs: 3 (pii-encryption-key, financial-data-key, analytics-shared-key)"
echo "  DEKs: 3 (sensitive-user-data, customer-profiles, payment-transactions)"
echo "  Exporters: 2 (dr-replica-us-west, analytics-mirror)"
echo "  Data rules: 4 (PII encryption, SSN validation, data quality)"
echo ""
echo "UI available at: http://localhost:8080/ui/"
echo "Login: admin / admin"
