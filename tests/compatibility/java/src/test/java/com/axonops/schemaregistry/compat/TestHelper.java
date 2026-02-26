package com.axonops.schemaregistry.compat;

import io.confluent.kafka.schemaregistry.avro.AvroSchemaProvider;
import io.confluent.kafka.schemaregistry.client.CachedSchemaRegistryClient;
import io.confluent.kafka.schemaregistry.client.SchemaRegistryClient;
import io.confluent.kafka.schemaregistry.json.JsonSchemaProvider;
import io.confluent.kafka.schemaregistry.protobuf.ProtobufSchemaProvider;
import io.confluent.kafka.serializers.KafkaAvroDeserializer;
import io.confluent.kafka.serializers.KafkaAvroSerializer;

import java.io.IOException;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.util.*;

/**
 * HTTP helper for data contract and CSFLE integration tests.
 * Uses java.net.http.HttpClient to register schemas with metadata/ruleSet
 * and interact with the DEK Registry API.
 */
public class TestHelper {

    private static final HttpClient HTTP = HttpClient.newHttpClient();
    private static final String CONTENT_TYPE = "application/vnd.schemaregistry.v1+json";

    /**
     * Register a schema with metadata and ruleSet via REST API.
     * Returns the global schema ID.
     */
    static int registerSchemaWithRules(String registryUrl, String subject, String body) {
        String url = registryUrl + "/subjects/" + subject + "/versions";
        HttpRequest request = HttpRequest.newBuilder()
                .uri(URI.create(url))
                .header("Content-Type", CONTENT_TYPE)
                .POST(HttpRequest.BodyPublishers.ofString(body))
                .build();
        try {
            HttpResponse<String> response = HTTP.send(request, HttpResponse.BodyHandlers.ofString());
            if (response.statusCode() != 200) {
                throw new RuntimeException("Failed to register schema under " + subject +
                        ": HTTP " + response.statusCode() + " - " + response.body());
            }
            // Parse {"id": N} from response
            String responseBody = response.body();
            int idStart = responseBody.indexOf("\"id\"") + 4;
            idStart = responseBody.indexOf(':', idStart) + 1;
            int idEnd = responseBody.indexOf('}', idStart);
            return Integer.parseInt(responseBody.substring(idStart, idEnd).trim());
        } catch (IOException | InterruptedException e) {
            throw new RuntimeException("Failed to register schema: " + e.getMessage(), e);
        }
    }

    /**
     * Set subject-level config (compatibility, defaultRuleSet, overrideRuleSet, defaultMetadata, overrideMetadata).
     */
    static void setSubjectConfig(String registryUrl, String subject, String body) {
        String url = registryUrl + "/config/" + subject;
        HttpRequest request = HttpRequest.newBuilder()
                .uri(URI.create(url))
                .header("Content-Type", CONTENT_TYPE)
                .PUT(HttpRequest.BodyPublishers.ofString(body))
                .build();
        try {
            HttpResponse<String> response = HTTP.send(request, HttpResponse.BodyHandlers.ofString());
            if (response.statusCode() != 200) {
                throw new RuntimeException("Failed to set config for " + subject +
                        ": HTTP " + response.statusCode() + " - " + response.body());
            }
        } catch (IOException | InterruptedException e) {
            throw new RuntimeException("Failed to set config: " + e.getMessage(), e);
        }
    }

    /**
     * Set global config (compatibility, defaultRuleSet, overrideRuleSet).
     */
    static void setGlobalConfig(String registryUrl, String body) {
        String url = registryUrl + "/config";
        HttpRequest request = HttpRequest.newBuilder()
                .uri(URI.create(url))
                .header("Content-Type", CONTENT_TYPE)
                .PUT(HttpRequest.BodyPublishers.ofString(body))
                .build();
        try {
            HttpResponse<String> response = HTTP.send(request, HttpResponse.BodyHandlers.ofString());
            if (response.statusCode() != 200) {
                throw new RuntimeException("Failed to set global config: HTTP " +
                        response.statusCode() + " - " + response.body());
            }
        } catch (IOException | InterruptedException e) {
            throw new RuntimeException("Failed to set global config: " + e.getMessage(), e);
        }
    }

