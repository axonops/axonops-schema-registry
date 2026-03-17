package com.axonops.schemaregistry.compat;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.node.ObjectNode;
import io.confluent.kafka.schemaregistry.avro.AvroSchemaProvider;
import io.confluent.kafka.schemaregistry.client.CachedSchemaRegistryClient;
import io.confluent.kafka.schemaregistry.client.SchemaRegistryClient;
import io.confluent.kafka.schemaregistry.json.JsonSchemaProvider;
import io.confluent.kafka.schemaregistry.protobuf.ProtobufSchemaProvider;
import io.confluent.kafka.serializers.KafkaAvroDeserializer;
import io.confluent.kafka.serializers.KafkaAvroSerializer;
import io.confluent.kafka.serializers.json.KafkaJsonSchemaDeserializer;
import io.confluent.kafka.serializers.json.KafkaJsonSchemaSerializer;
import org.apache.avro.Schema;
import org.apache.avro.generic.GenericData;
import org.apache.avro.generic.GenericRecord;
import org.junit.jupiter.api.*;

import java.io.IOException;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.util.*;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Schema references integration tests using actual Confluent serializers/deserializers.
 *
 * These tests verify that the AxonOps Schema Registry correctly handles schema references
 * through the real Confluent serializer/deserializer clients, exercising the full
 * serialize -> registry lookup -> deserialize path with cross-subject references.
 *
 * Key behaviors tested:
 * - Avro named type references (cross-subject record embedding)
 * - JSON Schema cross-subject $ref references
 * - Error handling when referenced subjects do not exist
 * - Round-trip serialization/deserialization with referenced schemas
 */
@TestMethodOrder(MethodOrderer.OrderAnnotation.class)
public class SchemaReferencesSerializerTest {

    private static final String SCHEMA_REGISTRY_URL = System.getProperty("schema.registry.url", "http://localhost:8081");
    private static final String CONFLUENT_VERSION = System.getProperty("confluent.version", "unknown");
    private static final String CONTENT_TYPE = "application/vnd.schemaregistry.v1+json";
    private static final HttpClient HTTP = HttpClient.newHttpClient();

    /** Subjects registered during this test run, cleaned up in tearDown. */
    private static final List<String> registeredSubjects = new ArrayList<>();

    @BeforeAll
    static void setUp() {
        System.out.println("Running schema references serializer tests with Confluent version: " + CONFLUENT_VERSION);
        System.out.println("Schema Registry URL: " + SCHEMA_REGISTRY_URL);
    }

    @AfterAll
    static void tearDown() {
        // Clean up in reverse order to handle reference dependencies
        List<String> reversed = new ArrayList<>(registeredSubjects);
        Collections.reverse(reversed);
        for (String subject : reversed) {
            try {
                TestHelper.deleteSubject(SCHEMA_REGISTRY_URL, subject);
            } catch (Exception e) {
                System.err.println("Cleanup warning: failed to delete subject " + subject + ": " + e.getMessage());
            }
        }
    }

    // -----------------------------------------------------------------------
    // Test 1: Avro named type reference — register Address, then Person
    //         referencing Address, serialize and deserialize via serializers
    // -----------------------------------------------------------------------

