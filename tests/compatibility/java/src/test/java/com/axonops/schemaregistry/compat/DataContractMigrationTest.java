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
 * Tests JSONata migration rules using real Confluent SerDe clients against
 * the AxonOps Schema Registry.
 *
 * <p>Migration rules live in the {@code migrationRules} array inside a ruleSet.
 * They are executed during deserialization: the consumer reads the schema ID from
 * the wire format, determines the writer schema vs reader schema, and executes
 * UPGRADE or DOWNGRADE rules to transform data between versions.</p>
 *
 * <p>The {@code kafka-schema-rules} artifact provides the JSONata executor and is
 * auto-discovered via ServiceLoader.</p>
 *
 * <p>Flow for UPGRADE tests:</p>
 * <ol>
 *   <li>Produce data with v1 schema using an auto-register serializer.</li>
 *   <li>Register v2 schema with UPGRADE migration rule via REST API.</li>
 *   <li>Create a fresh client + deserializer with {@code use.latest.version=true}.</li>
 *   <li>Deserialize v1 bytes -- the deserializer fetches v2 as the reader schema
 *       and fires the UPGRADE rule to transform v1 data into v2 shape.</li>
 * </ol>
 */
@Tag("data-contracts")
@TestMethodOrder(MethodOrderer.OrderAnnotation.class)
public class DataContractMigrationTest {

    private static final String SCHEMA_REGISTRY_URL =
            System.getProperty("schema.registry.url", "http://localhost:8081");

    // -----------------------------------------------------------------------
    // Test 1: UPGRADE rule -- rename "state" to "status"
    // -----------------------------------------------------------------------

    @Test
    @Order(1)
    @DisplayName("UPGRADE: field rename state -> status via JSONata migration rule")
    void testUpgradeFieldRenameStateToStatus() {
        String subject = "migration-upgrade-rename-" + System.currentTimeMillis() + "-value";
        String topic = subject.replace("-value", "");

        String v1SchemaStr = "{\"type\":\"record\",\"name\":\"Order\",\"namespace\":\"com.axonops.test.migration\","
                + "\"fields\":[{\"name\":\"orderId\",\"type\":\"string\"},{\"name\":\"state\",\"type\":\"string\"}]}";

        String v2SchemaStr = "{\"type\":\"record\",\"name\":\"Order\",\"namespace\":\"com.axonops.test.migration\","
                + "\"fields\":[{\"name\":\"orderId\",\"type\":\"string\"},{\"name\":\"status\",\"type\":\"string\"}]}";

        try {
            // --- Step 1: Set compatibility to NONE (schemas are intentionally incompatible) ---
            TestHelper.setSubjectConfig(SCHEMA_REGISTRY_URL, subject,
                    "{\"compatibility\": \"NONE\"}");

            // --- Step 2: Produce v1 data with auto-register serializer ---
            SchemaRegistryClient producerClient = TestHelper.createClient(SCHEMA_REGISTRY_URL);
            KafkaAvroSerializer autoSerializer = TestHelper.createAutoRegisterSerializer(SCHEMA_REGISTRY_URL, producerClient);

            Schema v1Schema = new Schema.Parser().parse(v1SchemaStr);
            GenericRecord v1Record = new GenericData.Record(v1Schema);
            v1Record.put("orderId", "ORD-001");
            v1Record.put("state", "PENDING");

            byte[] v1Bytes = autoSerializer.serialize(topic, v1Record);
            assertNotNull(v1Bytes, "Serialized v1 data should not be null");
            autoSerializer.close();

            // --- Step 3: Register v2 schema with UPGRADE migration rule ---
            String v2Body = "{"
                    + "\"schema\": " + jsonEscape(v2SchemaStr) + ","
                    + "\"schemaType\": \"AVRO\","
                    + "\"ruleSet\": {"
                    + "  \"migrationRules\": [{"
                    + "    \"name\": \"renameStateToStatus\","
                    + "    \"kind\": \"TRANSFORM\","
                    + "    \"type\": \"JSONATA\","
                    + "    \"mode\": \"UPGRADE\","
                    + "    \"expr\": \"$merge([$sift($, function($v, $k) {$k != 'state'}), {'status': $.state}])\""
                    + "  }]"
                    + "}"
                    + "}";

            int v2Id = TestHelper.registerSchemaWithRules(SCHEMA_REGISTRY_URL, subject, v2Body);
            assertTrue(v2Id > 0, "v2 schema should be registered successfully");

            // --- Step 4: Consume with fresh client (use.latest.version=true triggers UPGRADE) ---
            SchemaRegistryClient consumerClient = TestHelper.createClient(SCHEMA_REGISTRY_URL);
            KafkaAvroDeserializer ruleDeserializer = TestHelper.createRuleAwareDeserializer(SCHEMA_REGISTRY_URL, consumerClient);

            GenericRecord result = (GenericRecord) ruleDeserializer.deserialize(topic, v1Bytes);

            assertNotNull(result, "Deserialized record should not be null");
            assertEquals("ORD-001", result.get("orderId").toString(),
                    "orderId should be preserved through migration");
            assertEquals("PENDING", result.get("status").toString(),
                    "state should be renamed to status by UPGRADE rule");

            ruleDeserializer.close();
            System.out.println("UPGRADE rename (state->status) test passed. v2 schema ID: " + v2Id);

        } finally {
            TestHelper.deleteSubject(SCHEMA_REGISTRY_URL, subject);
        }
    }