    /**
     * GET a schema version to verify rules/metadata are present.
     */
    static String getSchemaVersion(String registryUrl, String subject, int version) {
        String url = registryUrl + "/subjects/" + subject + "/versions/" + version;
        HttpRequest request = HttpRequest.newBuilder()
                .uri(URI.create(url))
                .header("Accept", CONTENT_TYPE)
                .GET()
                .build();
        try {
            HttpResponse<String> response = HTTP.send(request, HttpResponse.BodyHandlers.ofString());
            if (response.statusCode() != 200) {
                throw new RuntimeException("Failed to get " + subject + " v" + version +
                        ": HTTP " + response.statusCode() + " - " + response.body());
            }
            return response.body();
        } catch (IOException | InterruptedException e) {
            throw new RuntimeException("Failed to get schema version: " + e.getMessage(), e);
        }
    }

    /**
     * GET a KEK from the DEK Registry.
     */
    static String getKEK(String registryUrl, String kekName) {
        String url = registryUrl + "/dek-registry/v1/keks/" + kekName;
        HttpRequest request = HttpRequest.newBuilder()
                .uri(URI.create(url))
                .header("Accept", CONTENT_TYPE)
                .GET()
                .build();
        try {
            HttpResponse<String> response = HTTP.send(request, HttpResponse.BodyHandlers.ofString());
            return response.statusCode() == 200 ? response.body() : null;
        } catch (IOException | InterruptedException e) {
            throw new RuntimeException("Failed to get KEK: " + e.getMessage(), e);
        }
    }

    /**
     * GET a DEK from the DEK Registry.
     */
    static String getDEK(String registryUrl, String kekName, String subject) {
        String url = registryUrl + "/dek-registry/v1/keks/" + kekName + "/deks/" + subject;
        HttpRequest request = HttpRequest.newBuilder()
                .uri(URI.create(url))
                .header("Accept", CONTENT_TYPE)
                .GET()
                .build();
        try {
            HttpResponse<String> response = HTTP.send(request, HttpResponse.BodyHandlers.ofString());
            return response.statusCode() == 200 ? response.body() : null;
        } catch (IOException | InterruptedException e) {
            throw new RuntimeException("Failed to get DEK: " + e.getMessage(), e);
        }
    }

    /**
     * Delete a subject (for test cleanup). Uses permanent=true.
     */
    static void deleteSubject(String registryUrl, String subject) {
        try {
            // Soft delete first
            String url = registryUrl + "/subjects/" + subject;
            HttpRequest softDelete = HttpRequest.newBuilder()
                    .uri(URI.create(url))
                    .DELETE()
                    .build();
            HTTP.send(softDelete, HttpResponse.BodyHandlers.ofString());

            // Then permanent delete
            HttpRequest hardDelete = HttpRequest.newBuilder()
                    .uri(URI.create(url + "?permanent=true"))
                    .DELETE()
                    .build();
            HTTP.send(hardDelete, HttpResponse.BodyHandlers.ofString());
        } catch (IOException | InterruptedException e) {
            // Ignore cleanup failures
        }
    }

    /**
     * Create a SchemaRegistryClient with all providers registered.
     */
    static SchemaRegistryClient createClient(String registryUrl) {
        return new CachedSchemaRegistryClient(
                Collections.singletonList(registryUrl),
                100,
                Arrays.asList(new AvroSchemaProvider(), new JsonSchemaProvider(), new ProtobufSchemaProvider()),
                Collections.emptyMap(),
                Collections.emptyMap()
        );
    }

    /**
     * Create a KafkaAvroSerializer configured for rule execution.
     * auto.register.schemas=false, use.latest.version=true.
     */
    static KafkaAvroSerializer createRuleAwareSerializer(String registryUrl, SchemaRegistryClient client) {
        Map<String, Object> config = new HashMap<>();
        config.put("schema.registry.url", registryUrl);
        config.put("auto.register.schemas", false);
        config.put("use.latest.version", true);

        KafkaAvroSerializer serializer = new KafkaAvroSerializer(client);
        serializer.configure(config, false);
        return serializer;
    }

