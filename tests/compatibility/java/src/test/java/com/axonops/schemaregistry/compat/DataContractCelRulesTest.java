package com.axonops.schemaregistry.compat;

import io.confluent.kafka.schemaregistry.client.SchemaRegistryClient;
import io.confluent.kafka.serializers.KafkaAvroDeserializer;
import io.confluent.kafka.serializers.KafkaAvroSerializer;
import org.apache.avro.Schema;
import org.apache.avro.generic.GenericData;
import org.apache.avro.generic.GenericRecord;
import org.junit.jupiter.api.*;

import java.util.ArrayList;
import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

/**
 * CEL and CEL_FIELD data contract rule tests.
 *
 * These tests verify that the AxonOps Schema Registry correctly stores and returns
 * ruleSet definitions so that Confluent serializer/deserializer clients can execute
 * CEL-based data contract rules at serialize/deserialize time.
 *
 * Key behaviors tested:
 * - CEL CONDITION rules that validate entire records (message.field access)
 * - CEL CONDITION rules that reject invalid data with RuleConditionException
 * - CEL CONDITION rules on READ (deserialization) path
 * - Multiple chained CEL CONDITION rules
 * - CEL_FIELD TRANSFORM rules with tag-based field selection
 * - CEL_FIELD CONDITION rules with tag-based validation
 * - CEL_FIELD TRANSFORM rules with field-name-based selection
 * - Disabled rules that are skipped during execution
 *
 * Important: The registry stores rules but does NOT execute them. Rule execution
 * happens entirely in the Confluent serializer/deserializer clients. The
 * kafka-schema-rules artifact provides CEL and CEL_FIELD executors auto-discovered
 * via ServiceLoader.
 */
@Tag("data-contracts")
@TestMethodOrder(MethodOrderer.OrderAnnotation.class)
public class DataContractCelRulesTest {

    private static final String SCHEMA_REGISTRY_URL = System.getProperty("schema.registry.url", "http://localhost:8081");
    private static final String CONFLUENT_VERSION = System.getProperty("confluent.version", "unknown");

    /** Subjects registered during this test run, cleaned up in tearDown. */
    private static final List<String> registeredSubjects = new ArrayList<>();

    @BeforeAll
    static void setUp() {
        System.out.println("Running CEL data contract rule tests with Confluent version: " + CONFLUENT_VERSION);
        System.out.println("Schema Registry URL: " + SCHEMA_REGISTRY_URL);
    }

    @AfterAll
    static void tearDown() {
        for (String subject : registeredSubjects) {
            try {
                TestHelper.deleteSubject(SCHEMA_REGISTRY_URL, subject);
            } catch (Exception e) {
                System.err.println("Cleanup warning: failed to delete subject " + subject + ": " + e.getMessage());
            }
        }
    }

    // -----------------------------------------------------------------------
    // Test 1: CEL CONDITION — valid data passes
    // -----------------------------------------------------------------------

