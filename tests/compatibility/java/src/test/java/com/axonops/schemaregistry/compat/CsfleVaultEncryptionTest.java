package com.axonops.schemaregistry.compat;

import io.confluent.kafka.schemaregistry.client.SchemaRegistryClient;
import io.confluent.kafka.serializers.KafkaAvroDeserializer;
import io.confluent.kafka.serializers.KafkaAvroSerializer;
import org.apache.avro.Schema;
import org.apache.avro.generic.GenericData;
import org.apache.avro.generic.GenericRecord;
import org.junit.jupiter.api.*;

import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.nio.charset.StandardCharsets;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Client-Side Field Level Encryption (CSFLE) tests using HashiCorp Vault Transit
 * as the KMS backend, exercised against the AxonOps Schema Registry.
 *
 * <p>CSFLE uses envelope encryption: a Key Encryption Key (KEK) stored in Vault Transit
 * wraps a Data Encryption Key (DEK) managed by the DEK Registry. The DEK encrypts
 * individual field values at the serializer level. ENCRYPT rules in the schema's
 * {@code domainRules} identify which fields to encrypt by matching {@code confluent:tags}
 * annotations on the Avro field definitions.</p>
 *
 * <p>Infrastructure requirements:
 * <ul>
 *   <li>Schema registry running at {@code http://localhost:8081} (configurable via {@code schema.registry.url})</li>
 *   <li>HashiCorp Vault running at {@code http://localhost:18200} (configurable via {@code vault.url})
 *       with dev root token {@code test-root-token} (configurable via {@code vault.token})</li>
 *   <li>Vault Transit secrets engine enabled with a key named {@code test-key}</li>
 * </ul>
 */
@Tag("csfle")
@TestMethodOrder(MethodOrderer.OrderAnnotation.class)
public class CsfleVaultEncryptionTest {

    private static final String SCHEMA_REGISTRY_URL =
            System.getProperty("schema.registry.url", "http://localhost:8081");
    private static final String VAULT_URL =
            System.getProperty("vault.url", "http://localhost:18200");
    private static final String VAULT_TOKEN =
            System.getProperty("vault.token", "test-root-token");

    /**
     * Vault host:port extracted from VAULT_URL for use in {@code encrypt.kms.key.id}.
     * The hcvault KMS key ID format is {@code hcvault://<host>:<port>/transit/keys/<keyname>}.
     */
    private static final String VAULT_HOST_PORT;

    static {
        // Strip the "http://" or "https://" prefix to get host:port
        String stripped = VAULT_URL;
        if (stripped.startsWith("https://")) {
            stripped = stripped.substring("https://".length());
        } else if (stripped.startsWith("http://")) {
            stripped = stripped.substring("http://".length());
        }
        // Remove any trailing slash
        if (stripped.endsWith("/")) {
            stripped = stripped.substring(0, stripped.length() - 1);
        }
        VAULT_HOST_PORT = stripped;
    }

    private static final String VAULT_TRANSIT_KEY = "test-key";

    // -- Avro schemas --

    private static final String CUSTOMER_SCHEMA = """
            {
                "type": "record",
                "name": "Customer",
                "namespace": "com.axonops.test.csfle",
                "fields": [
                    {"name": "customerId", "type": "string"},
                    {"name": "name", "type": "string"},
                    {"name": "ssn", "type": "string", "confluent:tags": ["PII"]}
                ]
            }""";

    private static final String USER_PROFILE_SCHEMA = """
            {
                "type": "record",
                "name": "UserProfile",
                "namespace": "com.axonops.test.csfle",
                "fields": [
                    {"name": "userId", "type": "string"},
                    {"name": "ssn", "type": "string", "confluent:tags": ["PII"]},
                    {"name": "email", "type": "string", "confluent:tags": ["PII"]},
                    {"name": "creditCard", "type": "string", "confluent:tags": ["PII"]}
                ]
            }""";

    private static final String PAYMENT_EVENT_SCHEMA = """
            {
                "type": "record",
                "name": "PaymentEvent",
                "namespace": "com.axonops.test.csfle",
                "fields": [
                    {"name": "customerId", "type": "string"},
                    {"name": "creditCardNumber", "type": "string", "confluent:tags": ["PII"]},
                    {"name": "amount", "type": "double"},
                    {"name": "merchantName", "type": "string"}
                ]
            }""";

    @BeforeAll
    static void checkInfrastructure() {
        // Verify Vault is reachable before running any tests
        try {
            HttpClient http = HttpClient.newHttpClient();
            HttpRequest request = HttpRequest.newBuilder()
                    .uri(URI.create(VAULT_URL + "/v1/sys/health"))
                    .GET()
                    .build();
            HttpResponse<String> response = http.send(request, HttpResponse.BodyHandlers.ofString());
            // Vault health returns 200 for initialized+unsealed (dev mode)
            assertTrue(response.statusCode() == 200 || response.statusCode() == 429 || response.statusCode() == 472,
                    "Vault is not healthy at " + VAULT_URL + ": HTTP " + response.statusCode());
        } catch (Exception e) {
            Assumptions.assumeTrue(false,
                    "Vault is not accessible at " + VAULT_URL + ": " + e.getMessage()
                            + ". Skipping CSFLE tests.");
        }

        // Verify schema registry is reachable
        try {
            HttpClient http = HttpClient.newHttpClient();
            HttpRequest request = HttpRequest.newBuilder()
                    .uri(URI.create(SCHEMA_REGISTRY_URL + "/subjects"))
                    .GET()
                    .build();
            HttpResponse<String> response = http.send(request, HttpResponse.BodyHandlers.ofString());
            assertTrue(response.statusCode() == 200,
                    "Schema Registry is not healthy at " + SCHEMA_REGISTRY_URL + ": HTTP " + response.statusCode());
        } catch (Exception e) {
            Assumptions.assumeTrue(false,
                    "Schema Registry is not accessible at " + SCHEMA_REGISTRY_URL + ": " + e.getMessage()
                            + ". Skipping CSFLE tests.");
        }

        System.out.println("CSFLE Vault Encryption Tests");
        System.out.println("  Schema Registry: " + SCHEMA_REGISTRY_URL);
        System.out.println("  Vault URL:       " + VAULT_URL);
        System.out.println("  Vault host:port: " + VAULT_HOST_PORT);
    }

    // -------------------------------------------------------------------------
    // Helpers
    // -------------------------------------------------------------------------

    /**
     * Build the JSON body for registering a schema with an ENCRYPT domain rule.
     */
    private static String buildSchemaWithEncryptRule(String avroSchema, String kekName) {
        // The ruleSet goes into domainRules, not encodingRules.
        // onFailure is "ERROR,NONE" — error on write failure, no-op on read failure (for test 5 scenario).
        return """
                {
                    "schemaType": "AVRO",
                    "schema": %s,
                    "ruleSet": {
                        "domainRules": [
                            {
                                "name": "encrypt-pii",
                                "kind": "TRANSFORM",
                                "type": "ENCRYPT",
                                "mode": "WRITEREAD",
                                "tags": ["PII"],
                                "params": {
                                    "encrypt.kek.name": "%s",
                                    "encrypt.kms.type": "hcvault",
                                    "encrypt.kms.key.id": "hcvault://%s/transit/keys/%s"
                                },
                                "onFailure": "ERROR,NONE"
                            }
                        ]
                    }
                }"""
                .formatted(
                        escapeJsonString(avroSchema),
                        kekName,
                        VAULT_HOST_PORT,
                        VAULT_TRANSIT_KEY
                );
    }

    /**
     * Escape a string value so it can be embedded as a JSON string value.
     * This wraps the raw schema in quotes and escapes inner quotes/newlines.
     */
    private static String escapeJsonString(String raw) {
        // Compact the schema to a single line and escape for JSON embedding
        String compacted = raw.replaceAll("\\s+", " ").trim();
        // Escape backslashes first, then double-quotes
        String escaped = compacted.replace("\\", "\\\\").replace("\"", "\\\"");
        return "\"" + escaped + "\"";
    }

    /**
     * Generate a unique name based on the test method name and current timestamp.
     */
    private static String uniqueName(String prefix) {
        return prefix + "-" + System.currentTimeMillis();
    }

    // -------------------------------------------------------------------------
    // Test 1: Encrypt/decrypt PII field round-trip
    // -------------------------------------------------------------------------

    @Test
    @Order(1)
    @DisplayName("Encrypt and decrypt PII field round-trip via Vault Transit")
    void testEncryptDecryptPiiFieldRoundTrip() {
        String subject = uniqueName("csfle-roundtrip") + "-value";
        String kekName = uniqueName("kek-roundtrip");

        SchemaRegistryClient client = TestHelper.createClient(SCHEMA_REGISTRY_URL);

        try {
            // Register schema with ENCRYPT rule
            String body = buildSchemaWithEncryptRule(CUSTOMER_SCHEMA, kekName);
            int schemaId = TestHelper.registerSchemaWithRules(SCHEMA_REGISTRY_URL, subject, body);
            assertTrue(schemaId > 0, "Schema registration should return a positive ID");

            // Create CSFLE-aware serializer and deserializer
            KafkaAvroSerializer serializer = TestHelper.createCsfleSerializer(
                    SCHEMA_REGISTRY_URL, client, VAULT_TOKEN);
            KafkaAvroDeserializer deserializer = TestHelper.createCsfleDeserializer(
                    SCHEMA_REGISTRY_URL, client, VAULT_TOKEN);

            try {
                // Build record with PII field
                Schema schema = new Schema.Parser().parse(CUSTOMER_SCHEMA);
                GenericRecord record = new GenericData.Record(schema);
                record.put("customerId", "CUST-001");
                record.put("name", "Jane Doe");
                record.put("ssn", "123-45-6789");

                // Serialize (encryption happens here)
                String topic = subject.replace("-value", "");
                byte[] encrypted = serializer.serialize(topic, record);
                assertNotNull(encrypted, "Serialized bytes should not be null");
                assertTrue(encrypted.length > 5, "Serialized data should contain magic byte + schema ID + payload");

                // Deserialize (decryption happens here)
                GenericRecord decrypted = (GenericRecord) deserializer.deserialize(topic, encrypted);
                assertNotNull(decrypted, "Deserialized record should not be null");

                assertEquals("CUST-001", decrypted.get("customerId").toString(),
                        "customerId should match original");
                assertEquals("Jane Doe", decrypted.get("name").toString(),
                        "name should match original");
                assertEquals("123-45-6789", decrypted.get("ssn").toString(),
                        "ssn should be decrypted back to original plaintext");

                System.out.println("Round-trip CSFLE test passed: ssn encrypted and decrypted correctly");
            } finally {
                serializer.close();
                deserializer.close();
            }
        } finally {
            TestHelper.deleteSubject(SCHEMA_REGISTRY_URL, subject);
        }
    }

    // -------------------------------------------------------------------------
    // Test 2: Raw bytes do not contain plaintext PII
    // -------------------------------------------------------------------------

    @Test
    @Order(2)
    @DisplayName("Raw serialized bytes do not contain plaintext PII value")
    void testRawBytesDoNotContainPlaintextPii() {
        String subject = uniqueName("csfle-rawbytes") + "-value";
        String kekName = uniqueName("kek-rawbytes");

        SchemaRegistryClient client = TestHelper.createClient(SCHEMA_REGISTRY_URL);

        try {
            String body = buildSchemaWithEncryptRule(CUSTOMER_SCHEMA, kekName);
            TestHelper.registerSchemaWithRules(SCHEMA_REGISTRY_URL, subject, body);

            KafkaAvroSerializer serializer = TestHelper.createCsfleSerializer(
                    SCHEMA_REGISTRY_URL, client, VAULT_TOKEN);

            try {
                Schema schema = new Schema.Parser().parse(CUSTOMER_SCHEMA);
                GenericRecord record = new GenericData.Record(schema);
                record.put("customerId", "CUST-002");
                record.put("name", "John Smith");
                record.put("ssn", "123-45-6789");

                String topic = subject.replace("-value", "");
                byte[] encrypted = serializer.serialize(topic, record);

                // Convert raw bytes to string and verify plaintext SSN is NOT present
                String rawString = new String(encrypted, StandardCharsets.ISO_8859_1);
                assertFalse(rawString.contains("123-45-6789"),
                        "Raw serialized bytes MUST NOT contain plaintext SSN; encryption did not occur");

                System.out.println("Raw bytes verification passed: plaintext SSN not found in "
                        + encrypted.length + " bytes");
            } finally {
                serializer.close();
            }
        } finally {
            TestHelper.deleteSubject(SCHEMA_REGISTRY_URL, subject);
        }
    }

    // -------------------------------------------------------------------------
    // Test 3: Multiple PII fields encrypted
    // -------------------------------------------------------------------------

    @Test
    @Order(3)
    @DisplayName("Multiple PII fields (ssn, email, creditCard) all encrypted and decryptable")
    void testMultiplePiiFieldsEncrypted() {
        String subject = uniqueName("csfle-multipii") + "-value";
        String kekName = uniqueName("kek-multipii");

        SchemaRegistryClient client = TestHelper.createClient(SCHEMA_REGISTRY_URL);

        try {
            String body = buildSchemaWithEncryptRule(USER_PROFILE_SCHEMA, kekName);
            TestHelper.registerSchemaWithRules(SCHEMA_REGISTRY_URL, subject, body);

            KafkaAvroSerializer serializer = TestHelper.createCsfleSerializer(
                    SCHEMA_REGISTRY_URL, client, VAULT_TOKEN);
            KafkaAvroDeserializer deserializer = TestHelper.createCsfleDeserializer(
                    SCHEMA_REGISTRY_URL, client, VAULT_TOKEN);

            try {
                Schema schema = new Schema.Parser().parse(USER_PROFILE_SCHEMA);
                GenericRecord record = new GenericData.Record(schema);
                record.put("userId", "USER-100");
                record.put("ssn", "987-65-4321");
                record.put("email", "secret@example.com");
                record.put("creditCard", "4111-1111-1111-1111");

                String topic = subject.replace("-value", "");
                byte[] encrypted = serializer.serialize(topic, record);

                // Verify none of the plaintext PII values appear in raw bytes
                String rawString = new String(encrypted, StandardCharsets.ISO_8859_1);
                assertFalse(rawString.contains("987-65-4321"),
                        "Raw bytes MUST NOT contain plaintext SSN");
                assertFalse(rawString.contains("secret@example.com"),
                        "Raw bytes MUST NOT contain plaintext email");
                assertFalse(rawString.contains("4111-1111-1111-1111"),
                        "Raw bytes MUST NOT contain plaintext credit card number");

                // Decrypt and verify all fields round-trip correctly
                GenericRecord decrypted = (GenericRecord) deserializer.deserialize(topic, encrypted);
                assertNotNull(decrypted, "Deserialized record should not be null");

                assertEquals("USER-100", decrypted.get("userId").toString(),
                        "userId should match original (non-PII)");
                assertEquals("987-65-4321", decrypted.get("ssn").toString(),
                        "ssn should be decrypted back to original");
                assertEquals("secret@example.com", decrypted.get("email").toString(),
                        "email should be decrypted back to original");
                assertEquals("4111-1111-1111-1111", decrypted.get("creditCard").toString(),
                        "creditCard should be decrypted back to original");

                System.out.println("Multiple PII field encryption test passed: all 3 fields encrypted and decrypted");
            } finally {
                serializer.close();
                deserializer.close();
            }
        } finally {
            TestHelper.deleteSubject(SCHEMA_REGISTRY_URL, subject);
        }
    }

    // -------------------------------------------------------------------------
    // Test 4: Credit card PII protection use case
    // -------------------------------------------------------------------------

    @Test
    @Order(4)
    @DisplayName("Real-world PaymentEvent: only creditCardNumber encrypted, non-PII fields intact")
    void testCreditCardPiiProtectionUseCase() {
        String subject = uniqueName("csfle-payment") + "-value";
        String kekName = uniqueName("kek-payment");

        SchemaRegistryClient client = TestHelper.createClient(SCHEMA_REGISTRY_URL);

        try {
            String body = buildSchemaWithEncryptRule(PAYMENT_EVENT_SCHEMA, kekName);
            TestHelper.registerSchemaWithRules(SCHEMA_REGISTRY_URL, subject, body);

            KafkaAvroSerializer serializer = TestHelper.createCsfleSerializer(
                    SCHEMA_REGISTRY_URL, client, VAULT_TOKEN);
            KafkaAvroDeserializer deserializer = TestHelper.createCsfleDeserializer(
                    SCHEMA_REGISTRY_URL, client, VAULT_TOKEN);

            try {
                Schema schema = new Schema.Parser().parse(PAYMENT_EVENT_SCHEMA);
                GenericRecord record = new GenericData.Record(schema);
                record.put("customerId", "CUST-PAY-001");
                record.put("creditCardNumber", "4532-0150-1234-5678");
                record.put("amount", 149.99);
                record.put("merchantName", "Coffee Shop");

                String topic = subject.replace("-value", "");
                byte[] encrypted = serializer.serialize(topic, record);

                // (a) Raw bytes must not contain the credit card number
                String rawString = new String(encrypted, StandardCharsets.ISO_8859_1);
                assertFalse(rawString.contains("4532-0150-1234-5678"),
                        "Raw bytes MUST NOT contain plaintext credit card number");

                // (b) Authorized consumer decrypts correctly
                GenericRecord decrypted = (GenericRecord) deserializer.deserialize(topic, encrypted);
                assertNotNull(decrypted, "Deserialized record should not be null");

                assertEquals("4532-0150-1234-5678", decrypted.get("creditCardNumber").toString(),
                        "Authorized consumer should decrypt creditCardNumber");

                // (c) Non-PII fields are intact
                assertEquals("CUST-PAY-001", decrypted.get("customerId").toString(),
                        "customerId (non-PII) should be intact");
                assertEquals(149.99, (double) decrypted.get("amount"), 0.001,
                        "amount (non-PII) should be intact");
                assertEquals("Coffee Shop", decrypted.get("merchantName").toString(),
                        "merchantName (non-PII) should be intact");

                System.out.println("PaymentEvent CSFLE test passed: card number encrypted, non-PII fields intact");
            } finally {
                serializer.close();
                deserializer.close();
            }
        } finally {
            TestHelper.deleteSubject(SCHEMA_REGISTRY_URL, subject);
        }
    }

    // -------------------------------------------------------------------------
    // Test 5: Consumer without KMS access cannot decrypt
    // -------------------------------------------------------------------------

    @Test
    @Order(5)
    @DisplayName("Consumer without Vault token cannot decrypt encrypted fields")
    void testConsumerWithoutKmsAccessCannotDecrypt() {
        String subject = uniqueName("csfle-nokms") + "-value";
        String kekName = uniqueName("kek-nokms");

        SchemaRegistryClient producerClient = TestHelper.createClient(SCHEMA_REGISTRY_URL);

        try {
            String body = buildSchemaWithEncryptRule(CUSTOMER_SCHEMA, kekName);
            TestHelper.registerSchemaWithRules(SCHEMA_REGISTRY_URL, subject, body);

            // Produce encrypted data with full KMS access
            KafkaAvroSerializer serializer = TestHelper.createCsfleSerializer(
                    SCHEMA_REGISTRY_URL, producerClient, VAULT_TOKEN);

            byte[] encrypted;
            String topic = subject.replace("-value", "");
            try {
                Schema schema = new Schema.Parser().parse(CUSTOMER_SCHEMA);
                GenericRecord record = new GenericData.Record(schema);
                record.put("customerId", "CUST-SECURE");
                record.put("name", "Secure User");
                record.put("ssn", "555-66-7777");

                encrypted = serializer.serialize(topic, record);
                assertNotNull(encrypted, "Encrypted data should not be null");
            } finally {
                serializer.close();
            }

            // Create a NEW client and deserializer WITHOUT the Vault token
            SchemaRegistryClient consumerClient = TestHelper.createClient(SCHEMA_REGISTRY_URL);
            KafkaAvroDeserializer noKmsDeserializer = TestHelper.createRuleAwareDeserializer(
                    SCHEMA_REGISTRY_URL, consumerClient);

            try {
                // Attempting to deserialize should fail because the deserializer cannot
                // unwrap the DEK without Vault access
                Exception thrown = assertThrows(Exception.class, () -> {
                    noKmsDeserializer.deserialize(topic, encrypted);
                }, "Deserialization without KMS access should throw an exception");

                // Walk the cause chain looking for KMS/decryption failure indicators
                Throwable cause = thrown;
                boolean foundKmsError = false;
                while (cause != null) {
                    String msg = cause.getMessage() != null ? cause.getMessage().toLowerCase() : "";
                    if (msg.contains("kms") || msg.contains("decrypt") || msg.contains("vault")
                            || msg.contains("unwrap") || msg.contains("key") || msg.contains("encrypt")
                            || msg.contains("rule") || msg.contains("executor")
                            || msg.contains("token") || msg.contains("forbidden")
                            || msg.contains("unauthorized") || msg.contains("permission")) {
                        foundKmsError = true;
                        break;
                    }
                    cause = cause.getCause();
                }

                assertTrue(foundKmsError,
                        "Exception should be related to KMS/decryption failure, got: " + thrown.getMessage());

                System.out.println("No-KMS-access test passed: deserialization correctly failed with: "
                        + thrown.getClass().getSimpleName() + ": " + thrown.getMessage());
            } finally {
                noKmsDeserializer.close();
            }
        } finally {
            TestHelper.deleteSubject(SCHEMA_REGISTRY_URL, subject);
        }
    }

    // -------------------------------------------------------------------------
    // Test 6: DEK created in registry after first produce
    // -------------------------------------------------------------------------

    @Test
    @Order(6)
    @DisplayName("DEK is auto-created in the DEK Registry after first serialization")
    void testDekCreatedInRegistryAfterFirstProduce() {
        String subject = uniqueName("csfle-dek") + "-value";
        String kekName = uniqueName("kek-dek");

        SchemaRegistryClient client = TestHelper.createClient(SCHEMA_REGISTRY_URL);

        try {
            String body = buildSchemaWithEncryptRule(CUSTOMER_SCHEMA, kekName);
            TestHelper.registerSchemaWithRules(SCHEMA_REGISTRY_URL, subject, body);

            // Verify DEK does not exist yet
            String dekBefore = TestHelper.getDEK(SCHEMA_REGISTRY_URL, kekName, subject);
            assertNull(dekBefore, "DEK should not exist before first serialization");

            // Serialize a record (triggers DEK creation)
            KafkaAvroSerializer serializer = TestHelper.createCsfleSerializer(
                    SCHEMA_REGISTRY_URL, client, VAULT_TOKEN);

            try {
                Schema schema = new Schema.Parser().parse(CUSTOMER_SCHEMA);
                GenericRecord record = new GenericData.Record(schema);
                record.put("customerId", "CUST-DEK-TEST");
                record.put("name", "DEK Test User");
                record.put("ssn", "111-22-3333");

                String topic = subject.replace("-value", "");
                byte[] encrypted = serializer.serialize(topic, record);
                assertNotNull(encrypted, "Serialized data should not be null");
            } finally {
                serializer.close();
            }

            // Now the DEK should exist in the DEK Registry
            String dekAfter = TestHelper.getDEK(SCHEMA_REGISTRY_URL, kekName, subject);
            assertNotNull(dekAfter,
                    "DEK should be created in the DEK Registry after first serialization");
            assertTrue(dekAfter.contains("encryptedKeyMaterial"),
                    "DEK response should contain encryptedKeyMaterial field");

            System.out.println("DEK auto-creation test passed. DEK response: "
                    + dekAfter.substring(0, Math.min(200, dekAfter.length())) + "...");
        } finally {
            TestHelper.deleteSubject(SCHEMA_REGISTRY_URL, subject);
        }
    }

    // -------------------------------------------------------------------------
    // Test 7: KEK auto-created from ENCRYPT rule params
    // -------------------------------------------------------------------------

    @Test
    @Order(7)
    @DisplayName("KEK is auto-created in the DEK Registry from ENCRYPT rule params after first serialization")
    void testKekAutoCreatedFromEncryptRuleParams() {
        String subject = uniqueName("csfle-kek") + "-value";
        String kekName = uniqueName("kek-autocreate");

        SchemaRegistryClient client = TestHelper.createClient(SCHEMA_REGISTRY_URL);

        try {
            // Do NOT pre-create the KEK. The serializer should create it from the ENCRYPT rule params.
            String kekBefore = TestHelper.getKEK(SCHEMA_REGISTRY_URL, kekName);
            assertNull(kekBefore, "KEK should not exist before schema registration and first produce");

            // Register schema with ENCRYPT rule referencing the KEK
            String body = buildSchemaWithEncryptRule(CUSTOMER_SCHEMA, kekName);
            TestHelper.registerSchemaWithRules(SCHEMA_REGISTRY_URL, subject, body);

            // Serialize a record (triggers KEK + DEK creation)
            KafkaAvroSerializer serializer = TestHelper.createCsfleSerializer(
                    SCHEMA_REGISTRY_URL, client, VAULT_TOKEN);

            try {
                Schema schema = new Schema.Parser().parse(CUSTOMER_SCHEMA);
                GenericRecord record = new GenericData.Record(schema);
                record.put("customerId", "CUST-KEK-TEST");
                record.put("name", "KEK Test User");
                record.put("ssn", "444-55-6666");

                String topic = subject.replace("-value", "");
                byte[] encrypted = serializer.serialize(topic, record);
                assertNotNull(encrypted, "Serialized data should not be null");
            } finally {
                serializer.close();
            }

            // Verify KEK was auto-created in the DEK Registry
            String kekAfter = TestHelper.getKEK(SCHEMA_REGISTRY_URL, kekName);
            assertNotNull(kekAfter,
                    "KEK should be auto-created in the DEK Registry after first serialization");
            assertTrue(kekAfter.contains("hcvault"),
                    "KEK response should indicate kmsType is hcvault, got: " + kekAfter);

            System.out.println("KEK auto-creation test passed. KEK response: "
                    + kekAfter.substring(0, Math.min(200, kekAfter.length())) + "...");
        } finally {
            TestHelper.deleteSubject(SCHEMA_REGISTRY_URL, subject);
        }
    }
}