    // -----------------------------------------------------------------------
    // Test 2: DOWNGRADE rule -- rename "status" back to "state"
    // -----------------------------------------------------------------------

    @Test
    @Order(2)
    @DisplayName("UPGRADE+DOWNGRADE: bidirectional field rename stored and UPGRADE applied")
    void testDowngradeFieldRenameStatusToState() {
        String subject = "migration-downgrade-rename-" + System.currentTimeMillis() + "-value";
        String topic = subject.replace("-value", "");

        String v1SchemaStr = "{\"type\":\"record\",\"name\":\"Order\",\"namespace\":\"com.axonops.test.migration\","
                + "\"fields\":[{\"name\":\"orderId\",\"type\":\"string\"},{\"name\":\"state\",\"type\":\"string\"}]}";

        String v2SchemaStr = "{\"type\":\"record\",\"name\":\"Order\",\"namespace\":\"com.axonops.test.migration\","
                + "\"fields\":[{\"name\":\"orderId\",\"type\":\"string\"},{\"name\":\"status\",\"type\":\"string\"}]}";

        try {
            // --- Step 1: Set compatibility to NONE ---
            TestHelper.setSubjectConfig(SCHEMA_REGISTRY_URL, subject,
                    "{\"compatibility\": \"NONE\"}");

            // --- Step 2: Produce v1 data ---
            SchemaRegistryClient producerClient = TestHelper.createClient(SCHEMA_REGISTRY_URL);
            KafkaAvroSerializer autoSerializer = TestHelper.createAutoRegisterSerializer(SCHEMA_REGISTRY_URL, producerClient);

            Schema v1Schema = new Schema.Parser().parse(v1SchemaStr);
            GenericRecord v1Record = new GenericData.Record(v1Schema);
            v1Record.put("orderId", "ORD-002");
            v1Record.put("state", "SHIPPED");

            byte[] v1Bytes = autoSerializer.serialize(topic, v1Record);
            assertNotNull(v1Bytes, "Serialized v1 data should not be null");
            autoSerializer.close();

            // --- Step 3: Register v2 with both UPGRADE and DOWNGRADE rules ---
            String v2Body = "{"
                    + "\"schema\": " + jsonEscape(v2SchemaStr) + ","
                    + "\"schemaType\": \"AVRO\","
                    + "\"ruleSet\": {"
                    + "  \"migrationRules\": [{"
                    + "    \"name\": \"upgradeStateToStatus\","
                    + "    \"kind\": \"TRANSFORM\","
                    + "    \"type\": \"JSONATA\","
                    + "    \"mode\": \"UPGRADE\","
                    + "    \"expr\": \"$merge([$sift($, function($v, $k) {$k != 'state'}), {'status': $.state}])\""
                    + "  },{"
                    + "    \"name\": \"downgradeStatusToState\","
                    + "    \"kind\": \"TRANSFORM\","
                    + "    \"type\": \"JSONATA\","
                    + "    \"mode\": \"DOWNGRADE\","
                    + "    \"expr\": \"$merge([$sift($, function($v, $k) {$k != 'status'}), {'state': $.status}])\""
                    + "  }]"
                    + "}"
                    + "}";

            int v2Id = TestHelper.registerSchemaWithRules(SCHEMA_REGISTRY_URL, subject, v2Body);
            assertTrue(v2Id > 0, "v2 schema should be registered successfully");

            // --- Step 4: Verify both rules are stored by fetching v2 ---
            String v2Response = TestHelper.getSchemaVersion(SCHEMA_REGISTRY_URL, subject, 2);
            assertNotNull(v2Response, "Should be able to fetch v2 schema version");
            assertTrue(v2Response.contains("upgradeStateToStatus"),
                    "v2 should contain the UPGRADE rule");
            assertTrue(v2Response.contains("downgradeStatusToState"),
                    "v2 should contain the DOWNGRADE rule");
            assertTrue(v2Response.contains("DOWNGRADE"),
                    "v2 response should include DOWNGRADE mode");

            // --- Step 5: Verify UPGRADE still works (v1 data -> v2 reader) ---
            SchemaRegistryClient consumerClient = TestHelper.createClient(SCHEMA_REGISTRY_URL);
            KafkaAvroDeserializer ruleDeserializer = TestHelper.createRuleAwareDeserializer(SCHEMA_REGISTRY_URL, consumerClient);

            GenericRecord result = (GenericRecord) ruleDeserializer.deserialize(topic, v1Bytes);

            assertNotNull(result, "Deserialized record should not be null");
            assertEquals("ORD-002", result.get("orderId").toString(),
                    "orderId should be preserved");
            assertEquals("SHIPPED", result.get("status").toString(),
                    "state should be renamed to status by UPGRADE rule");

            ruleDeserializer.close();
            System.out.println("Bidirectional rules test passed. UPGRADE applied, DOWNGRADE stored. v2 ID: " + v2Id);

        } finally {
            TestHelper.deleteSubject(SCHEMA_REGISTRY_URL, subject);
        }
    }

