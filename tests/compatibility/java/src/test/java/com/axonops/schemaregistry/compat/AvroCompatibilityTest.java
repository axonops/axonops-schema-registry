package com.axonops.schemaregistry.compat;

import io.confluent.kafka.schemaregistry.client.CachedSchemaRegistryClient;
import io.confluent.kafka.schemaregistry.client.SchemaRegistryClient;
import io.confluent.kafka.schemaregistry.client.rest.exceptions.RestClientException;
import io.confluent.kafka.serializers.KafkaAvroDeserializer;
import io.confluent.kafka.serializers.KafkaAvroSerializer;
import org.apache.avro.Schema;
import org.apache.avro.generic.GenericData;
import org.apache.avro.generic.GenericRecord;
import org.junit.jupiter.api.*;

import java.io.IOException;
import java.util.HashMap;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Avro serializer/deserializer compatibility tests.
 *
 * These tests verify that the AxonOps Schema Registry correctly handles
 * Avro schemas from different versions of Confluent serializers.
 */
@TestMethodOrder(MethodOrderer.OrderAnnotation.class)
public class AvroCompatibilityTest {

    private static final String SCHEMA_REGISTRY_URL = System.getProperty("schema.registry.url", "http://localhost:8081");
    private static final String CONFLUENT_VERSION = System.getProperty("confluent.version", "unknown");
    private static final String TOPIC_PREFIX = "avro-compat-test-";

    private static SchemaRegistryClient schemaRegistryClient;
    private static KafkaAvroSerializer serializer;
    private static KafkaAvroDeserializer deserializer;

    private static final String USER_SCHEMA_V1 = """
        {
            "type": "record",
            "name": "User",
            "namespace": "com.axonops.test",
            "fields": [
                {"name": "id", "type": "long"},
                {"name": "name", "type": "string"},
                {"name": "email", "type": "string"}
            ]
        }
        """;

    private static final String USER_SCHEMA_V2 = """
        {
            "type": "record",
            "name": "User",
            "namespace": "com.axonops.test",
            "fields": [
                {"name": "id", "type": "long"},
                {"name": "name", "type": "string"},
                {"name": "email", "type": "string"},
                {"name": "age", "type": ["null", "int"], "default": null}
            ]
        }
        """;

    @BeforeAll
    static void setUp() {
        System.out.println("Running Avro compatibility tests with Confluent version: " + CONFLUENT_VERSION);
        System.out.println("Schema Registry URL: " + SCHEMA_REGISTRY_URL);

        schemaRegistryClient = new CachedSchemaRegistryClient(SCHEMA_REGISTRY_URL, 100);

        Map<String, Object> serializerConfig = new HashMap<>();
        serializerConfig.put("schema.registry.url", SCHEMA_REGISTRY_URL);
        serializerConfig.put("auto.register.schemas", true);

        serializer = new KafkaAvroSerializer(schemaRegistryClient);
        serializer.configure(serializerConfig, false);

        deserializer = new KafkaAvroDeserializer(schemaRegistryClient);
        deserializer.configure(serializerConfig, false);
    }

    @AfterAll
    static void tearDown() {
        if (serializer != null) {
            serializer.close();
        }
        if (deserializer != null) {
            deserializer.close();
        }
    }

    @Test
    @Order(1)
    @DisplayName("Register Avro schema via serializer")
    void testSchemaRegistration() throws IOException, RestClientException {
        String topic = TOPIC_PREFIX + System.currentTimeMillis();
        String subject = topic + "-value";

        Schema schema = new Schema.Parser().parse(USER_SCHEMA_V1);
        GenericRecord record = new GenericData.Record(schema);
        record.put("id", 1L);
        record.put("name", "Test User");
        record.put("email", "test@example.com");

        // Serialize - this should auto-register the schema
        byte[] serialized = serializer.serialize(topic, record);

        assertNotNull(serialized, "Serialized data should not be null");
        assertTrue(serialized.length > 5, "Serialized data should have magic byte + schema ID + payload");
        assertEquals(0, serialized[0], "First byte should be magic byte 0");

        // Verify schema was registered
        int schemaId = schemaRegistryClient.getLatestSchemaMetadata(subject).getId();
        assertTrue(schemaId > 0, "Schema ID should be positive");

        System.out.println("Registered schema with ID: " + schemaId);
    }