    @Test
    @Order(1)
    @DisplayName("CEL CONDITION rule allows valid data and round-trips correctly")
    void testCelConditionValidDataPasses() {
        String subject = "cel-condition-valid-" + System.currentTimeMillis() + "-value";
        registeredSubjects.add(subject);

        String orderSchemaStr = "{\"type\":\"record\",\"name\":\"Order\",\"namespace\":\"com.axonops.test.cel\","
                + "\"fields\":["
                + "{\"name\":\"orderId\",\"type\":\"string\"},"
                + "{\"name\":\"amount\",\"type\":\"double\"},"
                + "{\"name\":\"currency\",\"type\":\"string\"}"
                + "]}";

        // Register schema with a CEL CONDITION rule: amount must be positive
        String body = "{"
                + "\"schemaType\":\"AVRO\","
                + "\"schema\":\"" + escapeJson(orderSchemaStr) + "\","
                + "\"ruleSet\":{"
                + "  \"domainRules\":["
                + "    {"
                + "      \"name\":\"amountPositive\","
                + "      \"kind\":\"CONDITION\","
                + "      \"type\":\"CEL\","
                + "      \"mode\":\"WRITE\","
                + "      \"expr\":\"message.amount > 0.0\","
                + "      \"onFailure\":\"ERROR\""
                + "    }"
                + "  ]"
                + "}"
                + "}";

        int schemaId = TestHelper.registerSchemaWithRules(SCHEMA_REGISTRY_URL, subject, body);
        assertTrue(schemaId > 0, "Schema should be registered with a positive ID");

        // Build a valid order record
        Schema schema = new Schema.Parser().parse(orderSchemaStr);
        GenericRecord order = new GenericData.Record(schema);
        order.put("orderId", "ORD-001");
        order.put("amount", 100.50);
        order.put("currency", "USD");

        // Create rule-aware serializer/deserializer
        SchemaRegistryClient client = TestHelper.createClient(SCHEMA_REGISTRY_URL);
        KafkaAvroSerializer serializer = TestHelper.createRuleAwareSerializer(SCHEMA_REGISTRY_URL, client);
        KafkaAvroDeserializer deserializer = TestHelper.createRuleAwareDeserializer(SCHEMA_REGISTRY_URL, client);

        try {
            // Serialize — CEL rule should pass (amount=100.50 > 0)
            String topic = subject.replace("-value", "");
            byte[] serialized = serializer.serialize(topic, order);
            assertNotNull(serialized, "Serialized bytes should not be null");
            assertTrue(serialized.length > 5, "Serialized data should contain magic byte + schema ID + payload");

            // Deserialize and verify round-trip
            GenericRecord deserialized = (GenericRecord) deserializer.deserialize(topic, serialized);
            assertNotNull(deserialized, "Deserialized record should not be null");
            assertEquals("ORD-001", deserialized.get("orderId").toString(), "orderId should match");
            assertEquals(100.50, (double) deserialized.get("amount"), 0.001, "amount should match");
            assertEquals("USD", deserialized.get("currency").toString(), "currency should match");

            System.out.println("CEL CONDITION valid data test passed: schema ID " + schemaId);
        } finally {
            serializer.close();
            deserializer.close();
        }
    }

    // -----------------------------------------------------------------------
    // Test 2: CEL CONDITION — invalid data rejected
    // -----------------------------------------------------------------------

    @Test
    @Order(2)
    @DisplayName("CEL CONDITION rule rejects invalid data with RuleConditionException")
    void testCelConditionInvalidDataRejected() {
        String subject = "cel-condition-invalid-" + System.currentTimeMillis() + "-value";
        registeredSubjects.add(subject);

        String orderSchemaStr = "{\"type\":\"record\",\"name\":\"Order\",\"namespace\":\"com.axonops.test.cel\","
                + "\"fields\":["
                + "{\"name\":\"orderId\",\"type\":\"string\"},"
                + "{\"name\":\"amount\",\"type\":\"double\"},"
                + "{\"name\":\"currency\",\"type\":\"string\"}"
                + "]}";

        String body = "{"
                + "\"schemaType\":\"AVRO\","
                + "\"schema\":\"" + escapeJson(orderSchemaStr) + "\","
                + "\"ruleSet\":{"
                + "  \"domainRules\":["
                + "    {"
                + "      \"name\":\"amountPositive\","
                + "      \"kind\":\"CONDITION\","
                + "      \"type\":\"CEL\","
                + "      \"mode\":\"WRITE\","
                + "      \"expr\":\"message.amount > 0.0\","
                + "      \"onFailure\":\"ERROR\""
                + "    }"
                + "  ]"
                + "}"
                + "}";

        int schemaId = TestHelper.registerSchemaWithRules(SCHEMA_REGISTRY_URL, subject, body);
        assertTrue(schemaId > 0, "Schema should be registered with a positive ID");

        // Build an invalid order record (negative amount)
        Schema schema = new Schema.Parser().parse(orderSchemaStr);
        GenericRecord order = new GenericData.Record(schema);
        order.put("orderId", "ORD-BAD");
        order.put("amount", -5.0);
        order.put("currency", "USD");

        SchemaRegistryClient client = TestHelper.createClient(SCHEMA_REGISTRY_URL);
        KafkaAvroSerializer serializer = TestHelper.createRuleAwareSerializer(SCHEMA_REGISTRY_URL, client);

        try {
            String topic = subject.replace("-value", "");

            // Serialize should throw — amount=-5.0 fails "message.amount > 0"
            Exception thrown = assertThrows(Exception.class, () -> {
                serializer.serialize(topic, order);
            }, "Serialization should fail due to CEL CONDITION rule violation");

            // Walk the cause chain looking for RuleConditionException
            assertTrue(isRuleConditionException(thrown),
                    "Exception cause chain should contain RuleConditionException, but got: "
                            + describeExceptionChain(thrown));

            System.out.println("CEL CONDITION invalid data correctly rejected: " + thrown.getClass().getSimpleName());
        } finally {
            serializer.close();
        }
    }