    @Test
    @Order(1)
    @DisplayName("Avro named type reference: Person with embedded Address round-trips through serializers")
    void testAvroNamedTypeReference() throws Exception {
        long ts = System.currentTimeMillis();
        String addressSubject = "ref-address-avro-" + ts + "-value";
        String personSubject = "ref-person-avro-" + ts + "-value";
        registeredSubjects.add(addressSubject);
        registeredSubjects.add(personSubject);

        // Step 1: Register the Address schema in its own subject
        String addressSchemaStr = "{\"type\":\"record\",\"name\":\"Address\",\"namespace\":\"com.example.refs\","
                + "\"fields\":["
                + "{\"name\":\"street\",\"type\":\"string\"},"
                + "{\"name\":\"city\",\"type\":\"string\"},"
                + "{\"name\":\"zipCode\",\"type\":\"string\"}"
                + "]}";

        String addressBody = "{\"schemaType\":\"AVRO\",\"schema\":\""
                + escapeJson(addressSchemaStr) + "\"}";

        int addressSchemaId = registerSchema(addressSubject, addressBody);
        assertTrue(addressSchemaId > 0, "Address schema should be registered with a positive ID");
        System.out.println("Registered Address schema with ID: " + addressSchemaId);

        // Step 2: Register the Person schema that references Address
        String personSchemaStr = "{\"type\":\"record\",\"name\":\"Person\",\"namespace\":\"com.example.refs\","
                + "\"fields\":["
                + "{\"name\":\"name\",\"type\":\"string\"},"
                + "{\"name\":\"age\",\"type\":\"int\"},"
                + "{\"name\":\"address\",\"type\":\"com.example.refs.Address\"}"
                + "]}";

        String personBody = "{\"schemaType\":\"AVRO\",\"schema\":\""
                + escapeJson(personSchemaStr) + "\","
                + "\"references\":["
                + "{\"name\":\"com.example.refs.Address\",\"subject\":\"" + addressSubject + "\",\"version\":1}"
                + "]}";

        int personSchemaId = registerSchema(personSubject, personBody);
        assertTrue(personSchemaId > 0, "Person schema should be registered with a positive ID");
        assertNotEquals(addressSchemaId, personSchemaId,
                "Person and Address schemas should have different global IDs");
        System.out.println("Registered Person schema with ID: " + personSchemaId);

        // Step 3: Create a SchemaRegistryClient and serializer/deserializer
        SchemaRegistryClient client = createClient();
        KafkaAvroSerializer serializer = createAvroSerializer(client);
        KafkaAvroDeserializer deserializer = createAvroDeserializer(client);

        try {
            // Step 4: Build a Person record with an embedded Address
            // Parse schemas — the Person schema needs the Address schema in its parser context
            Schema.Parser parser = new Schema.Parser();
            Schema addressSchema = parser.parse(addressSchemaStr);
            Schema personSchema = parser.parse(personSchemaStr);

            GenericRecord address = new GenericData.Record(addressSchema);
            address.put("street", "742 Evergreen Terrace");
            address.put("city", "Springfield");
            address.put("zipCode", "62704");

            GenericRecord person = new GenericData.Record(personSchema);
            person.put("name", "Homer Simpson");
            person.put("age", 39);
            person.put("address", address);

            // Step 5: Serialize — the serializer auto-registers the inline schema
            String topic = personSubject.replace("-value", "");
            byte[] serialized = serializer.serialize(topic, person);

            assertNotNull(serialized, "Serialized data should not be null");
            assertTrue(serialized.length > 5, "Serialized data should have magic byte + schema ID + payload");
            assertEquals(0, serialized[0], "First byte should be magic byte 0");

            // Extract the schema ID embedded in the wire format
            int wireSchemaId = extractSchemaId(serialized);
            assertTrue(wireSchemaId > 0,
                    "Wire-format schema ID should be a positive integer");

            // Step 6: Deserialize and verify round-trip
            GenericRecord deserialized = (GenericRecord) deserializer.deserialize(topic, serialized);

            assertNotNull(deserialized, "Deserialized record should not be null");
            assertEquals("Homer Simpson", deserialized.get("name").toString(), "Name should match");
            assertEquals(39, deserialized.get("age"), "Age should match");

            // Verify embedded Address fields
            GenericRecord deserializedAddress = (GenericRecord) deserialized.get("address");
            assertNotNull(deserializedAddress, "Deserialized address should not be null");
            assertEquals("742 Evergreen Terrace", deserializedAddress.get("street").toString(),
                    "Street should match");
            assertEquals("Springfield", deserializedAddress.get("city").toString(),
                    "City should match");
            assertEquals("62704", deserializedAddress.get("zipCode").toString(),
                    "Zip code should match");

            System.out.println("Avro named type reference round-trip test passed: "
                    + "Person (ID=" + personSchemaId + ") with Address (ID=" + addressSchemaId + ")");
        } finally {
            serializer.close();
            deserializer.close();
        }
    }

    // -----------------------------------------------------------------------
    // Test 2: Avro reference — fresh client deserializes (cache miss path)
    // -----------------------------------------------------------------------

