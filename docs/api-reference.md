<!-- This file is auto-generated from api/openapi.yaml -->
<!-- Do not edit manually. Regenerate with: make docs-api -->


# AxonOps Schema Registry
AxonOps Schema Registry is a drop-in replacement for the Confluent Schema Registry,
providing full API compatibility with extensions for enterprise security and
multi-backend storage. It supports **Avro**, **Protobuf**, and **JSON Schema**
formats for schema management.


## Contents

- [AxonOps Schema Registry](#axonops-schema-registry)
  - [Key Concepts](#key-concepts)
  - [Content Types](#content-types)
  - [Error Handling](#error-handling)
  - [Authentication](#authentication)
- [Authentication](#authentication)
- [Default](#default)
  - [Health check](#health-check)
  - [Prometheus metrics](#prometheus-metrics)
- [Schemas](#schemas)
  - [Get supported schema types](#get-supported-schema-types)
  - [List schemas](#list-schemas)
  - [Get schema by global ID](#get-schema-by-global-id)
  - [Get raw schema string by global ID](#get-raw-schema-string-by-global-id)
  - [Get subjects associated with a schema ID](#get-subjects-associated-with-a-schema-id)
  - [Get subject-version pairs for a schema ID](#get-subject-version-pairs-for-a-schema-id)
- [Subjects](#subjects)
  - [List subjects](#list-subjects)
  - [List versions under a subject](#list-versions-under-a-subject)
  - [Register a new schema under a subject](#register-a-new-schema-under-a-subject)
  - [Get a specific version of a subject](#get-a-specific-version-of-a-subject)
  - [Delete a specific version of a subject](#delete-a-specific-version-of-a-subject)
  - [Get raw schema string by subject version](#get-raw-schema-string-by-subject-version)
  - [Get schema IDs that reference this version](#get-schema-ids-that-reference-this-version)
  - [Look up schema under a subject](#look-up-schema-under-a-subject)
  - [Delete a subject](#delete-a-subject)
- [Config](#config)
  - [Get global compatibility configuration](#get-global-compatibility-configuration)
  - [Set global compatibility configuration](#set-global-compatibility-configuration)
  - [Delete global compatibility configuration](#delete-global-compatibility-configuration)
  - [Get subject-level compatibility configuration](#get-subject-level-compatibility-configuration)
  - [Set subject-level compatibility configuration](#set-subject-level-compatibility-configuration)
  - [Delete subject-level compatibility configuration](#delete-subject-level-compatibility-configuration)
- [Mode](#mode)
  - [Get global mode](#get-global-mode)
  - [Set global mode](#set-global-mode)
  - [Get subject-level mode](#get-subject-level-mode)
  - [Set subject-level mode](#set-subject-level-mode)
  - [Delete subject-level mode](#delete-subject-level-mode)
- [Compatibility](#compatibility)
  - [Check compatibility against a specific version](#check-compatibility-against-a-specific-version)
  - [Check compatibility against all versions](#check-compatibility-against-all-versions)
- [Import](#import)
  - [Bulk import schemas](#bulk-import-schemas)
- [Metadata](#metadata)
  - [Get schema registry contexts](#get-schema-registry-contexts)
  - [Get cluster ID](#get-cluster-id)
  - [Get server version](#get-server-version)
- [Account](#account)
  - [Get current user](#get-current-user)
  - [Change current user password](#change-current-user-password)
- [Admin](#admin)
  - [List all users](#list-all-users)
  - [Create a new user](#create-a-new-user)
  - [Get a user by ID](#get-a-user-by-id)
  - [Update a user](#update-a-user)
  - [Delete a user](#delete-a-user)
  - [List API keys](#list-api-keys)
  - [Create a new API key](#create-a-new-api-key)
  - [Get an API key by ID](#get-an-api-key-by-id)
  - [Update an API key](#update-an-api-key)
  - [Delete an API key](#delete-an-api-key)
  - [Revoke an API key](#revoke-an-api-key)
  - [Rotate an API key](#rotate-an-api-key)
  - [List available roles](#list-available-roles)
- [Documentation](#documentation)
  - [Swagger UI](#swagger-ui)
  - [OpenAPI specification](#openapi-specification)
- [Schemas](#schemas)
  - [Reference](#reference)
  - [Metadata](#metadata)
  - [Rule](#rule)
  - [RuleSet](#ruleset)
  - [RegisterSchemaRequest](#registerschemarequest)
  - [RegisterSchemaResponse](#registerschemaresponse)
  - [SchemaByIDResponse](#schemabyidresponse)
  - [SchemaResponse](#schemaresponse)
  - [SubjectVersionResponse](#subjectversionresponse)
  - [LookupSchemaRequest](#lookupschemarequest)
  - [LookupSchemaResponse](#lookupschemaresponse)
  - [SchemaListItem](#schemalistitem)
  - [SubjectVersionPair](#subjectversionpair)
  - [ConfigResponse](#configresponse)
  - [ConfigRequest](#configrequest)
  - [ModeResponse](#moderesponse)
  - [ModeRequest](#moderequest)
  - [CompatibilityCheckRequest](#compatibilitycheckrequest)
  - [CompatibilityCheckResponse](#compatibilitycheckresponse)
  - [ImportSchemasRequest](#importschemasrequest)
  - [ImportSchemaRequest](#importschemarequest)
  - [ImportSchemasResponse](#importschemasresponse)
  - [ImportSchemaResult](#importschemaresult)
  - [ServerClusterIDResponse](#serverclusteridresponse)
  - [ServerVersionResponse](#serverversionresponse)
  - [ErrorResponse](#errorresponse)
  - [CreateUserRequest](#createuserrequest)
  - [UpdateUserRequest](#updateuserrequest)
  - [UserResponse](#userresponse)
  - [UsersListResponse](#userslistresponse)
  - [ChangePasswordRequest](#changepasswordrequest)
  - [CreateAPIKeyRequest](#createapikeyrequest)
  - [UpdateAPIKeyRequest](#updateapikeyrequest)
  - [APIKeyResponse](#apikeyresponse)
  - [CreateAPIKeyResponse](#createapikeyresponse)
  - [APIKeysListResponse](#apikeyslistresponse)
  - [RotateAPIKeyRequest](#rotateapikeyrequest)
  - [RotateAPIKeyResponse](#rotateapikeyresponse)
  - [RoleInfo](#roleinfo)
  - [RolesListResponse](#roleslistresponse)

## Key Concepts

- **Schema**: A versioned definition of a data structure (Avro record, Protobuf message,
  or JSON Schema document). Schemas are stored centrally and assigned a globally unique ID.
- **Subject**: A named scope under which schema versions are registered. A subject typically
  maps to a Kafka topic (e.g. `my-topic-value`).
- **Version**: An incrementing integer assigned each time a new schema is registered under
  a subject.
- **Compatibility**: A policy that controls whether a new schema version is allowed based on
  its relationship to previous versions. Levels include BACKWARD, FORWARD, FULL, and
  their TRANSITIVE variants.
- **Mode**: Controls whether a subject (or the entire registry) accepts writes. Modes
  include READWRITE, READONLY, READONLY_OVERRIDE, and IMPORT.
- **Reference**: A pointer from one schema to another, enabling cross-subject schema
  composition (e.g. Avro named types, Protobuf imports, JSON Schema $ref).
- **Metadata**: Key-value tags and properties attached to a schema for data contract
  management.
- **RuleSet**: Data contract rules (migration and domain rules) attached to a schema.

## Content Types

The primary content type is `application/vnd.schemaregistry.v1+json`.
The registry also accepts `application/json`.

## Error Handling

All errors are returned as JSON objects with `error_code` and `message` fields.
Error codes follow the Confluent Schema Registry convention (e.g. 40401 for subject
not found, 42201 for invalid schema).

## Authentication

When security is enabled, the registry supports HTTP Basic authentication,
API key authentication (via the `X-API-Key` header), and JWT bearer tokens.
Public endpoints (health check, metrics, documentation) do not require authentication.

Base URLs:

* [http://localhost:8081](http://localhost:8081)

Web: [AxonOps](https://github.com/axonops/axonops-schema-registry) 
License: [Apache 2.0](https://www.apache.org/licenses/LICENSE-2.0)

# Authentication

- HTTP Authentication, scheme: basic HTTP Basic authentication. The username and password are verified against the registry's user database.

* API Key (apiKey)
    - Parameter Name: **X-API-Key**, in: header. API key authentication. Pass the full API key string in the `X-API-Key` header. API keys are created and managed via the admin endpoints.

- HTTP Authentication, scheme: bearer JWT bearer token authentication. Tokens may be issued by the configured OIDC provider or the registry itself.

# Default

## Health check


> Code samples

```shell
# You can also use wget
curl -X GET http://localhost:8081/ \
  -H 'Accept: application/json'

```

`GET /`

Returns an empty JSON object to indicate the service is healthy and accepting requests. This endpoint does not require authentication.

> Example responses

> 200 Response

```json
{}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The service is healthy.|Inline|

### Response Schema

Status Code **200**

*An empty JSON object.*

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|

> **Success:** 
This operation does not require authentication


## Prometheus metrics


> Code samples

```shell
# You can also use wget
curl -X GET http://localhost:8081/metrics \
  -H 'Accept: text/plain'

```

`GET /metrics`

Returns Prometheus-formatted metrics for the registry. This endpoint does not require authentication. Metrics include request counts, latencies, cache statistics, and storage health indicators.

> Example responses

> 200 Response

```
"string"
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|Prometheus metrics in text exposition format.|string|

> **Success:** 
This operation does not require authentication


# Schemas

Operations for retrieving schemas by their globally unique ID, listing all schemas, and querying supported schema types. Every schema registered in the registry receives a unique integer ID that persists across subjects.

## Get supported schema types


> Code samples

```shell
# You can also use wget
curl -X GET http://localhost:8081/schemas/types \
  -H 'Accept: application/vnd.schemaregistry.v1+json'

```

`GET /schemas/types`

Returns the list of schema types supported by this registry. The response is an array of strings. Currently supported types are AVRO, PROTOBUF, and JSON.

> Example responses

> 200 Response

```json
[
  "AVRO",
  "PROTOBUF",
  "JSON"
]
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|A list of supported schema type strings.|Inline|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

### Response Schema

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


## List schemas


> Code samples

```shell
# You can also use wget
curl -X GET http://localhost:8081/schemas \
  -H 'Accept: application/vnd.schemaregistry.v1+json'

```

`GET /schemas`

Returns a list of all schemas registered in the registry. Results MAY be filtered by subject prefix, soft-deleted status, and whether to return only the latest version per subject. Pagination is supported via offset and limit query parameters.

### Parameters

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|subjectPrefix|query|string|false|Filter results to schemas whose subject name starts with this prefix. If omitted, schemas across all subjects are returned.|
|deleted|query|boolean|false|When set to `true`, soft-deleted schemas are included in the results.|
|latestOnly|query|boolean|false|When set to `true`, only the latest version of each subject is returned.|
|offset|query|integer|false|The number of results to skip for pagination.|
|limit|query|integer|false|The maximum number of results to return. If omitted or set to 0, all results are returned.|

> Example responses

> 200 Response

```json
[
  {
    "subject": "string",
    "version": 0,
    "id": 0,
    "schemaType": "AVRO",
    "schema": "string",
    "references": [
      {
        "name": "com.example.Address",
        "subject": "address-value",
        "version": 1
      }
    ],
    "metadata": {
      "tags": {
        "team": [
          "platform",
          "data-eng"
        ]
      },
      "properties": {
        "owner": "data-platform-team",
        "classification": "internal"
      },
      "sensitive": [
        "ssn",
        "email"
      ]
    },
    "ruleSet": {
      "migrationRules": [
        {
          "name": "checkSensitiveFields",
          "doc": "Ensures PII fields are encrypted",
          "kind": "CONDITION",
          "mode": "WRITE",
          "type": "CEL",
          "tags": [
            "string"
          ],
          "params": {
            "property1": "string",
            "property2": "string"
          },
          "expr": "message.ssn != ''",
          "onSuccess": "string",
          "onFailure": "string",
          "disabled": false
        }
      ],
      "domainRules": [
        {
          "name": "checkSensitiveFields",
          "doc": "Ensures PII fields are encrypted",
          "kind": "CONDITION",
          "mode": "WRITE",
          "type": "CEL",
          "tags": [
            "string"
          ],
          "params": {
            "property1": "string",
            "property2": "string"
          },
          "expr": "message.ssn != ''",
          "onSuccess": "string",
          "onFailure": "string",
          "disabled": false
        }
      ]
    }
  }
]
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|A list of schema records.|Inline|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

### Response Schema

Status Code **200**

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|[[SchemaListItem](#schemaschemalistitem)]|false|none|[A single schema in the list schemas response.]|
|» subject|string|true|none|The subject name.|
|» version|integer|true|none|The version number.|
|» id|integer(int64)|true|none|The globally unique schema ID.|
|» schemaType|string|true|none|The type of the schema.|
|» schema|string|true|none|The schema definition as a string.|
|» references|[[Reference](#schemareference)]|false|none|References to other schemas.|
|»» name|string|true|none|The reference name. For Avro, this is the fully-qualified name of the referenced type. For Protobuf, this is the import path. For JSON Schema, this is the $ref URI.|
|»» subject|string|true|none|The subject under which the referenced schema is registered.|
|»» version|integer|true|none|The version of the referenced schema.|
|» metadata|[Metadata](#schemametadata)|false|none|Metadata associated with a schema for data contract management. Contains tags for categorization, properties for key-value data, and a list of field names that contain sensitive information.|
|»» tags|object|false|none|A map of tag names to arrays of tag values. Used for categorizing schemas.|
|»»» **additionalProperties**|[string]|false|none|none|
|»» properties|object|false|none|A map of property names to string values. Used for attaching arbitrary metadata to schemas.|
|»»» **additionalProperties**|string|false|none|none|
|»» sensitive|[string]|false|none|A list of field names that contain sensitive data (e.g. PII). Schema processing tools MAY use this to apply data masking or encryption.|
|» ruleSet|[RuleSet](#schemaruleset)|false|none|A set of data contract rules attached to a schema. Contains migration rules (applied during schema evolution) and domain rules (applied during data processing).|
|»» migrationRules|[[Rule](#schemarule)]|false|none|Rules applied during schema migration (evolution). These rules govern how data written with an older schema version is transformed when read with a newer version, or vice versa.|
|»»» name|string|true|none|The unique name of this rule.|
|»»» doc|string|false|none|A human-readable description of the rule's purpose.|
|»»» kind|string|true|none|The kind of rule. Common values include CONDITION (validation) and TRANSFORM (data transformation).|
|»»» mode|string|true|none|When the rule is applied in the data flow. Common values include WRITE (applied on produce), READ (applied on consume), and WRITEREAD (applied on both).|
|»»» type|string|false|none|The rule engine type (e.g. CEL, AVRO, JSONATA).|
|»»» tags|[string]|false|none|Tags that this rule applies to.|
|»»» params|object|false|none|Key-value parameters passed to the rule engine.|
|»»»» **additionalProperties**|string|false|none|none|
|»»» expr|string|false|none|The rule expression to evaluate. The syntax depends on the rule `type`.|
|»»» onSuccess|string|false|none|Action to take when the rule evaluates successfully (e.g. NONE, ERROR).|
|»»» onFailure|string|false|none|Action to take when the rule evaluation fails (e.g. NONE, ERROR, DLQ).|
|»»» disabled|boolean|false|none|Whether the rule is currently disabled.|
|»» domainRules|[[Rule](#schemarule)]|false|none|Rules applied during normal data processing. These rules define validation conditions and data transformations.|

#### Enumerated Values

|Property|Value|
|---|---|
|schemaType|AVRO|
|schemaType|PROTOBUF|
|schemaType|JSON|
|kind|CONDITION|
|kind|TRANSFORM|
|mode|WRITE|
|mode|READ|
|mode|WRITEREAD|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


## Get schema by global ID


> Code samples

```shell
# You can also use wget
curl -X GET http://localhost:8081/schemas/ids/{id} \
  -H 'Accept: application/vnd.schemaregistry.v1+json'

```

`GET /schemas/ids/{id}`

Retrieves a schema by its globally unique ID. The response includes the schema string, its type, any references, metadata, and ruleset. The optional `format` query parameter allows requesting the schema in an alternative serialization (e.g. `serialized` for Protobuf's normalized format). When `fetchMaxId` is set to `true`, the response includes the current maximum schema ID in the registry. The `subject` query parameter MAY be used as a hint but does not filter results.

### Parameters

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|id|path|integer(int64)|true|The globally unique integer ID of the schema.|
|format|query|string|false|An optional format hint for the returned schema string. For Protobuf schemas, passing `serialized` returns the normalized descriptor representation.|
|fetchMaxId|query|boolean|false|When set to `true`, the response includes a `maxId` field containing the current highest schema ID in the registry.|
|subject|query|string|false|An optional subject name filter. This parameter is accepted for Confluent API compatibility.|

> Example responses

> 200 Response

```json
{
  "schema": "string",
  "schemaType": "AVRO",
  "references": [
    {
      "name": "com.example.Address",
      "subject": "address-value",
      "version": 1
    }
  ],
  "metadata": {
    "tags": {
      "team": [
        "platform",
        "data-eng"
      ]
    },
    "properties": {
      "owner": "data-platform-team",
      "classification": "internal"
    },
    "sensitive": [
      "ssn",
      "email"
    ]
  },
  "ruleSet": {
    "migrationRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ],
    "domainRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ]
  },
  "maxId": 0
}
```

> 404 Response

```json
{
  "error_code": 40403,
  "message": "Schema not found"
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The schema identified by the given global ID.|[SchemaByIDResponse](#schemaschemabyidresponse)|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|Schema not found.|[ErrorResponse](#schemaerrorresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


## Get raw schema string by global ID


> Code samples

```shell
# You can also use wget
curl -X GET http://localhost:8081/schemas/ids/{id}/schema \
  -H 'Accept: application/vnd.schemaregistry.v1+json'

```

`GET /schemas/ids/{id}/schema`

Retrieves the raw schema string for the schema identified by its globally unique ID. Unlike `GET /schemas/ids/{id}`, this endpoint returns only the schema content as a string, without metadata or references. The optional `format` query parameter allows requesting an alternative serialization.

### Parameters

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|id|path|integer(int64)|true|The globally unique integer ID of the schema.|
|format|query|string|false|An optional format hint for the returned schema string.|

> Example responses

> 200 Response

```json
"string"
```

> 404 Response

```json
{
  "error_code": 40403,
  "message": "Schema not found"
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The raw schema string.|string|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|Schema not found.|[ErrorResponse](#schemaerrorresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


## Get subjects associated with a schema ID


> Code samples

```shell
# You can also use wget
curl -X GET http://localhost:8081/schemas/ids/{id}/subjects \
  -H 'Accept: application/vnd.schemaregistry.v1+json'

```

`GET /schemas/ids/{id}/subjects`

Returns the list of subject names that have registered the schema identified by the given global ID. This is useful for determining which subjects share the same schema content. Soft-deleted subjects MAY be included by setting `deleted=true`. An optional `subject` query parameter filters the results to a specific subject name.

### Parameters

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|id|path|integer(int64)|true|The globally unique integer ID of the schema.|
|deleted|query|boolean|false|When set to `true`, includes subjects that have been soft-deleted.|
|subject|query|string|false|An optional subject name to filter the result to only that subject.|

> Example responses

> 200 Response

```json
[
  "my-topic-value",
  "other-topic-value"
]
```

> 404 Response

```json
{
  "error_code": 40403,
  "message": "Schema not found"
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|A list of subject names.|Inline|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|Schema not found.|[ErrorResponse](#schemaerrorresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

### Response Schema

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


## Get subject-version pairs for a schema ID


> Code samples

```shell
# You can also use wget
curl -X GET http://localhost:8081/schemas/ids/{id}/versions \
  -H 'Accept: application/vnd.schemaregistry.v1+json'

```

`GET /schemas/ids/{id}/versions`

Returns all subject-version pairs where the schema identified by the given global ID is registered. A single schema MAY be registered under multiple subjects at different version numbers. Soft-deleted versions MAY be included by setting `deleted=true`. An optional `subject` filter restricts results to a specific subject name.

### Parameters

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|id|path|integer(int64)|true|The globally unique integer ID of the schema.|
|deleted|query|boolean|false|When set to `true`, includes soft-deleted subject-version pairs.|
|subject|query|string|false|An optional subject name to filter results to only versions under that subject.|

> Example responses

> 200 Response

```json
[
  {
    "subject": "my-topic-value",
    "version": 1
  }
]
```

> 404 Response

```json
{
  "error_code": 40403,
  "message": "Schema not found"
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|A list of subject-version pairs.|Inline|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|Schema not found.|[ErrorResponse](#schemaerrorresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

### Response Schema

Status Code **200**

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|[[SubjectVersionPair](#schemasubjectversionpair)]|false|none|[A pair identifying a specific subject and version.]|
|» subject|string|true|none|The subject name.|
|» version|integer|true|none|The version number.|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


# Subjects

Operations for managing subjects and their schema versions. A subject is a named scope (typically corresponding to a Kafka topic) under which one or more schema versions are registered. Subjects support soft-delete and permanent-delete semantics.

## List subjects


> Code samples

```shell
# You can also use wget
curl -X GET http://localhost:8081/subjects \
  -H 'Accept: application/vnd.schemaregistry.v1+json'

```

`GET /subjects`

Returns a list of all registered subject names. By default only active (non-deleted) subjects are returned. Use the `deleted` query parameter to include soft-deleted subjects, or `deletedOnly` to return only soft-deleted subjects. Results MAY be filtered by a subject name prefix and paginated with offset and limit.

### Parameters

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|deleted|query|boolean|false|When set to `true`, includes soft-deleted subjects alongside active ones.|
|deletedOnly|query|boolean|false|When set to `true`, returns only subjects that have been soft-deleted. This implicitly sets `deleted=true`.|
|subjectPrefix|query|string|false|Filters the results to subjects whose name starts with the given prefix.|
|offset|query|integer|false|The number of results to skip for pagination.|
|limit|query|integer|false|The maximum number of results to return. If omitted, all results are returned.|

> Example responses

> 200 Response

```json
[
  "my-topic-value",
  "other-topic-key"
]
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|A JSON array of subject name strings.|Inline|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

### Response Schema

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


## List versions under a subject


> Code samples

```shell
# You can also use wget
curl -X GET http://localhost:8081/subjects/{subject}/versions \
  -H 'Accept: application/vnd.schemaregistry.v1+json'

```

`GET /subjects/{subject}/versions`

Returns a list of version numbers registered under the specified subject. By default only active (non-deleted) versions are returned. Use `deleted=true` to include soft-deleted versions, or `deletedOnly=true` to return exclusively soft-deleted versions. Pagination is supported via offset and limit.

### Parameters

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|subject|path|string|true|The name of the subject. Subjects typically correspond to Kafka topic names with a `-key` or `-value` suffix (e.g. `my-topic-value`).|
|deleted|query|boolean|false|When set to `true`, includes soft-deleted versions alongside active ones.|
|deletedOnly|query|boolean|false|When set to `true`, returns only versions that have been soft-deleted. This implicitly sets `deleted=true`.|
|offset|query|integer|false|The number of results to skip for pagination.|
|limit|query|integer|false|The maximum number of results to return. If omitted, all versions are returned.|

> Example responses

> 200 Response

```json
[
  1,
  2,
  3
]
```

> 404 Response

```json
{
  "error_code": 40401,
  "message": "Subject not found"
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|A JSON array of version numbers (integers).|Inline|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|Subject not found.|[ErrorResponse](#schemaerrorresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

### Response Schema

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


## Register a new schema under a subject


> Code samples

```shell
# You can also use wget
curl -X POST http://localhost:8081/subjects/{subject}/versions \
  -H 'Content-Type: application/vnd.schemaregistry.v1+json' \
  -H 'Accept: application/vnd.schemaregistry.v1+json'

```

`POST /subjects/{subject}/versions`

Registers a new schema version under the specified subject. If an identical schema (by content fingerprint) already exists under this subject, the existing schema's ID is returned without creating a new version. The schema MUST be compatible with previous versions according to the subject's compatibility policy, unless the compatibility level is set to NONE.
The request body MUST include the `schema` field containing the schema definition as a string. The `schemaType` field defaults to `AVRO` if omitted. References to other schemas MAY be included via the `references` array.
When the `normalize` query parameter is set to `true`, the schema string is canonicalized before fingerprinting and storage.
If the request includes an explicit `id` field and the registry is in IMPORT mode, the schema is imported with that specific ID.
The subject's mode MUST be READWRITE or IMPORT for this operation to succeed. If the subject is in READONLY or READONLY_OVERRIDE mode, a 42205 error is returned.

> Body parameter

```json
{
  "schema": "{\"type\":\"record\",\"name\":\"User\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"}]}",
  "schemaType": "AVRO",
  "references": [
    {
      "name": "com.example.Address",
      "subject": "address-value",
      "version": 1
    }
  ],
  "id": 0,
  "metadata": {
    "tags": {
      "team": [
        "platform",
        "data-eng"
      ]
    },
    "properties": {
      "owner": "data-platform-team",
      "classification": "internal"
    },
    "sensitive": [
      "ssn",
      "email"
    ]
  },
  "ruleSet": {
    "migrationRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ],
    "domainRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ]
  }
}
```

### Parameters

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|subject|path|string|true|The name of the subject. Subjects typically correspond to Kafka topic names with a `-key` or `-value` suffix (e.g. `my-topic-value`).|
|normalize|query|boolean|false|When set to `true`, the schema is canonicalized (normalized) before storage and fingerprinting. This ensures semantically equivalent schemas produce the same fingerprint.|
|body|body|[RegisterSchemaRequest](#schemaregisterschemarequest)|true|none|

> Example responses

> 200 Response

```json
{
  "id": 1
}
```

> 409 Response

```json
{
  "error_code": 409,
  "message": "Schema being registered is incompatible with an earlier schema"
}
```

> 422 Response

```json
{
  "error_code": 42201,
  "message": "Invalid schema"
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The schema was registered successfully (or an identical schema already existed). Returns the globally unique schema ID.|[RegisterSchemaResponse](#schemaregisterschemaresponse)|
|409|[Conflict](https://tools.ietf.org/html/rfc7231#section-6.5.8)|The schema is incompatible with an existing version under this subject according to the configured compatibility policy.|[ErrorResponse](#schemaerrorresponse)|
|422|[Unprocessable Entity](https://tools.ietf.org/html/rfc2518#section-10.3)|The schema is invalid, the schema type is unsupported, references could not be resolved, or the operation is not permitted in the current mode.|[ErrorResponse](#schemaerrorresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


## Get a specific version of a subject


> Code samples

```shell
# You can also use wget
curl -X GET http://localhost:8081/subjects/{subject}/versions/{version} \
  -H 'Accept: application/vnd.schemaregistry.v1+json'

```

`GET /subjects/{subject}/versions/{version}`

Retrieves the schema registered under the given subject at the specified version number. The `version` path parameter accepts an integer (1 through 2^31-1) or the string `latest` to retrieve the most recently registered version.
When `deleted=true` is set, soft-deleted versions are also returned. The optional `format` query parameter allows requesting an alternative schema serialization.

### Parameters

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|subject|path|string|true|The name of the subject. Subjects typically correspond to Kafka topic names with a `-key` or `-value` suffix (e.g. `my-topic-value`).|
|version|path|any|true|The version number to operate on. MUST be a positive integer (1 through 2^31-1) or the string `latest` to refer to the most recently registered version. The value `-1` is also accepted as an alias for `latest`.|
|deleted|query|boolean|false|When set to `true`, soft-deleted versions are also retrievable.|
|format|query|string|false|An optional format hint for the returned schema string.|

> Example responses

> 200 Response

```json
{
  "subject": "my-topic-value",
  "id": 1,
  "version": 1,
  "schemaType": "AVRO",
  "schema": "string",
  "references": [
    {
      "name": "com.example.Address",
      "subject": "address-value",
      "version": 1
    }
  ],
  "metadata": {
    "tags": {
      "team": [
        "platform",
        "data-eng"
      ]
    },
    "properties": {
      "owner": "data-platform-team",
      "classification": "internal"
    },
    "sensitive": [
      "ssn",
      "email"
    ]
  },
  "ruleSet": {
    "migrationRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ],
    "domainRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ]
  }
}
```

> Subject or version not found.

```json
{
  "error_code": 40401,
  "message": "Subject not found"
}
```

```json
{
  "error_code": 40402,
  "message": "Version not found"
}
```

> 422 Response

```json
{
  "error_code": 42202,
  "message": "The specified version 'abc' is not a valid version id. Allowed values are between [1, 2^31-1] and the string \"latest\""
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The schema version detail.|[SubjectVersionResponse](#schemasubjectversionresponse)|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|Subject or version not found.|[ErrorResponse](#schemaerrorresponse)|
|422|[Unprocessable Entity](https://tools.ietf.org/html/rfc2518#section-10.3)|Invalid version identifier.|[ErrorResponse](#schemaerrorresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


## Delete a specific version of a subject


> Code samples

```shell
# You can also use wget
curl -X DELETE http://localhost:8081/subjects/{subject}/versions/{version} \
  -H 'Accept: application/vnd.schemaregistry.v1+json'

```

`DELETE /subjects/{subject}/versions/{version}`

Deletes a specific version of a schema under a subject. By default this performs a soft-delete, marking the version as deleted but retaining it in storage. To permanently remove the version, set `permanent=true`. A version MUST be soft-deleted before it can be permanently deleted.
The string `latest` and `-1` are NOT allowed for permanent deletes; an explicit numeric version MUST be specified.
If the schema version is referenced by other schemas, the delete operation fails with a 42206 error.
The subject's mode MUST NOT be READONLY or READONLY_OVERRIDE for this operation to succeed.

### Parameters

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|subject|path|string|true|The name of the subject. Subjects typically correspond to Kafka topic names with a `-key` or `-value` suffix (e.g. `my-topic-value`).|
|version|path|any|true|The version number to operate on. MUST be a positive integer (1 through 2^31-1) or the string `latest` to refer to the most recently registered version. The value `-1` is also accepted as an alias for `latest`.|
|permanent|query|boolean|false|When set to `true`, permanently removes the version from storage. The version MUST have been soft-deleted first.|

> Example responses

> 200 Response

```json
3
```

> Subject or version not found, or version not yet soft-deleted.

```json
{
  "error_code": 40401,
  "message": "Subject not found"
}
```

```json
{
  "error_code": 40402,
  "message": "Version not found"
}
```

```json
{
  "error_code": 40407,
  "message": "Subject 'my-subject' Version 3 was not deleted first before being permanently deleted"
}
```

> Invalid version, operation not permitted, or referenced by other schemas.

```json
{
  "error_code": 42202,
  "message": "The specified version 'abc' is not a valid version id."
}
```

```json
{
  "error_code": 42206,
  "message": "Schema is referenced by other schemas"
}
```

```json
{
  "error_code": 42205,
  "message": "Subject 'my-subject' is in READONLY mode"
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The version was deleted. Returns the version number that was deleted.|integer|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|Subject or version not found, or version not yet soft-deleted.|[ErrorResponse](#schemaerrorresponse)|
|422|[Unprocessable Entity](https://tools.ietf.org/html/rfc2518#section-10.3)|Invalid version, operation not permitted, or referenced by other schemas.|[ErrorResponse](#schemaerrorresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


## Get raw schema string by subject version


> Code samples

```shell
# You can also use wget
curl -X GET http://localhost:8081/subjects/{subject}/versions/{version}/schema \
  -H 'Accept: application/vnd.schemaregistry.v1+json'

```

`GET /subjects/{subject}/versions/{version}/schema`

Retrieves only the raw schema string for the specified subject and version, without any metadata envelope. The `version` path parameter accepts an integer or `latest`. The optional `format` query parameter allows requesting an alternative serialization.

### Parameters

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|subject|path|string|true|The name of the subject. Subjects typically correspond to Kafka topic names with a `-key` or `-value` suffix (e.g. `my-topic-value`).|
|version|path|any|true|The version number to operate on. MUST be a positive integer (1 through 2^31-1) or the string `latest` to refer to the most recently registered version. The value `-1` is also accepted as an alias for `latest`.|
|format|query|string|false|An optional format hint for the returned schema string.|

> Example responses

> 200 Response

```json
"string"
```

> Subject or version not found.

```json
{
  "error_code": 40401,
  "message": "Subject not found"
}
```

```json
{
  "error_code": 40402,
  "message": "Version not found"
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The raw schema string.|string|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|Subject or version not found.|[ErrorResponse](#schemaerrorresponse)|
|422|[Unprocessable Entity](https://tools.ietf.org/html/rfc2518#section-10.3)|Invalid version identifier.|[ErrorResponse](#schemaerrorresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


## Get schema IDs that reference this version


> Code samples

```shell
# You can also use wget
curl -X GET http://localhost:8081/subjects/{subject}/versions/{version}/referencedby \
  -H 'Accept: application/vnd.schemaregistry.v1+json'

```

`GET /subjects/{subject}/versions/{version}/referencedby`

Returns a list of schema IDs that reference the schema registered under the given subject and version. This is useful for determining downstream dependencies before deleting a schema.

### Parameters

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|subject|path|string|true|The name of the subject. Subjects typically correspond to Kafka topic names with a `-key` or `-value` suffix (e.g. `my-topic-value`).|
|version|path|any|true|The version number to operate on. MUST be a positive integer (1 through 2^31-1) or the string `latest` to refer to the most recently registered version. The value `-1` is also accepted as an alias for `latest`.|

> Example responses

> 200 Response

```json
[
  5,
  12
]
```

> Subject or version not found.

```json
{
  "error_code": 40401,
  "message": "Subject not found"
}
```

```json
{
  "error_code": 40402,
  "message": "Version not found"
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|A JSON array of schema IDs (integers) that reference this schema version.|Inline|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|Subject or version not found.|[ErrorResponse](#schemaerrorresponse)|
|422|[Unprocessable Entity](https://tools.ietf.org/html/rfc2518#section-10.3)|Invalid version identifier.|[ErrorResponse](#schemaerrorresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

### Response Schema

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


## Look up schema under a subject


> Code samples

```shell
# You can also use wget
curl -X POST http://localhost:8081/subjects/{subject} \
  -H 'Content-Type: application/vnd.schemaregistry.v1+json' \
  -H 'Accept: application/vnd.schemaregistry.v1+json'

```

`POST /subjects/{subject}`

Checks whether the given schema has been registered under the specified subject. If found, returns the subject name, schema ID, version, and the full schema details. This is a lookup-only operation that does NOT register the schema.
The `deleted` query parameter controls whether soft-deleted versions are considered during the lookup. When `normalize=true`, the schema is canonicalized before comparison.

> Body parameter

```json
{
  "schema": "string",
  "schemaType": "AVRO",
  "references": [
    {
      "name": "com.example.Address",
      "subject": "address-value",
      "version": 1
    }
  ]
}
```

### Parameters

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|subject|path|string|true|The name of the subject. Subjects typically correspond to Kafka topic names with a `-key` or `-value` suffix (e.g. `my-topic-value`).|
|deleted|query|boolean|false|When set to `true`, also searches among soft-deleted versions.|
|normalize|query|boolean|false|When set to `true`, the provided schema is canonicalized before comparison.|
|body|body|[LookupSchemaRequest](#schemalookupschemarequest)|true|none|

> Example responses

> 200 Response

```json
{
  "subject": "string",
  "id": 0,
  "version": 0,
  "schemaType": "AVRO",
  "schema": "string",
  "references": [
    {
      "name": "com.example.Address",
      "subject": "address-value",
      "version": 1
    }
  ],
  "metadata": {
    "tags": {
      "team": [
        "platform",
        "data-eng"
      ]
    },
    "properties": {
      "owner": "data-platform-team",
      "classification": "internal"
    },
    "sensitive": [
      "ssn",
      "email"
    ]
  },
  "ruleSet": {
    "migrationRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ],
    "domainRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ]
  }
}
```

> Subject or schema not found.

```json
{
  "error_code": 40401,
  "message": "Subject 'my-subject' not found."
}
```

```json
{
  "error_code": 40403,
  "message": "Schema not found"
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The schema was found under the given subject.|[LookupSchemaResponse](#schemalookupschemaresponse)|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|Subject or schema not found.|[ErrorResponse](#schemaerrorresponse)|
|422|[Unprocessable Entity](https://tools.ietf.org/html/rfc2518#section-10.3)|Invalid schema.|[ErrorResponse](#schemaerrorresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


## Delete a subject


> Code samples

```shell
# You can also use wget
curl -X DELETE http://localhost:8081/subjects/{subject} \
  -H 'Accept: application/vnd.schemaregistry.v1+json'

```

`DELETE /subjects/{subject}`

Deletes all schema versions registered under the specified subject. By default this performs a soft-delete, marking all versions as deleted but retaining them in storage. To permanently remove the subject and all its versions, set `permanent=true`. A subject MUST be soft-deleted before it can be permanently deleted.
If any schema version under this subject is referenced by schemas in other subjects, the delete operation fails with a 42206 error.
The subject's mode MUST NOT be READONLY or READONLY_OVERRIDE for this operation to succeed.

### Parameters

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|subject|path|string|true|The name of the subject. Subjects typically correspond to Kafka topic names with a `-key` or `-value` suffix (e.g. `my-topic-value`).|
|permanent|query|boolean|false|When set to `true`, permanently removes the subject and all its versions from storage. The subject MUST have been soft-deleted first.|

> Example responses

> 200 Response

```json
[
  1,
  2,
  3
]
```

> Subject not found or not in the expected delete state.

```json
{
  "error_code": 40401,
  "message": "Subject not found"
}
```

```json
{
  "error_code": 40404,
  "message": "Subject 'my-subject' was soft deleted. Set permanent=true to delete permanently"
}
```

```json
{
  "error_code": 40405,
  "message": "Subject 'my-subject' was not deleted first before being permanently deleted"
}
```

> Referenced by other schemas or operation not permitted.

```json
{
  "error_code": 42206,
  "message": "Schema is referenced by other schemas"
}
```

```json
{
  "error_code": 42205,
  "message": "Subject 'my-subject' is in READONLY mode"
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The subject was deleted. Returns a JSON array of the version numbers that were deleted.|Inline|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|Subject not found or not in the expected delete state.|[ErrorResponse](#schemaerrorresponse)|
|422|[Unprocessable Entity](https://tools.ietf.org/html/rfc2518#section-10.3)|Referenced by other schemas or operation not permitted.|[ErrorResponse](#schemaerrorresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

### Response Schema

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


# Config

Operations for managing compatibility configuration at the global and per-subject level. The compatibility level determines what changes are permitted when registering a new schema version. Supported levels are NONE, BACKWARD, BACKWARD_TRANSITIVE, FORWARD, FORWARD_TRANSITIVE, FULL, and FULL_TRANSITIVE.

## Get global compatibility configuration


> Code samples

```shell
# You can also use wget
curl -X GET http://localhost:8081/config \
  -H 'Accept: application/vnd.schemaregistry.v1+json'

```

`GET /config`

Returns the global compatibility configuration for the registry. The `defaultToGlobal` query parameter is accepted for Confluent API compatibility but has no effect on this endpoint (it is meaningful only on per-subject config endpoints).

### Parameters

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|defaultToGlobal|query|boolean|false|Accepted for compatibility. Has no effect on the global config endpoint.|

> Example responses

> 200 Response

```json
{
  "compatibilityLevel": "BACKWARD",
  "normalize": true,
  "validateFields": true,
  "alias": "string",
  "compatibilityGroup": "string",
  "defaultMetadata": {
    "tags": {
      "team": [
        "platform",
        "data-eng"
      ]
    },
    "properties": {
      "owner": "data-platform-team",
      "classification": "internal"
    },
    "sensitive": [
      "ssn",
      "email"
    ]
  },
  "overrideMetadata": {
    "tags": {
      "team": [
        "platform",
        "data-eng"
      ]
    },
    "properties": {
      "owner": "data-platform-team",
      "classification": "internal"
    },
    "sensitive": [
      "ssn",
      "email"
    ]
  },
  "defaultRuleSet": {
    "migrationRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ],
    "domainRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ]
  },
  "overrideRuleSet": {
    "migrationRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ],
    "domainRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ]
  }
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The global compatibility configuration.|[ConfigResponse](#schemaconfigresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


## Set global compatibility configuration


> Code samples

```shell
# You can also use wget
curl -X PUT http://localhost:8081/config \
  -H 'Content-Type: application/vnd.schemaregistry.v1+json' \
  -H 'Accept: application/vnd.schemaregistry.v1+json'

```

`PUT /config`

Updates the global compatibility configuration for the registry. The `compatibility` field MUST be set to a valid compatibility level. If the request body contains an empty `compatibility` field, the current global configuration is returned without modification (Confluent compatibility behavior).

> Body parameter

```json
{
  "compatibility": "NONE",
  "normalize": true,
  "validateFields": true,
  "alias": "string",
  "compatibilityGroup": "string",
  "defaultMetadata": {
    "tags": {
      "team": [
        "platform",
        "data-eng"
      ]
    },
    "properties": {
      "owner": "data-platform-team",
      "classification": "internal"
    },
    "sensitive": [
      "ssn",
      "email"
    ]
  },
  "overrideMetadata": {
    "tags": {
      "team": [
        "platform",
        "data-eng"
      ]
    },
    "properties": {
      "owner": "data-platform-team",
      "classification": "internal"
    },
    "sensitive": [
      "ssn",
      "email"
    ]
  },
  "defaultRuleSet": {
    "migrationRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ],
    "domainRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ]
  },
  "overrideRuleSet": {
    "migrationRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ],
    "domainRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ]
  }
}
```

### Parameters

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|body|body|[ConfigRequest](#schemaconfigrequest)|true|none|

> Example responses

> 200 Response

```json
{
  "compatibility": "NONE",
  "normalize": true,
  "validateFields": true,
  "alias": "string",
  "compatibilityGroup": "string",
  "defaultMetadata": {
    "tags": {
      "team": [
        "platform",
        "data-eng"
      ]
    },
    "properties": {
      "owner": "data-platform-team",
      "classification": "internal"
    },
    "sensitive": [
      "ssn",
      "email"
    ]
  },
  "overrideMetadata": {
    "tags": {
      "team": [
        "platform",
        "data-eng"
      ]
    },
    "properties": {
      "owner": "data-platform-team",
      "classification": "internal"
    },
    "sensitive": [
      "ssn",
      "email"
    ]
  },
  "defaultRuleSet": {
    "migrationRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ],
    "domainRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ]
  },
  "overrideRuleSet": {
    "migrationRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ],
    "domainRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ]
  }
}
```

> 422 Response

```json
{
  "error_code": 42203,
  "message": "Invalid compatibility level"
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The updated global compatibility configuration.|[ConfigRequest](#schemaconfigrequest)|
|422|[Unprocessable Entity](https://tools.ietf.org/html/rfc2518#section-10.3)|Invalid compatibility level.|[ErrorResponse](#schemaerrorresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


## Delete global compatibility configuration


> Code samples

```shell
# You can also use wget
curl -X DELETE http://localhost:8081/config \
  -H 'Accept: application/vnd.schemaregistry.v1+json'

```

`DELETE /config`

Deletes the global compatibility configuration, resetting it to the server default (BACKWARD). Returns the compatibility level that was in effect before deletion.

> Example responses

> 200 Response

```json
{
  "compatibilityLevel": "BACKWARD",
  "normalize": true,
  "validateFields": true,
  "alias": "string",
  "compatibilityGroup": "string",
  "defaultMetadata": {
    "tags": {
      "team": [
        "platform",
        "data-eng"
      ]
    },
    "properties": {
      "owner": "data-platform-team",
      "classification": "internal"
    },
    "sensitive": [
      "ssn",
      "email"
    ]
  },
  "overrideMetadata": {
    "tags": {
      "team": [
        "platform",
        "data-eng"
      ]
    },
    "properties": {
      "owner": "data-platform-team",
      "classification": "internal"
    },
    "sensitive": [
      "ssn",
      "email"
    ]
  },
  "defaultRuleSet": {
    "migrationRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ],
    "domainRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ]
  },
  "overrideRuleSet": {
    "migrationRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ],
    "domainRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ]
  }
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The compatibility level that was in effect before deletion.|[ConfigResponse](#schemaconfigresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


## Get subject-level compatibility configuration


> Code samples

```shell
# You can also use wget
curl -X GET http://localhost:8081/config/{subject} \
  -H 'Accept: application/vnd.schemaregistry.v1+json'

```

`GET /config/{subject}`

Returns the compatibility configuration for the specified subject. If the subject does not have a subject-level configuration and `defaultToGlobal` is set to `true`, the global configuration is returned as a fallback. If `defaultToGlobal` is `false` (the default) and no subject-level configuration exists, a 40408 error is returned.

### Parameters

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|subject|path|string|true|The name of the subject. Subjects typically correspond to Kafka topic names with a `-key` or `-value` suffix (e.g. `my-topic-value`).|
|defaultToGlobal|query|boolean|false|When set to `true`, falls back to the global compatibility configuration if the subject does not have its own configuration.|

> Example responses

> 200 Response

```json
{
  "compatibilityLevel": "BACKWARD",
  "normalize": true,
  "validateFields": true,
  "alias": "string",
  "compatibilityGroup": "string",
  "defaultMetadata": {
    "tags": {
      "team": [
        "platform",
        "data-eng"
      ]
    },
    "properties": {
      "owner": "data-platform-team",
      "classification": "internal"
    },
    "sensitive": [
      "ssn",
      "email"
    ]
  },
  "overrideMetadata": {
    "tags": {
      "team": [
        "platform",
        "data-eng"
      ]
    },
    "properties": {
      "owner": "data-platform-team",
      "classification": "internal"
    },
    "sensitive": [
      "ssn",
      "email"
    ]
  },
  "defaultRuleSet": {
    "migrationRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ],
    "domainRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ]
  },
  "overrideRuleSet": {
    "migrationRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ],
    "domainRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ]
  }
}
```

> 404 Response

```json
{
  "error_code": 40408,
  "message": "Subject 'my-subject' does not have subject-level compatibility configured"
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The subject-level compatibility configuration.|[ConfigResponse](#schemaconfigresponse)|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|Subject does not have subject-level compatibility configured.|[ErrorResponse](#schemaerrorresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


## Set subject-level compatibility configuration


> Code samples

```shell
# You can also use wget
curl -X PUT http://localhost:8081/config/{subject} \
  -H 'Content-Type: application/vnd.schemaregistry.v1+json' \
  -H 'Accept: application/vnd.schemaregistry.v1+json'

```

`PUT /config/{subject}`

Updates the compatibility configuration for the specified subject. The `compatibility` field MUST be set to a valid compatibility level. If the request body contains an empty `compatibility` field, the current configuration for this subject is returned without modification.

> Body parameter

```json
{
  "compatibility": "NONE",
  "normalize": true,
  "validateFields": true,
  "alias": "string",
  "compatibilityGroup": "string",
  "defaultMetadata": {
    "tags": {
      "team": [
        "platform",
        "data-eng"
      ]
    },
    "properties": {
      "owner": "data-platform-team",
      "classification": "internal"
    },
    "sensitive": [
      "ssn",
      "email"
    ]
  },
  "overrideMetadata": {
    "tags": {
      "team": [
        "platform",
        "data-eng"
      ]
    },
    "properties": {
      "owner": "data-platform-team",
      "classification": "internal"
    },
    "sensitive": [
      "ssn",
      "email"
    ]
  },
  "defaultRuleSet": {
    "migrationRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ],
    "domainRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ]
  },
  "overrideRuleSet": {
    "migrationRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ],
    "domainRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ]
  }
}
```

### Parameters

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|subject|path|string|true|The name of the subject. Subjects typically correspond to Kafka topic names with a `-key` or `-value` suffix (e.g. `my-topic-value`).|
|body|body|[ConfigRequest](#schemaconfigrequest)|true|none|

> Example responses

> 200 Response

```json
{
  "compatibility": "NONE",
  "normalize": true,
  "validateFields": true,
  "alias": "string",
  "compatibilityGroup": "string",
  "defaultMetadata": {
    "tags": {
      "team": [
        "platform",
        "data-eng"
      ]
    },
    "properties": {
      "owner": "data-platform-team",
      "classification": "internal"
    },
    "sensitive": [
      "ssn",
      "email"
    ]
  },
  "overrideMetadata": {
    "tags": {
      "team": [
        "platform",
        "data-eng"
      ]
    },
    "properties": {
      "owner": "data-platform-team",
      "classification": "internal"
    },
    "sensitive": [
      "ssn",
      "email"
    ]
  },
  "defaultRuleSet": {
    "migrationRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ],
    "domainRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ]
  },
  "overrideRuleSet": {
    "migrationRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ],
    "domainRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ]
  }
}
```

> 422 Response

```json
{
  "error_code": 42203,
  "message": "Invalid compatibility level"
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The updated subject-level compatibility configuration.|[ConfigRequest](#schemaconfigrequest)|
|422|[Unprocessable Entity](https://tools.ietf.org/html/rfc2518#section-10.3)|Invalid compatibility level.|[ErrorResponse](#schemaerrorresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


## Delete subject-level compatibility configuration


> Code samples

```shell
# You can also use wget
curl -X DELETE http://localhost:8081/config/{subject} \
  -H 'Accept: application/vnd.schemaregistry.v1+json'

```

`DELETE /config/{subject}`

Deletes the subject-level compatibility configuration for the specified subject, causing the subject to inherit the global configuration. Returns the compatibility level that was in effect before deletion.

### Parameters

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|subject|path|string|true|The name of the subject. Subjects typically correspond to Kafka topic names with a `-key` or `-value` suffix (e.g. `my-topic-value`).|

> Example responses

> 200 Response

```json
{
  "compatibilityLevel": "BACKWARD",
  "normalize": true,
  "validateFields": true,
  "alias": "string",
  "compatibilityGroup": "string",
  "defaultMetadata": {
    "tags": {
      "team": [
        "platform",
        "data-eng"
      ]
    },
    "properties": {
      "owner": "data-platform-team",
      "classification": "internal"
    },
    "sensitive": [
      "ssn",
      "email"
    ]
  },
  "overrideMetadata": {
    "tags": {
      "team": [
        "platform",
        "data-eng"
      ]
    },
    "properties": {
      "owner": "data-platform-team",
      "classification": "internal"
    },
    "sensitive": [
      "ssn",
      "email"
    ]
  },
  "defaultRuleSet": {
    "migrationRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ],
    "domainRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ]
  },
  "overrideRuleSet": {
    "migrationRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ],
    "domainRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ]
  }
}
```

> 404 Response

```json
{
  "error_code": 40401,
  "message": "Config not found for subject"
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The compatibility level that was in effect before deletion.|[ConfigResponse](#schemaconfigresponse)|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|No subject-level config found for this subject.|[ErrorResponse](#schemaerrorresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


# Mode

Operations for managing the registry mode at the global and per-subject level. The mode controls whether schema registration (writes) is permitted. Supported modes are READWRITE, READONLY, READONLY_OVERRIDE, and IMPORT.

## Get global mode


> Code samples

```shell
# You can also use wget
curl -X GET http://localhost:8081/mode \
  -H 'Accept: application/vnd.schemaregistry.v1+json'

```

`GET /mode`

Returns the global registry mode. The mode controls whether the registry accepts schema registration (write) operations. Possible modes are READWRITE (default), READONLY, READONLY_OVERRIDE, and IMPORT.

> Example responses

> 200 Response

```json
{
  "mode": "READWRITE"
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The global mode.|[ModeResponse](#schemamoderesponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


## Set global mode


> Code samples

```shell
# You can also use wget
curl -X PUT http://localhost:8081/mode \
  -H 'Content-Type: application/vnd.schemaregistry.v1+json' \
  -H 'Accept: application/vnd.schemaregistry.v1+json'

```

`PUT /mode`

Updates the global registry mode. The `mode` field MUST be set to a valid mode string (READWRITE, READONLY, READONLY_OVERRIDE, or IMPORT). The `force` query parameter MAY be set to `true` to bypass validation checks when changing mode.

> Body parameter

```json
{
  "mode": "READWRITE"
}
```

### Parameters

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|force|query|boolean|false|When set to `true`, forces the mode change even if validation would otherwise prevent it.|
|body|body|[ModeRequest](#schemamoderequest)|true|none|

> Example responses

> 200 Response

```json
{
  "mode": "READWRITE"
}
```

> Invalid mode or operation not permitted.

```json
{
  "error_code": 42204,
  "message": "Invalid mode"
}
```

```json
{
  "error_code": 42205,
  "message": "Operation not permitted"
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The updated global mode.|[ModeResponse](#schemamoderesponse)|
|422|[Unprocessable Entity](https://tools.ietf.org/html/rfc2518#section-10.3)|Invalid mode or operation not permitted.|[ErrorResponse](#schemaerrorresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


## Get subject-level mode


> Code samples

```shell
# You can also use wget
curl -X GET http://localhost:8081/mode/{subject} \
  -H 'Accept: application/vnd.schemaregistry.v1+json'

```

`GET /mode/{subject}`

Returns the mode configured for the specified subject. If the subject does not have a subject-level mode and `defaultToGlobal` is set to `true`, the global mode is returned. If `defaultToGlobal` is `false` (the default) and no subject-level mode exists, a 40409 error is returned.

### Parameters

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|subject|path|string|true|The name of the subject. Subjects typically correspond to Kafka topic names with a `-key` or `-value` suffix (e.g. `my-topic-value`).|
|defaultToGlobal|query|boolean|false|When set to `true`, falls back to the global mode if the subject does not have its own mode configured.|

> Example responses

> 200 Response

```json
{
  "mode": "READWRITE"
}
```

> 404 Response

```json
{
  "error_code": 40409,
  "message": "Subject 'my-subject' does not have subject-level mode configured"
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The subject-level mode.|[ModeResponse](#schemamoderesponse)|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|Subject does not have a subject-level mode configured.|[ErrorResponse](#schemaerrorresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


## Set subject-level mode


> Code samples

```shell
# You can also use wget
curl -X PUT http://localhost:8081/mode/{subject} \
  -H 'Content-Type: application/vnd.schemaregistry.v1+json' \
  -H 'Accept: application/vnd.schemaregistry.v1+json'

```

`PUT /mode/{subject}`

Updates the mode for the specified subject. The `mode` field MUST be a valid mode string. The `force` query parameter MAY be set to `true` to bypass validation.

> Body parameter

```json
{
  "mode": "READWRITE"
}
```

### Parameters

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|subject|path|string|true|The name of the subject. Subjects typically correspond to Kafka topic names with a `-key` or `-value` suffix (e.g. `my-topic-value`).|
|force|query|boolean|false|When set to `true`, forces the mode change even if validation would otherwise prevent it.|
|body|body|[ModeRequest](#schemamoderequest)|true|none|

> Example responses

> 200 Response

```json
{
  "mode": "READWRITE"
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The updated subject-level mode.|[ModeResponse](#schemamoderesponse)|
|422|[Unprocessable Entity](https://tools.ietf.org/html/rfc2518#section-10.3)|Invalid mode or operation not permitted.|[ErrorResponse](#schemaerrorresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


## Delete subject-level mode


> Code samples

```shell
# You can also use wget
curl -X DELETE http://localhost:8081/mode/{subject} \
  -H 'Accept: application/vnd.schemaregistry.v1+json'

```

`DELETE /mode/{subject}`

Deletes the subject-level mode for the specified subject, causing it to inherit the global mode. Returns the mode that was in effect before deletion.

### Parameters

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|subject|path|string|true|The name of the subject. Subjects typically correspond to Kafka topic names with a `-key` or `-value` suffix (e.g. `my-topic-value`).|

> Example responses

> 200 Response

```json
{
  "mode": "READWRITE"
}
```

> 404 Response

```json
{
  "error_code": 40401,
  "message": "Mode not found for subject"
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The mode that was in effect before deletion.|[ModeResponse](#schemamoderesponse)|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|No subject-level mode found for this subject.|[ErrorResponse](#schemaerrorresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


# Compatibility

Operations for testing whether a candidate schema is compatible with existing schema versions under a subject, without actually registering it.

## Check compatibility against a specific version


> Code samples

```shell
# You can also use wget
curl -X POST http://localhost:8081/compatibility/subjects/{subject}/versions/{version} \
  -H 'Content-Type: application/vnd.schemaregistry.v1+json' \
  -H 'Accept: application/vnd.schemaregistry.v1+json'

```

`POST /compatibility/subjects/{subject}/versions/{version}`

Tests whether a candidate schema is compatible with the schema registered under the given subject at the specified version, according to the subject's compatibility policy. This is a read-only check that does NOT register the schema.
The `version` path parameter accepts an integer or the string `latest`. When `verbose=true`, the response includes detailed compatibility messages explaining any incompatibilities found. When `normalize=true`, the candidate schema is canonicalized before comparison.

> Body parameter

```json
{
  "schema": "string",
  "schemaType": "AVRO",
  "references": [
    {
      "name": "com.example.Address",
      "subject": "address-value",
      "version": 1
    }
  ]
}
```

### Parameters

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|subject|path|string|true|The name of the subject. Subjects typically correspond to Kafka topic names with a `-key` or `-value` suffix (e.g. `my-topic-value`).|
|version|path|any|true|The version number to operate on. MUST be a positive integer (1 through 2^31-1) or the string `latest` to refer to the most recently registered version. The value `-1` is also accepted as an alias for `latest`.|
|verbose|query|boolean|false|When set to `true`, the response includes detailed compatibility messages.|
|normalize|query|boolean|false|When set to `true`, the candidate schema is canonicalized before comparison.|
|body|body|[CompatibilityCheckRequest](#schemacompatibilitycheckrequest)|true|none|

> Example responses

> 200 Response

```json
{
  "is_compatible": true,
  "messages": [
    "string"
  ]
}
```

> Subject or version not found.

```json
{
  "error_code": 40401,
  "message": "Subject not found"
}
```

```json
{
  "error_code": 40402,
  "message": "Version 5 not found"
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The compatibility check result.|[CompatibilityCheckResponse](#schemacompatibilitycheckresponse)|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|Subject or version not found.|[ErrorResponse](#schemaerrorresponse)|
|422|[Unprocessable Entity](https://tools.ietf.org/html/rfc2518#section-10.3)|Invalid schema or invalid version.|[ErrorResponse](#schemaerrorresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


## Check compatibility against all versions


> Code samples

```shell
# You can also use wget
curl -X POST http://localhost:8081/compatibility/subjects/{subject}/versions \
  -H 'Content-Type: application/vnd.schemaregistry.v1+json' \
  -H 'Accept: application/vnd.schemaregistry.v1+json'

```

`POST /compatibility/subjects/{subject}/versions`

Tests whether a candidate schema is compatible with all existing schema versions registered under the given subject, according to the subject's compatibility policy. This endpoint is equivalent to checking compatibility against `latest` when the compatibility level is non-transitive, or against all versions when the level is transitive. This is a read-only check that does NOT register the schema.
When `verbose=true`, the response includes detailed compatibility messages. When `normalize=true`, the candidate schema is canonicalized before comparison.

> Body parameter

```json
{
  "schema": "string",
  "schemaType": "AVRO",
  "references": [
    {
      "name": "com.example.Address",
      "subject": "address-value",
      "version": 1
    }
  ]
}
```

### Parameters

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|subject|path|string|true|The name of the subject. Subjects typically correspond to Kafka topic names with a `-key` or `-value` suffix (e.g. `my-topic-value`).|
|verbose|query|boolean|false|When set to `true`, the response includes detailed compatibility messages.|
|normalize|query|boolean|false|When set to `true`, the candidate schema is canonicalized before comparison.|
|body|body|[CompatibilityCheckRequest](#schemacompatibilitycheckrequest)|true|none|

> Example responses

> 200 Response

```json
{
  "is_compatible": true,
  "messages": [
    "string"
  ]
}
```

> 404 Response

```json
{
  "error_code": 40401,
  "message": "Subject not found"
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The compatibility check result.|[CompatibilityCheckResponse](#schemacompatibilitycheckresponse)|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|Subject not found.|[ErrorResponse](#schemaerrorresponse)|
|422|[Unprocessable Entity](https://tools.ietf.org/html/rfc2518#section-10.3)|Invalid schema.|[ErrorResponse](#schemaerrorresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


# Import

Operations for bulk-importing schemas from another schema registry, preserving original schema IDs. This is used for migration scenarios.

## Bulk import schemas


> Code samples

```shell
# You can also use wget
curl -X POST http://localhost:8081/import/schemas \
  -H 'Content-Type: application/vnd.schemaregistry.v1+json' \
  -H 'Accept: application/vnd.schemaregistry.v1+json'

```

`POST /import/schemas`

Imports multiple schemas in a single request, preserving the original schema IDs from the source registry. This endpoint is intended for migrating schemas from another schema registry (e.g. Confluent Schema Registry). Each schema in the request MUST include a specific `id`, `subject`, `version`, and `schema` string.
The response indicates how many schemas were successfully imported and how many failed, along with individual results for each schema in the request.

> Body parameter

```json
{
  "schemas": [
    {
      "id": 0,
      "subject": "string",
      "version": 0,
      "schemaType": "AVRO",
      "schema": "string",
      "references": [
        {
          "name": "com.example.Address",
          "subject": "address-value",
          "version": 1
        }
      ]
    }
  ]
}
```

### Parameters

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|body|body|[ImportSchemasRequest](#schemaimportschemasrequest)|true|none|

> Example responses

> 200 Response

```json
{
  "imported": 10,
  "errors": 2,
  "results": [
    {
      "id": 0,
      "subject": "string",
      "version": 0,
      "success": true,
      "error": "string"
    }
  ]
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|Import completed. Check the `imported` and `errors` counts and the individual `results` to determine whether all schemas were imported successfully.|[ImportSchemasResponse](#schemaimportschemasresponse)|
|400|[Bad Request](https://tools.ietf.org/html/rfc7231#section-6.5.1)|Invalid request body or no schemas provided.|[ErrorResponse](#schemaerrorresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


# Metadata

Operations for retrieving registry metadata such as the cluster ID and server version.

## Get schema registry contexts


> Code samples

```shell
# You can also use wget
curl -X GET http://localhost:8081/contexts \
  -H 'Accept: application/vnd.schemaregistry.v1+json'

```

`GET /contexts`

Returns the list of contexts defined in the registry.

In Confluent Schema Registry, **contexts** are a multi-tenancy feature that allows multiple schemas with the same subject names and IDs to coexist in separate namespaces (e.g. `.team-a`, `.team-b`). Subjects are qualified with a context prefix (e.g. `:.mycontext:my-subject`), and schema IDs are unique within each context. This is primarily used for Schema Linking and enterprise multi-tenant deployments.

AxonOps Schema Registry operates as a single-tenant registry. This endpoint always returns `["."]` (the default context only). Context-qualified subject names are not supported. See [#264](https://github.com/axonops/axonops-schema-registry/issues/264) for the feature request to add context support.

> Example responses

> 200 Response

```json
[
  "."
]
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|A list of context strings.|Inline|

### Response Schema

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


## Get cluster ID


> Code samples

```shell
# You can also use wget
curl -X GET http://localhost:8081/v1/metadata/id \
  -H 'Accept: application/vnd.schemaregistry.v1+json'

```

`GET /v1/metadata/id`

Returns the cluster ID of this schema registry instance. The cluster ID is a string identifier configured at server startup.

> Example responses

> 200 Response

```json
{
  "id": "default-cluster"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The cluster ID.|[ServerClusterIDResponse](#schemaserverclusteridresponse)|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


## Get server version


> Code samples

```shell
# You can also use wget
curl -X GET http://localhost:8081/v1/metadata/version \
  -H 'Accept: application/vnd.schemaregistry.v1+json'

```

`GET /v1/metadata/version`

Returns the version, commit hash, and build time of the running schema registry server.

> Example responses

> 200 Response

```json
{
  "version": "1.0.0",
  "commit": "abc123def",
  "build_time": "2025-01-15T10:30:00Z"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The server version information.|[ServerVersionResponse](#schemaserverversionresponse)|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


# Account

Self-service account endpoints for authenticated users to view their own profile and change their password.

## Get current user


> Code samples

```shell
# You can also use wget
curl -X GET http://localhost:8081/me \
  -H 'Accept: application/json'

```

`GET /me`

Returns the profile of the currently authenticated user. The caller MUST be authenticated. If the user record is not found in the database, a 404 error is returned.

> Example responses

> 200 Response

```json
{
  "id": 1,
  "username": "johndoe",
  "email": "johndoe@example.com",
  "role": "developer",
  "enabled": true,
  "created_at": "2025-01-15T10:30:00Z",
  "updated_at": "2025-01-15T10:30:00Z"
}
```

> 401 Response

```json
{
  "error_code": 40101,
  "message": "Authentication required"
}
```

> 404 Response

```json
{
  "error_code": 40404,
  "message": "User not found"
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The current user's profile.|[UserResponse](#schemauserresponse)|
|401|[Unauthorized](https://tools.ietf.org/html/rfc7235#section-3.1)|Authentication required.|[ErrorResponse](#schemaerrorresponse)|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|User not found.|[ErrorResponse](#schemaerrorresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


## Change current user password


> Code samples

```shell
# You can also use wget
curl -X POST http://localhost:8081/me/password \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json'

```

`POST /me/password`

Changes the password of the currently authenticated user. The request MUST include both the current (old) password for verification and the desired new password. Returns 204 No Content on success.

> Body parameter

```json
{
  "old_password": "pa$$word",
  "new_password": "pa$$word"
}
```

### Parameters

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|body|body|[ChangePasswordRequest](#schemachangepasswordrequest)|true|none|

> Example responses

> 400 Response

```json
{
  "error_code": 40401,
  "message": "Subject 'my-topic-value' not found"
}
```

> 401 Response

```json
{
  "error_code": 40101,
  "message": "Authentication required"
}
```

> 403 Response

```json
{
  "error_code": 40301,
  "message": "Current password is incorrect"
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|204|[No Content](https://tools.ietf.org/html/rfc7231#section-6.3.5)|Password changed successfully.|None|
|400|[Bad Request](https://tools.ietf.org/html/rfc7231#section-6.5.1)|Missing required fields.|[ErrorResponse](#schemaerrorresponse)|
|401|[Unauthorized](https://tools.ietf.org/html/rfc7235#section-3.1)|Authentication required.|[ErrorResponse](#schemaerrorresponse)|
|403|[Forbidden](https://tools.ietf.org/html/rfc7231#section-6.5.3)|Current password is incorrect.|[ErrorResponse](#schemaerrorresponse)|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|User not found.|[ErrorResponse](#schemaerrorresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


# Admin

Administrative endpoints for managing users, API keys, and roles. These endpoints require admin-level permissions.

## List all users


> Code samples

```shell
# You can also use wget
curl -X GET http://localhost:8081/admin/users \
  -H 'Accept: application/json'

```

`GET /admin/users`

Returns a list of all users in the registry. The caller MUST have admin read permissions.

> Example responses

> 200 Response

```json
{
  "users": [
    {
      "id": 1,
      "username": "johndoe",
      "email": "johndoe@example.com",
      "role": "developer",
      "enabled": true,
      "created_at": "2025-01-15T10:30:00Z",
      "updated_at": "2025-01-15T10:30:00Z"
    }
  ]
}
```

> 401 Response

```json
{
  "error_code": 40101,
  "message": "Authentication required"
}
```

> 403 Response

```json
{
  "error_code": 40301,
  "message": "Admin write permission required"
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|A list of all users.|[UsersListResponse](#schemauserslistresponse)|
|401|[Unauthorized](https://tools.ietf.org/html/rfc7235#section-3.1)|Authentication is required.|[ErrorResponse](#schemaerrorresponse)|
|403|[Forbidden](https://tools.ietf.org/html/rfc7231#section-6.5.3)|The authenticated user does not have sufficient permissions.|[ErrorResponse](#schemaerrorresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


## Create a new user


> Code samples

```shell
# You can also use wget
curl -X POST http://localhost:8081/admin/users \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json'

```

`POST /admin/users`

Creates a new user in the registry. The `username`, `password`, and `role` fields are required. The `role` MUST be one of `super_admin`, `admin`, `developer`, or `readonly`. The caller MUST have admin write permissions.

> Body parameter

```json
{
  "username": "johndoe",
  "email": "johndoe@example.com",
  "password": "SecureP@ss123",
  "role": "developer",
  "enabled": true
}
```

### Parameters

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|body|body|[CreateUserRequest](#schemacreateuserrequest)|true|none|

> Example responses

> 201 Response

```json
{
  "id": 1,
  "username": "johndoe",
  "email": "johndoe@example.com",
  "role": "developer",
  "enabled": true,
  "created_at": "2025-01-15T10:30:00Z",
  "updated_at": "2025-01-15T10:30:00Z"
}
```

> 401 Response

```json
{
  "error_code": 40101,
  "message": "Authentication required"
}
```

> 403 Response

```json
{
  "error_code": 40301,
  "message": "Admin write permission required"
}
```

> 409 Response

```json
{
  "error_code": 40901,
  "message": "User already exists"
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|201|[Created](https://tools.ietf.org/html/rfc7231#section-6.3.2)|The newly created user.|[UserResponse](#schemauserresponse)|
|400|[Bad Request](https://tools.ietf.org/html/rfc7231#section-6.5.1)|Missing required fields or invalid role.|[ErrorResponse](#schemaerrorresponse)|
|401|[Unauthorized](https://tools.ietf.org/html/rfc7235#section-3.1)|Authentication is required.|[ErrorResponse](#schemaerrorresponse)|
|403|[Forbidden](https://tools.ietf.org/html/rfc7231#section-6.5.3)|The authenticated user does not have sufficient permissions.|[ErrorResponse](#schemaerrorresponse)|
|409|[Conflict](https://tools.ietf.org/html/rfc7231#section-6.5.8)|A user with this username already exists.|[ErrorResponse](#schemaerrorresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


## Get a user by ID


> Code samples

```shell
# You can also use wget
curl -X GET http://localhost:8081/admin/users/{id} \
  -H 'Accept: application/json'

```

`GET /admin/users/{id}`

Retrieves the user with the specified ID. The caller MUST have admin read permissions.

### Parameters

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|id|path|integer(int64)|true|The unique integer ID of the resource.|

> Example responses

> 200 Response

```json
{
  "id": 1,
  "username": "johndoe",
  "email": "johndoe@example.com",
  "role": "developer",
  "enabled": true,
  "created_at": "2025-01-15T10:30:00Z",
  "updated_at": "2025-01-15T10:30:00Z"
}
```

> 401 Response

```json
{
  "error_code": 40101,
  "message": "Authentication required"
}
```

> 403 Response

```json
{
  "error_code": 40301,
  "message": "Admin write permission required"
}
```

> 404 Response

```json
{
  "error_code": 40404,
  "message": "User not found"
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The user record.|[UserResponse](#schemauserresponse)|
|400|[Bad Request](https://tools.ietf.org/html/rfc7231#section-6.5.1)|Invalid user ID.|[ErrorResponse](#schemaerrorresponse)|
|401|[Unauthorized](https://tools.ietf.org/html/rfc7235#section-3.1)|Authentication is required.|[ErrorResponse](#schemaerrorresponse)|
|403|[Forbidden](https://tools.ietf.org/html/rfc7231#section-6.5.3)|The authenticated user does not have sufficient permissions.|[ErrorResponse](#schemaerrorresponse)|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|User not found.|[ErrorResponse](#schemaerrorresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


## Update a user


> Code samples

```shell
# You can also use wget
curl -X PUT http://localhost:8081/admin/users/{id} \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json'

```

`PUT /admin/users/{id}`

Updates one or more fields of the user with the specified ID. Only the fields provided in the request body are modified; omitted fields remain unchanged. The caller MUST have admin write permissions.

> Body parameter

```json
{
  "email": "string",
  "password": "pa$$word",
  "role": "super_admin",
  "enabled": true
}
```

### Parameters

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|id|path|integer(int64)|true|The unique integer ID of the resource.|
|body|body|[UpdateUserRequest](#schemaupdateuserrequest)|true|none|

> Example responses

> 200 Response

```json
{
  "id": 1,
  "username": "johndoe",
  "email": "johndoe@example.com",
  "role": "developer",
  "enabled": true,
  "created_at": "2025-01-15T10:30:00Z",
  "updated_at": "2025-01-15T10:30:00Z"
}
```

> 401 Response

```json
{
  "error_code": 40101,
  "message": "Authentication required"
}
```

> 403 Response

```json
{
  "error_code": 40301,
  "message": "Admin write permission required"
}
```

> 404 Response

```json
{
  "error_code": 40404,
  "message": "User not found"
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The updated user record.|[UserResponse](#schemauserresponse)|
|400|[Bad Request](https://tools.ietf.org/html/rfc7231#section-6.5.1)|Invalid user ID or invalid role.|[ErrorResponse](#schemaerrorresponse)|
|401|[Unauthorized](https://tools.ietf.org/html/rfc7235#section-3.1)|Authentication is required.|[ErrorResponse](#schemaerrorresponse)|
|403|[Forbidden](https://tools.ietf.org/html/rfc7231#section-6.5.3)|The authenticated user does not have sufficient permissions.|[ErrorResponse](#schemaerrorresponse)|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|User not found.|[ErrorResponse](#schemaerrorresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


## Delete a user


> Code samples

```shell
# You can also use wget
curl -X DELETE http://localhost:8081/admin/users/{id} \
  -H 'Accept: application/json'

```

`DELETE /admin/users/{id}`

Permanently deletes the user with the specified ID. Returns 204 No Content on success. The caller MUST have admin write permissions.

### Parameters

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|id|path|integer(int64)|true|The unique integer ID of the resource.|

> Example responses

> 400 Response

```json
{
  "error_code": 40401,
  "message": "Subject 'my-topic-value' not found"
}
```

> 401 Response

```json
{
  "error_code": 40101,
  "message": "Authentication required"
}
```

> 403 Response

```json
{
  "error_code": 40301,
  "message": "Admin write permission required"
}
```

> 404 Response

```json
{
  "error_code": 40404,
  "message": "User not found"
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|204|[No Content](https://tools.ietf.org/html/rfc7231#section-6.3.5)|User deleted successfully.|None|
|400|[Bad Request](https://tools.ietf.org/html/rfc7231#section-6.5.1)|Invalid user ID.|[ErrorResponse](#schemaerrorresponse)|
|401|[Unauthorized](https://tools.ietf.org/html/rfc7235#section-3.1)|Authentication is required.|[ErrorResponse](#schemaerrorresponse)|
|403|[Forbidden](https://tools.ietf.org/html/rfc7231#section-6.5.3)|The authenticated user does not have sufficient permissions.|[ErrorResponse](#schemaerrorresponse)|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|User not found.|[ErrorResponse](#schemaerrorresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


## List API keys


> Code samples

```shell
# You can also use wget
curl -X GET http://localhost:8081/admin/apikeys \
  -H 'Accept: application/json'

```

`GET /admin/apikeys`

Returns a list of all API keys in the registry. The optional `user_id` query parameter filters results to API keys owned by a specific user. The caller MUST have admin read permissions. The raw key secret is never included in list responses.

### Parameters

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|user_id|query|integer(int64)|false|Filter API keys by the owning user's ID. If omitted, all API keys are returned.|

> Example responses

> 200 Response

```json
{
  "api_keys": [
    {
      "id": 1,
      "key_prefix": "axon_abc",
      "name": "ci-pipeline-key",
      "role": "developer",
      "user_id": 1,
      "username": "johndoe",
      "enabled": true,
      "created_at": "2025-01-15T10:30:00Z",
      "expires_at": "2025-02-14T10:30:00Z",
      "last_used": "2019-08-24T14:15:22Z"
    }
  ]
}
```

> 401 Response

```json
{
  "error_code": 40101,
  "message": "Authentication required"
}
```

> 403 Response

```json
{
  "error_code": 40301,
  "message": "Admin write permission required"
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|A list of API keys.|[APIKeysListResponse](#schemaapikeyslistresponse)|
|400|[Bad Request](https://tools.ietf.org/html/rfc7231#section-6.5.1)|Invalid user ID.|[ErrorResponse](#schemaerrorresponse)|
|401|[Unauthorized](https://tools.ietf.org/html/rfc7235#section-3.1)|Authentication is required.|[ErrorResponse](#schemaerrorresponse)|
|403|[Forbidden](https://tools.ietf.org/html/rfc7231#section-6.5.3)|The authenticated user does not have sufficient permissions.|[ErrorResponse](#schemaerrorresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


## Create a new API key


> Code samples

```shell
# You can also use wget
curl -X POST http://localhost:8081/admin/apikeys \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json'

```

`POST /admin/apikeys`

Creates a new API key. The `name`, `role`, and `expires_in` fields are required. The `name` MUST be unique per user. The `expires_in` value is a duration in seconds from the current time (e.g. 2592000 for 30 days). The `role` MUST be one of `super_admin`, `admin`, `developer`, or `readonly`.
The raw API key secret is returned ONLY in the creation response and cannot be retrieved afterward. Clients SHOULD store the key securely immediately after creation.
Super admins MAY create API keys for other users by specifying the `for_user_id` field.

> Body parameter

```json
{
  "name": "ci-pipeline-key",
  "role": "developer",
  "expires_in": 2592000,
  "for_user_id": 0
}
```

### Parameters

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|body|body|[CreateAPIKeyRequest](#schemacreateapikeyrequest)|true|none|

> Example responses

> 201 Response

```json
{
  "id": 0,
  "key": "axon_abcdefghijklmnopqrstuvwxyz1234567890",
  "key_prefix": "string",
  "name": "string",
  "role": "string",
  "user_id": 0,
  "username": "string",
  "enabled": true,
  "created_at": "2019-08-24T14:15:22Z",
  "expires_at": "2019-08-24T14:15:22Z"
}
```

> 401 Response

```json
{
  "error_code": 40101,
  "message": "Authentication required"
}
```

> 403 Response

```json
{
  "error_code": 40301,
  "message": "Admin write permission required"
}
```

> 409 Response

```json
{
  "error_code": 40902,
  "message": "API key name already exists for this user"
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|201|[Created](https://tools.ietf.org/html/rfc7231#section-6.3.2)|The newly created API key, including the raw key secret. This is the ONLY time the raw key is returned.|[CreateAPIKeyResponse](#schemacreateapikeyresponse)|
|400|[Bad Request](https://tools.ietf.org/html/rfc7231#section-6.5.1)|Missing required fields, invalid role, or invalid expires_in.|[ErrorResponse](#schemaerrorresponse)|
|401|[Unauthorized](https://tools.ietf.org/html/rfc7235#section-3.1)|Authentication is required.|[ErrorResponse](#schemaerrorresponse)|
|403|[Forbidden](https://tools.ietf.org/html/rfc7231#section-6.5.3)|The authenticated user does not have sufficient permissions.|[ErrorResponse](#schemaerrorresponse)|
|409|[Conflict](https://tools.ietf.org/html/rfc7231#section-6.5.8)|An API key with this name already exists for the user.|[ErrorResponse](#schemaerrorresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


## Get an API key by ID


> Code samples

```shell
# You can also use wget
curl -X GET http://localhost:8081/admin/apikeys/{id} \
  -H 'Accept: application/json'

```

`GET /admin/apikeys/{id}`

Retrieves the API key with the specified ID. The raw key secret is NOT included in the response. The caller MUST have admin read permissions.

### Parameters

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|id|path|integer(int64)|true|The unique integer ID of the resource.|

> Example responses

> 200 Response

```json
{
  "id": 1,
  "key_prefix": "axon_abc",
  "name": "ci-pipeline-key",
  "role": "developer",
  "user_id": 1,
  "username": "johndoe",
  "enabled": true,
  "created_at": "2025-01-15T10:30:00Z",
  "expires_at": "2025-02-14T10:30:00Z",
  "last_used": "2019-08-24T14:15:22Z"
}
```

> 401 Response

```json
{
  "error_code": 40101,
  "message": "Authentication required"
}
```

> 403 Response

```json
{
  "error_code": 40301,
  "message": "Admin write permission required"
}
```

> 404 Response

```json
{
  "error_code": 40405,
  "message": "API key not found"
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The API key record.|[APIKeyResponse](#schemaapikeyresponse)|
|400|[Bad Request](https://tools.ietf.org/html/rfc7231#section-6.5.1)|Invalid API key ID.|[ErrorResponse](#schemaerrorresponse)|
|401|[Unauthorized](https://tools.ietf.org/html/rfc7235#section-3.1)|Authentication is required.|[ErrorResponse](#schemaerrorresponse)|
|403|[Forbidden](https://tools.ietf.org/html/rfc7231#section-6.5.3)|The authenticated user does not have sufficient permissions.|[ErrorResponse](#schemaerrorresponse)|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|API key not found.|[ErrorResponse](#schemaerrorresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


## Update an API key


> Code samples

```shell
# You can also use wget
curl -X PUT http://localhost:8081/admin/apikeys/{id} \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json'

```

`PUT /admin/apikeys/{id}`

Updates one or more fields of the API key with the specified ID. Only the fields provided in the request body are modified; omitted fields remain unchanged. The caller MUST have admin write permissions.

> Body parameter

```json
{
  "name": "string",
  "role": "super_admin",
  "enabled": true
}
```

### Parameters

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|id|path|integer(int64)|true|The unique integer ID of the resource.|
|body|body|[UpdateAPIKeyRequest](#schemaupdateapikeyrequest)|true|none|

> Example responses

> 200 Response

```json
{
  "id": 1,
  "key_prefix": "axon_abc",
  "name": "ci-pipeline-key",
  "role": "developer",
  "user_id": 1,
  "username": "johndoe",
  "enabled": true,
  "created_at": "2025-01-15T10:30:00Z",
  "expires_at": "2025-02-14T10:30:00Z",
  "last_used": "2019-08-24T14:15:22Z"
}
```

> 401 Response

```json
{
  "error_code": 40101,
  "message": "Authentication required"
}
```

> 403 Response

```json
{
  "error_code": 40301,
  "message": "Admin write permission required"
}
```

> 404 Response

```json
{
  "error_code": 40405,
  "message": "API key not found"
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The updated API key record.|[APIKeyResponse](#schemaapikeyresponse)|
|400|[Bad Request](https://tools.ietf.org/html/rfc7231#section-6.5.1)|Invalid API key ID or invalid role.|[ErrorResponse](#schemaerrorresponse)|
|401|[Unauthorized](https://tools.ietf.org/html/rfc7235#section-3.1)|Authentication is required.|[ErrorResponse](#schemaerrorresponse)|
|403|[Forbidden](https://tools.ietf.org/html/rfc7231#section-6.5.3)|The authenticated user does not have sufficient permissions.|[ErrorResponse](#schemaerrorresponse)|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|API key not found.|[ErrorResponse](#schemaerrorresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


## Delete an API key


> Code samples

```shell
# You can also use wget
curl -X DELETE http://localhost:8081/admin/apikeys/{id} \
  -H 'Accept: application/json'

```

`DELETE /admin/apikeys/{id}`

Permanently deletes the API key with the specified ID. Returns 204 No Content on success. The caller MUST have admin write permissions.

### Parameters

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|id|path|integer(int64)|true|The unique integer ID of the resource.|

> Example responses

> 400 Response

```json
{
  "error_code": 40401,
  "message": "Subject 'my-topic-value' not found"
}
```

> 401 Response

```json
{
  "error_code": 40101,
  "message": "Authentication required"
}
```

> 403 Response

```json
{
  "error_code": 40301,
  "message": "Admin write permission required"
}
```

> 404 Response

```json
{
  "error_code": 40405,
  "message": "API key not found"
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|204|[No Content](https://tools.ietf.org/html/rfc7231#section-6.3.5)|API key deleted successfully.|None|
|400|[Bad Request](https://tools.ietf.org/html/rfc7231#section-6.5.1)|Invalid API key ID.|[ErrorResponse](#schemaerrorresponse)|
|401|[Unauthorized](https://tools.ietf.org/html/rfc7235#section-3.1)|Authentication is required.|[ErrorResponse](#schemaerrorresponse)|
|403|[Forbidden](https://tools.ietf.org/html/rfc7231#section-6.5.3)|The authenticated user does not have sufficient permissions.|[ErrorResponse](#schemaerrorresponse)|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|API key not found.|[ErrorResponse](#schemaerrorresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


## Revoke an API key


> Code samples

```shell
# You can also use wget
curl -X POST http://localhost:8081/admin/apikeys/{id}/revoke \
  -H 'Accept: application/json'

```

`POST /admin/apikeys/{id}/revoke`

Revokes the API key with the specified ID by disabling it. The API key remains in the database but can no longer be used for authentication. The caller MUST have admin write permissions.

### Parameters

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|id|path|integer(int64)|true|The unique integer ID of the resource.|

> Example responses

> 200 Response

```json
{
  "id": 1,
  "key_prefix": "axon_abc",
  "name": "ci-pipeline-key",
  "role": "developer",
  "user_id": 1,
  "username": "johndoe",
  "enabled": true,
  "created_at": "2025-01-15T10:30:00Z",
  "expires_at": "2025-02-14T10:30:00Z",
  "last_used": "2019-08-24T14:15:22Z"
}
```

> 401 Response

```json
{
  "error_code": 40101,
  "message": "Authentication required"
}
```

> 403 Response

```json
{
  "error_code": 40301,
  "message": "Admin write permission required"
}
```

> 404 Response

```json
{
  "error_code": 40405,
  "message": "API key not found"
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The revoked API key record (with enabled set to false).|[APIKeyResponse](#schemaapikeyresponse)|
|400|[Bad Request](https://tools.ietf.org/html/rfc7231#section-6.5.1)|Invalid API key ID.|[ErrorResponse](#schemaerrorresponse)|
|401|[Unauthorized](https://tools.ietf.org/html/rfc7235#section-3.1)|Authentication is required.|[ErrorResponse](#schemaerrorresponse)|
|403|[Forbidden](https://tools.ietf.org/html/rfc7231#section-6.5.3)|The authenticated user does not have sufficient permissions.|[ErrorResponse](#schemaerrorresponse)|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|API key not found.|[ErrorResponse](#schemaerrorresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


## Rotate an API key


> Code samples

```shell
# You can also use wget
curl -X POST http://localhost:8081/admin/apikeys/{id}/rotate \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json'

```

`POST /admin/apikeys/{id}/rotate`

Rotates the API key with the specified ID by revoking the existing key and creating a new one with the same name and role but a fresh secret and expiry. The `expires_in` field in the request body specifies the duration in seconds for the new key. The raw secret for the new key is returned ONLY in this response.
The caller MUST have admin write permissions.

> Body parameter

```json
{
  "expires_in": 2592000
}
```

### Parameters

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|id|path|integer(int64)|true|The unique integer ID of the resource.|
|body|body|[RotateAPIKeyRequest](#schemarotateapikeyrequest)|true|none|

> Example responses

> 200 Response

```json
{
  "new_key": {
    "id": 0,
    "key": "axon_abcdefghijklmnopqrstuvwxyz1234567890",
    "key_prefix": "string",
    "name": "string",
    "role": "string",
    "user_id": 0,
    "username": "string",
    "enabled": true,
    "created_at": "2019-08-24T14:15:22Z",
    "expires_at": "2019-08-24T14:15:22Z"
  },
  "revoked_id": 1
}
```

> 401 Response

```json
{
  "error_code": 40101,
  "message": "Authentication required"
}
```

> 403 Response

```json
{
  "error_code": 40301,
  "message": "Admin write permission required"
}
```

> 404 Response

```json
{
  "error_code": 40405,
  "message": "API key not found"
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The rotation result containing the new API key (with its raw secret) and the ID of the revoked key.|[RotateAPIKeyResponse](#schemarotateapikeyresponse)|
|400|[Bad Request](https://tools.ietf.org/html/rfc7231#section-6.5.1)|Invalid API key ID or invalid expires_in.|[ErrorResponse](#schemaerrorresponse)|
|401|[Unauthorized](https://tools.ietf.org/html/rfc7235#section-3.1)|Authentication is required.|[ErrorResponse](#schemaerrorresponse)|
|403|[Forbidden](https://tools.ietf.org/html/rfc7231#section-6.5.3)|The authenticated user does not have sufficient permissions.|[ErrorResponse](#schemaerrorresponse)|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|API key not found.|[ErrorResponse](#schemaerrorresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


## List available roles


> Code samples

```shell
# You can also use wget
curl -X GET http://localhost:8081/admin/roles \
  -H 'Accept: application/json'

```

`GET /admin/roles`

Returns the list of roles available in the registry along with their descriptions and associated permissions. The caller MUST have admin read permissions.

> Example responses

> 200 Response

```json
{
  "roles": [
    {
      "name": "developer",
      "description": "Can register and read schemas",
      "permissions": [
        "schema:read",
        "schema:write",
        "subject:read"
      ]
    }
  ]
}
```

> 401 Response

```json
{
  "error_code": 40101,
  "message": "Authentication required"
}
```

> 403 Response

```json
{
  "error_code": 40301,
  "message": "Admin write permission required"
}
```

> 500 Response

```json
{
  "error_code": 50001,
  "message": "Internal server error"
}
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|A list of available roles with their permissions.|[RolesListResponse](#schemaroleslistresponse)|
|401|[Unauthorized](https://tools.ietf.org/html/rfc7235#section-3.1)|Authentication is required.|[ErrorResponse](#schemaerrorresponse)|
|403|[Forbidden](https://tools.ietf.org/html/rfc7231#section-6.5.3)|The authenticated user does not have sufficient permissions.|[ErrorResponse](#schemaerrorresponse)|
|500|[Internal Server Error](https://tools.ietf.org/html/rfc7231#section-6.6.1)|An internal server error occurred.|[ErrorResponse](#schemaerrorresponse)|

> **Warning:** 
To perform this operation, you must be authenticated by means of one of the following methods:
basicAuth, apiKey, bearerAuth


# Documentation

Endpoints for serving the interactive API documentation (Swagger UI) and the raw OpenAPI specification. Available only when the server is configured with docs_enabled.

## Swagger UI


> Code samples

```shell
# You can also use wget
curl -X GET http://localhost:8081/docs \
  -H 'Accept: text/html'

```

`GET /docs`

Serves the interactive Swagger UI for exploring the API. This endpoint is only available when the server is configured with `docs_enabled: true`. It does not require authentication.

> Example responses

> 200 Response

```
"string"
```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The Swagger UI HTML page.|string|

> **Success:** 
This operation does not require authentication


## OpenAPI specification


> Code samples

```shell
# You can also use wget
curl -X GET http://localhost:8081/openapi.yaml \
  -H 'Accept: application/x-yaml'

```

`GET /openapi.yaml`

Returns the raw OpenAPI 3.0.3 specification for this API in YAML format. This endpoint is only available when the server is configured with `docs_enabled: true`. It does not require authentication.

> Example responses

> 200 Response

```yaml
string

```

### Responses

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The OpenAPI specification in YAML format.|string|

> **Success:** 
This operation does not require authentication


# Schemas

## Reference
<!-- backwards compatibility -->

```json
{
  "name": "com.example.Address",
  "subject": "address-value",
  "version": 1
}

```

A reference from one schema to another. References enable schema composition across subjects. For Avro, this corresponds to named type references. For Protobuf, this corresponds to import statements. For JSON Schema, this corresponds to $ref URIs.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|name|string|true|none|The reference name. For Avro, this is the fully-qualified name of the referenced type. For Protobuf, this is the import path. For JSON Schema, this is the $ref URI.|
|subject|string|true|none|The subject under which the referenced schema is registered.|
|version|integer|true|none|The version of the referenced schema.|

## Metadata
<!-- backwards compatibility -->

```json
{
  "tags": {
    "team": [
      "platform",
      "data-eng"
    ]
  },
  "properties": {
    "owner": "data-platform-team",
    "classification": "internal"
  },
  "sensitive": [
    "ssn",
    "email"
  ]
}

```

Metadata associated with a schema for data contract management. Contains tags for categorization, properties for key-value data, and a list of field names that contain sensitive information.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|tags|object|false|none|A map of tag names to arrays of tag values. Used for categorizing schemas.|
|» **additionalProperties**|[string]|false|none|none|
|properties|object|false|none|A map of property names to string values. Used for attaching arbitrary metadata to schemas.|
|» **additionalProperties**|string|false|none|none|
|sensitive|[string]|false|none|A list of field names that contain sensitive data (e.g. PII). Schema processing tools MAY use this to apply data masking or encryption.|

## Rule
<!-- backwards compatibility -->

```json
{
  "name": "checkSensitiveFields",
  "doc": "Ensures PII fields are encrypted",
  "kind": "CONDITION",
  "mode": "WRITE",
  "type": "CEL",
  "tags": [
    "string"
  ],
  "params": {
    "property1": "string",
    "property2": "string"
  },
  "expr": "message.ssn != ''",
  "onSuccess": "string",
  "onFailure": "string",
  "disabled": false
}

```

A single data contract rule. Rules define validations, transformations, or governance policies applied to data flowing through schemas.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|name|string|true|none|The unique name of this rule.|
|doc|string|false|none|A human-readable description of the rule's purpose.|
|kind|string|true|none|The kind of rule. Common values include CONDITION (validation) and TRANSFORM (data transformation).|
|mode|string|true|none|When the rule is applied in the data flow. Common values include WRITE (applied on produce), READ (applied on consume), and WRITEREAD (applied on both).|
|type|string|false|none|The rule engine type (e.g. CEL, AVRO, JSONATA).|
|tags|[string]|false|none|Tags that this rule applies to.|
|params|object|false|none|Key-value parameters passed to the rule engine.|
|» **additionalProperties**|string|false|none|none|
|expr|string|false|none|The rule expression to evaluate. The syntax depends on the rule `type`.|
|onSuccess|string|false|none|Action to take when the rule evaluates successfully (e.g. NONE, ERROR).|
|onFailure|string|false|none|Action to take when the rule evaluation fails (e.g. NONE, ERROR, DLQ).|
|disabled|boolean|false|none|Whether the rule is currently disabled.|

#### Enumerated Values

|Property|Value|
|---|---|
|kind|CONDITION|
|kind|TRANSFORM|
|mode|WRITE|
|mode|READ|
|mode|WRITEREAD|

## RuleSet
<!-- backwards compatibility -->

```json
{
  "migrationRules": [
    {
      "name": "checkSensitiveFields",
      "doc": "Ensures PII fields are encrypted",
      "kind": "CONDITION",
      "mode": "WRITE",
      "type": "CEL",
      "tags": [
        "string"
      ],
      "params": {
        "property1": "string",
        "property2": "string"
      },
      "expr": "message.ssn != ''",
      "onSuccess": "string",
      "onFailure": "string",
      "disabled": false
    }
  ],
  "domainRules": [
    {
      "name": "checkSensitiveFields",
      "doc": "Ensures PII fields are encrypted",
      "kind": "CONDITION",
      "mode": "WRITE",
      "type": "CEL",
      "tags": [
        "string"
      ],
      "params": {
        "property1": "string",
        "property2": "string"
      },
      "expr": "message.ssn != ''",
      "onSuccess": "string",
      "onFailure": "string",
      "disabled": false
    }
  ]
}

```

A set of data contract rules attached to a schema. Contains migration rules (applied during schema evolution) and domain rules (applied during data processing).

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|migrationRules|[[Rule](#schemarule)]|false|none|Rules applied during schema migration (evolution). These rules govern how data written with an older schema version is transformed when read with a newer version, or vice versa.|
|domainRules|[[Rule](#schemarule)]|false|none|Rules applied during normal data processing. These rules define validation conditions and data transformations.|

## RegisterSchemaRequest
<!-- backwards compatibility -->

```json
{
  "schema": "{\"type\":\"record\",\"name\":\"User\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"}]}",
  "schemaType": "AVRO",
  "references": [
    {
      "name": "com.example.Address",
      "subject": "address-value",
      "version": 1
    }
  ],
  "id": 0,
  "metadata": {
    "tags": {
      "team": [
        "platform",
        "data-eng"
      ]
    },
    "properties": {
      "owner": "data-platform-team",
      "classification": "internal"
    },
    "sensitive": [
      "ssn",
      "email"
    ]
  },
  "ruleSet": {
    "migrationRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ],
    "domainRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ]
  }
}

```

The request body for registering a new schema under a subject.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|schema|string|true|none|The schema definition as a string. For Avro, this is a JSON string. For Protobuf, this is the `.proto` file content. For JSON Schema, this is a JSON Schema document as a string.|
|schemaType|string|false|none|The type of the schema. Defaults to `AVRO` if omitted.|
|references|[[Reference](#schemareference)]|false|none|References to other schemas that this schema depends on.|
|id|integer(int64)|false|none|An explicit schema ID to assign. This is used in IMPORT mode for migrating schemas while preserving their original IDs.|
|metadata|[Metadata](#schemametadata)|false|none|Metadata associated with a schema for data contract management. Contains tags for categorization, properties for key-value data, and a list of field names that contain sensitive information.|
|ruleSet|[RuleSet](#schemaruleset)|false|none|A set of data contract rules attached to a schema. Contains migration rules (applied during schema evolution) and domain rules (applied during data processing).|

#### Enumerated Values

|Property|Value|
|---|---|
|schemaType|AVRO|
|schemaType|PROTOBUF|
|schemaType|JSON|

## RegisterSchemaResponse
<!-- backwards compatibility -->

```json
{
  "id": 1
}

```

The response returned after successfully registering a schema. Contains the globally unique ID assigned to the schema.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|id|integer(int64)|true|none|The globally unique ID assigned to the registered schema.|

## SchemaByIDResponse
<!-- backwards compatibility -->

```json
{
  "schema": "string",
  "schemaType": "AVRO",
  "references": [
    {
      "name": "com.example.Address",
      "subject": "address-value",
      "version": 1
    }
  ],
  "metadata": {
    "tags": {
      "team": [
        "platform",
        "data-eng"
      ]
    },
    "properties": {
      "owner": "data-platform-team",
      "classification": "internal"
    },
    "sensitive": [
      "ssn",
      "email"
    ]
  },
  "ruleSet": {
    "migrationRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ],
    "domainRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ]
  },
  "maxId": 0
}

```

The full response when retrieving a schema by its global ID.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|schema|string|true|none|The schema definition as a string.|
|schemaType|string|true|none|The type of the schema (AVRO, PROTOBUF, or JSON).|
|references|[[Reference](#schemareference)]|false|none|References to other schemas that this schema depends on.|
|metadata|[Metadata](#schemametadata)|false|none|Metadata associated with a schema for data contract management. Contains tags for categorization, properties for key-value data, and a list of field names that contain sensitive information.|
|ruleSet|[RuleSet](#schemaruleset)|false|none|A set of data contract rules attached to a schema. Contains migration rules (applied during schema evolution) and domain rules (applied during data processing).|
|maxId|integer(int64)|false|none|The current maximum schema ID in the registry. Only present when the `fetchMaxId=true` query parameter is set.|

#### Enumerated Values

|Property|Value|
|---|---|
|schemaType|AVRO|
|schemaType|PROTOBUF|
|schemaType|JSON|

## SchemaResponse
<!-- backwards compatibility -->

```json
{
  "schema": "string",
  "schemaType": "AVRO",
  "references": [
    {
      "name": "com.example.Address",
      "subject": "address-value",
      "version": 1
    }
  ]
}

```

A schema response containing the schema string, type, and references.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|schema|string|true|none|The schema definition as a string.|
|schemaType|string|true|none|The type of the schema.|
|references|[[Reference](#schemareference)]|false|none|References to other schemas.|

#### Enumerated Values

|Property|Value|
|---|---|
|schemaType|AVRO|
|schemaType|PROTOBUF|
|schemaType|JSON|

## SubjectVersionResponse
<!-- backwards compatibility -->

```json
{
  "subject": "my-topic-value",
  "id": 1,
  "version": 1,
  "schemaType": "AVRO",
  "schema": "string",
  "references": [
    {
      "name": "com.example.Address",
      "subject": "address-value",
      "version": 1
    }
  ],
  "metadata": {
    "tags": {
      "team": [
        "platform",
        "data-eng"
      ]
    },
    "properties": {
      "owner": "data-platform-team",
      "classification": "internal"
    },
    "sensitive": [
      "ssn",
      "email"
    ]
  },
  "ruleSet": {
    "migrationRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ],
    "domainRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ]
  }
}

```

The detailed response for a specific subject version, including the subject name, version number, schema ID, schema content, and optional metadata.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|subject|string|true|none|The subject name.|
|id|integer(int64)|true|none|The globally unique schema ID.|
|version|integer|true|none|The version number under the subject.|
|schemaType|string|true|none|The type of the schema.|
|schema|string|true|none|The schema definition as a string.|
|references|[[Reference](#schemareference)]|false|none|References to other schemas.|
|metadata|[Metadata](#schemametadata)|false|none|Metadata associated with a schema for data contract management. Contains tags for categorization, properties for key-value data, and a list of field names that contain sensitive information.|
|ruleSet|[RuleSet](#schemaruleset)|false|none|A set of data contract rules attached to a schema. Contains migration rules (applied during schema evolution) and domain rules (applied during data processing).|

#### Enumerated Values

|Property|Value|
|---|---|
|schemaType|AVRO|
|schemaType|PROTOBUF|
|schemaType|JSON|

## LookupSchemaRequest
<!-- backwards compatibility -->

```json
{
  "schema": "string",
  "schemaType": "AVRO",
  "references": [
    {
      "name": "com.example.Address",
      "subject": "address-value",
      "version": 1
    }
  ]
}

```

The request body for looking up whether a schema exists under a subject.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|schema|string|true|none|The schema definition to search for.|
|schemaType|string|false|none|The type of the schema. Defaults to `AVRO` if omitted.|
|references|[[Reference](#schemareference)]|false|none|References to other schemas.|

#### Enumerated Values

|Property|Value|
|---|---|
|schemaType|AVRO|
|schemaType|PROTOBUF|
|schemaType|JSON|

## LookupSchemaResponse
<!-- backwards compatibility -->

```json
{
  "subject": "string",
  "id": 0,
  "version": 0,
  "schemaType": "AVRO",
  "schema": "string",
  "references": [
    {
      "name": "com.example.Address",
      "subject": "address-value",
      "version": 1
    }
  ],
  "metadata": {
    "tags": {
      "team": [
        "platform",
        "data-eng"
      ]
    },
    "properties": {
      "owner": "data-platform-team",
      "classification": "internal"
    },
    "sensitive": [
      "ssn",
      "email"
    ]
  },
  "ruleSet": {
    "migrationRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ],
    "domainRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ]
  }
}

```

The response when a schema lookup finds a match under the specified subject.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|subject|string|true|none|The subject name where the schema was found.|
|id|integer(int64)|true|none|The globally unique schema ID.|
|version|integer|true|none|The version number under the subject.|
|schemaType|string|true|none|The type of the schema.|
|schema|string|true|none|The schema definition as a string.|
|references|[[Reference](#schemareference)]|false|none|References to other schemas.|
|metadata|[Metadata](#schemametadata)|false|none|Metadata associated with a schema for data contract management. Contains tags for categorization, properties for key-value data, and a list of field names that contain sensitive information.|
|ruleSet|[RuleSet](#schemaruleset)|false|none|A set of data contract rules attached to a schema. Contains migration rules (applied during schema evolution) and domain rules (applied during data processing).|

#### Enumerated Values

|Property|Value|
|---|---|
|schemaType|AVRO|
|schemaType|PROTOBUF|
|schemaType|JSON|

## SchemaListItem
<!-- backwards compatibility -->

```json
{
  "subject": "string",
  "version": 0,
  "id": 0,
  "schemaType": "AVRO",
  "schema": "string",
  "references": [
    {
      "name": "com.example.Address",
      "subject": "address-value",
      "version": 1
    }
  ],
  "metadata": {
    "tags": {
      "team": [
        "platform",
        "data-eng"
      ]
    },
    "properties": {
      "owner": "data-platform-team",
      "classification": "internal"
    },
    "sensitive": [
      "ssn",
      "email"
    ]
  },
  "ruleSet": {
    "migrationRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ],
    "domainRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ]
  }
}

```

A single schema in the list schemas response.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|subject|string|true|none|The subject name.|
|version|integer|true|none|The version number.|
|id|integer(int64)|true|none|The globally unique schema ID.|
|schemaType|string|true|none|The type of the schema.|
|schema|string|true|none|The schema definition as a string.|
|references|[[Reference](#schemareference)]|false|none|References to other schemas.|
|metadata|[Metadata](#schemametadata)|false|none|Metadata associated with a schema for data contract management. Contains tags for categorization, properties for key-value data, and a list of field names that contain sensitive information.|
|ruleSet|[RuleSet](#schemaruleset)|false|none|A set of data contract rules attached to a schema. Contains migration rules (applied during schema evolution) and domain rules (applied during data processing).|

#### Enumerated Values

|Property|Value|
|---|---|
|schemaType|AVRO|
|schemaType|PROTOBUF|
|schemaType|JSON|

## SubjectVersionPair
<!-- backwards compatibility -->

```json
{
  "subject": "my-topic-value",
  "version": 1
}

```

A pair identifying a specific subject and version.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|subject|string|true|none|The subject name.|
|version|integer|true|none|The version number.|

## ConfigResponse
<!-- backwards compatibility -->

```json
{
  "compatibilityLevel": "BACKWARD",
  "normalize": true,
  "validateFields": true,
  "alias": "string",
  "compatibilityGroup": "string",
  "defaultMetadata": {
    "tags": {
      "team": [
        "platform",
        "data-eng"
      ]
    },
    "properties": {
      "owner": "data-platform-team",
      "classification": "internal"
    },
    "sensitive": [
      "ssn",
      "email"
    ]
  },
  "overrideMetadata": {
    "tags": {
      "team": [
        "platform",
        "data-eng"
      ]
    },
    "properties": {
      "owner": "data-platform-team",
      "classification": "internal"
    },
    "sensitive": [
      "ssn",
      "email"
    ]
  },
  "defaultRuleSet": {
    "migrationRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ],
    "domainRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ]
  },
  "overrideRuleSet": {
    "migrationRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ],
    "domainRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ]
  }
}

```

The compatibility configuration for a subject or the global default.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|compatibilityLevel|string|true|none|The compatibility level. Determines what schema changes are permitted when registering new versions.|
|normalize|boolean|false|none|Whether schemas are normalized before storage and compatibility checks.|
|validateFields|boolean|false|none|Whether reserved field validation is enforced for this subject.|
|alias|string|false|none|An alias that redirects operations on this subject to another subject.|
|compatibilityGroup|string|false|none|A group name used to partition compatibility checks. Schemas in the same group are checked for compatibility independently from other groups.|
|defaultMetadata|[Metadata](#schemametadata)|false|none|Metadata associated with a schema for data contract management. Contains tags for categorization, properties for key-value data, and a list of field names that contain sensitive information.|
|overrideMetadata|[Metadata](#schemametadata)|false|none|Metadata associated with a schema for data contract management. Contains tags for categorization, properties for key-value data, and a list of field names that contain sensitive information.|
|defaultRuleSet|[RuleSet](#schemaruleset)|false|none|A set of data contract rules attached to a schema. Contains migration rules (applied during schema evolution) and domain rules (applied during data processing).|
|overrideRuleSet|[RuleSet](#schemaruleset)|false|none|A set of data contract rules attached to a schema. Contains migration rules (applied during schema evolution) and domain rules (applied during data processing).|

#### Enumerated Values

|Property|Value|
|---|---|
|compatibilityLevel|NONE|
|compatibilityLevel|BACKWARD|
|compatibilityLevel|BACKWARD_TRANSITIVE|
|compatibilityLevel|FORWARD|
|compatibilityLevel|FORWARD_TRANSITIVE|
|compatibilityLevel|FULL|
|compatibilityLevel|FULL_TRANSITIVE|

## ConfigRequest
<!-- backwards compatibility -->

```json
{
  "compatibility": "NONE",
  "normalize": true,
  "validateFields": true,
  "alias": "string",
  "compatibilityGroup": "string",
  "defaultMetadata": {
    "tags": {
      "team": [
        "platform",
        "data-eng"
      ]
    },
    "properties": {
      "owner": "data-platform-team",
      "classification": "internal"
    },
    "sensitive": [
      "ssn",
      "email"
    ]
  },
  "overrideMetadata": {
    "tags": {
      "team": [
        "platform",
        "data-eng"
      ]
    },
    "properties": {
      "owner": "data-platform-team",
      "classification": "internal"
    },
    "sensitive": [
      "ssn",
      "email"
    ]
  },
  "defaultRuleSet": {
    "migrationRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ],
    "domainRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ]
  },
  "overrideRuleSet": {
    "migrationRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ],
    "domainRules": [
      {
        "name": "checkSensitiveFields",
        "doc": "Ensures PII fields are encrypted",
        "kind": "CONDITION",
        "mode": "WRITE",
        "type": "CEL",
        "tags": [
          "string"
        ],
        "params": {
          "property1": "string",
          "property2": "string"
        },
        "expr": "message.ssn != ''",
        "onSuccess": "string",
        "onFailure": "string",
        "disabled": false
      }
    ]
  }
}

```

The request body for setting compatibility configuration.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|compatibility|string|false|none|The desired compatibility level. MUST be one of the supported levels.|
|normalize|boolean|false|none|Whether schemas SHOULD be normalized before storage and compatibility checks.|
|validateFields|boolean|false|none|Whether reserved field validation SHOULD be enforced.|
|alias|string|false|none|An alias that redirects operations on this subject to another subject.|
|compatibilityGroup|string|false|none|A group name used to partition compatibility checks.|
|defaultMetadata|[Metadata](#schemametadata)|false|none|Metadata associated with a schema for data contract management. Contains tags for categorization, properties for key-value data, and a list of field names that contain sensitive information.|
|overrideMetadata|[Metadata](#schemametadata)|false|none|Metadata associated with a schema for data contract management. Contains tags for categorization, properties for key-value data, and a list of field names that contain sensitive information.|
|defaultRuleSet|[RuleSet](#schemaruleset)|false|none|A set of data contract rules attached to a schema. Contains migration rules (applied during schema evolution) and domain rules (applied during data processing).|
|overrideRuleSet|[RuleSet](#schemaruleset)|false|none|A set of data contract rules attached to a schema. Contains migration rules (applied during schema evolution) and domain rules (applied during data processing).|

#### Enumerated Values

|Property|Value|
|---|---|
|compatibility|NONE|
|compatibility|BACKWARD|
|compatibility|BACKWARD_TRANSITIVE|
|compatibility|FORWARD|
|compatibility|FORWARD_TRANSITIVE|
|compatibility|FULL|
|compatibility|FULL_TRANSITIVE|

## ModeResponse
<!-- backwards compatibility -->

```json
{
  "mode": "READWRITE"
}

```

The mode for a subject or the global default.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|mode|string|true|none|The current mode. READWRITE allows normal operations. READONLY blocks schema registration. READONLY_OVERRIDE blocks registration but allows configuration changes. IMPORT allows importing schemas with explicit IDs.|

#### Enumerated Values

|Property|Value|
|---|---|
|mode|READWRITE|
|mode|READONLY|
|mode|READONLY_OVERRIDE|
|mode|IMPORT|

## ModeRequest
<!-- backwards compatibility -->

```json
{
  "mode": "READWRITE"
}

```

The request body for setting the registry mode.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|mode|string|true|none|The desired mode. MUST be one of the supported mode values.|

#### Enumerated Values

|Property|Value|
|---|---|
|mode|READWRITE|
|mode|READONLY|
|mode|READONLY_OVERRIDE|
|mode|IMPORT|

## CompatibilityCheckRequest
<!-- backwards compatibility -->

```json
{
  "schema": "string",
  "schemaType": "AVRO",
  "references": [
    {
      "name": "com.example.Address",
      "subject": "address-value",
      "version": 1
    }
  ]
}

```

The request body for checking schema compatibility. Contains the candidate schema to test against existing versions.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|schema|string|true|none|The candidate schema definition to check for compatibility.|
|schemaType|string|false|none|The type of the candidate schema. Defaults to `AVRO` if omitted.|
|references|[[Reference](#schemareference)]|false|none|References to other schemas that the candidate schema depends on.|

#### Enumerated Values

|Property|Value|
|---|---|
|schemaType|AVRO|
|schemaType|PROTOBUF|
|schemaType|JSON|

## CompatibilityCheckResponse
<!-- backwards compatibility -->

```json
{
  "is_compatible": true,
  "messages": [
    "string"
  ]
}

```

The result of a compatibility check.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|is_compatible|boolean|true|none|Whether the candidate schema is compatible with the existing schema(s) according to the configured compatibility policy.|
|messages|[string]|false|none|Detailed messages describing compatibility issues. Only populated when the `verbose=true` query parameter is set and the schema is incompatible.|

## ImportSchemasRequest
<!-- backwards compatibility -->

```json
{
  "schemas": [
    {
      "id": 0,
      "subject": "string",
      "version": 0,
      "schemaType": "AVRO",
      "schema": "string",
      "references": [
        {
          "name": "com.example.Address",
          "subject": "address-value",
          "version": 1
        }
      ]
    }
  ]
}

```

The request body for bulk-importing schemas from another registry.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|schemas|[[ImportSchemaRequest](#schemaimportschemarequest)]|true|none|The list of schemas to import. Each entry MUST include an explicit ID, subject, version, and schema content.|

## ImportSchemaRequest
<!-- backwards compatibility -->

```json
{
  "id": 0,
  "subject": "string",
  "version": 0,
  "schemaType": "AVRO",
  "schema": "string",
  "references": [
    {
      "name": "com.example.Address",
      "subject": "address-value",
      "version": 1
    }
  ]
}

```

A single schema to import with a specific ID.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|id|integer(int64)|true|none|The original schema ID to preserve during import.|
|subject|string|true|none|The subject to register the schema under.|
|version|integer|true|none|The version number to assign.|
|schemaType|string|false|none|The type of the schema. Defaults to `AVRO` if omitted.|
|schema|string|true|none|The schema definition as a string.|
|references|[[Reference](#schemareference)]|false|none|References to other schemas.|

#### Enumerated Values

|Property|Value|
|---|---|
|schemaType|AVRO|
|schemaType|PROTOBUF|
|schemaType|JSON|

## ImportSchemasResponse
<!-- backwards compatibility -->

```json
{
  "imported": 10,
  "errors": 2,
  "results": [
    {
      "id": 0,
      "subject": "string",
      "version": 0,
      "success": true,
      "error": "string"
    }
  ]
}

```

The response for a bulk import operation. Includes counts of successful and failed imports, along with individual results for each schema.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|imported|integer|true|none|The number of schemas successfully imported.|
|errors|integer|true|none|The number of schemas that failed to import.|
|results|[[ImportSchemaResult](#schemaimportschemaresult)]|true|none|Individual import results for each schema in the request.|

## ImportSchemaResult
<!-- backwards compatibility -->

```json
{
  "id": 0,
  "subject": "string",
  "version": 0,
  "success": true,
  "error": "string"
}

```

The result of importing a single schema.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|id|integer(int64)|true|none|The schema ID.|
|subject|string|true|none|The subject the schema was imported under.|
|version|integer|true|none|The version number.|
|success|boolean|true|none|Whether the import was successful.|
|error|string|false|none|The error message if the import failed. Empty when `success` is `true`.|

## ServerClusterIDResponse
<!-- backwards compatibility -->

```json
{
  "id": "default-cluster"
}

```

The cluster ID of this schema registry instance.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|id|string|true|none|The cluster identifier string.|

## ServerVersionResponse
<!-- backwards compatibility -->

```json
{
  "version": "1.0.0",
  "commit": "abc123def",
  "build_time": "2025-01-15T10:30:00Z"
}

```

Version information for the running server.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|version|string|true|none|The server version string.|
|commit|string|false|none|The git commit hash of the build.|
|build_time|string|false|none|The build timestamp in RFC 3339 format.|

## ErrorResponse
<!-- backwards compatibility -->

```json
{
  "error_code": 40401,
  "message": "Subject 'my-topic-value' not found"
}

```

The standard error response format for all API errors. Error codes follow the Confluent Schema Registry convention. Common error codes include:
| Code  | Meaning                       | |-------|-------------------------------| | 40101 | Unauthorized                  | | 40103 | API key expired               | | 40104 | API key disabled              | | 40105 | User disabled                 | | 40301 | Forbidden                     | | 40401 | Subject not found             | | 40402 | Version not found             | | 40403 | Schema not found              | | 40404 | Subject soft-deleted          | | 40405 | Subject not soft-deleted      | | 40406 | Schema version soft-deleted   | | 40407 | Version not soft-deleted       | | 40408 | Subject compat config not found | | 40409 | Subject mode not found        | | 409   | Incompatible schema           | | 40901 | User already exists           | | 40902 | API key already exists        | | 42201 | Invalid schema                | | 42202 | Invalid schema type or version | | 42203 | Invalid compatibility level   | | 42204 | Invalid mode                  | | 42205 | Operation not permitted       | | 42206 | Reference exists              | | 42207 | Invalid role                  | | 42208 | Invalid password              | | 50001 | Internal server error         | | 50002 | Storage error                 |

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|error_code|integer|true|none|The application-specific error code. These codes provide more granular detail than HTTP status codes.|
|message|string|true|none|A human-readable description of the error.|

## CreateUserRequest
<!-- backwards compatibility -->

```json
{
  "username": "johndoe",
  "email": "johndoe@example.com",
  "password": "SecureP@ss123",
  "role": "developer",
  "enabled": true
}

```

The request body for creating a new user.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|username|string|true|none|The unique username for the new user.|
|email|string|false|none|The email address for the user.|
|password|string(password)|true|none|The password for the new user.|
|role|string|true|none|The role to assign to the user. Determines the user's permissions.|
|enabled|boolean|false|none|Whether the user account is enabled. Defaults to `true` if omitted.|

#### Enumerated Values

|Property|Value|
|---|---|
|role|super_admin|
|role|admin|
|role|developer|
|role|readonly|

## UpdateUserRequest
<!-- backwards compatibility -->

```json
{
  "email": "string",
  "password": "pa$$word",
  "role": "super_admin",
  "enabled": true
}

```

The request body for updating an existing user. Only provided fields are updated; omitted fields remain unchanged.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|email|string|false|none|The new email address.|
|password|string(password)|false|none|The new password.|
|role|string|false|none|The new role.|
|enabled|boolean|false|none|Whether the user account is enabled.|

#### Enumerated Values

|Property|Value|
|---|---|
|role|super_admin|
|role|admin|
|role|developer|
|role|readonly|

## UserResponse
<!-- backwards compatibility -->

```json
{
  "id": 1,
  "username": "johndoe",
  "email": "johndoe@example.com",
  "role": "developer",
  "enabled": true,
  "created_at": "2025-01-15T10:30:00Z",
  "updated_at": "2025-01-15T10:30:00Z"
}

```

The representation of a user in API responses.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|id|integer(int64)|true|none|The unique user ID.|
|username|string|true|none|The username.|
|email|string|false|none|The email address.|
|role|string|true|none|The user's role.|
|enabled|boolean|true|none|Whether the user account is enabled.|
|created_at|string(date-time)|true|none|The timestamp when the user was created (RFC 3339).|
|updated_at|string(date-time)|true|none|The timestamp when the user was last updated (RFC 3339).|

## UsersListResponse
<!-- backwards compatibility -->

```json
{
  "users": [
    {
      "id": 1,
      "username": "johndoe",
      "email": "johndoe@example.com",
      "role": "developer",
      "enabled": true,
      "created_at": "2025-01-15T10:30:00Z",
      "updated_at": "2025-01-15T10:30:00Z"
    }
  ]
}

```

The response for listing all users.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|users|[[UserResponse](#schemauserresponse)]|true|none|The list of users.|

## ChangePasswordRequest
<!-- backwards compatibility -->

```json
{
  "old_password": "pa$$word",
  "new_password": "pa$$word"
}

```

The request body for changing the current user's password.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|old_password|string(password)|true|none|The current password for verification.|
|new_password|string(password)|true|none|The desired new password.|

## CreateAPIKeyRequest
<!-- backwards compatibility -->

```json
{
  "name": "ci-pipeline-key",
  "role": "developer",
  "expires_in": 2592000,
  "for_user_id": 0
}

```

The request body for creating a new API key.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|name|string|true|none|A human-readable name for the API key. MUST be unique per user.|
|role|string|true|none|The role to assign to the API key. Determines what operations the key can perform.|
|expires_in|integer(int64)|true|none|The duration in seconds until the API key expires. For example, 2592000 for 30 days. MUST be a positive integer.|
|for_user_id|integer(int64)|false|none|The ID of the user who SHOULD own this API key. Only super admins MAY create API keys for other users. If omitted, the key is created for the authenticated user.|

#### Enumerated Values

|Property|Value|
|---|---|
|role|super_admin|
|role|admin|
|role|developer|
|role|readonly|

## UpdateAPIKeyRequest
<!-- backwards compatibility -->

```json
{
  "name": "string",
  "role": "super_admin",
  "enabled": true
}

```

The request body for updating an existing API key. Only provided fields are updated; omitted fields remain unchanged.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|name|string|false|none|The new name for the API key.|
|role|string|false|none|The new role for the API key.|
|enabled|boolean|false|none|Whether the API key is enabled.|

#### Enumerated Values

|Property|Value|
|---|---|
|role|super_admin|
|role|admin|
|role|developer|
|role|readonly|

## APIKeyResponse
<!-- backwards compatibility -->

```json
{
  "id": 1,
  "key_prefix": "axon_abc",
  "name": "ci-pipeline-key",
  "role": "developer",
  "user_id": 1,
  "username": "johndoe",
  "enabled": true,
  "created_at": "2025-01-15T10:30:00Z",
  "expires_at": "2025-02-14T10:30:00Z",
  "last_used": "2019-08-24T14:15:22Z"
}

```

The representation of an API key in API responses. The raw key secret is never included; only the key prefix is shown for identification.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|id|integer(int64)|true|none|The unique API key ID.|
|key_prefix|string|true|none|The first 8 characters of the API key for display and identification.|
|name|string|true|none|The human-readable name of the API key.|
|role|string|true|none|The role assigned to this API key.|
|user_id|integer(int64)|true|none|The ID of the user who owns this API key.|
|username|string|true|none|The username of the user who owns this API key.|
|enabled|boolean|true|none|Whether the API key is currently enabled.|
|created_at|string(date-time)|true|none|The timestamp when the API key was created (RFC 3339).|
|expires_at|string(date-time)|true|none|The timestamp when the API key expires (RFC 3339).|
|last_used|string(date-time)|false|none|The timestamp when the API key was last used for authentication (RFC 3339). May be null if the key has never been used.|

## CreateAPIKeyResponse
<!-- backwards compatibility -->

```json
{
  "id": 0,
  "key": "axon_abcdefghijklmnopqrstuvwxyz1234567890",
  "key_prefix": "string",
  "name": "string",
  "role": "string",
  "user_id": 0,
  "username": "string",
  "enabled": true,
  "created_at": "2019-08-24T14:15:22Z",
  "expires_at": "2019-08-24T14:15:22Z"
}

```

The response returned when creating a new API key. This is the ONLY time the raw API key secret is included in a response. Clients MUST store the key securely.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|id|integer(int64)|true|none|The unique API key ID.|
|key|string|true|none|The raw API key secret. This value is shown ONLY once at creation time and cannot be retrieved later.|
|key_prefix|string|true|none|The first 8 characters of the API key for identification.|
|name|string|true|none|The name of the API key.|
|role|string|true|none|The role assigned to this API key.|
|user_id|integer(int64)|true|none|The ID of the user who owns this API key.|
|username|string|true|none|The username of the user who owns this API key.|
|enabled|boolean|true|none|Whether the API key is enabled.|
|created_at|string(date-time)|true|none|The creation timestamp (RFC 3339).|
|expires_at|string(date-time)|true|none|The expiration timestamp (RFC 3339).|

## APIKeysListResponse
<!-- backwards compatibility -->

```json
{
  "api_keys": [
    {
      "id": 1,
      "key_prefix": "axon_abc",
      "name": "ci-pipeline-key",
      "role": "developer",
      "user_id": 1,
      "username": "johndoe",
      "enabled": true,
      "created_at": "2025-01-15T10:30:00Z",
      "expires_at": "2025-02-14T10:30:00Z",
      "last_used": "2019-08-24T14:15:22Z"
    }
  ]
}

```

The response for listing API keys.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|api_keys|[[APIKeyResponse](#schemaapikeyresponse)]|true|none|The list of API keys.|

## RotateAPIKeyRequest
<!-- backwards compatibility -->

```json
{
  "expires_in": 2592000
}

```

The request body for rotating an API key. The old key is revoked and a new key is created with the same name and role.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|expires_in|integer(int64)|true|none|The duration in seconds until the new API key expires. MUST be a positive integer.|

## RotateAPIKeyResponse
<!-- backwards compatibility -->

```json
{
  "new_key": {
    "id": 0,
    "key": "axon_abcdefghijklmnopqrstuvwxyz1234567890",
    "key_prefix": "string",
    "name": "string",
    "role": "string",
    "user_id": 0,
    "username": "string",
    "enabled": true,
    "created_at": "2019-08-24T14:15:22Z",
    "expires_at": "2019-08-24T14:15:22Z"
  },
  "revoked_id": 1
}

```

The response for rotating an API key. Contains the new key (with its raw secret) and the ID of the revoked key.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|new_key|[CreateAPIKeyResponse](#schemacreateapikeyresponse)|true|none|The response returned when creating a new API key. This is the ONLY time the raw API key secret is included in a response. Clients MUST store the key securely.|
|revoked_id|integer(int64)|true|none|The ID of the API key that was revoked.|

## RoleInfo
<!-- backwards compatibility -->

```json
{
  "name": "developer",
  "description": "Can register and read schemas",
  "permissions": [
    "schema:read",
    "schema:write",
    "subject:read"
  ]
}

```

Information about a role, including its name, description, and the permissions it grants.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|name|string|true|none|The role name.|
|description|string|true|none|A human-readable description of the role.|
|permissions|[string]|true|none|The list of permissions granted by this role.|

## RolesListResponse
<!-- backwards compatibility -->

```json
{
  "roles": [
    {
      "name": "developer",
      "description": "Can register and read schemas",
      "permissions": [
        "schema:read",
        "schema:write",
        "subject:read"
      ]
    }
  ]
}

```

The response for listing available roles.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|roles|[[RoleInfo](#schemaroleinfo)]|true|none|The list of available roles.|