    // -----------------------------------------------------------------------
    // Test 3: CEL CONDITION on READ — rejected at deserialization
    // -----------------------------------------------------------------------

    @Test
    @Order(3)
    @DisplayName("CEL CONDITION rule on READ rejects data during deserialization")
    void testCelConditionOnRead() {
        String subject = "cel-condition-read-" + System.currentTimeMillis() + "-value";
        registeredSubjects.add(subject);

        String statusSchemaStr = "{\"type\":\"record\",\"name\":\"OrderStatus\",\"namespace\":\"com.axonops.test.cel\","
                + "\"fields\":["
                + "{\"name\":\"orderId\",\"type\":\"string\"},"
                + "{\"name\":\"status\",\"type\":\"string\"}"
                + "]}";

        // Rule on READ: status must not be CANCELLED
        String body = "{"
                + "\"schemaType\":\"AVRO\","
                + "\"schema\":\"" + escapeJson(statusSchemaStr) + "\","
                + "\"ruleSet\":{"
                + "  \"domainRules\":["
                + "    {"
                + "      \"name\":\"notCancelled\","
                + "      \"kind\":\"CONDITION\","
                + "      \"type\":\"CEL\","
                + "      \"mode\":\"READ\","
                + "      \"expr\":\"message.status != 'CANCELLED'\","
                + "      \"onFailure\":\"ERROR\""
                + "    }"
                + "  ]"
                + "}"
                + "}";

        int schemaId = TestHelper.registerSchemaWithRules(SCHEMA_REGISTRY_URL, subject, body);
        assertTrue(schemaId > 0, "Schema should be registered with a positive ID");

        Schema schema = new Schema.Parser().parse(statusSchemaStr);
        GenericRecord record = new GenericData.Record(schema);
        record.put("orderId", "ORD-CANCEL");
        record.put("status", "CANCELLED");

        String topic = subject.replace("-value", "");

        // Use an auto-registering serializer so the WRITE path has no rule
        // (the rule is mode=READ, so it only fires on deserialization)
        SchemaRegistryClient writeClient = TestHelper.createClient(SCHEMA_REGISTRY_URL);
        KafkaAvroSerializer writeSerializer = TestHelper.createRuleAwareSerializer(SCHEMA_REGISTRY_URL, writeClient);

        byte[] serialized;
        try {
            // Serialize should succeed — the rule is READ-only
            serialized = writeSerializer.serialize(topic, record);
            assertNotNull(serialized, "Serialization should succeed (rule is READ mode)");
        } finally {
            writeSerializer.close();
        }

        // Now deserialize with a rule-aware deserializer — should fail
        SchemaRegistryClient readClient = TestHelper.createClient(SCHEMA_REGISTRY_URL);
        KafkaAvroDeserializer readDeserializer = TestHelper.createRuleAwareDeserializer(SCHEMA_REGISTRY_URL, readClient);

        try {
            Exception thrown = assertThrows(Exception.class, () -> {
                readDeserializer.deserialize(topic, serialized);
            }, "Deserialization should fail due to READ CONDITION rule violation");

            assertTrue(isRuleConditionException(thrown),
                    "Exception cause chain should contain RuleConditionException, but got: "
                            + describeExceptionChain(thrown));

            System.out.println("CEL CONDITION on READ correctly rejected: " + thrown.getClass().getSimpleName());
        } finally {
            readDeserializer.close();
        }
    }