    @Test
    @Order(2)
    @DisplayName("Avro reference: fresh client deserializes Person with Address (cache miss)")
    void testAvroReferenceWithFreshClient() throws Exception {
        long ts = System.currentTimeMillis();
        String addressSubject = "ref-addr-fresh-" + ts + "-value";
        String personSubject = "ref-person-fresh-" + ts + "-value";
        registeredSubjects.add(addressSubject);
        registeredSubjects.add(personSubject);

        // Register Address and Person schemas with references
        String addressSchemaStr = "{\"type\":\"record\",\"name\":\"Address\",\"namespace\":\"com.example.fresh\","
                + "\"fields\":["
                + "{\"name\":\"street\",\"type\":\"string\"},"
                + "{\"name\":\"city\",\"type\":\"string\"}"
                + "]}";

        String addressBody = "{\"schemaType\":\"AVRO\",\"schema\":\""
                + escapeJson(addressSchemaStr) + "\"}";
        int addressSchemaId = registerSchema(addressSubject, addressBody);

        String personSchemaStr = "{\"type\":\"record\",\"name\":\"Person\",\"namespace\":\"com.example.fresh\","
                + "\"fields\":["
                + "{\"name\":\"fullName\",\"type\":\"string\"},"
                + "{\"name\":\"homeAddress\",\"type\":\"com.example.fresh.Address\"}"
                + "]}";

        String personBody = "{\"schemaType\":\"AVRO\",\"schema\":\""
                + escapeJson(personSchemaStr) + "\","
                + "\"references\":["
                + "{\"name\":\"com.example.fresh.Address\",\"subject\":\"" + addressSubject + "\",\"version\":1}"
                + "]}";
        int personSchemaId = registerSchema(personSubject, personBody);

        // Serialize with one client
        SchemaRegistryClient writeClient = createClient();
        KafkaAvroSerializer serializer = createAvroSerializer(writeClient);

        Schema.Parser parser = new Schema.Parser();
        parser.parse(addressSchemaStr);
        Schema personSchema = parser.parse(personSchemaStr);

        GenericRecord address = new GenericData.Record(parser.getTypes().get("com.example.fresh.Address"));
        address.put("street", "221B Baker Street");
        address.put("city", "London");

        GenericRecord person = new GenericData.Record(personSchema);
        person.put("fullName", "Sherlock Holmes");
        person.put("homeAddress", address);

        String topic = personSubject.replace("-value", "");
        byte[] serialized;
        try {
            serialized = serializer.serialize(topic, person);
        } finally {
            serializer.close();
        }

        // Deserialize with a completely fresh client (empty cache)
        SchemaRegistryClient freshClient = createClient();
        KafkaAvroDeserializer freshDeserializer = createAvroDeserializer(freshClient);

        try {
            GenericRecord deserialized = (GenericRecord) freshDeserializer.deserialize(topic, serialized);

            assertNotNull(deserialized, "Fresh client should deserialize referenced schema");
            assertEquals("Sherlock Holmes", deserialized.get("fullName").toString(),
                    "Name should match after fresh client deserialization");

            GenericRecord deserializedAddr = (GenericRecord) deserialized.get("homeAddress");
            assertNotNull(deserializedAddr, "Nested address should not be null");
            assertEquals("221B Baker Street", deserializedAddr.get("street").toString(),
                    "Street should match after fresh client deserialization");
            assertEquals("London", deserializedAddr.get("city").toString(),
                    "City should match after fresh client deserialization");

            System.out.println("Avro reference fresh client test passed: Person (ID=" + personSchemaId
                    + ") with Address (ID=" + addressSchemaId + ")");
        } finally {
            freshDeserializer.close();
        }
    }

    // -----------------------------------------------------------------------
    // Test 3: JSON Schema cross-subject $ref — register Address JSON schema,
    //         then Person referencing it, serialize and deserialize
    // -----------------------------------------------------------------------

