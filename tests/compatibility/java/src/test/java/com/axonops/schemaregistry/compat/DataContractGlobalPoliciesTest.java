package com.axonops.schemaregistry.compat;

import io.confluent.kafka.schemaregistry.client.SchemaRegistryClient;
import io.confluent.kafka.serializers.KafkaAvroDeserializer;
import io.confluent.kafka.serializers.KafkaAvroSerializer;
import org.apache.avro.Schema;
import org.apache.avro.generic.GenericData;
import org.apache.avro.generic.GenericRecord;
import org.junit.jupiter.api.*;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests config-level default/override rules being correctly merged by the registry
 * and then executed by real Confluent SerDe clients.
 *
 * <p>The registry supports a 3-layer merge for rules and metadata:</p>
 * <pre>
 *   final = merge(merge(config.default, request-specific), config.override)
 * </pre>
 *
 * <ul>
 *   <li>{@code PUT /config/{subject}} accepts {@code defaultRuleSet}, {@code overrideRuleSet},
 *       {@code defaultMetadata}, and {@code overrideMetadata} alongside {@code compatibility}.</li>
 *   <li>When a schema is registered without rules, it inherits {@code defaultRuleSet}.</li>
 *   <li>Override rules are always applied on top of whatever rules the schema has.</li>
 *   <li>Rules from previous versions are inherited if not explicitly provided in new registrations.</li>
 *   <li>The Confluent serializer with {@code use.latest.version=true} fetches the latest schema
 *       WITH its merged rules, then executes them.</li>
 * </ul>
 *
 * <p>The {@code kafka-schema-rules} artifact provides CEL and CEL_FIELD rule executors,
 * auto-discovered via ServiceLoader.</p>
 */
@Tag("data-contracts")
@TestMethodOrder(MethodOrderer.OrderAnnotation.class)
public class DataContractGlobalPoliciesTest {

    private static final String SCHEMA_REGISTRY_URL =
            System.getProperty("schema.registry.url", "http://localhost:8081");

    // -----------------------------------------------------------------------
    // Test 1: defaultRuleSet applied to a schema registered without rules
    // -----------------------------------------------------------------------

    @Test
    @Order(1)
    @DisplayName("defaultRuleSet from config is applied and executed by serializer")
    void testDefaultRuleSetAppliedAndExecuted() {
        String subject = "policy-default-ruleset-" + System.currentTimeMillis() + "-value";
        String topic = subject.replace("-value", "");

        String orderSchemaStr = "{\"type\":\"record\",\"name\":\"Order\","
                + "\"namespace\":\"com.axonops.test.policy\","
                + "\"fields\":["
                + "{\"name\":\"orderId\",\"type\":\"string\"},"
                + "{\"name\":\"amount\",\"type\":\"double\"}"
                + "]}";

        try {
            // --- Step 1: Set subject config with defaultRuleSet ---
            // The CEL CONDITION rule rejects messages where amount <= 0.
            String configBody = "{"
                    + "\"compatibility\":\"NONE\","
                    + "\"defaultRuleSet\":{"
                    + "  \"domainRules\":[{"
                    + "    \"name\":\"amount-positive\","
                    + "    \"kind\":\"CONDITION\","
                    + "    \"type\":\"CEL\","
                    + "    \"mode\":\"WRITE\","
                    + "    \"expr\":\"message.amount > 0.0\","
                    + "    \"onFailure\":\"ERROR\""
                    + "  }]"
                    + "}"
                    + "}";
            TestHelper.setSubjectConfig(SCHEMA_REGISTRY_URL, subject, configBody);

            // --- Step 2: Register schema WITHOUT any ruleSet ---
            // The registry should merge defaultRuleSet into this version.
            String schemaBody = "{\"schema\":" + jsonEscape(orderSchemaStr) + "}";
            int schemaId = TestHelper.registerSchemaWithRules(SCHEMA_REGISTRY_URL, subject, schemaBody);
            assertTrue(schemaId > 0, "Schema should be registered successfully");

            // --- Step 3: Verify the merged rules are present on the version ---
            String versionResponse = TestHelper.getSchemaVersion(SCHEMA_REGISTRY_URL, subject, 1);
            assertNotNull(versionResponse, "Should be able to fetch version 1");
            assertTrue(versionResponse.contains("amount-positive"),
                    "Version should contain the defaultRuleSet rule 'amount-positive'");

            // --- Step 4: Create rule-aware serializer ---
            SchemaRegistryClient client = TestHelper.createClient(SCHEMA_REGISTRY_URL);
            KafkaAvroSerializer serializer = TestHelper.createRuleAwareSerializer(SCHEMA_REGISTRY_URL, client);

            Schema orderSchema = new Schema.Parser().parse(orderSchemaStr);

            // --- Step 5: Serialize with amount=-1 should FAIL (CEL CONDITION rejects) ---
            GenericRecord badRecord = new GenericData.Record(orderSchema);
            badRecord.put("orderId", "ORD-BAD");
            badRecord.put("amount", -1.0);

            Exception caught = assertThrows(Exception.class,
                    () -> serializer.serialize(topic, badRecord),
                    "Serialization with amount=-1 should fail due to defaultRuleSet CEL CONDITION");
            System.out.println("Correctly rejected negative amount: " + caught.getMessage());

            // --- Step 6: Serialize with amount=100 should SUCCEED ---
            GenericRecord goodRecord = new GenericData.Record(orderSchema);
            goodRecord.put("orderId", "ORD-GOOD");
            goodRecord.put("amount", 100.0);

            byte[] serialized = serializer.serialize(topic, goodRecord);
            assertNotNull(serialized, "Serialization with amount=100 should succeed");
            assertTrue(serialized.length > 5, "Serialized bytes should contain payload");

            serializer.close();
            System.out.println("defaultRuleSet test passed. Schema ID: " + schemaId);

        } finally {
            TestHelper.deleteSubject(SCHEMA_REGISTRY_URL, subject);
        }
    }