    @Test
    @Order(2)
    @DisplayName("Serialize and deserialize round-trip")
    void testSerializeDeserializeRoundTrip() {
        String topic = TOPIC_PREFIX + "roundtrip-" + System.currentTimeMillis();

        Schema schema = new Schema.Parser().parse(USER_SCHEMA_V1);
        GenericRecord original = new GenericData.Record(schema);
        original.put("id", 42L);
        original.put("name", "Round Trip User");
        original.put("email", "roundtrip@example.com");

        // Serialize
        byte[] serialized = serializer.serialize(topic, original);

        // Deserialize
        GenericRecord deserialized = (GenericRecord) deserializer.deserialize(topic, serialized);

        assertNotNull(deserialized, "Deserialized record should not be null");
        assertEquals(original.get("id"), deserialized.get("id"), "ID should match");
        assertEquals(original.get("name").toString(), deserialized.get("name").toString(), "Name should match");
        assertEquals(original.get("email").toString(), deserialized.get("email").toString(), "Email should match");
    }

    @Test
    @Order(3)
    @DisplayName("Schema deduplication - same schema returns same ID")
    void testSchemaDeduplication() throws IOException, RestClientException {
        String topic = TOPIC_PREFIX + "dedup-" + System.currentTimeMillis();
        String subject = topic + "-value";

        Schema schema = new Schema.Parser().parse(USER_SCHEMA_V1);

        // Create two records with same schema
        GenericRecord record1 = new GenericData.Record(schema);
        record1.put("id", 1L);
        record1.put("name", "User 1");
        record1.put("email", "user1@example.com");

        GenericRecord record2 = new GenericData.Record(schema);
        record2.put("id", 2L);
        record2.put("name", "User 2");
        record2.put("email", "user2@example.com");

        // Serialize both
        byte[] serialized1 = serializer.serialize(topic, record1);
        byte[] serialized2 = serializer.serialize(topic, record2);

        // Extract schema IDs (bytes 1-4, big-endian)
        int schemaId1 = extractSchemaId(serialized1);
        int schemaId2 = extractSchemaId(serialized2);

        assertEquals(schemaId1, schemaId2, "Same schema should return same ID (deduplication)");
        System.out.println("Schema deduplication working: both records use schema ID " + schemaId1);
    }

    @Test
    @Order(4)
    @DisplayName("Schema evolution - backward compatible change")
    void testSchemaEvolution() throws IOException, RestClientException {
        String topic = TOPIC_PREFIX + "evolution-" + System.currentTimeMillis();
        String subject = topic + "-value";

        // Register v1 schema
        Schema schemaV1 = new Schema.Parser().parse(USER_SCHEMA_V1);
        GenericRecord recordV1 = new GenericData.Record(schemaV1);
        recordV1.put("id", 1L);
        recordV1.put("name", "User V1");
        recordV1.put("email", "v1@example.com");

        byte[] serializedV1 = serializer.serialize(topic, recordV1);
        int schemaIdV1 = extractSchemaId(serializedV1);

        // Register v2 schema (backward compatible)
        Schema schemaV2 = new Schema.Parser().parse(USER_SCHEMA_V2);
        GenericRecord recordV2 = new GenericData.Record(schemaV2);
        recordV2.put("id", 2L);
        recordV2.put("name", "User V2");
        recordV2.put("email", "v2@example.com");
        recordV2.put("age", 25);

        byte[] serializedV2 = serializer.serialize(topic, recordV2);
        int schemaIdV2 = extractSchemaId(serializedV2);

        assertNotEquals(schemaIdV1, schemaIdV2, "Different schemas should have different IDs");
        assertTrue(schemaIdV2 > schemaIdV1, "V2 schema ID should be greater than V1");

        // Both should deserialize correctly
        GenericRecord deserializedV1 = (GenericRecord) deserializer.deserialize(topic, serializedV1);
        GenericRecord deserializedV2 = (GenericRecord) deserializer.deserialize(topic, serializedV2);

        assertNotNull(deserializedV1, "V1 record should deserialize");
        assertNotNull(deserializedV2, "V2 record should deserialize");
        assertEquals(25, deserializedV2.get("age"), "V2 age field should be present");

        System.out.println("Schema evolution working: V1 ID=" + schemaIdV1 + ", V2 ID=" + schemaIdV2);
    }