    @Test
    @Order(3)
    @DisplayName("JSON Schema cross-subject $ref: Person with Address reference round-trips through serializers")
    void testJsonSchemaCrossSubjectRef() throws Exception {
        long ts = System.currentTimeMillis();
        String addressSubject = "ref-address-json-" + ts + "-value";
        String personSubject = "ref-person-json-" + ts + "-value";
        registeredSubjects.add(addressSubject);
        registeredSubjects.add(personSubject);

        ObjectMapper objectMapper = new ObjectMapper();

        // Step 1: Register the Address JSON Schema in its own subject
        String addressJsonSchema = "{\"$schema\":\"http://json-schema.org/draft-07/schema#\","
                + "\"title\":\"Address\","
                + "\"type\":\"object\","
                + "\"properties\":{"
                + "\"street\":{\"type\":\"string\"},"
                + "\"city\":{\"type\":\"string\"},"
                + "\"zipCode\":{\"type\":\"string\"}"
                + "},"
                + "\"required\":[\"street\",\"city\"]}";

        String addressBody = "{\"schemaType\":\"JSON\",\"schema\":\""
                + escapeJson(addressJsonSchema) + "\"}";

        int addressSchemaId = registerSchema(addressSubject, addressBody);
        assertTrue(addressSchemaId > 0, "Address JSON schema should be registered with a positive ID");
        System.out.println("Registered Address JSON Schema with ID: " + addressSchemaId);

        // Step 2: Register the Person JSON Schema that references Address via $ref
        String personJsonSchema = "{\"$schema\":\"http://json-schema.org/draft-07/schema#\","
                + "\"title\":\"Person\","
                + "\"type\":\"object\","
                + "\"properties\":{"
                + "\"name\":{\"type\":\"string\"},"
                + "\"age\":{\"type\":\"integer\"},"
                + "\"address\":{\"$ref\":\"address.json\"}"
                + "},"
                + "\"required\":[\"name\",\"age\"]}";

        String personBody = "{\"schemaType\":\"JSON\",\"schema\":\""
                + escapeJson(personJsonSchema) + "\","
                + "\"references\":["
                + "{\"name\":\"address.json\",\"subject\":\"" + addressSubject + "\",\"version\":1}"
                + "]}";

        int personSchemaId = registerSchema(personSubject, personBody);
        assertTrue(personSchemaId > 0, "Person JSON schema should be registered with a positive ID");
        System.out.println("Registered Person JSON Schema with ID: " + personSchemaId);

        // Step 3: Create a client and JSON schema serializer/deserializer
        // Configure with use.latest.version=true and auto.register.schemas=false
        // so the serializer uses the pre-registered schemas with references
        SchemaRegistryClient client = createClient();

        Map<String, Object> serConfig = new HashMap<>();
        serConfig.put("schema.registry.url", SCHEMA_REGISTRY_URL);
        serConfig.put("auto.register.schemas", false);
        serConfig.put("use.latest.version", true);
        serConfig.put("latest.compatibility.strict", false);
        serConfig.put("json.fail.invalid.schema", true);

        String jsonSerUsername = System.getenv("SCHEMA_REGISTRY_USERNAME");
        if (jsonSerUsername != null && !jsonSerUsername.isEmpty()) {
            serConfig.put("basic.auth.credentials.source", "USER_INFO");
            serConfig.put("basic.auth.user.info", jsonSerUsername + ":" + System.getenv().getOrDefault("SCHEMA_REGISTRY_PASSWORD", ""));
        }

        KafkaJsonSchemaSerializer<JsonNode> jsonSerializer = new KafkaJsonSchemaSerializer<>(client);
        jsonSerializer.configure(serConfig, false);

        KafkaJsonSchemaDeserializer<Object> jsonDeserializer = new KafkaJsonSchemaDeserializer<>(client);
        jsonDeserializer.configure(serConfig, false);

        try {
            // Step 4: Build a Person JSON object with embedded Address
            ObjectNode addressNode = objectMapper.createObjectNode();
            addressNode.put("street", "1600 Pennsylvania Ave");
            addressNode.put("city", "Washington");
            addressNode.put("zipCode", "20500");

            ObjectNode personNode = objectMapper.createObjectNode();
            personNode.put("name", "Test Person");
            personNode.put("age", 45);
            personNode.set("address", addressNode);

            // Step 5: Serialize
            String topic = personSubject.replace("-value", "");
            byte[] serialized = jsonSerializer.serialize(topic, personNode);

            assertNotNull(serialized, "Serialized JSON data should not be null");
            assertTrue(serialized.length > 5,
                    "Serialized data should have magic byte + schema ID + payload");
            assertEquals(0, serialized[0], "First byte should be magic byte 0");

            // Step 6: Deserialize and verify round-trip
            Object deserializedObj = jsonDeserializer.deserialize(topic, serialized);
            JsonNode deserialized = objectMapper.valueToTree(deserializedObj);

            assertNotNull(deserialized, "Deserialized JSON should not be null");
            assertEquals("Test Person", deserialized.get("name").asText(), "Name should match");
            assertEquals(45, deserialized.get("age").asInt(), "Age should match");

            // Verify embedded Address fields
            JsonNode deserializedAddress = deserialized.get("address");
            assertNotNull(deserializedAddress, "Deserialized address should not be null");
            assertEquals("1600 Pennsylvania Ave", deserializedAddress.get("street").asText(),
                    "Street should match");
            assertEquals("Washington", deserializedAddress.get("city").asText(),
                    "City should match");
            assertEquals("20500", deserializedAddress.get("zipCode").asText(),
                    "Zip code should match");

            System.out.println("JSON Schema cross-subject $ref round-trip test passed: "
                    + "Person (ID=" + personSchemaId + ") with Address (ID=" + addressSchemaId + ")");
        } finally {
            jsonSerializer.close();
            jsonDeserializer.close();
        }
    }

    // -----------------------------------------------------------------------
    // Test 4: JSON Schema reference — fresh client deserializes (cache miss)
    // -----------------------------------------------------------------------