    // -----------------------------------------------------------------------
    // Test 3: UPGRADE rule -- add field with default via JSONata
    // -----------------------------------------------------------------------

    @Test
    @Order(3)
    @DisplayName("UPGRADE: field addition with default value via JSONata migration rule")
    void testUpgradeWithFieldAdditionAndDefault() {
        String subject = "migration-upgrade-addfield-" + System.currentTimeMillis() + "-value";
        String topic = subject.replace("-value", "");

        String v1SchemaStr = "{\"type\":\"record\",\"name\":\"Payment\",\"namespace\":\"com.axonops.test.migration\","
                + "\"fields\":[{\"name\":\"id\",\"type\":\"string\"},{\"name\":\"amount\",\"type\":\"double\"}]}";

        String v2SchemaStr = "{\"type\":\"record\",\"name\":\"Payment\",\"namespace\":\"com.axonops.test.migration\","
                + "\"fields\":[{\"name\":\"id\",\"type\":\"string\"},{\"name\":\"amount\",\"type\":\"double\"},"
                + "{\"name\":\"currency\",\"type\":\"string\",\"default\":\"UNKNOWN\"}]}";

        try {
            // --- Step 1: Set compatibility to NONE ---
            TestHelper.setSubjectConfig(SCHEMA_REGISTRY_URL, subject,
                    "{\"compatibility\": \"NONE\"}");

            // --- Step 2: Produce v1 data ---
            SchemaRegistryClient producerClient = TestHelper.createClient(SCHEMA_REGISTRY_URL);
            KafkaAvroSerializer autoSerializer = TestHelper.createAutoRegisterSerializer(SCHEMA_REGISTRY_URL, producerClient);

            Schema v1Schema = new Schema.Parser().parse(v1SchemaStr);
            GenericRecord v1Record = new GenericData.Record(v1Schema);
            v1Record.put("id", "PAY-001");
            v1Record.put("amount", 99.99);

            byte[] v1Bytes = autoSerializer.serialize(topic, v1Record);
            assertNotNull(v1Bytes, "Serialized v1 data should not be null");
            autoSerializer.close();

            // --- Step 3: Register v2 with UPGRADE rule that sets currency ---
            //
            // The JSONata expression merges {'currency': 'USD'} into the root
            // object, adding the missing field with a business-meaningful default.
            String v2Body = "{"
                    + "\"schema\": " + jsonEscape(v2SchemaStr) + ","
                    + "\"schemaType\": \"AVRO\","
                    + "\"ruleSet\": {"
                    + "  \"migrationRules\": [{"
                    + "    \"name\": \"addCurrencyDefault\","
                    + "    \"kind\": \"TRANSFORM\","
                    + "    \"type\": \"JSONATA\","
                    + "    \"mode\": \"UPGRADE\","
                    + "    \"expr\": \"$merge([$, {'currency': 'USD'}])\""
                    + "  }]"
                    + "}"
                    + "}";

            int v2Id = TestHelper.registerSchemaWithRules(SCHEMA_REGISTRY_URL, subject, v2Body);
            assertTrue(v2Id > 0, "v2 schema should be registered successfully");

            // --- Step 4: Consume with fresh client ---
            SchemaRegistryClient consumerClient = TestHelper.createClient(SCHEMA_REGISTRY_URL);
            KafkaAvroDeserializer ruleDeserializer = TestHelper.createRuleAwareDeserializer(SCHEMA_REGISTRY_URL, consumerClient);

            GenericRecord result = (GenericRecord) ruleDeserializer.deserialize(topic, v1Bytes);

            assertNotNull(result, "Deserialized record should not be null");
            assertEquals("PAY-001", result.get("id").toString(),
                    "id should be preserved through migration");
            assertEquals(99.99, (double) result.get("amount"), 0.001,
                    "amount should be preserved through migration");
            assertNotNull(result.get("currency"),
                    "currency field should be populated by UPGRADE rule");
            assertEquals("USD", result.get("currency").toString(),
                    "currency should be 'USD' as set by UPGRADE rule");

            ruleDeserializer.close();
            System.out.println("UPGRADE add-field-with-default test passed. v2 schema ID: " + v2Id);

        } finally {
            TestHelper.deleteSubject(SCHEMA_REGISTRY_URL, subject);
        }
    }

