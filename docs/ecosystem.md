# Schema Registry Ecosystem

The Kafka Schema Registry is a critical component in any event-driven architecture — it ensures that producers and consumers agree on data formats, prevents breaking changes from reaching production, and enables safe schema evolution over time. This document provides an overview of the schema registry ecosystem and explains where AxonOps Schema Registry fits in.

## A Brief History

Confluent created the original Schema Registry as part of [Confluent Platform](https://www.confluent.io/product/confluent-platform/), and in doing so defined the REST API that has become the de facto standard for schema management in the Kafka ecosystem. Every major Kafka serializer, deserializer, and client library speaks this API. Confluent's contribution in establishing this standard has been foundational to the entire community.

Over time, the ecosystem has grown. Several open-source projects now implement the same API, each with different design goals, language choices, and trade-offs. This is a healthy sign of a mature ecosystem — teams can choose the implementation that best fits their operational environment.

## Ecosystem Overview

| Registry | Language | License | Storage | Maintained By |
|----------|----------|---------|---------|---------------|
| [Confluent Schema Registry](https://github.com/confluentinc/schema-registry) | Java | [Confluent Community License](https://www.confluent.io/confluent-community-license/) | Kafka (`_schemas` topic) | Confluent |
| [Confluent Platform](https://www.confluent.io/product/confluent-platform/) (Enterprise) | Java | Commercial | Kafka (`_schemas` topic) | Confluent |
| [Karapace](https://github.com/Aiven-Open/karapace) | Python | Apache 2.0 | Kafka (`_schemas` topic) | Aiven |
| [Apicurio Registry](https://github.com/Apicurio/apicurio-registry) | Java | Apache 2.0 | PostgreSQL, Kafka, SQL | Red Hat |
| **AxonOps Schema Registry** | Go | Apache 2.0 | PostgreSQL, MySQL, Cassandra | [AxonOps](https://axonops.com) |

Each project serves different needs:

- **Confluent Schema Registry** is the reference implementation and the most widely deployed. The Community edition is freely available under the Confluent Community License (not OSI-approved open source). The Enterprise edition adds features like RBAC, data contracts, client-side field-level encryption (CSFLE), multi-tenant contexts, and schema linking, under a commercial license.

- **Karapace** is a Python-based, API-compatible alternative created by Aiven. It uses Kafka for storage (like Confluent) and is a good choice for teams already running Aiven's managed Kafka platform.

- **Apicurio Registry** is a Java-based registry from Red Hat that supports multiple schema formats beyond Kafka (OpenAPI, GraphQL, AsyncAPI) and multiple storage backends. It targets broader API governance use cases.

- **AxonOps Schema Registry** is a Go-based, API-compatible implementation built by [AxonOps](https://axonops.com) with a focus on operational simplicity, enterprise features without licensing costs, and AI-assisted schema management.

## Why We Built AxonOps Schema Registry

At [AxonOps](https://axonops.com), we work with organizations running Apache Kafka and Apache Cassandra at scale. We saw a recurring set of challenges:

1. **Licensing constraints** — Many enterprise features that teams need (RBAC, data contracts, CSFLE, multi-tenant contexts, audit logging) are locked behind commercial licenses. Teams on a budget either go without or build workarounds.

2. **Operational complexity** — Using Kafka as the storage backend for the schema registry creates a circular dependency: you need Kafka running to start the registry, but you need the registry to validate schemas before producing to Kafka. Teams wanted a simpler operational model.

3. **Storage flexibility** — Organizations already running PostgreSQL, MySQL, or Cassandra wanted to use their existing database infrastructure rather than managing a separate Kafka cluster just for schema storage.

4. **AI integration** — As AI-assisted development becomes the norm, we saw an opportunity to make schema management more accessible through natural language — helping developers design better schemas, troubleshoot compatibility issues, and plan migrations without being schema format experts.

AxonOps Schema Registry addresses all of these. It implements the full Confluent Schema Registry REST API (Community and Enterprise), adds additional REST APIs for schema analysis and administration, and includes a built-in MCP server for AI-assisted workflows — all under the Apache 2.0 license, in a single binary with no Kafka dependency for storage.

## Compatibility Philosophy

We believe in ecosystem compatibility. AxonOps Schema Registry is a drop-in replacement — existing Kafka serializers, deserializers, and client libraries work without modification. We run automated compatibility tests against the Confluent Schema Registry to verify behavioral parity, and our BDD test suite includes thousands of scenarios covering every API endpoint, compatibility rule, and error code.

When we extend the API surface with additional endpoints (schema analysis, quality scoring, admin management), we do so alongside the standard API — never by modifying existing endpoints. Existing tools and integrations continue to work exactly as they do with Confluent.

## Choosing a Schema Registry

The right choice depends on your environment and priorities:

- **If you need the reference implementation** and are comfortable with Kafka-based storage, Confluent Schema Registry (Community or Enterprise) is the established choice.

- **If you want enterprise features without licensing costs**, want to use standard databases instead of Kafka for storage, or want AI-assisted schema management, AxonOps Schema Registry is purpose-built for these needs.

- **If you need broad API governance** beyond Kafka (OpenAPI, GraphQL, AsyncAPI), Apicurio Registry's multi-format support may be a better fit.

- **If you're running Aiven's managed Kafka platform**, Karapace integrates naturally with that ecosystem.

All of these projects contribute to a healthy, competitive ecosystem that ultimately benefits the Kafka community.

## AxonOps and Open Source

AxonOps Schema Registry is part of our commitment to the open-source Apache Kafka community. We believe that essential infrastructure components like schema registries should be freely available, fully featured, and not gated by licensing. By releasing under the Apache 2.0 license, we ensure that anyone — whether they use [AxonOps](https://axonops.com) or not — can deploy, modify, and build upon this project.

We built AxonOps Schema Registry to make schema management an integral part of any Kafka deployment, regardless of scale, budget, or vendor. If you find it useful, we would love your feedback, contributions, and feature requests on [GitHub](https://github.com/axonops/axonops-schema-registry).

For organizations that need commercial support, SLAs, or professional services, [AxonOps](https://axonops.com) offers support plans.
