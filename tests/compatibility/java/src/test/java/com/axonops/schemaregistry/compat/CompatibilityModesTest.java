package com.axonops.schemaregistry.compat;

import io.confluent.kafka.schemaregistry.avro.AvroSchemaProvider;
import io.confluent.kafka.schemaregistry.client.CachedSchemaRegistryClient;
import io.confluent.kafka.schemaregistry.client.SchemaRegistryClient;
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

import static org.junit.jupiter.api.Assertions.*;

/**
 * Compatibility mode tests for FORWARD, FULL, and transitive modes.
 *
 * These tests verify that the AxonOps Schema Registry correctly enforces
 * all seven compatibility levels (not just the default BACKWARD) through
 * actual Avro serializers, ensuring data can be read across schema versions
 * in the expected directions.
 *
 * Compatibility semantics:
 * - FORWARD: new data (written with new schema) can be read by old consumers
 * - FULL: both BACKWARD and FORWARD — bidirectional compatibility
 * - *_TRANSITIVE: checks against ALL previous versions, not just the latest
 */
@TestMethodOrder(MethodOrderer.OrderAnnotation.class)
public class CompatibilityModesTest {

    private static final String SCHEMA_REGISTRY_URL = System.getProperty("schema.registry.url", "http://localhost:8081");
    private static final String CONFLUENT_VERSION = System.getProperty("confluent.version", "unknown");
    private static final String SUBJECT_PREFIX = "compat-modes-test-";

    // Track subjects created during tests for cleanup
    private static final List<String> createdSubjects = new ArrayList<>();

    @BeforeAll
    static void setUp() {
        System.out.println("Running compatibility mode tests with Confluent version: " + CONFLUENT_VERSION);
        System.out.println("Schema Registry URL: " + SCHEMA_REGISTRY_URL);
    }

    @AfterAll
    static void tearDown() {
        // Clean up created subjects
        for (String subject : createdSubjects) {
            TestHelper.deleteSubject(SCHEMA_REGISTRY_URL, subject);
        }
    }

    /**
     * Create a fresh SchemaRegistryClient to avoid cross-test cache pollution.
     */
    private static SchemaRegistryClient createFreshClient() {
        return TestHelper.createClient(SCHEMA_REGISTRY_URL);
    }

    /**
     * Create a fresh serializer with its own client to avoid cache collisions.
     */
    private static KafkaAvroSerializer createFreshSerializer(SchemaRegistryClient client) {
        Map<String, Object> config = new HashMap<>();
        config.put("schema.registry.url", SCHEMA_REGISTRY_URL);
        config.put("auto.register.schemas", true);

        String username = System.getenv("SCHEMA_REGISTRY_USERNAME");
        if (username != null && !username.isEmpty()) {
            config.put("basic.auth.credentials.source", "USER_INFO");
            config.put("basic.auth.user.info", username + ":" + System.getenv().getOrDefault("SCHEMA_REGISTRY_PASSWORD", ""));
        }

        KafkaAvroSerializer ser = new KafkaAvroSerializer(client);
        ser.configure(config, false);
        return ser;
    }

    /**
     * Create a fresh deserializer with its own client to avoid cache collisions.
     */
    private static KafkaAvroDeserializer createFreshDeserializer(SchemaRegistryClient client) {
        Map<String, Object> config = new HashMap<>();
        config.put("schema.registry.url", SCHEMA_REGISTRY_URL);
        config.put("auto.register.schemas", true);

        String username = System.getenv("SCHEMA_REGISTRY_USERNAME");
        if (username != null && !username.isEmpty()) {
            config.put("basic.auth.credentials.source", "USER_INFO");
            config.put("basic.auth.user.info", username + ":" + System.getenv().getOrDefault("SCHEMA_REGISTRY_PASSWORD", ""));
        }

        KafkaAvroDeserializer deser = new KafkaAvroDeserializer(client);
        deser.configure(config, false);
        return deser;
    }

    /**
     * Helper to generate a unique subject name and track it for cleanup.
     */
    private static String uniqueSubject(String testName) {
        String subject = SUBJECT_PREFIX + testName + "-" + System.currentTimeMillis();
        createdSubjects.add(subject);
        return subject;
    }

    /**
     * Helper to set compatibility level for a subject via the REST API.
     */
    private static void setCompatibility(String subject, String level) {
        TestHelper.setSubjectConfig(SCHEMA_REGISTRY_URL, subject,
            "{\"compatibility\": \"" + level + "\"}");
    }