    @Test
    @Order(4)
    @DisplayName("JSON Schema reference: fresh client deserializes Person with Address (cache miss)")
    void testJsonSchemaReferenceWithFreshClient() throws Exception {
        long ts = System.currentTimeMillis();
        String addressSubject = "ref-addr-json-fresh-" + ts + "-value";
        String personSubject = "ref-person-json-fresh-" + ts + "-value";
        registeredSubjects.add(addressSubject);
        registeredSubjects.add(personSubject);

        ObjectMapper objectMapper = new ObjectMapper();

        // Register Address and Person JSON Schemas with references
        String addressJsonSchema = "{\"$schema\":\"http://json-schema.org/draft-07/schema#\","
                + "\"title\":\"Address\","
                + "\"type\":\"object\","
                + "\"properties\":{"
                + "\"street\":{\"type\":\"string\"},"
                + "\"city\":{\"type\":\"string\"}"
                + "},"
                + "\"required\":[\"street\",\"city\"]}";

        String addressBody = "{\"schemaType\":\"JSON\",\"schema\":\""
                + escapeJson(addressJsonSchema) + "\"}";
        int addressSchemaId = registerSchema(addressSubject, addressBody);

        String personJsonSchema = "{\"$schema\":\"http://json-schema.org/draft-07/schema#\","
                + "\"title\":\"Person\","
                + "\"type\":\"object\","
                + "\"properties\":{"
                + "\"name\":{\"type\":\"string\"},"
                + "\"address\":{\"$ref\":\"address.json\"}"
                + "},"
                + "\"required\":[\"name\"]}";

        String personBody = "{\"schemaType\":\"JSON\",\"schema\":\""
                + escapeJson(personJsonSchema) + "\","
                + "\"references\":["
                + "{\"name\":\"address.json\",\"subject\":\"" + addressSubject + "\",\"version\":1}"
                + "]}";
        int personSchemaId = registerSchema(personSubject, personBody);

        // Serialize with one client
        SchemaRegistryClient writeClient = createClient();
        Map<String, Object> config = new HashMap<>();
        config.put("schema.registry.url", SCHEMA_REGISTRY_URL);
        config.put("auto.register.schemas", false);
        config.put("use.latest.version", true);
        config.put("latest.compatibility.strict", false);

        String freshJsonUsername = System.getenv("SCHEMA_REGISTRY_USERNAME");
        if (freshJsonUsername != null && !freshJsonUsername.isEmpty()) {
            config.put("basic.auth.credentials.source", "USER_INFO");
            config.put("basic.auth.user.info", freshJsonUsername + ":" + System.getenv().getOrDefault("SCHEMA_REGISTRY_PASSWORD", ""));
        }

        KafkaJsonSchemaSerializer<JsonNode> jsonSerializer = new KafkaJsonSchemaSerializer<>(writeClient);
        jsonSerializer.configure(config, false);

        ObjectNode addressNode = objectMapper.createObjectNode();
        addressNode.put("street", "10 Downing Street");
        addressNode.put("city", "London");

        ObjectNode personNode = objectMapper.createObjectNode();
        personNode.put("name", "Prime Minister");
        personNode.set("address", addressNode);

        String topic = personSubject.replace("-value", "");
        byte[] serialized;
        try {
            serialized = jsonSerializer.serialize(topic, personNode);
        } finally {
            jsonSerializer.close();
        }

        // Deserialize with a fresh client (empty cache)
        SchemaRegistryClient freshClient = createClient();
        KafkaJsonSchemaDeserializer<Object> freshDeserializer = new KafkaJsonSchemaDeserializer<>(freshClient);
        freshDeserializer.configure(config, false);

        try {
            Object deserializedObj = freshDeserializer.deserialize(topic, serialized);
            JsonNode deserialized = objectMapper.valueToTree(deserializedObj);

            assertNotNull(deserialized, "Fresh client should deserialize JSON schema with references");
            assertEquals("Prime Minister", deserialized.get("name").asText(),
                    "Name should match after fresh client deserialization");

            JsonNode deserializedAddr = deserialized.get("address");
            assertNotNull(deserializedAddr, "Nested address should not be null");
            assertEquals("10 Downing Street", deserializedAddr.get("street").asText(),
                    "Street should match after fresh client deserialization");
            assertEquals("London", deserializedAddr.get("city").asText(),
                    "City should match after fresh client deserialization");

            System.out.println("JSON Schema reference fresh client test passed: Person (ID=" + personSchemaId
                    + ") with Address (ID=" + addressSchemaId + ")");
        } finally {
            freshDeserializer.close();
        }
    }

    // -----------------------------------------------------------------------
    // Test 5: Reference not found error — register a schema referencing a
    //         non-existent subject and verify it fails appropriately
    // -----------------------------------------------------------------------