    @Test
    @Order(5)
    @DisplayName("Fetch schema by ID")
    void testFetchSchemaById() throws IOException, RestClientException {
        String topic = TOPIC_PREFIX + "fetch-" + System.currentTimeMillis();

        Schema schema = new Schema.Parser().parse(USER_SCHEMA_V1);
        GenericRecord record = new GenericData.Record(schema);
        record.put("id", 1L);
        record.put("name", "Fetch Test");
        record.put("email", "fetch@example.com");

        byte[] serialized = serializer.serialize(topic, record);
        int schemaId = extractSchemaId(serialized);

        // Fetch schema by ID
        org.apache.avro.Schema fetchedSchema = schemaRegistryClient.getById(schemaId);

        assertNotNull(fetchedSchema, "Fetched schema should not be null");
        assertEquals("User", fetchedSchema.getName(), "Schema name should match");
        assertEquals("com.axonops.test", fetchedSchema.getNamespace(), "Schema namespace should match");

        System.out.println("Successfully fetched schema by ID: " + schemaId);
    }

    @Test
    @Order(6)
    @DisplayName("List subjects")
    void testListSubjects() throws IOException, RestClientException {
        // First register a schema to ensure at least one subject exists
        String topic = TOPIC_PREFIX + "list-" + System.currentTimeMillis();

        Schema schema = new Schema.Parser().parse(USER_SCHEMA_V1);
        GenericRecord record = new GenericData.Record(schema);
        record.put("id", 1L);
        record.put("name", "List Test");
        record.put("email", "list@example.com");

        serializer.serialize(topic, record);

        // List all subjects
        var subjects = schemaRegistryClient.getAllSubjects();

        assertNotNull(subjects, "Subjects list should not be null");
        assertFalse(subjects.isEmpty(), "Subjects list should not be empty");
        assertTrue(subjects.contains(topic + "-value"), "Subject should be in the list");

        System.out.println("Found " + subjects.size() + " subjects");
    }

    @Test
    @Order(7)
    @DisplayName("Schema fingerprint consistency")
    void testSchemaFingerprint() throws IOException, RestClientException {
        String topic1 = TOPIC_PREFIX + "fingerprint1-" + System.currentTimeMillis();
        String topic2 = TOPIC_PREFIX + "fingerprint2-" + System.currentTimeMillis();

        // Same schema registered under different subjects
        Schema schema = new Schema.Parser().parse(USER_SCHEMA_V1);

        GenericRecord record1 = new GenericData.Record(schema);
        record1.put("id", 1L);
        record1.put("name", "Test 1");
        record1.put("email", "test1@example.com");

        GenericRecord record2 = new GenericData.Record(schema);
        record2.put("id", 2L);
        record2.put("name", "Test 2");
        record2.put("email", "test2@example.com");

        byte[] serialized1 = serializer.serialize(topic1, record1);
        byte[] serialized2 = serializer.serialize(topic2, record2);

        int schemaId1 = extractSchemaId(serialized1);
        int schemaId2 = extractSchemaId(serialized2);

        // Schema IDs may differ per subject, but fetching either should return equivalent schemas
        Schema fetched1 = schemaRegistryClient.getById(schemaId1);
        Schema fetched2 = schemaRegistryClient.getById(schemaId2);

        assertEquals(fetched1.toString(), fetched2.toString(),
            "Same schema content should produce identical canonical form");

        System.out.println("Schema fingerprint consistency verified");
    }

    private int extractSchemaId(byte[] serialized) {
        return ((serialized[1] & 0xFF) << 24) |
               ((serialized[2] & 0xFF) << 16) |
               ((serialized[3] & 0xFF) << 8) |
               (serialized[4] & 0xFF);
    }
}