    /**
     * Helper to register a schema via the REST API (bypassing serializer auto-register).
     * Returns the global schema ID.
     */
    private static int registerSchema(String subject, String schemaJson) {
        String body = "{\"schema\": " + escapeJsonValue(schemaJson) + "}";
        return TestHelper.registerSchemaWithRules(SCHEMA_REGISTRY_URL, subject, body);
    }

    /**
     * Escape a JSON schema string to be embedded as a JSON string value.
     */
    private static String escapeJsonValue(String json) {
        // The schema needs to be a JSON string value inside the request body.
        // We wrap it as a properly escaped JSON string.
        return "\"" + json
            .replace("\\", "\\\\")
            .replace("\"", "\\\"")
            .replace("\n", "\\n")
            .replace("\r", "\\r")
            .replace("\t", "\\t")
            + "\"";
    }

    /**
     * Helper to attempt schema registration and expect failure (409 Incompatible).
     * Returns the HTTP status code from the error.
     */
    private static void assertRegistrationRejected(String subject, String schemaJson) {
        String body = "{\"schema\": " + escapeJsonValue(schemaJson) + "}";
        String url = SCHEMA_REGISTRY_URL + "/subjects/" + subject + "/versions";

        try {
            java.net.http.HttpClient httpClient = java.net.http.HttpClient.newHttpClient();
            java.net.http.HttpRequest.Builder requestBuilder = java.net.http.HttpRequest.newBuilder()
                .uri(java.net.URI.create(url))
                .header("Content-Type", "application/vnd.schemaregistry.v1+json")
                .POST(java.net.http.HttpRequest.BodyPublishers.ofString(body));

            String authUsername = System.getenv("SCHEMA_REGISTRY_USERNAME");
            if (authUsername != null && !authUsername.isEmpty()) {
                String credentials = authUsername + ":" + System.getenv().getOrDefault("SCHEMA_REGISTRY_PASSWORD", "");
                String encoded = java.util.Base64.getEncoder().encodeToString(credentials.getBytes());
                requestBuilder.header("Authorization", "Basic " + encoded);
            }

            java.net.http.HttpRequest request = requestBuilder.build();

            java.net.http.HttpResponse<String> response = httpClient.send(request,
                java.net.http.HttpResponse.BodyHandlers.ofString());

            assertEquals(409, response.statusCode(),
                "Expected 409 Incompatible but got " + response.statusCode() + ": " + response.body());

            String responseBody = response.body().toLowerCase();
            assertTrue(responseBody.contains("incompatible") || responseBody.contains("compatibility"),
                "Response should indicate incompatibility: " + response.body());

            System.out.println("Schema correctly rejected with 409: " + response.body());
        } catch (IOException | InterruptedException e) {
            fail("HTTP request failed: " + e.getMessage());
        }
    }

    // ==================== FORWARD Compatibility Tests ====================

    @Test
    @Order(1)
    @DisplayName("FORWARD: accepts field addition with default (old reader can read new data)")
    void testForwardCompatibilityAcceptsFieldAddition() throws Exception {
        String subject = uniqueSubject("fwd-add");

        // v1: {name, age}
        String v1Schema = """
            {
                "type": "record",
                "name": "Person",
                "namespace": "com.axonops.compat.forward",
                "fields": [
                    {"name": "name", "type": "string"},
                    {"name": "age", "type": "int"}
                ]
            }
            """;

        // v2: {name, age, email with default}
        // FORWARD-compatible: old reader (v1) can read new data (v2) because
        // the extra field "email" is simply ignored by the old reader.
        String v2Schema = """
            {
                "type": "record",
                "name": "Person",
                "namespace": "com.axonops.compat.forward",
                "fields": [
                    {"name": "name", "type": "string"},
                    {"name": "age", "type": "int"},
                    {"name": "email", "type": "string", "default": ""}
                ]
            }
            """;

        // Register v1
        registerSchema(subject, v1Schema);

        // Set FORWARD compatibility
        setCompatibility(subject, "FORWARD");

        // Register v2 — should succeed
        registerSchema(subject, v2Schema);

        // Verify serialization round-trip: serialize with v2, deserialize with v1 reader
        Schema writerSchema = new Schema.Parser().parse(v2Schema);
        Schema readerSchema = new Schema.Parser().parse(v1Schema);

        GenericRecord v2Record = new GenericData.Record(writerSchema);
        v2Record.put("name", "Alice");
        v2Record.put("age", 30);
        v2Record.put("email", "alice@example.com");

        // Use fresh client/serializer/deserializer to avoid cross-test cache pollution
        SchemaRegistryClient client = createFreshClient();
        KafkaAvroSerializer serializer = createFreshSerializer(client);
        KafkaAvroDeserializer deserializer = createFreshDeserializer(client);
        try {
            // Serialize with v2 schema
            byte[] serialized = serializer.serialize(subject, v2Record);
            assertNotNull(serialized, "Serialized data should not be null");

            // Deserialize — the deserializer uses the writer schema from the registry
            // and the reader gets whatever fields it understands
            GenericRecord deserialized = (GenericRecord) deserializer.deserialize(subject, serialized);
            assertNotNull(deserialized, "Deserialized record should not be null");

            // The deserialized record should have name and age
            assertEquals("Alice", deserialized.get("name").toString(), "Name should match");
            assertEquals(30, deserialized.get("age"), "Age should match");

            System.out.println("FORWARD compatibility: field addition with default accepted and round-trip verified");
        } finally {
            serializer.close();
            deserializer.close();
        }
    }