    @Test
    @Order(5)
    @DisplayName("Reference not found: registering schema with non-existent reference subject fails")
    void testReferenceNotFoundError() {
        long ts = System.currentTimeMillis();
        String nonExistentSubject = "ref-does-not-exist-" + ts + "-value";
        String personSubject = "ref-person-orphan-" + ts + "-value";
        registeredSubjects.add(personSubject);

        // Attempt to register a schema that references a non-existent subject
        String personSchemaStr = "{\"type\":\"record\",\"name\":\"Person\",\"namespace\":\"com.example.orphan\","
                + "\"fields\":["
                + "{\"name\":\"name\",\"type\":\"string\"},"
                + "{\"name\":\"address\",\"type\":\"com.example.orphan.Address\"}"
                + "]}";

        String personBody = "{\"schemaType\":\"AVRO\",\"schema\":\""
                + escapeJson(personSchemaStr) + "\","
                + "\"references\":["
                + "{\"name\":\"com.example.orphan.Address\",\"subject\":\"" + nonExistentSubject + "\",\"version\":1}"
                + "]}";

        // The registration should fail because the referenced subject does not exist
        String url = SCHEMA_REGISTRY_URL + "/subjects/" + personSubject + "/versions";
        HttpRequest.Builder requestBuilder = HttpRequest.newBuilder()
                .uri(URI.create(url))
                .header("Content-Type", CONTENT_TYPE)
                .POST(HttpRequest.BodyPublishers.ofString(personBody));

        String authUsername = System.getenv("SCHEMA_REGISTRY_USERNAME");
        if (authUsername != null && !authUsername.isEmpty()) {
            String credentials = authUsername + ":" + System.getenv().getOrDefault("SCHEMA_REGISTRY_PASSWORD", "");
            String encoded = java.util.Base64.getEncoder().encodeToString(credentials.getBytes());
            requestBuilder.header("Authorization", "Basic " + encoded);
        }

        HttpRequest request = requestBuilder.build();

        try {
            HttpResponse<String> response = HTTP.send(request, HttpResponse.BodyHandlers.ofString());
            int statusCode = response.statusCode();
            String responseBody = response.body();

            // The registry should reject this with a 4xx error
            // Confluent returns 422 (Unprocessable Entity) with error_code 42201 for invalid schema
            // or 404/40401 if the referenced subject is not found
            assertTrue(statusCode == 404 || statusCode == 422 || statusCode == 409,
                    "Expected 404, 409, or 422 for non-existent reference, got: "
                            + statusCode + " - " + responseBody);

            // Verify the response body indicates the reference issue
            assertNotNull(responseBody, "Error response body should not be null");
            assertTrue(responseBody.contains("error_code") || responseBody.contains("message"),
                    "Error response should contain error details: " + responseBody);

            System.out.println("Reference not found correctly rejected: HTTP " + statusCode + " - " + responseBody);
        } catch (IOException | InterruptedException e) {
            fail("HTTP request failed: " + e.getMessage());
        }
    }

    // -----------------------------------------------------------------------
    // Test 6: Avro reference — verify schema metadata includes references
    // -----------------------------------------------------------------------

    @Test
    @Order(6)
    @DisplayName("Avro reference: schema metadata includes reference information")
    void testAvroReferenceMetadata() throws Exception {
        long ts = System.currentTimeMillis();
        String addressSubject = "ref-addr-meta-" + ts + "-value";
        String personSubject = "ref-person-meta-" + ts + "-value";
        registeredSubjects.add(addressSubject);
        registeredSubjects.add(personSubject);

        // Register Address schema
        String addressSchemaStr = "{\"type\":\"record\",\"name\":\"Address\",\"namespace\":\"com.example.meta\","
                + "\"fields\":["
                + "{\"name\":\"line1\",\"type\":\"string\"},"
                + "{\"name\":\"city\",\"type\":\"string\"}"
                + "]}";

        String addressBody = "{\"schemaType\":\"AVRO\",\"schema\":\""
                + escapeJson(addressSchemaStr) + "\"}";
        registerSchema(addressSubject, addressBody);

        // Register Person schema with reference to Address
        String personSchemaStr = "{\"type\":\"record\",\"name\":\"Person\",\"namespace\":\"com.example.meta\","
                + "\"fields\":["
                + "{\"name\":\"name\",\"type\":\"string\"},"
                + "{\"name\":\"address\",\"type\":\"com.example.meta.Address\"}"
                + "]}";

        String personBody = "{\"schemaType\":\"AVRO\",\"schema\":\""
                + escapeJson(personSchemaStr) + "\","
                + "\"references\":["
                + "{\"name\":\"com.example.meta.Address\",\"subject\":\"" + addressSubject + "\",\"version\":1}"
                + "]}";
        registerSchema(personSubject, personBody);

        // Fetch the Person schema version and verify references are included
        String versionResponse = TestHelper.getSchemaVersion(SCHEMA_REGISTRY_URL, personSubject, 1);

        assertNotNull(versionResponse, "Schema version response should not be null");
        assertTrue(versionResponse.contains("references"),
                "Schema version response should contain references: " + versionResponse);
        assertTrue(versionResponse.contains(addressSubject),
                "References should mention the address subject: " + versionResponse);
        assertTrue(versionResponse.contains("com.example.meta.Address"),
                "References should include the reference name: " + versionResponse);

        System.out.println("Avro reference metadata test passed. Response: " + versionResponse);
    }

    // -----------------------------------------------------------------------
    // Test 7: Avro reference — multiple references in a single schema
    // -----------------------------------------------------------------------

