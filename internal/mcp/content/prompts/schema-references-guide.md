Guide for cross-subject schema references.

## What Are References?
References allow one schema to depend on another schema registered in a different subject. This enables reusable, independently versioned type definitions.

## Reference Structure
Each reference has three fields:
- **name** -- how the referencing schema refers to this dependency
- **subject** -- subject where the referenced schema is registered
- **version** -- version number of the referenced schema

## Per-Format Name Semantics

### Avro
The **name** field is the fully qualified type name (namespace + name):
    name: "com.example.Address"
    subject: "com.example.Address"
    version: 1

In the schema, reference it by its fully qualified name in the type field.

### Protobuf
The **name** field is the import path:
    name: "address.proto"
    subject: "address-value"
    version: 1

In the .proto file, use: import "address.proto";

### JSON Schema
The **name** field is the reference URL:
    name: "address.json"
    subject: "address-value"
    version: 1

In the schema, use: "$ref": "address.json"

## Registering with References
Use **register_schema** with the references array:
    register_schema(
      subject: "order-value",
      schema: "...",
      schema_type: "AVRO",
      references: [
        {"name": "com.example.Address", "subject": "address-value", "version": 1}
      ]
    )

## Important Rules
1. Referenced schemas MUST be registered before the schemas that depend on them.
2. A schema that is referenced by others cannot be permanently deleted.
3. Use **get_referenced_by** to find all schemas that reference a given subject.
4. Use **get_dependency_graph** to visualize the full reference tree.
5. Use FULL_TRANSITIVE compatibility for shared referenced types.

## Resolving References
Pass ?referenceFormat=RESOLVED when fetching schemas to get resolved (inline) references.

For domain knowledge, read: schema://glossary/schema-types