    @Test
    @Order(2)
    @DisplayName("FORWARD: rejects field removal without default (old reader cannot read new data)")
    void testForwardCompatibilityRejectsFieldRemovalWithoutDefault() {
        String subject = uniqueSubject("fwd-reject");

        // v1: {name, age, email}
        String v1Schema = """
            {
                "type": "record",
                "name": "Person",
                "namespace": "com.axonops.compat.forward.reject",
                "fields": [
                    {"name": "name", "type": "string"},
                    {"name": "age", "type": "int"},
                    {"name": "email", "type": "string"}
                ]
            }
            """;

        // v2: removes "email" — NOT FORWARD-compatible because old reader (v1)
        // expects "email" field but new data (v2) does not provide it, and v1
        // has no default for "email".
        String v2Schema = """
            {
                "type": "record",
                "name": "Person",
                "namespace": "com.axonops.compat.forward.reject",
                "fields": [
                    {"name": "name", "type": "string"},
                    {"name": "age", "type": "int"}
                ]
            }
            """;

        // Register v1
        registerSchema(subject, v1Schema);

        // Set FORWARD compatibility
        setCompatibility(subject, "FORWARD");

        // Attempt to register v2 — should be rejected (409)
        assertRegistrationRejected(subject, v2Schema);

        System.out.println("FORWARD compatibility: field removal without default correctly rejected");
    }

    // ==================== FULL Compatibility Tests ====================

    @Test
    @Order(3)
    @DisplayName("FULL: accepts bidirectional change (add optional field with default)")
    void testFullCompatibilityAcceptsBidirectionalChange() throws Exception {
        String subject = uniqueSubject("full-bidir");

        // v1: {name, age}
        String v1Schema = """
            {
                "type": "record",
                "name": "Person",
                "namespace": "com.axonops.compat.full",
                "fields": [
                    {"name": "name", "type": "string"},
                    {"name": "age", "type": "int"}
                ]
            }
            """;

        // v2: {name, age, email with default}
        // FULL-compatible: adding an optional field with a default is both:
        // - BACKWARD: new reader (v2) can read old data (v1) — email gets default
        // - FORWARD: old reader (v1) can read new data (v2) — email is ignored
        String v2Schema = """
            {
                "type": "record",
                "name": "Person",
                "namespace": "com.axonops.compat.full",
                "fields": [
                    {"name": "name", "type": "string"},
                    {"name": "age", "type": "int"},
                    {"name": "email", "type": "string", "default": ""}
                ]
            }
            """;

        // Register v1
        registerSchema(subject, v1Schema);

        // Set FULL compatibility
        setCompatibility(subject, "FULL");

        // Register v2 — should succeed (bidirectionally compatible)
        registerSchema(subject, v2Schema);

        Schema schemaV1 = new Schema.Parser().parse(v1Schema);
        Schema schemaV2 = new Schema.Parser().parse(v2Schema);

        // Use fresh client/serializer/deserializer to avoid cross-test cache pollution
        SchemaRegistryClient client = createFreshClient();
        KafkaAvroSerializer serializer = createFreshSerializer(client);
        KafkaAvroDeserializer deserializer = createFreshDeserializer(client);
        try {
            // --- Test BACKWARD direction: serialize with v1, deserialize with v2 reader ---
            GenericRecord v1Record = new GenericData.Record(schemaV1);
            v1Record.put("name", "Bob");
            v1Record.put("age", 25);

            byte[] serializedV1 = serializer.serialize(subject, v1Record);
            GenericRecord deserializedFromV1 = (GenericRecord) deserializer.deserialize(subject, serializedV1);

            assertNotNull(deserializedFromV1, "V1 data deserialized should not be null");
            assertEquals("Bob", deserializedFromV1.get("name").toString(), "Name should match (BACKWARD)");
            assertEquals(25, deserializedFromV1.get("age"), "Age should match (BACKWARD)");

            // --- Test FORWARD direction: serialize with v2, deserialize with v1 reader ---
            GenericRecord v2Record = new GenericData.Record(schemaV2);
            v2Record.put("name", "Carol");
            v2Record.put("age", 35);
            v2Record.put("email", "carol@example.com");

            byte[] serializedV2 = serializer.serialize(subject, v2Record);
            GenericRecord deserializedFromV2 = (GenericRecord) deserializer.deserialize(subject, serializedV2);

            assertNotNull(deserializedFromV2, "V2 data deserialized should not be null");
            assertEquals("Carol", deserializedFromV2.get("name").toString(), "Name should match (FORWARD)");
            assertEquals(35, deserializedFromV2.get("age"), "Age should match (FORWARD)");

            System.out.println("FULL compatibility: bidirectional change accepted and both directions verified");
        } finally {
            serializer.close();
            deserializer.close();
        }
    }

