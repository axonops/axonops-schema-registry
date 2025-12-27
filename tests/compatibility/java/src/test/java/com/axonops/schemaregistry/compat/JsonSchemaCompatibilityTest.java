package com.axonops.schemaregistry.compat;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.node.ObjectNode;
import io.confluent.kafka.schemaregistry.avro.AvroSchemaProvider;
import io.confluent.kafka.schemaregistry.client.CachedSchemaRegistryClient;
import io.confluent.kafka.schemaregistry.client.SchemaRegistryClient;
import io.confluent.kafka.schemaregistry.client.rest.exceptions.RestClientException;
import io.confluent.kafka.schemaregistry.json.JsonSchemaProvider;
import io.confluent.kafka.schemaregistry.protobuf.ProtobufSchemaProvider;
import io.confluent.kafka.serializers.json.KafkaJsonSchemaDeserializer;
import io.confluent.kafka.serializers.json.KafkaJsonSchemaSerializer;
import org.junit.jupiter.api.*;

import java.io.IOException;
import java.util.*;
import java.util.concurrent.*;

import static org.junit.jupiter.api.Assertions.*;

/**
 * JSON Schema serializer/deserializer compatibility tests.
 *
 * These tests verify that the AxonOps Schema Registry correctly handles
 * JSON Schema from different versions of Confluent serializers.
 *
 * Key behaviors tested:
 * - Wire format (magic byte + 4-byte schema ID)
 * - Global schema ID space (same schema = same ID across subjects)
 * - Schema deduplication
 * - Concurrent registration
 * - Config endpoints
 * - Incompatible schema rejection
 */
@TestMethodOrder(MethodOrderer.OrderAnnotation.class)
public class JsonSchemaCompatibilityTest {

    private static final String SCHEMA_REGISTRY_URL = System.getProperty("schema.registry.url", "http://localhost:8081");
    private static final String CONFLUENT_VERSION = System.getProperty("confluent.version", "unknown");
    private static final String TOPIC_PREFIX = "jsonschema-compat-test-";

    private static SchemaRegistryClient schemaRegistryClient;
    private static KafkaJsonSchemaSerializer<JsonNode> serializer;
    private static KafkaJsonSchemaDeserializer<Object> deserializer;
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

        schemaRegistryClient = new CachedSchemaRegistryClient(
            Collections.singletonList(SCHEMA_REGISTRY_URL),
            100,
            Arrays.asList(new AvroSchemaProvider(), new JsonSchemaProvider(), new ProtobufSchemaProvider()),
            Collections.emptyMap(),
            Collections.emptyMap()
        );
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

        // Deserialize - returns Object (LinkedHashMap), convert to JsonNode for assertions
        Object deserializedObj = deserializer.deserialize(topic, serialized);
        JsonNode deserialized = objectMapper.valueToTree(deserializedObj);

