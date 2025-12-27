package com.axonops.schemaregistry.compat;

import com.google.protobuf.DynamicMessage;
import com.google.protobuf.Descriptors;
import io.confluent.kafka.schemaregistry.avro.AvroSchemaProvider;
import io.confluent.kafka.schemaregistry.client.CachedSchemaRegistryClient;
import io.confluent.kafka.schemaregistry.client.SchemaRegistryClient;
import io.confluent.kafka.schemaregistry.client.rest.exceptions.RestClientException;
import io.confluent.kafka.schemaregistry.json.JsonSchemaProvider;
import io.confluent.kafka.schemaregistry.protobuf.ProtobufSchema;
import io.confluent.kafka.schemaregistry.protobuf.ProtobufSchemaProvider;
import io.confluent.kafka.serializers.protobuf.KafkaProtobufDeserializer;
import io.confluent.kafka.serializers.protobuf.KafkaProtobufSerializer;
import org.junit.jupiter.api.*;

import java.io.IOException;
import java.util.*;
import java.util.concurrent.*;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Protobuf serializer/deserializer compatibility tests.
 *
 * These tests verify that the AxonOps Schema Registry correctly handles
 * Protobuf schemas from different versions of Confluent serializers.
 *
 * Key behaviors tested:
 * - Wire format (magic byte + 4-byte schema ID + message index)
 * - Global schema ID space (same schema = same ID across subjects)
 * - Schema deduplication
 * - Concurrent registration
 * - Config endpoints
 * - Incompatible schema rejection
 */
@TestMethodOrder(MethodOrderer.OrderAnnotation.class)
public class ProtobufCompatibilityTest {

    private static final String SCHEMA_REGISTRY_URL = System.getProperty("schema.registry.url", "http://localhost:8081");
    private static final String CONFLUENT_VERSION = System.getProperty("confluent.version", "unknown");
    private static final String TOPIC_PREFIX = "protobuf-compat-test-";

    private static SchemaRegistryClient schemaRegistryClient;
    private static KafkaProtobufSerializer<DynamicMessage> serializer;
    private static KafkaProtobufDeserializer<DynamicMessage> deserializer;

    private static final String USER_PROTO_V1 = """
        syntax = "proto3";
        package com.axonops.test;

        message User {
            int64 id = 1;
            string name = 2;
            string email = 3;
        }
        """;

    private static final String USER_PROTO_V2 = """
        syntax = "proto3";
        package com.axonops.test;

        message User {
            int64 id = 1;
            string name = 2;
            string email = 3;
            int32 age = 4;
        }
        """;