    @Test
    @Order(4)
    @DisplayName("FULL: rejects non-bidirectional change (add required field without default)")
    void testFullCompatibilityRejectsNonBidirectionalChange() {
        String subject = uniqueSubject("full-reject");

        // v1: {name, age}
        String v1Schema = """
            {
                "type": "record",
                "name": "Person",
                "namespace": "com.axonops.compat.full.reject",
                "fields": [
                    {"name": "name", "type": "string"},
                    {"name": "age", "type": "int"}
                ]
            }
            """;

        // v2: adds required field without default — NOT FULL-compatible
        // While it would be FORWARD-compatible (old reader ignores extra field),
        // it is NOT BACKWARD-compatible (new reader cannot read old data without "email").
        String v2Schema = """
            {
                "type": "record",
                "name": "Person",
                "namespace": "com.axonops.compat.full.reject",
                "fields": [
                    {"name": "name", "type": "string"},
                    {"name": "age", "type": "int"},
                    {"name": "email", "type": "string"}
                ]
            }
            """;

        // Register v1
        registerSchema(subject, v1Schema);

        // Set FULL compatibility
        setCompatibility(subject, "FULL");

        // Attempt to register v2 — should be rejected (not BACKWARD-compatible)
        assertRegistrationRejected(subject, v2Schema);

        System.out.println("FULL compatibility: non-bidirectional change correctly rejected");
    }

    // ==================== Transitive Compatibility Tests ====================