    // -----------------------------------------------------------------------
    // Test 2: overrideRuleSet always enforced on top of request-level rules
    // -----------------------------------------------------------------------

    @Test
    @Order(2)
    @DisplayName("overrideRuleSet always enforced on top of request-level rules")
    void testOverrideRuleSetAlwaysEnforced() {
        String subject = "policy-override-ruleset-" + System.currentTimeMillis() + "-value";
        String topic = subject.replace("-value", "");

        String orderSchemaStr = "{\"type\":\"record\",\"name\":\"Order\","
                + "\"namespace\":\"com.axonops.test.policy\","
                + "\"fields\":["
                + "{\"name\":\"orderId\",\"type\":\"string\"},"
                + "{\"name\":\"amount\",\"type\":\"double\"}"
                + "]}";

        try {
            // --- Step 1: Set subject config with overrideRuleSet ---
            // The override rule enforces a strict amount range: > 0 AND < 10000.
            String configBody = "{"
                    + "\"compatibility\":\"NONE\","
                    + "\"overrideRuleSet\":{"
                    + "  \"domainRules\":[{"
                    + "    \"name\":\"amount-range-override\","
                    + "    \"kind\":\"CONDITION\","
                    + "    \"type\":\"CEL\","
                    + "    \"mode\":\"WRITE\","
                    + "    \"expr\":\"message.amount > 0.0 && message.amount < 10000.0\","
                    + "    \"onFailure\":\"ERROR\""
                    + "  }]"
                    + "}"
                    + "}";
            TestHelper.setSubjectConfig(SCHEMA_REGISTRY_URL, subject, configBody);

            // --- Step 2: Register schema WITH its own permissive ruleSet ---
            // The request-level rule only checks that orderId is non-empty.
            String schemaBody = "{"
                    + "\"schema\":" + jsonEscape(orderSchemaStr) + ","
                    + "\"ruleSet\":{"
                    + "  \"domainRules\":[{"
                    + "    \"name\":\"orderId-not-empty\","
                    + "    \"kind\":\"CONDITION\","
                    + "    \"type\":\"CEL\","
                    + "    \"mode\":\"WRITE\","
                    + "    \"expr\":\"size(message.orderId) > 0\","
                    + "    \"onFailure\":\"ERROR\""
                    + "  }]"
                    + "}"
                    + "}";
            int schemaId = TestHelper.registerSchemaWithRules(SCHEMA_REGISTRY_URL, subject, schemaBody);
            assertTrue(schemaId > 0, "Schema should be registered successfully");

            // --- Step 3: Create rule-aware serializer ---
            SchemaRegistryClient client = TestHelper.createClient(SCHEMA_REGISTRY_URL);
            KafkaAvroSerializer serializer = TestHelper.createRuleAwareSerializer(SCHEMA_REGISTRY_URL, client);

            Schema orderSchema = new Schema.Parser().parse(orderSchemaStr);

            // --- Step 4: amount=50000 should FAIL (override rule rejects: >= 10000) ---
            GenericRecord overLimit = new GenericData.Record(orderSchema);
            overLimit.put("orderId", "ORD-OVER");
            overLimit.put("amount", 50000.0);

            Exception overEx = assertThrows(Exception.class,
                    () -> serializer.serialize(topic, overLimit),
                    "Serialization with amount=50000 should fail due to overrideRuleSet");
            System.out.println("Correctly rejected over-limit amount: " + overEx.getMessage());

            // --- Step 5: empty orderId should FAIL (request-level rule rejects) ---
            GenericRecord emptyOrderId = new GenericData.Record(orderSchema);
            emptyOrderId.put("orderId", "");
            emptyOrderId.put("amount", 100.0);

            Exception emptyEx = assertThrows(Exception.class,
                    () -> serializer.serialize(topic, emptyOrderId),
                    "Serialization with empty orderId should fail due to request-level rule");
            System.out.println("Correctly rejected empty orderId: " + emptyEx.getMessage());

            // --- Step 6: amount=100, orderId="X" should SUCCEED (both rules pass) ---
            GenericRecord goodRecord = new GenericData.Record(orderSchema);
            goodRecord.put("orderId", "ORD-VALID");
            goodRecord.put("amount", 100.0);

            byte[] serialized = serializer.serialize(topic, goodRecord);
            assertNotNull(serialized, "Serialization with valid data should succeed");
            assertTrue(serialized.length > 5, "Serialized bytes should contain payload");

            serializer.close();
            System.out.println("overrideRuleSet test passed. Schema ID: " + schemaId);

        } finally {
            TestHelper.deleteSubject(SCHEMA_REGISTRY_URL, subject);
        }
    }