        assertNotNull(deserialized, "Deserialized object should not be null");
        assertEquals(42, deserialized.get("id").asInt(), "ID should match");
        assertEquals("Round Trip User", deserialized.get("name").asText(), "Name should match");
        assertEquals("roundtrip@example.com", deserialized.get("email").asText(), "Email should match");
    }

    @Test
    @Order(3)
    @DisplayName("Schema deduplication - same schema returns same ID within subject")
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
        JsonNode deserializedV1 = objectMapper.valueToTree(deserializer.deserialize(topic, serializedV1));
        JsonNode deserializedV2 = objectMapper.valueToTree(deserializer.deserialize(topic, serializedV2));

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
    @DisplayName("List subjects")
    void testListSubjects() throws IOException, RestClientException {
        // First register a schema to ensure at least one subject exists
        String topic = TOPIC_PREFIX + "list-" + System.currentTimeMillis();

        ObjectNode user = objectMapper.createObjectNode();
        user.put("id", 1);
        user.put("name", "List Test");
        user.put("email", "list@example.com");

        serializer.serialize(topic, user);

        // List all subjects
        var subjects = schemaRegistryClient.getAllSubjects();

        assertNotNull(subjects, "Subjects list should not be null");
        assertFalse(subjects.isEmpty(), "Subjects list should not be empty");
        assertTrue(subjects.contains(topic + "-value"), "Subject should be in the list");

        System.out.println("Found " + subjects.size() + " subjects");
    }

    @Test
    @Order(7)
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
        JsonNode deserialized = objectMapper.valueToTree(deserializer.deserialize(topic, serialized));

        assertNotNull(deserialized, "Deserialized object should not be null");
        assertNotNull(deserialized.get("address"), "Address should be present");
        assertEquals("123 Main St", deserialized.get("address").get("street").asText(),
            "Nested street should match");

        System.out.println("Nested object handling working correctly");
    }

    @Test
    @Order(8)
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
        JsonNode deserialized = objectMapper.valueToTree(deserializer.deserialize(topic, serialized));

        assertNotNull(deserialized, "Deserialized object should not be null");
        assertTrue(deserialized.get("tags").isArray(), "Tags should be an array");
        assertEquals(3, deserialized.get("tags").size(), "Array should have 3 elements");

        System.out.println("Array handling working correctly");
    }

    @Test
    @Order(9)
    @DisplayName("Global schema ID space - same schema under different subjects returns same ID")
    void testGlobalSchemaIdSpace() throws IOException, RestClientException {
        String topic1 = TOPIC_PREFIX + "global1-" + System.currentTimeMillis();
        String topic2 = TOPIC_PREFIX + "global2-" + System.currentTimeMillis();

        ObjectNode user1 = objectMapper.createObjectNode();
        user1.put("id", 1);
        user1.put("name", "Test 1");
        user1.put("email", "test1@example.com");

        ObjectNode user2 = objectMapper.createObjectNode();
        user2.put("id", 2);
        user2.put("name", "Test 2");
        user2.put("email", "test2@example.com");

        byte[] serialized1 = serializer.serialize(topic1, user1);
        byte[] serialized2 = serializer.serialize(topic2, user2);

        int schemaId1 = extractSchemaId(serialized1);
        int schemaId2 = extractSchemaId(serialized2);

        // Same schema content should produce same global ID (Confluent-compatible behavior)
        assertEquals(schemaId1, schemaId2,
            "Same JSON schema under different subjects should return same global ID");

        // Structural verification - fetch and compare key properties
        var fetched1 = schemaRegistryClient.getSchemaById(schemaId1);
        var fetched2 = schemaRegistryClient.getSchemaById(schemaId2);

        assertEquals(fetched1.schemaType(), fetched2.schemaType(), "Schema types should match");
        assertEquals(fetched1.canonicalString(), fetched2.canonicalString(), "Canonical schemas should match");

        System.out.println("Global schema ID space verified: both subjects use schema ID " + schemaId1);
    }

    @Test
    @Order(10)
    @DisplayName("Concurrent schema registration returns consistent IDs")
    void testConcurrentRegistration() throws Exception {
        String topic = TOPIC_PREFIX + "concurrent-" + System.currentTimeMillis();
        String subject = topic + "-value";

        int numThreads = 10;
        ExecutorService executor = Executors.newFixedThreadPool(numThreads);
        CountDownLatch readyLatch = new CountDownLatch(numThreads);
        CountDownLatch startLatch = new CountDownLatch(1);
        List<Future<Integer>> futures = new ArrayList<>();

        // Create tasks with thread-local clients for genuine parallel HTTP POSTs
        for (int i = 0; i < numThreads; i++) {
            final int idx = i;
            futures.add(executor.submit(() -> {
                SchemaRegistryClient threadClient = new CachedSchemaRegistryClient(
                    Collections.singletonList(SCHEMA_REGISTRY_URL),
                    100,
                    Arrays.asList(new AvroSchemaProvider(), new JsonSchemaProvider(), new ProtobufSchemaProvider()),
                    Collections.emptyMap(),
                    Collections.emptyMap()
                );

                Map<String, Object> config = new HashMap<>();
                config.put("schema.registry.url", SCHEMA_REGISTRY_URL);
                config.put("auto.register.schemas", true);
                config.put("json.fail.invalid.schema", true);

                KafkaJsonSchemaSerializer<JsonNode> threadSerializer = new KafkaJsonSchemaSerializer<>(threadClient);
                threadSerializer.configure(config, false);

                try {
                    readyLatch.countDown();
                    startLatch.await();

                    ObjectNode user = objectMapper.createObjectNode();
                    user.put("id", idx);
                    user.put("name", "Concurrent User " + idx);
                    user.put("email", "concurrent" + idx + "@example.com");

                    byte[] serialized = threadSerializer.serialize(topic, user);
                    return extractSchemaId(serialized);
                } finally {
                    threadSerializer.close();
                    if (threadClient instanceof AutoCloseable) {
                        try {
                            ((AutoCloseable) threadClient).close();
                        } catch (Exception ignored) {
                        }
                    }
                }
            }));
        }

        // Wait for all threads to be ready, then release them simultaneously
        assertTrue(readyLatch.await(30, TimeUnit.SECONDS),
            "Threads did not become ready in time - concurrency test invalid");
        startLatch.countDown();

        // Collect all schema IDs
        Set<Integer> schemaIds = new HashSet<>();
        for (Future<Integer> future : futures) {
            schemaIds.add(future.get(30, TimeUnit.SECONDS));
        }

        executor.shutdown();
        executor.awaitTermination(30, TimeUnit.SECONDS);

        // All concurrent registrations should return the same ID
        assertEquals(1, schemaIds.size(),
            "All concurrent registrations should return the same schema ID");

        // Verify only one version was created
        var versions = schemaRegistryClient.getAllVersions(subject);
        assertEquals(1, versions.size(),
            "Only one version should exist after concurrent registration");

        // Also verify via getLatestSchemaMetadata
        var latestMeta = schemaRegistryClient.getLatestSchemaMetadata(subject);
        assertEquals(1, latestMeta.getVersion(),
            "Latest version should be 1 after concurrent registration");

        System.out.println("Concurrent registration test passed: " + numThreads +
            " threads all got schema ID " + schemaIds.iterator().next());
    }

    @Test
    @Order(11)
    @DisplayName("Config endpoints - get global compatibility")
    void testConfigEndpoints() throws IOException, RestClientException {
        // Known Confluent compatibility levels
        Set<String> validCompatibilityLevels = Set.of(
            "NONE", "BACKWARD", "FORWARD", "FULL",
            "BACKWARD_TRANSITIVE", "FORWARD_TRANSITIVE", "FULL_TRANSITIVE"
        );

        // Get global compatibility (should have a default)
        String globalCompat = schemaRegistryClient.getCompatibility(null);
        assertNotNull(globalCompat, "Global compatibility should not be null");
        assertTrue(validCompatibilityLevels.contains(globalCompat),
            "Global compatibility should be a valid Confluent level, got: " + globalCompat);

        System.out.println("Global compatibility: " + globalCompat);
    }

    @Test
    @Order(12)
    @DisplayName("Incompatible schema evolution fails with correct error")
    void testIncompatibleSchemaEvolution() throws IOException, RestClientException {
        String topic = TOPIC_PREFIX + "incompat-" + System.currentTimeMillis();
        String subject = topic + "-value";

        // First, register v1 schema by serializing an object
        ObjectNode userV1 = objectMapper.createObjectNode();
        userV1.put("id", 1);
        userV1.put("name", "User V1");
        userV1.put("email", "v1@example.com");

        serializer.serialize(topic, userV1);

        // Set subject compatibility to BACKWARD
        schemaRegistryClient.updateCompatibility(subject, "BACKWARD");

        // Verify compatibility was actually set
        String actualCompat = schemaRegistryClient.getCompatibility(subject);
        assertEquals("BACKWARD", actualCompat,
            "Subject compatibility should be BACKWARD after update");

        // Create incompatible schema: change email type from string to integer (breaking change)
        // We need to force registration of an incompatible schema via the client API
        String incompatibleSchema = """
            {
                "$schema": "http://json-schema.org/draft-07/schema#",
                "title": "User",
                "type": "object",
                "properties": {
                    "id": {"type": "integer"},
                    "name": {"type": "string"},
                    "email": {"type": "integer"}
                },
                "required": ["id", "name", "email"]
            }
            """;

        try {
            io.confluent.kafka.schemaregistry.json.JsonSchema badSchema =
                new io.confluent.kafka.schemaregistry.json.JsonSchema(incompatibleSchema);
            schemaRegistryClient.register(subject, badSchema);
            fail("Expected registration to fail due to incompatible schema");
        } catch (RestClientException rce) {
            assertTrue(rce.getStatus() == 409 || rce.getStatus() == 422,
                "Expected 409 or 422 for incompatible schema, got: " + rce.getStatus());

            String errorMsg = rce.getMessage() != null ? rce.getMessage().toLowerCase() : "";
            Integer errorCode = null;
            try {
                errorCode = (Integer) rce.getClass().getMethod("getErrorCode").invoke(rce);
            } catch (Exception ignored) {
            }

            boolean isIncompatError = errorMsg.contains("incompatible")
                || errorMsg.contains("compatibility")
                || (errorCode != null && (errorCode == 409 || errorCode == 40901));
            assertTrue(isIncompatError,
                "Expected incompatibility error, got: " + rce.getMessage());

            System.out.println("Incompatible schema correctly rejected with status: " + rce.getStatus());
        }
    }

    @Test
    @Order(13)
    @DisplayName("Fresh client cache miss - schema fetch works after cache bypass")
    void testCacheBehavior() throws IOException, RestClientException {
        String topic = TOPIC_PREFIX + "cache-" + System.currentTimeMillis();

        ObjectNode user = objectMapper.createObjectNode();
        user.put("id", 1);
        user.put("name", "Cache Test");
        user.put("email", "cache@example.com");

        byte[] serialized = serializer.serialize(topic, user);
        int schemaId = extractSchemaId(serialized);

        // Create a completely new client (empty cache)
        SchemaRegistryClient freshClient = new CachedSchemaRegistryClient(
            Collections.singletonList(SCHEMA_REGISTRY_URL),
            100,
            Arrays.asList(new AvroSchemaProvider(), new JsonSchemaProvider(), new ProtobufSchemaProvider()),
            Collections.emptyMap(),
            Collections.emptyMap()
        );

        // Fetch schema with fresh client (cache miss, must hit registry)
        var fetchedSchema = freshClient.getSchemaById(schemaId);

        assertNotNull(fetchedSchema, "Fresh client should fetch schema by ID");
        assertEquals("JSON", fetchedSchema.schemaType(), "Schema type should be JSON");

        // Verify deserialization works with fresh client
        KafkaJsonSchemaDeserializer<Object> freshDeserializer = new KafkaJsonSchemaDeserializer<>(freshClient);
        Map<String, Object> config = new HashMap<>();
        config.put("schema.registry.url", SCHEMA_REGISTRY_URL);
        freshDeserializer.configure(config, false);

        Object deserializedObj = freshDeserializer.deserialize(topic, serialized);
        JsonNode deserialized = objectMapper.valueToTree(deserializedObj);
        assertNotNull(deserialized, "Fresh client should deserialize successfully");
        assertEquals(1, deserialized.get("id").asInt(), "Deserialized ID should match");

        freshDeserializer.close();
        if (freshClient instanceof AutoCloseable) {
            try {
                ((AutoCloseable) freshClient).close();
            } catch (Exception ignored) {
            }
        }
        System.out.println("Cache behavior test passed");
    }

    @Test
    @Order(14)
    @DisplayName("Schema canonicalization - same schema with different formatting returns same ID")
    void testSchemaCanonicalisation() throws IOException, RestClientException {
        // Same JSON Schema content but with different formatting
        // This tests that the registry canonicalizes schemas before comparison
        //
        // NOTE: Some serializer versions may canonicalize client-side before POSTing,
        // so this test may pass even if server-side canonicalization is broken.
        // For strict server-side canonicalization validation, register via REST API directly.

        // Compact format (minimal whitespace)
        String compactSchema = "{\"$schema\":\"http://json-schema.org/draft-07/schema#\",\"title\":\"Canonical\",\"type\":\"object\",\"properties\":{\"id\":{\"type\":\"integer\"},\"value\":{\"type\":\"string\"}},\"required\":[\"id\",\"value\"]}";

        // Verbose format (extra whitespace)
        String verboseSchema = """
            {
                "$schema" : "http://json-schema.org/draft-07/schema#",
                "title" : "Canonical",
                "type" : "object",
                "properties" : {
                    "id" : { "type" : "integer" },
                    "value" : { "type" : "string" }
                },
                "required" : [ "id", "value" ]
            }
            """;

        String topic1 = TOPIC_PREFIX + "canon1-" + System.currentTimeMillis();
        String topic2 = TOPIC_PREFIX + "canon2-" + System.currentTimeMillis();
        String subject1 = topic1 + "-value";
        String subject2 = topic2 + "-value";

        // Register compact schema directly
        io.confluent.kafka.schemaregistry.json.JsonSchema schema1 =
            new io.confluent.kafka.schemaregistry.json.JsonSchema(compactSchema);
        int schemaId1 = schemaRegistryClient.register(subject1, schema1);

        // Register verbose schema (should be canonicalized to same schema)
        io.confluent.kafka.schemaregistry.json.JsonSchema schema2 =
            new io.confluent.kafka.schemaregistry.json.JsonSchema(verboseSchema);
        int schemaId2 = schemaRegistryClient.register(subject2, schema2);

        // Same schema content (after canonicalization) should produce same global ID
        assertEquals(schemaId1, schemaId2,
            "Same JSON schema with different formatting should return same global ID (canonicalization)");

        System.out.println("Schema canonicalization verified: both formats use schema ID " + schemaId1);
    }

    private int extractSchemaId(byte[] serialized) {
        return ((serialized[1] & 0xFF) << 24) |
               ((serialized[2] & 0xFF) << 16) |
               ((serialized[3] & 0xFF) << 8) |
               (serialized[4] & 0xFF);
    }
}