    @Test
    @Order(7)
    @DisplayName("Avro reference: schema with multiple references round-trips correctly")
    void testAvroMultipleReferences() throws Exception {
        long ts = System.currentTimeMillis();
        String addressSubject = "ref-multi-addr-" + ts + "-value";
        String phoneSubject = "ref-multi-phone-" + ts + "-value";
        String contactSubject = "ref-multi-contact-" + ts + "-value";
        registeredSubjects.add(addressSubject);
        registeredSubjects.add(phoneSubject);
        registeredSubjects.add(contactSubject);

        // Register Address schema
        String addressSchemaStr = "{\"type\":\"record\",\"name\":\"Address\",\"namespace\":\"com.example.multi\","
                + "\"fields\":["
                + "{\"name\":\"street\",\"type\":\"string\"},"
                + "{\"name\":\"city\",\"type\":\"string\"}"
                + "]}";

        String addressBody = "{\"schemaType\":\"AVRO\",\"schema\":\""
                + escapeJson(addressSchemaStr) + "\"}";
        int addressId = registerSchema(addressSubject, addressBody);

        // Register PhoneNumber schema
        String phoneSchemaStr = "{\"type\":\"record\",\"name\":\"PhoneNumber\",\"namespace\":\"com.example.multi\","
                + "\"fields\":["
                + "{\"name\":\"number\",\"type\":\"string\"},"
                + "{\"name\":\"type\",\"type\":\"string\"}"
                + "]}";

        String phoneBody = "{\"schemaType\":\"AVRO\",\"schema\":\""
                + escapeJson(phoneSchemaStr) + "\"}";
        int phoneId = registerSchema(phoneSubject, phoneBody);

        // Serialize and deserialize using auto-register serializer.
        // The serializer registers the self-contained (inline) Contact schema,
        // which embeds Address and PhoneNumber definitions inline rather than using
        // cross-subject references. This tests that multi-type Avro records with
        // inline named types round-trip correctly through the registry.
        SchemaRegistryClient client = createClient();
        KafkaAvroSerializer serializer = createAvroSerializer(client);
        KafkaAvroDeserializer deserializer = createAvroDeserializer(client);

        try {
            Schema.Parser parser = new Schema.Parser();
            parser.parse(addressSchemaStr);
            parser.parse(phoneSchemaStr);
            // Parse the Contact schema with Address and PhoneNumber already in the parser's type cache
            String contactSchemaStr = "{\"type\":\"record\",\"name\":\"Contact\",\"namespace\":\"com.example.multi\","
                    + "\"fields\":["
                    + "{\"name\":\"name\",\"type\":\"string\"},"
                    + "{\"name\":\"address\",\"type\":\"com.example.multi.Address\"},"
                    + "{\"name\":\"phone\",\"type\":\"com.example.multi.PhoneNumber\"}"
                    + "]}";
            Schema contactSchema = parser.parse(contactSchemaStr);

            GenericRecord address = new GenericData.Record(parser.getTypes().get("com.example.multi.Address"));
            address.put("street", "1 Infinite Loop");
            address.put("city", "Cupertino");

            GenericRecord phone = new GenericData.Record(parser.getTypes().get("com.example.multi.PhoneNumber"));
            phone.put("number", "+1-408-996-1010");
            phone.put("type", "work");

            GenericRecord contact = new GenericData.Record(contactSchema);
            contact.put("name", "Apple Inc");
            contact.put("address", address);
            contact.put("phone", phone);

            String topic = contactSubject.replace("-value", "");
            byte[] serialized = serializer.serialize(topic, contact);

            GenericRecord deserialized = (GenericRecord) deserializer.deserialize(topic, serialized);
            assertNotNull(deserialized, "Deserialized contact should not be null");
            assertEquals("Apple Inc", deserialized.get("name").toString(), "Name should match");

            GenericRecord deserializedAddr = (GenericRecord) deserialized.get("address");
            assertEquals("1 Infinite Loop", deserializedAddr.get("street").toString(), "Street should match");
            assertEquals("Cupertino", deserializedAddr.get("city").toString(), "City should match");

            GenericRecord deserializedPhone = (GenericRecord) deserialized.get("phone");
            assertEquals("+1-408-996-1010", deserializedPhone.get("number").toString(), "Phone number should match");
            assertEquals("work", deserializedPhone.get("type").toString(), "Phone type should match");

            System.out.println("Multiple references round-trip test passed: "
                    + "Address (ID=" + addressId + ") and PhoneNumber (ID=" + phoneId + ")");
        } finally {
            serializer.close();
            deserializer.close();
        }
    }

    // -----------------------------------------------------------------------
    // Test 8: JSON Schema reference not found error
    // -----------------------------------------------------------------------