    // -----------------------------------------------------------------------
    // Test 4: Multiple chained CEL CONDITION rules
    // -----------------------------------------------------------------------

    @Test
    @Order(4)
    @DisplayName("Multiple chained CEL CONDITION rules — second rule fails on short currency")
    void testMultipleCelConditionsChained() {
        String subject = "cel-condition-chained-" + System.currentTimeMillis() + "-value";
        registeredSubjects.add(subject);

        String orderSchemaStr = "{\"type\":\"record\",\"name\":\"Order\",\"namespace\":\"com.axonops.test.cel\","
                + "\"fields\":["
                + "{\"name\":\"orderId\",\"type\":\"string\"},"
                + "{\"name\":\"amount\",\"type\":\"double\"},"
                + "{\"name\":\"currency\",\"type\":\"string\"}"
                + "]}";

        // Two chained conditions: amount > 0 AND currency must be exactly 3 chars
        String body = "{"
                + "\"schemaType\":\"AVRO\","
                + "\"schema\":\"" + escapeJson(orderSchemaStr) + "\","
                + "\"ruleSet\":{"
                + "  \"domainRules\":["
                + "    {"
                + "      \"name\":\"amountPositive\","
                + "      \"kind\":\"CONDITION\","
                + "      \"type\":\"CEL\","
                + "      \"mode\":\"WRITE\","
                + "      \"expr\":\"message.amount > 0.0\","
                + "      \"onFailure\":\"ERROR\""
                + "    },"
                + "    {"
                + "      \"name\":\"currencyLength\","
                + "      \"kind\":\"CONDITION\","
                + "      \"type\":\"CEL\","
                + "      \"mode\":\"WRITE\","
                + "      \"expr\":\"size(message.currency) == 3\","
                + "      \"onFailure\":\"ERROR\""
                + "    }"
                + "  ]"
                + "}"
                + "}";

        int schemaId = TestHelper.registerSchemaWithRules(SCHEMA_REGISTRY_URL, subject, body);
        assertTrue(schemaId > 0, "Schema should be registered with a positive ID");

        Schema schema = new Schema.Parser().parse(orderSchemaStr);
        String topic = subject.replace("-value", "");

        // --- Case A: amount=100, currency="US" (2 chars) -> second rule fails ---
        GenericRecord badCurrency = new GenericData.Record(schema);
        badCurrency.put("orderId", "ORD-SHORT");
        badCurrency.put("amount", 100.0);
        badCurrency.put("currency", "US");

        SchemaRegistryClient clientA = TestHelper.createClient(SCHEMA_REGISTRY_URL);
        KafkaAvroSerializer serializerA = TestHelper.createRuleAwareSerializer(SCHEMA_REGISTRY_URL, clientA);

        try {
            Exception thrown = assertThrows(Exception.class, () -> {
                serializerA.serialize(topic, badCurrency);
            }, "Serialization should fail — currency 'US' has length 2, not 3");

            assertTrue(isRuleConditionException(thrown),
                    "Exception cause chain should contain RuleConditionException for currency length, but got: "
                            + describeExceptionChain(thrown));

            System.out.println("Chained rules: short currency correctly rejected");
        } finally {
            serializerA.close();
        }

        // --- Case B: amount=100, currency="USD" (3 chars) -> both rules pass ---
        GenericRecord goodOrder = new GenericData.Record(schema);
        goodOrder.put("orderId", "ORD-GOOD");
        goodOrder.put("amount", 100.0);
        goodOrder.put("currency", "USD");

        SchemaRegistryClient clientB = TestHelper.createClient(SCHEMA_REGISTRY_URL);
        KafkaAvroSerializer serializerB = TestHelper.createRuleAwareSerializer(SCHEMA_REGISTRY_URL, clientB);
        KafkaAvroDeserializer deserializerB = TestHelper.createRuleAwareDeserializer(SCHEMA_REGISTRY_URL, clientB);

        try {
            byte[] serialized = serializerB.serialize(topic, goodOrder);
            assertNotNull(serialized, "Valid order should serialize successfully");

            GenericRecord deserialized = (GenericRecord) deserializerB.deserialize(topic, serialized);
            assertEquals("ORD-GOOD", deserialized.get("orderId").toString());
            assertEquals(100.0, (double) deserialized.get("amount"), 0.001);
            assertEquals("USD", deserialized.get("currency").toString());

            System.out.println("Chained rules: valid order passed both conditions");
        } finally {
            serializerB.close();
            deserializerB.close();
        }
    }

