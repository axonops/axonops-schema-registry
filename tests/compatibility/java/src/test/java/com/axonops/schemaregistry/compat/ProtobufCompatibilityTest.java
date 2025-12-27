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

import static org.junit.jupiter.api.Assertions.*;

/**
 * Protobuf serializer/deserializer compatibility tests.
 *
 * These tests verify that the AxonOps Schema Registry correctly handles
 * Protobuf schemas from different versions of Confluent serializers.
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
    @DisplayName("Schema deduplication - same schema returns same ID")
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
    @DisplayName("Schema evolution - adding field")
    void testSchemaEvolution() throws IOException, RestClientException {
        String topic = TOPIC_PREFIX + "evolution-" + System.currentTimeMillis();
        String subject = topic + "-value";

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

        // Register v2 schema (with new field)
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

        assertNotEquals(schemaIdV1, schemaIdV2, "Different schemas should have different IDs");
        assertTrue(schemaIdV2 > schemaIdV1, "V2 schema ID should be greater than V1");

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

    private int extractSchemaId(byte[] serialized) {
        return ((serialized[1] & 0xFF) << 24) |
               ((serialized[2] & 0xFF) << 16) |
               ((serialized[3] & 0xFF) << 8) |
               (serialized[4] & 0xFF);
    }
}