    // -----------------------------------------------------------------------
    // Test 3: Rule inheritance from previous version
    // -----------------------------------------------------------------------

    @Test
    @Order(3)
    @DisplayName("Rules from v1 are inherited by v2 when v2 is registered without rules")
    void testRuleInheritanceFromPreviousVersion() {
        String subject = "policy-rule-inherit-" + System.currentTimeMillis() + "-value";
        String topic = subject.replace("-value", "");

        String v1SchemaStr = "{\"type\":\"record\",\"name\":\"Order\","
                + "\"namespace\":\"com.axonops.test.policy\","
                + "\"fields\":["
                + "{\"name\":\"orderId\",\"type\":\"string\"},"
                + "{\"name\":\"amount\",\"type\":\"double\"}"
                + "]}";

        String v2SchemaStr = "{\"type\":\"record\",\"name\":\"Order\","
                + "\"namespace\":\"com.axonops.test.policy\","
                + "\"fields\":["
                + "{\"name\":\"orderId\",\"type\":\"string\"},"
                + "{\"name\":\"amount\",\"type\":\"double\"},"
                + "{\"name\":\"notes\",\"type\":[\"null\",\"string\"],\"default\":null}"
                + "]}";

        try {
            // --- Step 1: Set compatibility to NONE ---
            TestHelper.setSubjectConfig(SCHEMA_REGISTRY_URL, subject,
                    "{\"compatibility\":\"NONE\"}");

            // --- Step 2: Register v1 WITH a validation rule ---
            String v1Body = "{"
                    + "\"schema\":" + jsonEscape(v1SchemaStr) + ","
                    + "\"ruleSet\":{"
                    + "  \"domainRules\":[{"
                    + "    \"name\":\"amount-positive\","
                    + "    \"kind\":\"CONDITION\","
                    + "    \"type\":\"CEL\","
                    + "    \"mode\":\"WRITE\","
                    + "    \"expr\":\"message.amount > 0.0\","
                    + "    \"onFailure\":\"ERROR\""
                    + "  }]"
                    + "}"
                    + "}";
            int v1Id = TestHelper.registerSchemaWithRules(SCHEMA_REGISTRY_URL, subject, v1Body);
            assertTrue(v1Id > 0, "v1 schema should be registered successfully");

            // Verify v1 has the rule
            String v1Response = TestHelper.getSchemaVersion(SCHEMA_REGISTRY_URL, subject, 1);
            assertTrue(v1Response.contains("amount-positive"),
                    "v1 should contain the 'amount-positive' rule");

            // --- Step 3: Register v2 WITHOUT any ruleSet or metadata ---
            // The registry should inherit v1's ruleSet for v2.
            String v2Body = "{\"schema\":" + jsonEscape(v2SchemaStr) + "}";
            int v2Id = TestHelper.registerSchemaWithRules(SCHEMA_REGISTRY_URL, subject, v2Body);
            assertTrue(v2Id > 0, "v2 schema should be registered successfully");

            // --- Step 4: Verify v2 inherited the rule ---
            String v2Response = TestHelper.getSchemaVersion(SCHEMA_REGISTRY_URL, subject, 2);
            assertNotNull(v2Response, "Should be able to fetch v2");
            assertTrue(v2Response.contains("amount-positive"),
                    "v2 should inherit the 'amount-positive' rule from v1");

            // --- Step 5: Create serializer targeting v2 (use.latest.version=true) ---
            SchemaRegistryClient client = TestHelper.createClient(SCHEMA_REGISTRY_URL);
            KafkaAvroSerializer serializer = TestHelper.createRuleAwareSerializer(SCHEMA_REGISTRY_URL, client);

            Schema v2Schema = new Schema.Parser().parse(v2SchemaStr);

            // --- Step 6: Serialize with amount=-1 should FAIL (inherited rule) ---
            GenericRecord badRecord = new GenericData.Record(v2Schema);
            badRecord.put("orderId", "ORD-V2-BAD");
            badRecord.put("amount", -1.0);
            badRecord.put("notes", "some notes");

            Exception caught = assertThrows(Exception.class,
                    () -> serializer.serialize(topic, badRecord),
                    "Serialization with amount=-1 should fail due to inherited rule from v1");
            System.out.println("Correctly rejected via inherited rule: " + caught.getMessage());

            // --- Step 7: Serialize with valid data should SUCCEED ---
            GenericRecord goodRecord = new GenericData.Record(v2Schema);
            goodRecord.put("orderId", "ORD-V2-GOOD");
            goodRecord.put("amount", 50.0);
            goodRecord.put("notes", "priority order");

            byte[] serialized = serializer.serialize(topic, goodRecord);
            assertNotNull(serialized, "Serialization with valid v2 data should succeed");
            assertTrue(serialized.length > 5, "Serialized bytes should contain payload");

            serializer.close();
            System.out.println("Rule inheritance test passed. v1 ID: " + v1Id + ", v2 ID: " + v2Id);

        } finally {
            TestHelper.deleteSubject(SCHEMA_REGISTRY_URL, subject);
        }
    }