    // -----------------------------------------------------------------------
    // Test 5: CEL_FIELD TRANSFORM — mask SSN on READ using PII tags
    // -----------------------------------------------------------------------

    @Test
    @Order(5)
    @DisplayName("CEL_FIELD TRANSFORM masks SSN on READ using PII tag")
    void testCelFieldTransformMaskSsnOnRead() {
        String subject = "cel-field-mask-ssn-" + System.currentTimeMillis() + "-value";
        registeredSubjects.add(subject);

        String userSchemaStr = "{\"type\":\"record\",\"name\":\"User\",\"namespace\":\"com.axonops.test.cel\","
                + "\"fields\":["
                + "{\"name\":\"name\",\"type\":\"string\"},"
                + "{\"name\":\"ssn\",\"type\":\"string\",\"confluent:tags\":[\"PII\"]}"
                + "]}";

        // CEL_FIELD TRANSFORM on READ: mask PII-tagged STRING fields
        // Expression format: <guard> ; <body>
        // Guard: typeName == 'STRING' — only apply to string fields
        // Body: 'XXX-XX-' + value.substring(7) — mask all but last 4 digits
        String body = "{"
                + "\"schemaType\":\"AVRO\","
                + "\"schema\":\"" + escapeJson(userSchemaStr) + "\","
                + "\"ruleSet\":{"
                + "  \"domainRules\":["
                + "    {"
                + "      \"name\":\"maskPii\","
                + "      \"kind\":\"TRANSFORM\","
                + "      \"type\":\"CEL_FIELD\","
                + "      \"mode\":\"READ\","
                + "      \"tags\":[\"PII\"],"
                + "      \"expr\":\"typeName == 'STRING' ; 'XXX-XX-' + value.substring(7)\""
                + "    }"
                + "  ]"
                + "}"
                + "}";

        int schemaId = TestHelper.registerSchemaWithRules(SCHEMA_REGISTRY_URL, subject, body);
        assertTrue(schemaId > 0, "Schema should be registered with a positive ID");

        Schema schema = new Schema.Parser().parse(userSchemaStr);
        GenericRecord user = new GenericData.Record(schema);
        user.put("name", "Jane Doe");
        user.put("ssn", "123-45-6789");

        String topic = subject.replace("-value", "");

        // Serialize (WRITE path) — no WRITE rule, should succeed as-is
        SchemaRegistryClient writeClient = TestHelper.createClient(SCHEMA_REGISTRY_URL);
        KafkaAvroSerializer serializer = TestHelper.createRuleAwareSerializer(SCHEMA_REGISTRY_URL, writeClient);

        byte[] serialized;
        try {
            serialized = serializer.serialize(topic, user);
            assertNotNull(serialized, "Serialization should succeed (TRANSFORM rule is READ mode)");
        } finally {
            serializer.close();
        }

        // Deserialize (READ path) — CEL_FIELD TRANSFORM should mask the SSN
        SchemaRegistryClient readClient = TestHelper.createClient(SCHEMA_REGISTRY_URL);
        KafkaAvroDeserializer deserializer = TestHelper.createRuleAwareDeserializer(SCHEMA_REGISTRY_URL, readClient);

        try {
            GenericRecord deserialized = (GenericRecord) deserializer.deserialize(topic, serialized);
            assertNotNull(deserialized, "Deserialized record should not be null");
            assertEquals("Jane Doe", deserialized.get("name").toString(),
                    "name field should NOT be masked (no PII tag)");
            assertEquals("XXX-XX-6789", deserialized.get("ssn").toString(),
                    "ssn field should be masked to 'XXX-XX-6789'");

            System.out.println("CEL_FIELD TRANSFORM SSN masking test passed: ssn=" + deserialized.get("ssn"));
        } finally {
            deserializer.close();
        }
    }