    @BeforeAll
    static void setUp() {
        System.out.println("Running Protobuf compatibility tests with Confluent version: " + CONFLUENT_VERSION);
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

        serializer = new KafkaProtobufSerializer<>(schemaRegistryClient);
        serializer.configure(serializerConfig, false);

        deserializer = new KafkaProtobufDeserializer<>(schemaRegistryClient);
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
    @DisplayName("Register Protobuf schema via serializer")
    void testSchemaRegistration() throws IOException, RestClientException {
        String topic = TOPIC_PREFIX + System.currentTimeMillis();
        String subject = topic + "-value";

        ProtobufSchema schema = new ProtobufSchema(USER_PROTO_V1);
        Descriptors.Descriptor descriptor = schema.toDescriptor();

        DynamicMessage message = DynamicMessage.newBuilder(descriptor)
            .setField(descriptor.findFieldByName("id"), 1L)
            .setField(descriptor.findFieldByName("name"), "Test User")
            .setField(descriptor.findFieldByName("email"), "test@example.com")
            .build();

        // Serialize - this should auto-register the schema
        byte[] serialized = serializer.serialize(topic, message);

        assertNotNull(serialized, "Serialized data should not be null");
        assertTrue(serialized.length > 5, "Serialized data should have magic byte + schema ID + payload");
        assertEquals(0, serialized[0], "First byte should be magic byte 0");

        // Verify schema was registered
        int schemaId = schemaRegistryClient.getLatestSchemaMetadata(subject).getId();
        assertTrue(schemaId > 0, "Schema ID should be positive");

        System.out.println("Registered Protobuf schema with ID: " + schemaId);
    }

    @Test
    @Order(2)
    @DisplayName("Serialize and deserialize round-trip")
    void testSerializeDeserializeRoundTrip() {
        String topic = TOPIC_PREFIX + "roundtrip-" + System.currentTimeMillis();

        ProtobufSchema schema = new ProtobufSchema(USER_PROTO_V1);
        Descriptors.Descriptor descriptor = schema.toDescriptor();

        DynamicMessage original = DynamicMessage.newBuilder(descriptor)
            .setField(descriptor.findFieldByName("id"), 42L)
            .setField(descriptor.findFieldByName("name"), "Round Trip User")
            .setField(descriptor.findFieldByName("email"), "roundtrip@example.com")
            .build();

        // Serialize
        byte[] serialized = serializer.serialize(topic, original);

        // Deserialize
        DynamicMessage deserialized = deserializer.deserialize(topic, serialized);

        assertNotNull(deserialized, "Deserialized message should not be null");
        assertEquals(42L, deserialized.getField(deserialized.getDescriptorForType().findFieldByName("id")),
            "ID should match");
        assertEquals("Round Trip User",
            deserialized.getField(deserialized.getDescriptorForType().findFieldByName("name")),
            "Name should match");
        assertEquals("roundtrip@example.com",
            deserialized.getField(deserialized.getDescriptorForType().findFieldByName("email")),
            "Email should match");
    }

    @Test
    @Order(3)
    @DisplayName("Schema deduplication - same schema returns same ID within subject")
    void testSchemaDeduplication() throws IOException, RestClientException {
        String topic = TOPIC_PREFIX + "dedup-" + System.currentTimeMillis();

        ProtobufSchema schema = new ProtobufSchema(USER_PROTO_V1);
        Descriptors.Descriptor descriptor = schema.toDescriptor();

        // Create two messages with same schema
        DynamicMessage message1 = DynamicMessage.newBuilder(descriptor)
            .setField(descriptor.findFieldByName("id"), 1L)
            .setField(descriptor.findFieldByName("name"), "User 1")
            .setField(descriptor.findFieldByName("email"), "user1@example.com")
            .build();

        DynamicMessage message2 = DynamicMessage.newBuilder(descriptor)
            .setField(descriptor.findFieldByName("id"), 2L)
            .setField(descriptor.findFieldByName("name"), "User 2")
            .setField(descriptor.findFieldByName("email"), "user2@example.com")
            .build();

        // Serialize both
        byte[] serialized1 = serializer.serialize(topic, message1);
        byte[] serialized2 = serializer.serialize(topic, message2);

        // Extract schema IDs (bytes 1-4, big-endian)
        int schemaId1 = extractSchemaId(serialized1);
        int schemaId2 = extractSchemaId(serialized2);

        assertEquals(schemaId1, schemaId2, "Same schema should return same ID (deduplication)");
        System.out.println("Protobuf schema deduplication working: both messages use schema ID " + schemaId1);
    }

    @Test
    @Order(4)
    @DisplayName("Schema evolution - backward compatible change")
    void testSchemaEvolution() throws IOException, RestClientException {
        String topic = TOPIC_PREFIX + "evolution-" + System.currentTimeMillis();

        // Register v1 schema
        ProtobufSchema schemaV1 = new ProtobufSchema(USER_PROTO_V1);
        Descriptors.Descriptor descriptorV1 = schemaV1.toDescriptor();

        DynamicMessage messageV1 = DynamicMessage.newBuilder(descriptorV1)
            .setField(descriptorV1.findFieldByName("id"), 1L)
            .setField(descriptorV1.findFieldByName("name"), "User V1")
            .setField(descriptorV1.findFieldByName("email"), "v1@example.com")
            .build();

        byte[] serializedV1 = serializer.serialize(topic, messageV1);
        int schemaIdV1 = extractSchemaId(serializedV1);

        // Register v2 schema (backward compatible - adds optional field)
        ProtobufSchema schemaV2 = new ProtobufSchema(USER_PROTO_V2);
        Descriptors.Descriptor descriptorV2 = schemaV2.toDescriptor();

        DynamicMessage messageV2 = DynamicMessage.newBuilder(descriptorV2)
            .setField(descriptorV2.findFieldByName("id"), 2L)
            .setField(descriptorV2.findFieldByName("name"), "User V2")
            .setField(descriptorV2.findFieldByName("email"), "v2@example.com")
            .setField(descriptorV2.findFieldByName("age"), 25)
            .build();

        byte[] serializedV2 = serializer.serialize(topic, messageV2);
        int schemaIdV2 = extractSchemaId(serializedV2);

        // Different schemas should have different IDs
        assertNotEquals(schemaIdV1, schemaIdV2, "Different schemas should have different IDs");

        // Both should deserialize correctly
        DynamicMessage deserializedV1 = deserializer.deserialize(topic, serializedV1);
        DynamicMessage deserializedV2 = deserializer.deserialize(topic, serializedV2);

        assertNotNull(deserializedV1, "V1 message should deserialize");
        assertNotNull(deserializedV2, "V2 message should deserialize");

        System.out.println("Protobuf schema evolution working: V1 ID=" + schemaIdV1 + ", V2 ID=" + schemaIdV2);
    }

    @Test
    @Order(5)
    @DisplayName("Fetch Protobuf schema by ID")
    void testFetchSchemaById() throws IOException, RestClientException {
        String topic = TOPIC_PREFIX + "fetch-" + System.currentTimeMillis();

        ProtobufSchema schema = new ProtobufSchema(USER_PROTO_V1);
        Descriptors.Descriptor descriptor = schema.toDescriptor();

        DynamicMessage message = DynamicMessage.newBuilder(descriptor)
            .setField(descriptor.findFieldByName("id"), 1L)
            .setField(descriptor.findFieldByName("name"), "Fetch Test")
            .setField(descriptor.findFieldByName("email"), "fetch@example.com")
            .build();

        byte[] serialized = serializer.serialize(topic, message);
        int schemaId = extractSchemaId(serialized);

        // Fetch schema by ID
        var fetchedSchema = schemaRegistryClient.getSchemaById(schemaId);

        assertNotNull(fetchedSchema, "Fetched schema should not be null");
        assertEquals("PROTOBUF", fetchedSchema.schemaType(), "Schema type should be PROTOBUF");

        System.out.println("Successfully fetched Protobuf schema by ID: " + schemaId);
    }

    @Test
    @Order(6)
    @DisplayName("List subjects")
    void testListSubjects() throws IOException, RestClientException {
        // First register a schema to ensure at least one subject exists
        String topic = TOPIC_PREFIX + "list-" + System.currentTimeMillis();

        ProtobufSchema schema = new ProtobufSchema(USER_PROTO_V1);
        Descriptors.Descriptor descriptor = schema.toDescriptor();

        DynamicMessage message = DynamicMessage.newBuilder(descriptor)
            .setField(descriptor.findFieldByName("id"), 1L)
            .setField(descriptor.findFieldByName("name"), "List Test")
            .setField(descriptor.findFieldByName("email"), "list@example.com")
            .build();

        serializer.serialize(topic, message);

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

        ProtobufSchema schema = new ProtobufSchema(USER_PROTO_V1);
        Descriptors.Descriptor descriptor = schema.toDescriptor();

        DynamicMessage message1 = DynamicMessage.newBuilder(descriptor)
            .setField(descriptor.findFieldByName("id"), 1L)
            .setField(descriptor.findFieldByName("name"), "Test 1")
            .setField(descriptor.findFieldByName("email"), "test1@example.com")
            .build();

        DynamicMessage message2 = DynamicMessage.newBuilder(descriptor)
            .setField(descriptor.findFieldByName("id"), 2L)
            .setField(descriptor.findFieldByName("name"), "Test 2")
            .setField(descriptor.findFieldByName("email"), "test2@example.com")
            .build();

        byte[] serialized1 = serializer.serialize(topic1, message1);
        byte[] serialized2 = serializer.serialize(topic2, message2);

        int schemaId1 = extractSchemaId(serialized1);
        int schemaId2 = extractSchemaId(serialized2);

        // Same schema content should produce same global ID (Confluent-compatible behavior)
        assertEquals(schemaId1, schemaId2,
            "Same Protobuf schema under different subjects should return same global ID");

        // Structural verification - fetch and compare key properties
        var fetched1 = schemaRegistryClient.getSchemaById(schemaId1);
        var fetched2 = schemaRegistryClient.getSchemaById(schemaId2);

        assertEquals(fetched1.schemaType(), fetched2.schemaType(), "Schema types should match");
        // For Protobuf, compare the canonical form
        assertEquals(fetched1.canonicalString(), fetched2.canonicalString(), "Canonical schemas should match");

        System.out.println("Global schema ID space verified: both subjects use schema ID " + schemaId1);
    }

    @Test
    @Order(8)
    @DisplayName("Concurrent schema registration returns consistent IDs")
    void testConcurrentRegistration() throws Exception {
        String topic = TOPIC_PREFIX + "concurrent-" + System.currentTimeMillis();
        String subject = topic + "-value";

        ProtobufSchema schema = new ProtobufSchema(USER_PROTO_V1);
        Descriptors.Descriptor descriptor = schema.toDescriptor();

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

                KafkaProtobufSerializer<DynamicMessage> threadSerializer = new KafkaProtobufSerializer<>(threadClient);
                threadSerializer.configure(config, false);

                try {
                    readyLatch.countDown();
                    startLatch.await();

                    DynamicMessage message = DynamicMessage.newBuilder(descriptor)
                        .setField(descriptor.findFieldByName("id"), (long) idx)
                        .setField(descriptor.findFieldByName("name"), "Concurrent User " + idx)
                        .setField(descriptor.findFieldByName("email"), "concurrent" + idx + "@example.com")
                        .build();

                    byte[] serialized = threadSerializer.serialize(topic, message);
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
        ProtobufSchema schemaV1 = new ProtobufSchema(USER_PROTO_V1);
        Descriptors.Descriptor descriptorV1 = schemaV1.toDescriptor();

        DynamicMessage messageV1 = DynamicMessage.newBuilder(descriptorV1)
            .setField(descriptorV1.findFieldByName("id"), 1L)
            .setField(descriptorV1.findFieldByName("name"), "User V1")
            .setField(descriptorV1.findFieldByName("email"), "v1@example.com")
            .build();

        serializer.serialize(topic, messageV1);

        // Set subject compatibility to BACKWARD
        schemaRegistryClient.updateCompatibility(subject, "BACKWARD");

        // Verify compatibility was actually set
        String actualCompat = schemaRegistryClient.getCompatibility(subject);
        assertEquals("BACKWARD", actualCompat,
            "Subject compatibility should be BACKWARD after update");

        // Create incompatible schema: change email type from string to int64 (breaking change)
        String incompatibleProto = """
            syntax = "proto3";
            package com.axonops.test;

            message User {
                int64 id = 1;
                string name = 2;
                int64 email = 3;
            }
            """;

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

        KafkaProtobufSerializer<DynamicMessage> freshSerializer = new KafkaProtobufSerializer<>(freshClient);
        freshSerializer.configure(config, false);

        try {
            ProtobufSchema badSchema = new ProtobufSchema(incompatibleProto);
            Descriptors.Descriptor badDescriptor = badSchema.toDescriptor();

            DynamicMessage badMessage = DynamicMessage.newBuilder(badDescriptor)
                .setField(badDescriptor.findFieldByName("id"), 2L)
                .setField(badDescriptor.findFieldByName("name"), "Bad User")
                .setField(badDescriptor.findFieldByName("email"), 12345L)
                .build();

            freshSerializer.serialize(topic, badMessage);
            fail("Expected serialization to fail due to incompatible schema");
        } catch (Exception e) {
            Throwable cause = e;
            boolean foundRestClientException = false;
            while (cause != null) {
                if (cause instanceof RestClientException rce) {
                    foundRestClientException = true;
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

        ProtobufSchema schema = new ProtobufSchema(USER_PROTO_V1);
        Descriptors.Descriptor descriptor = schema.toDescriptor();

        DynamicMessage message = DynamicMessage.newBuilder(descriptor)
            .setField(descriptor.findFieldByName("id"), 1L)
            .setField(descriptor.findFieldByName("name"), "Cache Test")
            .setField(descriptor.findFieldByName("email"), "cache@example.com")
            .build();

        byte[] serialized = serializer.serialize(topic, message);
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
        assertEquals("PROTOBUF", fetchedSchema.schemaType(), "Schema type should be PROTOBUF");

        // Verify deserialization works with fresh client
        KafkaProtobufDeserializer<DynamicMessage> freshDeserializer = new KafkaProtobufDeserializer<>(freshClient);
        Map<String, Object> config = new HashMap<>();
        config.put("schema.registry.url", SCHEMA_REGISTRY_URL);
        freshDeserializer.configure(config, false);

        DynamicMessage deserialized = freshDeserializer.deserialize(topic, serialized);
        assertNotNull(deserialized, "Fresh client should deserialize successfully");
        assertEquals(1L, deserialized.getField(deserialized.getDescriptorForType().findFieldByName("id")),
            "Deserialized ID should match");

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
    @DisplayName("Schema canonicalization - same schema with different formatting returns same ID")
    void testSchemaCanonicalisation() throws IOException, RestClientException {
        // Same Protobuf schema content but with different formatting
        // This tests that the registry canonicalizes schemas before comparison
        //
        // NOTE: Some serializer versions may canonicalize client-side before POSTing,
        // so this test may pass even if server-side canonicalization is broken.
        // For strict server-side canonicalization validation, register via REST API directly.

        // Compact format
        String compactProto = "syntax = \"proto3\"; package com.axonops.canon; message Canonical { int64 id = 1; string value = 2; }";

        // Verbose format (extra whitespace, comments stripped by parser anyway)
        String verboseProto = """
            syntax = "proto3";

            package com.axonops.canon;

            message Canonical {
                int64 id = 1;
                string value = 2;
            }
            """;

        String topic1 = TOPIC_PREFIX + "canon1-" + System.currentTimeMillis();
        String topic2 = TOPIC_PREFIX + "canon2-" + System.currentTimeMillis();

        // Register compact schema
        ProtobufSchema schema1 = new ProtobufSchema(compactProto);
        Descriptors.Descriptor descriptor1 = schema1.toDescriptor();

        DynamicMessage message1 = DynamicMessage.newBuilder(descriptor1)
            .setField(descriptor1.findFieldByName("id"), 1L)
            .setField(descriptor1.findFieldByName("value"), "test1")
            .build();

        byte[] serialized1 = serializer.serialize(topic1, message1);
        int schemaId1 = extractSchemaId(serialized1);

        // Register verbose schema (should be canonicalized to same schema)
        ProtobufSchema schema2 = new ProtobufSchema(verboseProto);
        Descriptors.Descriptor descriptor2 = schema2.toDescriptor();

        DynamicMessage message2 = DynamicMessage.newBuilder(descriptor2)
            .setField(descriptor2.findFieldByName("id"), 2L)
            .setField(descriptor2.findFieldByName("value"), "test2")
            .build();

        byte[] serialized2 = serializer.serialize(topic2, message2);
        int schemaId2 = extractSchemaId(serialized2);

        // Same schema content (after canonicalization) should produce same global ID
        assertEquals(schemaId1, schemaId2,
            "Same Protobuf schema with different formatting should return same global ID (canonicalization)");

        System.out.println("Schema canonicalization verified: both formats use schema ID " + schemaId1);
    }

    private int extractSchemaId(byte[] serialized) {
        return ((serialized[1] & 0xFF) << 24) |
               ((serialized[2] & 0xFF) << 16) |
               ((serialized[3] & 0xFF) << 8) |
               (serialized[4] & 0xFF);
    }
}