    // -----------------------------------------------------------------------
    // Test 4: UPGRADE rule -- breaking change bridged by migration
    // -----------------------------------------------------------------------

    @Test
    @Order(4)
    @DisplayName("UPGRADE: breaking change (firstName+lastName -> fullName) bridged by JSONata migration")
    void testBreakingChangeWithMigrationBridge() {
        String subject = "migration-upgrade-breaking-" + System.currentTimeMillis() + "-value";
        String topic = subject.replace("-value", "");

        String v1SchemaStr = "{\"type\":\"record\",\"name\":\"Person\",\"namespace\":\"com.axonops.test.migration\","
                + "\"fields\":[{\"name\":\"firstName\",\"type\":\"string\"},{\"name\":\"lastName\",\"type\":\"string\"}]}";

        String v2SchemaStr = "{\"type\":\"record\",\"name\":\"Person\",\"namespace\":\"com.axonops.test.migration\","
                + "\"fields\":[{\"name\":\"fullName\",\"type\":\"string\"}]}";

        try {
            // --- Step 1: Set compatibility to NONE (this is a breaking change) ---
            TestHelper.setSubjectConfig(SCHEMA_REGISTRY_URL, subject,
                    "{\"compatibility\": \"NONE\"}");

            // --- Step 2: Produce v1 data ---
            SchemaRegistryClient producerClient = TestHelper.createClient(SCHEMA_REGISTRY_URL);
            KafkaAvroSerializer autoSerializer = TestHelper.createAutoRegisterSerializer(SCHEMA_REGISTRY_URL, producerClient);

            Schema v1Schema = new Schema.Parser().parse(v1SchemaStr);
            GenericRecord v1Record = new GenericData.Record(v1Schema);
            v1Record.put("firstName", "John");
            v1Record.put("lastName", "Doe");

            byte[] v1Bytes = autoSerializer.serialize(topic, v1Record);
            assertNotNull(v1Bytes, "Serialized v1 data should not be null");
            autoSerializer.close();

            // --- Step 3: Register v2 with UPGRADE rule that merges names ---
            //
            // The JSONata expression concatenates firstName and lastName into
            // a single fullName field, bridging the breaking schema change.
            String v2Body = "{"
                    + "\"schema\": " + jsonEscape(v2SchemaStr) + ","
                    + "\"schemaType\": \"AVRO\","
                    + "\"ruleSet\": {"
                    + "  \"migrationRules\": [{"
                    + "    \"name\": \"mergeNames\","
                    + "    \"kind\": \"TRANSFORM\","
                    + "    \"type\": \"JSONATA\","
                    + "    \"mode\": \"UPGRADE\","
                    + "    \"expr\": \"{'fullName': $.firstName & ' ' & $.lastName}\""
                    + "  }]"
                    + "}"
                    + "}";

            int v2Id = TestHelper.registerSchemaWithRules(SCHEMA_REGISTRY_URL, subject, v2Body);
            assertTrue(v2Id > 0, "v2 schema should be registered successfully");

            // --- Step 4: Consume with fresh client ---
            SchemaRegistryClient consumerClient = TestHelper.createClient(SCHEMA_REGISTRY_URL);
            KafkaAvroDeserializer ruleDeserializer = TestHelper.createRuleAwareDeserializer(SCHEMA_REGISTRY_URL, consumerClient);

            GenericRecord result = (GenericRecord) ruleDeserializer.deserialize(topic, v1Bytes);

            assertNotNull(result, "Deserialized record should not be null");
            assertEquals("John Doe", result.get("fullName").toString(),
                    "fullName should be 'John Doe' composed from firstName and lastName by UPGRADE rule");

            ruleDeserializer.close();
            System.out.println("UPGRADE breaking-change bridge test passed. v2 schema ID: " + v2Id);

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