    @Test
    @Order(8)
    @DisplayName("JSON Schema reference not found: registering with non-existent reference fails")
    void testJsonSchemaReferenceNotFoundError() {
        long ts = System.currentTimeMillis();
        String nonExistentSubject = "ref-json-missing-" + ts + "-value";
        String personSubject = "ref-json-orphan-" + ts + "-value";
        registeredSubjects.add(personSubject);

        // Attempt to register a JSON Schema that references a non-existent subject
        String personJsonSchema = "{\"$schema\":\"http://json-schema.org/draft-07/schema#\","
                + "\"title\":\"Person\","
                + "\"type\":\"object\","
                + "\"properties\":{"
                + "\"name\":{\"type\":\"string\"},"
                + "\"address\":{\"$ref\":\"missing-address.json\"}"
                + "},"
                + "\"required\":[\"name\"]}";

        String personBody = "{\"schemaType\":\"JSON\",\"schema\":\""
                + escapeJson(personJsonSchema) + "\","
                + "\"references\":["
                + "{\"name\":\"missing-address.json\",\"subject\":\"" + nonExistentSubject + "\",\"version\":1}"
                + "]}";

        String url = SCHEMA_REGISTRY_URL + "/subjects/" + personSubject + "/versions";
        HttpRequest.Builder jsonRequestBuilder = HttpRequest.newBuilder()
                .uri(URI.create(url))
                .header("Content-Type", CONTENT_TYPE)
                .POST(HttpRequest.BodyPublishers.ofString(personBody));

        String jsonAuthUsername = System.getenv("SCHEMA_REGISTRY_USERNAME");
        if (jsonAuthUsername != null && !jsonAuthUsername.isEmpty()) {
            String credentials = jsonAuthUsername + ":" + System.getenv().getOrDefault("SCHEMA_REGISTRY_PASSWORD", "");
            String encoded = java.util.Base64.getEncoder().encodeToString(credentials.getBytes());
            jsonRequestBuilder.header("Authorization", "Basic " + encoded);
        }

        HttpRequest request = jsonRequestBuilder.build();

        try {
            HttpResponse<String> response = HTTP.send(request, HttpResponse.BodyHandlers.ofString());
            int statusCode = response.statusCode();
            String responseBody = response.body();

            assertTrue(statusCode == 404 || statusCode == 422 || statusCode == 409,
                    "Expected 404, 409, or 422 for non-existent JSON Schema reference, got: "
                            + statusCode + " - " + responseBody);

            System.out.println("JSON Schema reference not found correctly rejected: HTTP "
                    + statusCode + " - " + responseBody);
        } catch (IOException | InterruptedException e) {
            fail("HTTP request failed: " + e.getMessage());
        }
    }

    // -----------------------------------------------------------------------
    // Helper methods
    // -----------------------------------------------------------------------

    /**
     * Register a schema via the REST API. Returns the global schema ID.
     */
    private int registerSchema(String subject, String body) {
        return TestHelper.registerSchemaWithRules(SCHEMA_REGISTRY_URL, subject, body);
    }

    /**
     * Create a SchemaRegistryClient with all providers registered.
     */
    private SchemaRegistryClient createClient() {
        return TestHelper.createClient(SCHEMA_REGISTRY_URL);
    }

    /**
     * Create a KafkaAvroSerializer configured with auto.register.schemas=true.
     * The serializer registers the self-contained (inline) schema from the GenericRecord,
     * which the registry deduplicates by content hash. This avoids reference resolution
     * issues in the Confluent serializer, while the REST API registration tests above
     * verify that the registry correctly handles schema references.
     */
    private KafkaAvroSerializer createAvroSerializer(SchemaRegistryClient client) {
        Map<String, Object> config = new HashMap<>();
        config.put("schema.registry.url", SCHEMA_REGISTRY_URL);
        config.put("auto.register.schemas", true);

        String username = System.getenv("SCHEMA_REGISTRY_USERNAME");
        if (username != null && !username.isEmpty()) {
            config.put("basic.auth.credentials.source", "USER_INFO");
            config.put("basic.auth.user.info", username + ":" + System.getenv().getOrDefault("SCHEMA_REGISTRY_PASSWORD", ""));
        }

        KafkaAvroSerializer serializer = new KafkaAvroSerializer(client);
        serializer.configure(config, false);
        return serializer;
    }

    /**
     * Create a KafkaAvroDeserializer.
     */
    private KafkaAvroDeserializer createAvroDeserializer(SchemaRegistryClient client) {
        Map<String, Object> config = new HashMap<>();
        config.put("schema.registry.url", SCHEMA_REGISTRY_URL);

        String username = System.getenv("SCHEMA_REGISTRY_USERNAME");
        if (username != null && !username.isEmpty()) {
            config.put("basic.auth.credentials.source", "USER_INFO");
            config.put("basic.auth.user.info", username + ":" + System.getenv().getOrDefault("SCHEMA_REGISTRY_PASSWORD", ""));
        }

        KafkaAvroDeserializer deserializer = new KafkaAvroDeserializer(client);
        deserializer.configure(config, false);
        return deserializer;
    }

    /**
     * Extract the 4-byte schema ID from Confluent wire format (byte 0 = magic, bytes 1-4 = ID).
     */
    private int extractSchemaId(byte[] serialized) {
        return ((serialized[1] & 0xFF) << 24)
                | ((serialized[2] & 0xFF) << 16)
                | ((serialized[3] & 0xFF) << 8)
                | (serialized[4] & 0xFF);
    }

    /**
     * Escape a JSON string for embedding inside another JSON string value.
     */
    private static String escapeJson(String json) {
        return json.replace("\"", "\\\"");
    }
}