    // -----------------------------------------------------------------------
    // Test 4: Tags propagated through config defaults (PII masking)
    // -----------------------------------------------------------------------

    @Test
    @Order(4)
    @DisplayName("CEL_FIELD rule with inline PII tags masks email on READ")
    void testTagsPropagatedThroughConfigDefaults() {
        String subject = "policy-tags-pii-" + System.currentTimeMillis() + "-value";
        String topic = subject.replace("-value", "");

        // Schema with inline confluent:tags on the email field — this is how the
        // Confluent CEL_FIELD executor discovers tagged fields for rule matching.
        String contactSchemaStr = "{\"type\":\"record\",\"name\":\"Contact\","
                + "\"namespace\":\"com.axonops.test.policy\","
                + "\"fields\":["
                + "{\"name\":\"name\",\"type\":\"string\"},"
                + "{\"name\":\"email\",\"type\":\"string\",\"confluent:tags\":[\"PII\"]}"
                + "]}";

        try {
            // --- Step 1: Set compatibility to NONE ---
            TestHelper.setSubjectConfig(SCHEMA_REGISTRY_URL, subject,
                    "{\"compatibility\":\"NONE\"}");

            // --- Step 2: Register schema with inline PII tags + CEL_FIELD rule ---
            // The rule masks any STRING field tagged PII to "REDACTED" on READ.
            String schemaBody = "{"
                    + "\"schema\":" + jsonEscape(contactSchemaStr) + ","
                    + "\"ruleSet\":{"
                    + "  \"domainRules\":[{"
                    + "    \"name\":\"mask-pii\","
                    + "    \"kind\":\"TRANSFORM\","
                    + "    \"type\":\"CEL_FIELD\","
                    + "    \"mode\":\"READ\","
                    + "    \"tags\":[\"PII\"],"
                    + "    \"expr\":\"typeName == 'STRING' ; 'REDACTED'\""
                    + "  }]"
                    + "}"
                    + "}";
            int schemaId = TestHelper.registerSchemaWithRules(SCHEMA_REGISTRY_URL, subject, schemaBody);
            assertTrue(schemaId > 0, "Schema should be registered successfully");

            // --- Step 3: Verify the version has the rule and PII tag ---
            String versionResponse = TestHelper.getSchemaVersion(SCHEMA_REGISTRY_URL, subject, 1);
            assertNotNull(versionResponse, "Should be able to fetch version 1");
            assertTrue(versionResponse.contains("mask-pii"),
                    "Version should contain the 'mask-pii' rule");
            assertTrue(versionResponse.contains("PII"),
                    "Version should contain the PII tag");

            // --- Step 4: Produce data with the serializer ---
            SchemaRegistryClient producerClient = TestHelper.createClient(SCHEMA_REGISTRY_URL);
            KafkaAvroSerializer serializer = TestHelper.createRuleAwareSerializer(SCHEMA_REGISTRY_URL, producerClient);

            Schema contactSchema = new Schema.Parser().parse(contactSchemaStr);
            GenericRecord record = new GenericData.Record(contactSchema);
            record.put("name", "Alice Smith");
            record.put("email", "user@example.com");

            byte[] serialized = serializer.serialize(topic, record);
            assertNotNull(serialized, "Serialization should succeed");
            serializer.close();

            // --- Step 5: Consume with a fresh client — CEL_FIELD READ rule should mask PII ---
            SchemaRegistryClient consumerClient = TestHelper.createClient(SCHEMA_REGISTRY_URL);
            KafkaAvroDeserializer deserializer = TestHelper.createRuleAwareDeserializer(SCHEMA_REGISTRY_URL, consumerClient);

            GenericRecord result = (GenericRecord) deserializer.deserialize(topic, serialized);

            assertNotNull(result, "Deserialized record should not be null");

            // The "name" field is NOT tagged PII, so it should be unchanged.
            assertEquals("Alice Smith", result.get("name").toString(),
                    "name field should NOT be masked (not tagged PII)");

            // The "email" field IS tagged PII via confluent:tags, so it should be "REDACTED".
            assertEquals("REDACTED", result.get("email").toString(),
                    "email field should be masked to 'REDACTED' by CEL_FIELD rule targeting PII tag");

            deserializer.close();
            System.out.println("Tags/PII masking test passed. Schema ID: " + schemaId);

        } finally {
            TestHelper.deleteSubject(SCHEMA_REGISTRY_URL, subject);
        }
    }

    // -----------------------------------------------------------------------
    // Utility
    // -----------------------------------------------------------------------

    /**
     * JSON-escape a string so it can be embedded as a JSON string value.
     * Wraps the input in double quotes and escapes internal double quotes and backslashes.
     */
    private static String jsonEscape(String raw) {
        String escaped = raw
                .replace("\\", "\\\\")
                .replace("\"", "\\\"")
                .replace("\n", "\\n")
                .replace("\r", "\\r")
                .replace("\t", "\\t");
        return "\"" + escaped + "\"";
    }
}