    // -----------------------------------------------------------------------
    // Test 6: CEL_FIELD CONDITION — validate PII-tagged fields not empty
    // -----------------------------------------------------------------------

    @Test
    @Order(6)
    @DisplayName("CEL_FIELD CONDITION with PII tags rejects empty SSN on WRITE")
    void testCelFieldConditionWithTagsValidatePiiNotEmpty() {
        String subject = "cel-field-pii-notempty-" + System.currentTimeMillis() + "-value";
        registeredSubjects.add(subject);

        String userSchemaStr = "{\"type\":\"record\",\"name\":\"User\",\"namespace\":\"com.axonops.test.cel\","
                + "\"fields\":["
                + "{\"name\":\"name\",\"type\":\"string\"},"
                + "{\"name\":\"ssn\",\"type\":\"string\",\"confluent:tags\":[\"PII\"]}"
                + "]}";

        // CEL_FIELD CONDITION on WRITE: PII-tagged string fields must not be empty
        String body = "{"
                + "\"schemaType\":\"AVRO\","
                + "\"schema\":\"" + escapeJson(userSchemaStr) + "\","
                + "\"ruleSet\":{"
                + "  \"domainRules\":["
                + "    {"
                + "      \"name\":\"piiNotEmpty\","
                + "      \"kind\":\"CONDITION\","
                + "      \"type\":\"CEL_FIELD\","
                + "      \"mode\":\"WRITE\","
                + "      \"tags\":[\"PII\"],"
                + "      \"expr\":\"typeName == 'STRING' ; value != ''\","
                + "      \"onFailure\":\"ERROR\""
                + "    }"
                + "  ]"
                + "}"
                + "}";

        int schemaId = TestHelper.registerSchemaWithRules(SCHEMA_REGISTRY_URL, subject, body);
        assertTrue(schemaId > 0, "Schema should be registered with a positive ID");

        Schema schema = new Schema.Parser().parse(userSchemaStr);
        String topic = subject.replace("-value", "");

        // --- Case A: empty SSN -> should fail ---
        GenericRecord emptySSN = new GenericData.Record(schema);
        emptySSN.put("name", "Empty SSN User");
        emptySSN.put("ssn", "");

        SchemaRegistryClient clientA = TestHelper.createClient(SCHEMA_REGISTRY_URL);
        KafkaAvroSerializer serializerA = TestHelper.createRuleAwareSerializer(SCHEMA_REGISTRY_URL, clientA);

        try {
            Exception thrown = assertThrows(Exception.class, () -> {
                serializerA.serialize(topic, emptySSN);
            }, "Serialization should fail — SSN is empty but tagged PII");

            assertTrue(isRuleConditionException(thrown),
                    "Exception cause chain should contain RuleConditionException for empty PII field, but got: "
                            + describeExceptionChain(thrown));

            System.out.println("CEL_FIELD CONDITION: empty PII SSN correctly rejected");
        } finally {
            serializerA.close();
        }

        // --- Case B: non-empty SSN -> should pass ---
        GenericRecord validSSN = new GenericData.Record(schema);
        validSSN.put("name", "Valid SSN User");
        validSSN.put("ssn", "123-45-6789");

        SchemaRegistryClient clientB = TestHelper.createClient(SCHEMA_REGISTRY_URL);
        KafkaAvroSerializer serializerB = TestHelper.createRuleAwareSerializer(SCHEMA_REGISTRY_URL, clientB);

        try {
            byte[] serialized = serializerB.serialize(topic, validSSN);
            assertNotNull(serialized, "Non-empty SSN should serialize successfully");

            System.out.println("CEL_FIELD CONDITION: non-empty PII SSN correctly accepted");
        } finally {
            serializerB.close();
        }
    }

    // -----------------------------------------------------------------------
    // Test 7: CEL_FIELD TRANSFORM on WRITE — normalize country to uppercase
    // -----------------------------------------------------------------------

