package com.axonops.schemaregistry.compat;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.node.ObjectNode;
import io.confluent.kafka.schemaregistry.client.CachedSchemaRegistryClient;
import io.confluent.kafka.schemaregistry.client.SchemaRegistryClient;
import io.confluent.kafka.schemaregistry.client.rest.exceptions.RestClientException;
import io.confluent.kafka.serializers.json.KafkaJsonSchemaDeserializer;
import io.confluent.kafka.serializers.json.KafkaJsonSchemaSerializer;
import org.junit.jupiter.api.*;

import java.io.IOException;
import java.util.HashMap;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

/**
 * JSON Schema serializer/deserializer compatibility tests.
 *
 * These tests verify that the AxonOps Schema Registry correctly handles
 * JSON Schema from different versions of Confluent serializers.
 */
@TestMethodOrder(MethodOrderer.OrderAnnotation.class)
public class JsonSchemaCompatibilityTest {

    private static final String SCHEMA_REGISTRY_URL = System.getProperty("schema.registry.url", "http://localhost:8081");
    private static final String CONFLUENT_VERSION = System.getProperty("confluent.version", "unknown");
    private static final String TOPIC_PREFIX = "jsonschema-compat-test-";

    private static SchemaRegistryClient schemaRegistryClient;
    private static KafkaJsonSchemaSerializer<JsonNode> serializer;
    private static KafkaJsonSchemaDeserializer<JsonNode> deserializer;
    private static ObjectMapper objectMapper;

    private static final String USER_SCHEMA_V1 = """
        {
            "$schema": "http://json-schema.org/draft-07/schema#",
            "title": "User",
            "type": "object",
            "properties": {
                "id": {"type": "integer"},
                "name": {"type": "string"},
                "email": {"type": "string"}
            },
            "required": ["id", "name", "email"]
        }
        """;

    private static final String USER_SCHEMA_V2 = """
        {
            "$schema": "http://json-schema.org/draft-07/schema#",
            "title": "User",
            "type": "object",
            "properties": {
                "id": {"type": "integer"},
                "name": {"type": "string"},
                "email": {"type": "string"},
                "age": {"type": "integer"}
            },
            "required": ["id", "name", "email"]
        }
        """;