    /**
     * Create a KafkaAvroSerializer configured for CSFLE rule execution with Vault credentials.
     */
    static KafkaAvroSerializer createCsfleSerializer(String registryUrl, SchemaRegistryClient client, String vaultToken) {
        Map<String, Object> config = new HashMap<>();
        config.put("schema.registry.url", registryUrl);
        config.put("auto.register.schemas", false);
        config.put("use.latest.version", true);
        config.put("rule.executors._default_.param.token.id", vaultToken);

        KafkaAvroSerializer serializer = new KafkaAvroSerializer(client);
        serializer.configure(config, false);
        return serializer;
    }

    /**
     * Create a KafkaAvroDeserializer configured for rule execution.
     */
    static KafkaAvroDeserializer createRuleAwareDeserializer(String registryUrl, SchemaRegistryClient client) {
        Map<String, Object> config = new HashMap<>();
        config.put("schema.registry.url", registryUrl);
        config.put("auto.register.schemas", false);
        config.put("use.latest.version", true);

        KafkaAvroDeserializer deserializer = new KafkaAvroDeserializer(client);
        deserializer.configure(config, false);
        return deserializer;
    }

    /**
     * Create a KafkaAvroDeserializer configured for CSFLE with Vault credentials.
     */
    static KafkaAvroDeserializer createCsfleDeserializer(String registryUrl, SchemaRegistryClient client, String vaultToken) {
        Map<String, Object> config = new HashMap<>();
        config.put("schema.registry.url", registryUrl);
        config.put("auto.register.schemas", false);
        config.put("use.latest.version", true);
        config.put("rule.executors._default_.param.token.id", vaultToken);

        KafkaAvroDeserializer deserializer = new KafkaAvroDeserializer(client);
        deserializer.configure(config, false);
        return deserializer;
    }

    /**
     * Create a KafkaAvroSerializer that auto-registers schemas (for producing initial v1 data).
     */
    static KafkaAvroSerializer createAutoRegisterSerializer(String registryUrl, SchemaRegistryClient client) {
        Map<String, Object> config = new HashMap<>();
        config.put("schema.registry.url", registryUrl);
        config.put("auto.register.schemas", true);

        KafkaAvroSerializer serializer = new KafkaAvroSerializer(client);
        serializer.configure(config, false);
        return serializer;
    }

    /**
     * Create a KafkaAvroDeserializer that pins to a specific schema version via metadata.
     * This is used for DOWNGRADE rule testing: by targeting an older schema version as the
     * reader schema, the deserializer will execute DOWNGRADE migration rules to transform
     * newer (writer) data into the older (reader) shape.
     *
     * <p>The {@code use.latest.with.metadata} config tells the deserializer to find the
     * latest schema version whose metadata properties match the given map, rather than
     * using the absolute latest version.</p>
     *
     * <p>NOTE: The Confluent client expects this config value to be a JSON string
     * (e.g., {@code "{\"major\":\"1\"}"}) not a {@code Map} object.</p>
     */
    static KafkaAvroDeserializer createMetadataPinnedDeserializer(
            String registryUrl, SchemaRegistryClient client, Map<String, String> metadata) {
        // The Confluent client's use.latest.with.metadata config expects a
        // comma-separated key=value string (Kafka ConfigDef MAP format).
        // e.g., Map.of("major", "1") -> "major=1"
        StringBuilder sb = new StringBuilder();
        boolean first = true;
        for (Map.Entry<String, String> entry : metadata.entrySet()) {
            if (!first) sb.append(",");
            sb.append(entry.getKey()).append("=").append(entry.getValue());
            first = false;
        }
        String metadataStr = sb.toString();

        Map<String, Object> config = new HashMap<>();
        config.put("schema.registry.url", registryUrl);
        config.put("auto.register.schemas", false);
        config.put("use.latest.with.metadata", metadataStr);

        KafkaAvroDeserializer deserializer = new KafkaAvroDeserializer(client);
        deserializer.configure(config, false);
        return deserializer;
    }

    /**
     * Register a schema with metadata properties via REST API.
     * Returns the global schema ID. The metadata is included as a top-level
     * "metadata" object in the request body alongside "schema" and "ruleSet".
     */
    static int registerSchemaWithMetadata(String registryUrl, String subject, String body) {
        return registerSchemaWithRules(registryUrl, subject, body);
    }
}