    @Test
    @Order(7)
    @DisplayName("CEL_FIELD TRANSFORM on WRITE normalizes country field to uppercase")
    void testCelFieldTransformOnWriteNormalizeCountry() {
        String subject = "cel-field-normalize-" + System.currentTimeMillis() + "-value";
        registeredSubjects.add(subject);

        String addressSchemaStr = "{\"type\":\"record\",\"name\":\"Address\",\"namespace\":\"com.axonops.test.cel\","
                + "\"fields\":["
                + "{\"name\":\"street\",\"type\":\"string\"},"
                + "{\"name\":\"country\",\"type\":\"string\"}"
                + "]}";

        // CEL_FIELD TRANSFORM on WRITE: uppercase the 'country' field
        // Guard: name == 'country'
        // Body: value.upperAscii()
        String body = "{"
                + "\"schemaType\":\"AVRO\","
                + "\"schema\":\"" + escapeJson(addressSchemaStr) + "\","
                + "\"ruleSet\":{"
                + "  \"domainRules\":["
                + "    {"
                + "      \"name\":\"normalizeCountry\","
                + "      \"kind\":\"TRANSFORM\","
                + "      \"type\":\"CEL_FIELD\","
                + "      \"mode\":\"WRITE\","
                + "      \"expr\":\"name == 'country' ; value.upperAscii()\""
                + "    }"
                + "  ]"
                + "}"
                + "}";

        int schemaId = TestHelper.registerSchemaWithRules(SCHEMA_REGISTRY_URL, subject, body);
        assertTrue(schemaId > 0, "Schema should be registered with a positive ID");

        Schema schema = new Schema.Parser().parse(addressSchemaStr);
        GenericRecord address = new GenericData.Record(schema);
        address.put("street", "123 Main St");
        address.put("country", "us");

        String topic = subject.replace("-value", "");

        // Serialize with WRITE TRANSFORM — 'us' should become 'US'
        SchemaRegistryClient writeClient = TestHelper.createClient(SCHEMA_REGISTRY_URL);
        KafkaAvroSerializer serializer = TestHelper.createRuleAwareSerializer(SCHEMA_REGISTRY_URL, writeClient);

        byte[] serialized;
        try {
            serialized = serializer.serialize(topic, address);
            assertNotNull(serialized, "Serialization should succeed with TRANSFORM rule");
        } finally {
            serializer.close();
        }

        // Deserialize and verify the transform was applied
        SchemaRegistryClient readClient = TestHelper.createClient(SCHEMA_REGISTRY_URL);
        KafkaAvroDeserializer deserializer = TestHelper.createRuleAwareDeserializer(SCHEMA_REGISTRY_URL, readClient);

        try {
            GenericRecord deserialized = (GenericRecord) deserializer.deserialize(topic, serialized);
            assertNotNull(deserialized, "Deserialized record should not be null");
            assertEquals("123 Main St", deserialized.get("street").toString(),
                    "street should be unchanged");
            assertEquals("US", deserialized.get("country").toString(),
                    "country should be uppercased to 'US' by WRITE TRANSFORM");

            System.out.println("CEL_FIELD TRANSFORM on WRITE: country normalized from 'us' to '"
                    + deserialized.get("country") + "'");
        } finally {
            deserializer.close();
        }
    }

    // -----------------------------------------------------------------------
    // Test 8: Disabled rule is skipped
    // -----------------------------------------------------------------------

