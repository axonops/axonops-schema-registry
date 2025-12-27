package com.axonops.schemaregistry.compat;

import io.confluent.kafka.schemaregistry.avro.AvroSchemaProvider;
import io.confluent.kafka.schemaregistry.client.CachedSchemaRegistryClient;
import io.confluent.kafka.schemaregistry.client.SchemaRegistryClient;
import io.confluent.kafka.schemaregistry.client.rest.exceptions.RestClientException;
import io.confluent.kafka.schemaregistry.json.JsonSchemaProvider;
import io.confluent.kafka.schemaregistry.protobuf.ProtobufSchemaProvider;
import io.confluent.kafka.serializers.KafkaAvroDeserializer;
import io.confluent.kafka.serializers.KafkaAvroSerializer;
import org.apache.avro.Schema;
import org.apache.avro.generic.GenericData;
import org.apache.avro.generic.GenericRecord;
import org.junit.jupiter.api.*;

import java.io.IOException;
import java.util.*;
import java.util.concurrent.*;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Avro serializer/deserializer compatibility tests.
 *
 * These tests verify that the AxonOps Schema Registry correctly handles
 * Avro schemas from different versions of Confluent serializers.
 *
 * Key behaviors tested:
 * - Wire format (magic byte + 4-byte schema ID)
 * - Global schema ID space (same schema = same ID across subjects)
 * - Schema deduplication
 * - Concurrent registration
 * - Config endpoints
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

        schemaRegistryClient = new CachedSchemaRegistryClient(
            Collections.singletonList(SCHEMA_REGISTRY_URL),
            100,
            Arrays.asList(new AvroSchemaProvider(), new JsonSchemaProvider(), new ProtobufSchemaProvider()),
            Collections.emptyMap(),
            Collections.emptyMap()
        );

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
    @DisplayName("Schema deduplication - same schema returns same ID within subject")
    void testSchemaDeduplication() throws IOException, RestClientException {
        String topic = TOPIC_PREFIX + "dedup-" + System.currentTimeMillis();

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

        // Register v1 schema
        Schema schemaV1 = new Schema.Parser().parse(USER_SCHEMA_V1);
        GenericRecord recordV1 = new GenericData.Record(schemaV1);
        recordV1.put("id", 1L);
        recordV1.put("name", "User V1");
        recordV1.put("email", "v1@example.com");

        byte[] serializedV1 = serializer.serialize(topic, recordV1);
        int schemaIdV1 = extractSchemaId(serializedV1);

        // Register v2 schema (backward compatible - adds optional field with default)
        Schema schemaV2 = new Schema.Parser().parse(USER_SCHEMA_V2);
        GenericRecord recordV2 = new GenericData.Record(schemaV2);
        recordV2.put("id", 2L);
        recordV2.put("name", "User V2");
        recordV2.put("email", "v2@example.com");
        recordV2.put("age", 25);

        byte[] serializedV2 = serializer.serialize(topic, recordV2);
        int schemaIdV2 = extractSchemaId(serializedV2);

        // Different schemas should have different IDs
        assertNotEquals(schemaIdV1, schemaIdV2, "Different schemas should have different IDs");

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
    @DisplayName("Global schema ID space - same schema under different subjects returns same ID")
    void testGlobalSchemaIdSpace() throws IOException, RestClientException {
        String topic1 = TOPIC_PREFIX + "global1-" + System.currentTimeMillis();
        String topic2 = TOPIC_PREFIX + "global2-" + System.currentTimeMillis();

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

        // Same schema content should produce same global ID (Confluent-compatible behavior)
        assertEquals(schemaId1, schemaId2,
            "Same schema under different subjects should return same global ID");

        // Structural verification - compare by fetching and checking key properties
        Schema fetched1 = schemaRegistryClient.getById(schemaId1);
        Schema fetched2 = schemaRegistryClient.getById(schemaId2);

        // Compare structural properties (more reliable than toString())
        assertEquals(fetched1.getName(), fetched2.getName(), "Schema names should match");
        assertEquals(fetched1.getNamespace(), fetched2.getNamespace(), "Schema namespaces should match");
        assertEquals(fetched1.getFields().size(), fetched2.getFields().size(), "Field count should match");

        // Deep structural comparison: field names and types
        for (int i = 0; i < fetched1.getFields().size(); i++) {
            assertEquals(fetched1.getFields().get(i).name(), fetched2.getFields().get(i).name(),
                "Field " + i + " name should match");
            assertEquals(fetched1.getFields().get(i).schema().getType(), fetched2.getFields().get(i).schema().getType(),
                "Field " + i + " type should match");
        }

        System.out.println("Global schema ID space verified: both subjects use schema ID " + schemaId1);
    }

    @Test
    @Order(8)
    @DisplayName("Concurrent schema registration returns consistent IDs")
    void testConcurrentRegistration() throws Exception {
        String topic = TOPIC_PREFIX + "concurrent-" + System.currentTimeMillis();
        String subject = topic + "-value";

        Schema schema = new Schema.Parser().parse(USER_SCHEMA_V1);
        int numThreads = 10;
        ExecutorService executor = Executors.newFixedThreadPool(numThreads);
        CountDownLatch readyLatch = new CountDownLatch(numThreads); // Signals all threads are ready
        CountDownLatch startLatch = new CountDownLatch(1); // Signals threads to start
        List<Future<Integer>> futures = new ArrayList<>();

        // Create tasks that will all try to register the same schema at once
        // Each thread gets its own client and serializer for genuine parallel HTTP POSTs
        for (int i = 0; i < numThreads; i++) {
            final int idx = i;
            futures.add(executor.submit(() -> {
                // Create thread-local client and serializer for true concurrent HTTP requests
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

                KafkaAvroSerializer threadSerializer = new KafkaAvroSerializer(threadClient);
                threadSerializer.configure(config, false);

                try {
                    // Signal this thread is ready and wait for all threads
                    readyLatch.countDown();
                    startLatch.await();

                    GenericRecord record = new GenericData.Record(schema);
                    record.put("id", (long) idx);
                    record.put("name", "Concurrent User " + idx);
                    record.put("email", "concurrent" + idx + "@example.com");

                    byte[] serialized = threadSerializer.serialize(topic, record);
                    return extractSchemaId(serialized);
                } finally {
                    threadSerializer.close();
                    // Close client if it implements AutoCloseable (some versions do)
                    if (threadClient instanceof AutoCloseable) {
                        try {
                            ((AutoCloseable) threadClient).close();
                        } catch (Exception ignored) {
                            // Ignore close exceptions
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
    @Order(9)
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
    @Order(10)
    @DisplayName("Incompatible schema evolution fails with correct error")
    void testIncompatibleSchemaEvolution() throws IOException, RestClientException {
        String topic = TOPIC_PREFIX + "incompat-" + System.currentTimeMillis();
        String subject = topic + "-value";

        // First, register v1 schema
        Schema schemaV1 = new Schema.Parser().parse(USER_SCHEMA_V1);
        GenericRecord recordV1 = new GenericData.Record(schemaV1);
        recordV1.put("id", 1L);
        recordV1.put("name", "User V1");
        recordV1.put("email", "v1@example.com");

        serializer.serialize(topic, recordV1);

        // Set subject compatibility to BACKWARD (default in most setups)
        schemaRegistryClient.updateCompatibility(subject, "BACKWARD");

        // Verify compatibility was actually set (catches "endpoint exists but does nothing" bugs)
        String actualCompat = schemaRegistryClient.getCompatibility(subject);
        assertEquals("BACKWARD", actualCompat,
            "Subject compatibility should be BACKWARD after update");

        // Create incompatible schema: change email type from string to long (breaking change)
        String incompatibleSchema = """
            {
                "type": "record",
                "name": "User",
                "namespace": "com.axonops.test",
                "fields": [
                    {"name": "id", "type": "long"},
                    {"name": "name", "type": "string"},
                    {"name": "email", "type": "long"}
                ]
            }
            """;

        // Create a new serializer that won't use cached schemas
        SchemaRegistryClient freshClient = new CachedSchemaRegistryClient(
            Collections.singletonList(SCHEMA_REGISTRY_URL),
            100,
            Arrays.asList(new AvroSchemaProvider(), new JsonSchemaProvider(), new ProtobufSchemaProvider()),
            Collections.emptyMap(),
            Collections.emptyMap()
        );

        Map<String, Object> config = new HashMap<>();
        config.put("schema.registry.url", SCHEMA_REGISTRY_URL);
        config.put("auto.register.schemas", true);

        KafkaAvroSerializer freshSerializer = new KafkaAvroSerializer(freshClient);
        freshSerializer.configure(config, false);

        try {
            Schema badSchema = new Schema.Parser().parse(incompatibleSchema);
            GenericRecord badRecord = new GenericData.Record(badSchema);
            badRecord.put("id", 2L);
            badRecord.put("name", "Bad User");
            badRecord.put("email", 12345L);

            // This should throw due to incompatibility
            freshSerializer.serialize(topic, badRecord);
            fail("Expected serialization to fail due to incompatible schema");
        } catch (Exception e) {
            // Expected: RestClientException with 409 Conflict or 422 Unprocessable Entity
            // Different SR implementations/versions may use different status codes
            // The serializer wraps the exception, so check the cause chain
            Throwable cause = e;
            boolean foundRestClientException = false;
            while (cause != null) {
                if (cause instanceof RestClientException rce) {
                    foundRestClientException = true;
                    // 409 Conflict or 422 Unprocessable Entity are valid responses
                    assertTrue(rce.getStatus() == 409 || rce.getStatus() == 422,
                        "Expected 409 or 422 for incompatible schema, got: " + rce.getStatus());

                    // Verify this is actually an incompatibility error, not some other 409/422
                    String errorMsg = rce.getMessage() != null ? rce.getMessage().toLowerCase() : "";

                    // Use reflection for getErrorCode() to handle API drift across Confluent versions
                    Integer errorCode = null;
                    try {
                        errorCode = (Integer) rce.getClass().getMethod("getErrorCode").invoke(rce);
                    } catch (Exception ignored) {
                        // getErrorCode() may not exist in all versions
                    }

                    boolean isIncompatError = errorMsg.contains("incompatible")
                        || errorMsg.contains("compatibility")
                        || (errorCode != null && (errorCode == 409 || errorCode == 40901));
                    assertTrue(isIncompatError,
                        "Expected incompatibility error, got: " + rce.getMessage());

                    System.out.println("Incompatible schema correctly rejected with status: " + rce.getStatus());
                    break;
                }
                cause = cause.getCause();
            }
            assertTrue(foundRestClientException,
                "Should have received RestClientException for incompatible schema");
        } finally {
            freshSerializer.close();
            if (freshClient instanceof AutoCloseable) {
                try {
                    ((AutoCloseable) freshClient).close();
                } catch (Exception ignored) {
                }
            }
        }
    }

    @Test
    @Order(11)
    @DisplayName("Fresh client cache miss - schema fetch works after cache bypass")
    void testCacheBehavior() throws IOException, RestClientException {
        String topic = TOPIC_PREFIX + "cache-" + System.currentTimeMillis();

        Schema schema = new Schema.Parser().parse(USER_SCHEMA_V1);
        GenericRecord record = new GenericData.Record(schema);
        record.put("id", 1L);
        record.put("name", "Cache Test");
        record.put("email", "cache@example.com");

        byte[] serialized = serializer.serialize(topic, record);
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
        Schema fetchedSchema = freshClient.getById(schemaId);

        assertNotNull(fetchedSchema, "Fresh client should fetch schema by ID");
        assertEquals("User", fetchedSchema.getName(), "Schema name should match");
        assertEquals("com.axonops.test", fetchedSchema.getNamespace(), "Namespace should match");

        // Verify deserialization works with fresh client
        KafkaAvroDeserializer freshDeserializer = new KafkaAvroDeserializer(freshClient);
        Map<String, Object> config = new HashMap<>();
        config.put("schema.registry.url", SCHEMA_REGISTRY_URL);
        freshDeserializer.configure(config, false);

        GenericRecord deserialized = (GenericRecord) freshDeserializer.deserialize(topic, serialized);
        assertNotNull(deserialized, "Fresh client should deserialize successfully");
        assertEquals(1L, deserialized.get("id"), "Deserialized ID should match");

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
    @Order(12)
    @DisplayName("Schema canonicalization - same schema with different JSON formatting returns same ID")
    void testSchemaCanonicalisation() throws IOException, RestClientException {
        // Same schema content but with different JSON formatting
        // This tests that the registry canonicalizes schemas before comparison
        //
        // NOTE: Some serializer versions may canonicalize client-side before POSTing,
        // so this test may pass even if server-side canonicalization is broken.
        // For strict server-side canonicalization validation, register via REST API directly.

        // Compact format (minimal whitespace)
        String compactSchema = "{\"type\":\"record\",\"name\":\"Canonical\",\"namespace\":\"com.axonops.test\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"},{\"name\":\"value\",\"type\":\"string\"}]}";

        // Verbose format (extra whitespace, reordered properties where valid)
        String verboseSchema = """
            {
                "type" : "record",
                "namespace" : "com.axonops.test",
                "name" : "Canonical",
                "fields" : [
                    { "name" : "id", "type" : "long" },
                    { "name" : "value", "type" : "string" }
                ]
            }
            """;

        String topic1 = TOPIC_PREFIX + "canon1-" + System.currentTimeMillis();
        String topic2 = TOPIC_PREFIX + "canon2-" + System.currentTimeMillis();

        // Register compact schema
        Schema schema1 = new Schema.Parser().parse(compactSchema);
        GenericRecord record1 = new GenericData.Record(schema1);
        record1.put("id", 1L);
        record1.put("value", "test1");

        byte[] serialized1 = serializer.serialize(topic1, record1);
        int schemaId1 = extractSchemaId(serialized1);

        // Register verbose schema (should be canonicalized to same schema)
        Schema schema2 = new Schema.Parser().parse(verboseSchema);
        GenericRecord record2 = new GenericData.Record(schema2);
        record2.put("id", 2L);
        record2.put("value", "test2");

        byte[] serialized2 = serializer.serialize(topic2, record2);
        int schemaId2 = extractSchemaId(serialized2);

        // Same schema content (after canonicalization) should produce same global ID
        assertEquals(schemaId1, schemaId2,
            "Same schema with different JSON formatting should return same global ID (canonicalization)");

        System.out.println("Schema canonicalization verified: both formats use schema ID " + schemaId1);
    }

    private int extractSchemaId(byte[] serialized) {
        return ((serialized[1] & 0xFF) << 24) |
               ((serialized[2] & 0xFF) << 16) |
               ((serialized[3] & 0xFF) << 8) |
               (serialized[4] & 0xFF);
    }
}