    @BeforeAll
    static void setUp() {
        System.out.println("Running JSON Schema compatibility tests with Confluent version: " + CONFLUENT_VERSION);
        System.out.println("Schema Registry URL: " + SCHEMA_REGISTRY_URL);

        schemaRegistryClient = new CachedSchemaRegistryClient(SCHEMA_REGISTRY_URL, 100);
        objectMapper = new ObjectMapper();

        Map<String, Object> serializerConfig = new HashMap<>();
        serializerConfig.put("schema.registry.url", SCHEMA_REGISTRY_URL);
        serializerConfig.put("auto.register.schemas", true);
        serializerConfig.put("json.fail.invalid.schema", true);

        serializer = new KafkaJsonSchemaSerializer<>(schemaRegistryClient);
        serializer.configure(serializerConfig, false);

        deserializer = new KafkaJsonSchemaDeserializer<>(schemaRegistryClient);
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
    @DisplayName("Register JSON Schema via serializer")
    void testSchemaRegistration() throws IOException, RestClientException {
        String topic = TOPIC_PREFIX + System.currentTimeMillis();
        String subject = topic + "-value";

        ObjectNode user = objectMapper.createObjectNode();
        user.put("id", 1);
        user.put("name", "Test User");
        user.put("email", "test@example.com");

        // Serialize - this should auto-register the schema
        byte[] serialized = serializer.serialize(topic, user);

        assertNotNull(serialized, "Serialized data should not be null");
        assertTrue(serialized.length > 5, "Serialized data should have magic byte + schema ID + payload");
        assertEquals(0, serialized[0], "First byte should be magic byte 0");

        // Verify schema was registered
        int schemaId = schemaRegistryClient.getLatestSchemaMetadata(subject).getId();
        assertTrue(schemaId > 0, "Schema ID should be positive");

        System.out.println("Registered JSON Schema with ID: " + schemaId);
    }

    @Test
    @Order(2)
    @DisplayName("Serialize and deserialize round-trip")
    void testSerializeDeserializeRoundTrip() {
        String topic = TOPIC_PREFIX + "roundtrip-" + System.currentTimeMillis();

        ObjectNode original = objectMapper.createObjectNode();
        original.put("id", 42);
        original.put("name", "Round Trip User");
        original.put("email", "roundtrip@example.com");

        // Serialize
        byte[] serialized = serializer.serialize(topic, original);

        // Deserialize
        JsonNode deserialized = deserializer.deserialize(topic, serialized);

        assertNotNull(deserialized, "Deserialized object should not be null");
        assertEquals(42, deserialized.get("id").asInt(), "ID should match");
        assertEquals("Round Trip User", deserialized.get("name").asText(), "Name should match");
        assertEquals("roundtrip@example.com", deserialized.get("email").asText(), "Email should match");
    }

    @Test
    @Order(3)
    @DisplayName("Schema deduplication - same schema returns same ID")
    void testSchemaDeduplication() throws IOException, RestClientException {
        String topic = TOPIC_PREFIX + "dedup-" + System.currentTimeMillis();

        // Create two objects with same schema
        ObjectNode user1 = objectMapper.createObjectNode();
        user1.put("id", 1);
        user1.put("name", "User 1");
        user1.put("email", "user1@example.com");

        ObjectNode user2 = objectMapper.createObjectNode();
        user2.put("id", 2);
        user2.put("name", "User 2");
        user2.put("email", "user2@example.com");

        // Serialize both
        byte[] serialized1 = serializer.serialize(topic, user1);
        byte[] serialized2 = serializer.serialize(topic, user2);

        // Extract schema IDs (bytes 1-4, big-endian)
        int schemaId1 = extractSchemaId(serialized1);
        int schemaId2 = extractSchemaId(serialized2);

        assertEquals(schemaId1, schemaId2, "Same schema should return same ID (deduplication)");
        System.out.println("JSON Schema deduplication working: both objects use schema ID " + schemaId1);
    }

    @Test
    @Order(4)
    @DisplayName("Schema evolution - adding optional field")
    void testSchemaEvolution() throws IOException, RestClientException {
        String topic = TOPIC_PREFIX + "evolution-" + System.currentTimeMillis();

        // Create v1 object
        ObjectNode userV1 = objectMapper.createObjectNode();
        userV1.put("id", 1);
        userV1.put("name", "User V1");
        userV1.put("email", "v1@example.com");

        byte[] serializedV1 = serializer.serialize(topic, userV1);
        int schemaIdV1 = extractSchemaId(serializedV1);

        // Create v2 object (with new field)
        ObjectNode userV2 = objectMapper.createObjectNode();
        userV2.put("id", 2);
        userV2.put("name", "User V2");
        userV2.put("email", "v2@example.com");
        userV2.put("age", 25);

        byte[] serializedV2 = serializer.serialize(topic, userV2);
        int schemaIdV2 = extractSchemaId(serializedV2);

        // Note: For JSON Schema, adding an optional field may or may not create a new schema
        // depending on the serializer configuration and schema inference
        System.out.println("JSON Schema evolution: V1 ID=" + schemaIdV1 + ", V2 ID=" + schemaIdV2);

        // Both should deserialize correctly
        JsonNode deserializedV1 = deserializer.deserialize(topic, serializedV1);
        JsonNode deserializedV2 = deserializer.deserialize(topic, serializedV2);

        assertNotNull(deserializedV1, "V1 object should deserialize");
        assertNotNull(deserializedV2, "V2 object should deserialize");

        assertEquals(25, deserializedV2.get("age").asInt(), "V2 age field should be present");
    }

    @Test
    @Order(5)
    @DisplayName("Fetch JSON Schema by ID")
    void testFetchSchemaById() throws IOException, RestClientException {
        String topic = TOPIC_PREFIX + "fetch-" + System.currentTimeMillis();

        ObjectNode user = objectMapper.createObjectNode();
        user.put("id", 1);
        user.put("name", "Fetch Test");
        user.put("email", "fetch@example.com");

        byte[] serialized = serializer.serialize(topic, user);
        int schemaId = extractSchemaId(serialized);

        // Fetch schema by ID
        var fetchedSchema = schemaRegistryClient.getSchemaById(schemaId);

        assertNotNull(fetchedSchema, "Fetched schema should not be null");
        assertEquals("JSON", fetchedSchema.schemaType(), "Schema type should be JSON");

        System.out.println("Successfully fetched JSON Schema by ID: " + schemaId);
    }

    @Test
    @Order(6)
    @DisplayName("Nested object handling")
    void testNestedObjects() {
        String topic = TOPIC_PREFIX + "nested-" + System.currentTimeMillis();

        ObjectNode address = objectMapper.createObjectNode();
        address.put("street", "123 Main St");
        address.put("city", "Test City");
        address.put("zip", "12345");

        ObjectNode user = objectMapper.createObjectNode();
        user.put("id", 1);
        user.put("name", "Nested User");
        user.put("email", "nested@example.com");
        user.set("address", address);

        // Serialize
        byte[] serialized = serializer.serialize(topic, user);

        // Deserialize
        JsonNode deserialized = deserializer.deserialize(topic, serialized);

        assertNotNull(deserialized, "Deserialized object should not be null");
        assertNotNull(deserialized.get("address"), "Address should be present");
        assertEquals("123 Main St", deserialized.get("address").get("street").asText(),
            "Nested street should match");

        System.out.println("Nested object handling working correctly");
    }

    @Test
    @Order(7)
    @DisplayName("Array handling")
    void testArrayHandling() {
        String topic = TOPIC_PREFIX + "array-" + System.currentTimeMillis();

        ObjectNode user = objectMapper.createObjectNode();
        user.put("id", 1);
        user.put("name", "Array User");
        user.put("email", "array@example.com");
        user.putArray("tags").add("tag1").add("tag2").add("tag3");

        // Serialize
        byte[] serialized = serializer.serialize(topic, user);

        // Deserialize
        JsonNode deserialized = deserializer.deserialize(topic, serialized);

        assertNotNull(deserialized, "Deserialized object should not be null");
        assertTrue(deserialized.get("tags").isArray(), "Tags should be an array");
        assertEquals(3, deserialized.get("tags").size(), "Array should have 3 elements");

        System.out.println("Array handling working correctly");
    }

    private int extractSchemaId(byte[] serialized) {
        return ((serialized[1] & 0xFF) << 24) |
               ((serialized[2] & 0xFF) << 16) |
               ((serialized[3] & 0xFF) << 8) |
               (serialized[4] & 0xFF);
    }
}