    @Test
    @Order(8)
    @DisplayName("Disabled CEL CONDITION rule is skipped and does not reject data")
    void testDisabledRuleSkipped() {
        String subject = "cel-condition-disabled-" + System.currentTimeMillis() + "-value";
        registeredSubjects.add(subject);

        String orderSchemaStr = "{\"type\":\"record\",\"name\":\"Order\",\"namespace\":\"com.axonops.test.cel\","
                + "\"fields\":["
                + "{\"name\":\"orderId\",\"type\":\"string\"},"
                + "{\"name\":\"amount\",\"type\":\"double\"},"
                + "{\"name\":\"currency\",\"type\":\"string\"}"
                + "]}";

        // CONDITION rule that would fail (amount > 1000) but is disabled
        String body = "{"
                + "\"schemaType\":\"AVRO\","
                + "\"schema\":\"" + escapeJson(orderSchemaStr) + "\","
                + "\"ruleSet\":{"
                + "  \"domainRules\":["
                + "    {"
                + "      \"name\":\"highValueOnly\","
                + "      \"kind\":\"CONDITION\","
                + "      \"type\":\"CEL\","
                + "      \"mode\":\"WRITE\","
                + "      \"expr\":\"message.amount > 1000.0\","
                + "      \"onFailure\":\"ERROR\","
                + "      \"disabled\":true"
                + "    }"
                + "  ]"
                + "}"
                + "}";

        int schemaId = TestHelper.registerSchemaWithRules(SCHEMA_REGISTRY_URL, subject, body);
        assertTrue(schemaId > 0, "Schema should be registered with a positive ID");

        // Build an order with amount=5 (which would fail the rule if it were enabled)
        Schema schema = new Schema.Parser().parse(orderSchemaStr);
        GenericRecord order = new GenericData.Record(schema);
        order.put("orderId", "ORD-LOW");
        order.put("amount", 5.0);
        order.put("currency", "EUR");

        String topic = subject.replace("-value", "");

        SchemaRegistryClient client = TestHelper.createClient(SCHEMA_REGISTRY_URL);
        KafkaAvroSerializer serializer = TestHelper.createRuleAwareSerializer(SCHEMA_REGISTRY_URL, client);
        KafkaAvroDeserializer deserializer = TestHelper.createRuleAwareDeserializer(SCHEMA_REGISTRY_URL, client);

        try {
            // Serialize should succeed — rule is disabled
            byte[] serialized = serializer.serialize(topic, order);
            assertNotNull(serialized, "Serialization should succeed with disabled rule");

            // Verify round-trip
            GenericRecord deserialized = (GenericRecord) deserializer.deserialize(topic, serialized);
            assertNotNull(deserialized, "Deserialized record should not be null");
            assertEquals("ORD-LOW", deserialized.get("orderId").toString(), "orderId should match");
            assertEquals(5.0, (double) deserialized.get("amount"), 0.001, "amount should match");
            assertEquals("EUR", deserialized.get("currency").toString(), "currency should match");

            System.out.println("Disabled rule test passed: amount=5 was accepted (rule skipped), schema ID " + schemaId);
        } finally {
            serializer.close();
            deserializer.close();
        }
    }

    // -----------------------------------------------------------------------
    // Helper methods
    // -----------------------------------------------------------------------

    /**
     * Escape a JSON string for embedding inside another JSON string value.
     * Replaces double quotes with backslash-escaped double quotes.
     */
    private static String escapeJson(String json) {
        return json.replace("\"", "\\\"");
    }

    /**
     * Walk the exception cause chain to determine if a RuleConditionException
     * is present. The Confluent serializer wraps rule failures in a
     * SerializationException, so the actual RuleConditionException may be
     * several levels deep.
     */
    private static boolean isRuleConditionException(Throwable t) {
        Throwable current = t;
        while (current != null) {
            String className = current.getClass().getName();
            // Check by class name to avoid compile-time dependency on internal class
            if (className.contains("RuleConditionException")
                    || className.contains("RuleException")) {
                return true;
            }
            // Some versions wrap the message with "Rule failed:" prefix
            if (current.getMessage() != null && current.getMessage().contains("Rule failed")) {
                return true;
            }
            current = current.getCause();
        }
        return false;
    }

    /**
     * Build a human-readable description of the exception cause chain
     * for diagnostic assertion messages.
     */
    private static String describeExceptionChain(Throwable t) {
        StringBuilder sb = new StringBuilder();
        Throwable current = t;
        int depth = 0;
        while (current != null && depth < 10) {
            if (depth > 0) {
                sb.append(" -> ");
            }
            sb.append(current.getClass().getName());
            if (current.getMessage() != null) {
                sb.append("(").append(truncate(current.getMessage(), 80)).append(")");
            }
            current = current.getCause();
            depth++;
        }
        return sb.toString();
    }

    /**
     * Truncate a string to the given max length, appending "..." if truncated.
     */
    private static String truncate(String s, int maxLen) {
        if (s.length() <= maxLen) {
            return s;
        }
        return s.substring(0, maxLen) + "...";
    }
}