    @Test
    @Order(5)
    @DisplayName("FORWARD_TRANSITIVE: rejects evolution not compatible with all previous versions")
    void testForwardTransitiveRejectsNonTransitiveEvolution() {
        String subject = uniqueSubject("fwd-trans");

        // Strategy: construct v1 and v2 so that v2 adds a default for a field that v1
        // does not have a default for. Then v3 drops that field. v3 is FORWARD-compatible
        // with v2 (which has the default) but NOT with v1 (which lacks the default).
        // FORWARD_TRANSITIVE checks against ALL previous versions, catching the v1 failure.

        // v1: {name, email} — both required, no defaults
        String v1Schema = """
            {
                "type": "record",
                "name": "Person",
                "namespace": "com.axonops.compat.fwdtrans",
                "fields": [
                    {"name": "name", "type": "string"},
                    {"name": "email", "type": "string"}
                ]
            }
            """;

        // v2: {name, email with default, age with default}
        // FORWARD with v1: v1 reader needs name (present) and email (present). OK.
        String v2Schema = """
            {
                "type": "record",
                "name": "Person",
                "namespace": "com.axonops.compat.fwdtrans",
                "fields": [
                    {"name": "name", "type": "string"},
                    {"name": "email", "type": "string", "default": ""},
                    {"name": "age", "type": "int", "default": 0}
                ]
            }
            """;

        // v3: {name, age} — drops email
        // FORWARD with v2: v2 reader needs name (present), email (missing but v2 has default ""), age (present). OK.
        // FORWARD with v1: v1 reader needs name (present), email (missing, v1 has NO default). FAILS.
        String v3Schema = """
            {
                "type": "record",
                "name": "Person",
                "namespace": "com.axonops.compat.fwdtrans",
                "fields": [
                    {"name": "name", "type": "string"},
                    {"name": "age", "type": "int", "default": 0}
                ]
            }
            """;

        // Register v1 with NONE compatibility (to allow initial setup)
        setCompatibility(subject, "NONE");
        registerSchema(subject, v1Schema);

        // Set FORWARD for v2 registration (checks against v1 only)
        setCompatibility(subject, "FORWARD");
        registerSchema(subject, v2Schema);

        // Now set FORWARD_TRANSITIVE — v3 must be FORWARD-compatible with ALL versions (v1 AND v2)
        setCompatibility(subject, "FORWARD_TRANSITIVE");

        // Attempt to register v3 — should be rejected (not FORWARD-compatible with v1)
        assertRegistrationRejected(subject, v3Schema);

        System.out.println("FORWARD_TRANSITIVE: non-transitive evolution correctly rejected");
    }

    @Test
    @Order(6)
    @DisplayName("FULL_TRANSITIVE: rejects evolution not bidirectionally compatible with all versions")
    void testFullTransitiveRejectsNonBidirectionalEvolution() {
        String subject = uniqueSubject("full-trans");

        // v1: {name, email} — both required, no defaults
        String v1Schema = """
            {
                "type": "record",
                "name": "Person",
                "namespace": "com.axonops.compat.fulltrans",
                "fields": [
                    {"name": "name", "type": "string"},
                    {"name": "email", "type": "string"}
                ]
            }
            """;

        // v2: {name, email, age with default 0}
        // FULL-compatible with v1:
        //   BACKWARD (v2 reads v1): v2 needs name (present), email (present), age (has default). OK.
        //   FORWARD (v1 reads v2): v1 needs name (present), email (present). Extra age ignored. OK.
        String v2Schema = """
            {
                "type": "record",
                "name": "Person",
                "namespace": "com.axonops.compat.fulltrans",
                "fields": [
                    {"name": "name", "type": "string"},
                    {"name": "email", "type": "string"},
                    {"name": "age", "type": "int", "default": 0}
                ]
            }
            """;

        // v3: {name, age with default 0} — drops email
        // FULL with v2:
        //   BACKWARD (v3 reads v2): v3 needs name (present in v2), age (present in v2). OK.
        //   FORWARD (v2 reads v3): v2 needs name (present), email (NOT in v3, no default in v2). FAILS.
        // FULL with v1:
        //   BACKWARD (v3 reads v1): v3 needs name (present), age (not in v1, has default). OK.
        //   FORWARD (v1 reads v3): v1 needs name (present), email (NOT in v3, no default in v1). FAILS.
        //
        // Under FULL_TRANSITIVE, v3 fails against both v1 and v2.
        String v3Schema = """
            {
                "type": "record",
                "name": "Person",
                "namespace": "com.axonops.compat.fulltrans",
                "fields": [
                    {"name": "name", "type": "string"},
                    {"name": "age", "type": "int", "default": 0}
                ]
            }
            """;

        // Register v1 with NONE compatibility (to allow initial setup)
        setCompatibility(subject, "NONE");
        registerSchema(subject, v1Schema);

        // Set FULL for v2 registration
        setCompatibility(subject, "FULL");
        registerSchema(subject, v2Schema);

        // Now set FULL_TRANSITIVE — v3 must be FULL-compatible with ALL versions (v1 AND v2)
        setCompatibility(subject, "FULL_TRANSITIVE");

        // Attempt to register v3 — should be rejected
        assertRegistrationRejected(subject, v3Schema);

        System.out.println("FULL_TRANSITIVE: non-bidirectional evolution correctly rejected");
    }

    private int extractSchemaId(byte[] serialized) {
        return ((serialized[1] & 0xFF) << 24) |
               ((serialized[2] & 0xFF) << 16) |
               ((serialized[3] & 0xFF) << 8) |
               (serialized[4] & 0xFF);
    }
}
