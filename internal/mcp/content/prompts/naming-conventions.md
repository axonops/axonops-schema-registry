Guide to subject naming strategies in the schema registry.

## Naming strategies

### topic_name (default)
Pattern: `{topic}-key` or `{topic}-value`
Examples: `orders-key`, `orders-value`, `user-events-value`
Use when: one schema per Kafka topic, simple key/value distinction.

### record_name
Pattern: `{fully.qualified.RecordName}`
Examples: `com.example.Order`, `com.example.UserEvent`
Use when: multiple event types share a topic, schemas identified by record name.

### topic_record_name
Pattern: `{topic}-{fully.qualified.RecordName}`
Examples: `orders-com.example.OrderCreated`, `orders-com.example.OrderShipped`
Use when: multiple event types per topic and you want topic context in the subject name.

## Validation
Use **validate_subject_name** to check if a name conforms to a strategy:
- validate_subject_name(subject: "orders-value", strategy: "topic_name")
- validate_subject_name(subject: "com.example.Order", strategy: "record_name")

## Best practices
- Pick one strategy per environment and use it consistently
- Use **detect_schema_patterns** to check current naming convention coverage
- Use **match_subjects** to find subjects that deviate from the dominant pattern
- Avoid mixing strategies in the same registry context
- Use lowercase with hyphens for topic names: `user-events` not `UserEvents`
- Use reverse-domain namespace for record names: `com.company.domain.Type`
