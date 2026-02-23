package serde_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Migration Rule Tests — JSONata transforms applied during schema evolution
//
// Pattern:
//   1. Set compatibility to NONE (so v2 can break v1 freely).
//   2. Register v1 schema, serialize data with v1 as latest.
//   3. Register v2 schema with a JSONata migration rule.
//   4. Create a FRESH client+deserializer (avoids cached v1 metadata).
//   5. Deserialize the v1-encoded bytes — the migration rule transforms the
//      payload into the v2 shape automatically.
// =============================================================================

func TestUpgradeFieldRename(t *testing.T) {
	subject := uniqueSubject("migrate-rename")
	defer deleteSubject(t, subject)

	// Set compatibility to NONE so we can make a breaking change.
	setSubjectConfig(t, subject, `{"compatibility":"NONE"}`)

	// -- v1: Order with "state" field --
	v1Schema := `{
		"type": "record",
		"name": "OrderV1",
		"namespace": "com.example",
		"fields": [
			{"name": "orderId", "type": "string"},
			{"name": "state",   "type": "string"}
		]
	}`
	v1Body := `{"schema": ` + jsonQuote(v1Schema) + `}`
	registerSchemaViaHTTP(t, subject, v1Body)

	// Serialize with v1 as latest.
	client1 := newClient(t)
	ser := newRuleSerializer(t, client1)
	original := OrderV1{OrderID: "ORD-001", State: "PENDING"}
	bytes, err := ser.Serialize(topicFromSubject(subject), &original)
	require.NoError(t, err, "serialize OrderV1")

	// -- v2: Order with "status" field + UPGRADE migration rule --
	v2Schema := `{
		"type": "record",
		"name": "OrderV2",
		"namespace": "com.example",
		"fields": [
			{"name": "orderId", "type": "string"},
			{"name": "status",  "type": "string"}
		]
	}`
	v2Body := `{
		"schema": ` + jsonQuote(v2Schema) + `,
		"schemaType": "AVRO",
		"ruleSet": {
			"migrationRules": [{
				"name": "renameStateToStatus",
				"kind": "TRANSFORM",
				"type": "JSONATA",
				"mode": "UPGRADE",
				"expr": "$merge([$sift($, function($v, $k) {$k != 'state'}), {'status': $.state}])"
			}]
		}
	}`
	registerSchemaViaHTTP(t, subject, v2Body)

	// Fresh client + deserializer to pick up v2 metadata.
	client2 := newClient(t)
	deser := newRuleDeserializer(t, client2)

	var result OrderV2
	err = deser.DeserializeInto(topicFromSubject(subject), bytes, &result)
	require.NoError(t, err, "deserialize into OrderV2")

	assert.Equal(t, "ORD-001", result.OrderID)
	assert.Equal(t, "PENDING", result.Status, "migration should rename state->status")
}

func TestBidirectionalUpgradeDowngrade(t *testing.T) {
	subject := uniqueSubject("migrate-bidir")
	defer deleteSubject(t, subject)

	setSubjectConfig(t, subject, `{"compatibility":"NONE"}`)

	// -- v1 schema --
	v1Schema := `{
		"type": "record",
		"name": "OrderV1",
		"namespace": "com.example",
		"fields": [
			{"name": "orderId", "type": "string"},
			{"name": "state",   "type": "string"}
		]
	}`
	v1Body := `{"schema": ` + jsonQuote(v1Schema) + `}`
	registerSchemaViaHTTP(t, subject, v1Body)

	// Serialize with v1 as latest.
	client1 := newClient(t)
	ser := newRuleSerializer(t, client1)
	original := OrderV1{OrderID: "ORD-002", State: "SHIPPED"}
	bytes, err := ser.Serialize(topicFromSubject(subject), &original)
	require.NoError(t, err, "serialize OrderV1")

	// -- v2 schema with BOTH upgrade and downgrade rules --
	v2Schema := `{
		"type": "record",
		"name": "OrderV2",
		"namespace": "com.example",
		"fields": [
			{"name": "orderId", "type": "string"},
			{"name": "status",  "type": "string"}
		]
	}`
	v2Body := `{
		"schema": ` + jsonQuote(v2Schema) + `,
		"schemaType": "AVRO",
		"ruleSet": {
			"migrationRules": [
				{
					"name": "upgradeStateToStatus",
					"kind": "TRANSFORM",
					"type": "JSONATA",
					"mode": "UPGRADE",
					"expr": "$merge([$sift($, function($v, $k) {$k != 'state'}), {'status': $.state}])"
				},
				{
					"name": "downgradeStatusToState",
					"kind": "TRANSFORM",
					"type": "JSONATA",
					"mode": "DOWNGRADE",
					"expr": "$merge([$sift($, function($v, $k) {$k != 'status'}), {'state': $.status}])"
				}
			]
		}
	}`
	registerSchemaViaHTTP(t, subject, v2Body)

	// Verify both rules are stored in the schema version response.
	versionResp := getSchemaVersionResponse(t, subject, 2)
	assert.Contains(t, versionResp, "upgradeStateToStatus",
		"v2 response should contain upgrade rule name")
	assert.Contains(t, versionResp, "downgradeStatusToState",
		"v2 response should contain downgrade rule name")

	// Fresh client + deserializer.
	client2 := newClient(t)
	deser := newRuleDeserializer(t, client2)

	var result OrderV2
	err = deser.DeserializeInto(topicFromSubject(subject), bytes, &result)
	require.NoError(t, err, "deserialize into OrderV2")

	assert.Equal(t, "ORD-002", result.OrderID)
	assert.Equal(t, "SHIPPED", result.Status, "upgrade migration should rename state->status")
}

func TestUpgradeFieldAdditionWithDefault(t *testing.T) {
	subject := uniqueSubject("migrate-addfield")
	defer deleteSubject(t, subject)

	setSubjectConfig(t, subject, `{"compatibility":"NONE"}`)

	// -- v1: Payment with id + amount --
	v1Schema := `{
		"type": "record",
		"name": "PaymentV1",
		"namespace": "com.example",
		"fields": [
			{"name": "id",     "type": "string"},
			{"name": "amount", "type": "double"}
		]
	}`
	v1Body := `{"schema": ` + jsonQuote(v1Schema) + `}`
	registerSchemaViaHTTP(t, subject, v1Body)

	// Serialize with v1 as latest.
	client1 := newClient(t)
	ser := newRuleSerializer(t, client1)
	original := PaymentV1{ID: "PAY-001", Amount: 99.99}
	bytes, err := ser.Serialize(topicFromSubject(subject), &original)
	require.NoError(t, err, "serialize PaymentV1")

	// -- v2: Payment with id + amount + currency, migration sets currency to "USD" --
	v2Schema := `{
		"type": "record",
		"name": "PaymentV2",
		"namespace": "com.example",
		"fields": [
			{"name": "id",       "type": "string"},
			{"name": "amount",   "type": "double"},
			{"name": "currency", "type": "string", "default": "UNKNOWN"}
		]
	}`
	v2Body := `{
		"schema": ` + jsonQuote(v2Schema) + `,
		"schemaType": "AVRO",
		"ruleSet": {
			"migrationRules": [{
				"name": "addCurrencyDefault",
				"kind": "TRANSFORM",
				"type": "JSONATA",
				"mode": "UPGRADE",
				"expr": "$merge([$, {'currency': 'USD'}])"
			}]
		}
	}`
	registerSchemaViaHTTP(t, subject, v2Body)

	// Fresh client + deserializer.
	client2 := newClient(t)
	deser := newRuleDeserializer(t, client2)

	var result PaymentV2
	err = deser.DeserializeInto(topicFromSubject(subject), bytes, &result)
	require.NoError(t, err, "deserialize into PaymentV2")

	assert.Equal(t, "PAY-001", result.ID)
	assert.Equal(t, 99.99, result.Amount)
	assert.Equal(t, "USD", result.Currency, "migration should set currency to USD")
}

func TestBreakingChangeBridgedByMigration(t *testing.T) {
	subject := uniqueSubject("migrate-breaking")
	defer deleteSubject(t, subject)

	setSubjectConfig(t, subject, `{"compatibility":"NONE"}`)

	// -- v1: Person with firstName + lastName --
	v1Schema := `{
		"type": "record",
		"name": "PersonV1",
		"namespace": "com.example",
		"fields": [
			{"name": "firstName", "type": "string"},
			{"name": "lastName",  "type": "string"}
		]
	}`
	v1Body := `{"schema": ` + jsonQuote(v1Schema) + `}`
	registerSchemaViaHTTP(t, subject, v1Body)

	// Serialize with v1 as latest.
	client1 := newClient(t)
	ser := newRuleSerializer(t, client1)
	original := PersonV1{FirstName: "John", LastName: "Doe"}
	bytes, err := ser.Serialize(topicFromSubject(subject), &original)
	require.NoError(t, err, "serialize PersonV1")

	// -- v2: Person with fullName only — completely breaking change --
	v2Schema := `{
		"type": "record",
		"name": "PersonV2",
		"namespace": "com.example",
		"fields": [
			{"name": "fullName", "type": "string"}
		]
	}`
	v2Body := `{
		"schema": ` + jsonQuote(v2Schema) + `,
		"schemaType": "AVRO",
		"ruleSet": {
			"migrationRules": [{
				"name": "mergeNames",
				"kind": "TRANSFORM",
				"type": "JSONATA",
				"mode": "UPGRADE",
				"expr": "{'fullName': $.firstName & ' ' & $.lastName}"
			}]
		}
	}`
	registerSchemaViaHTTP(t, subject, v2Body)

	// Fresh client + deserializer.
	client2 := newClient(t)
	deser := newRuleDeserializer(t, client2)

	var result PersonV2
	err = deser.DeserializeInto(topicFromSubject(subject), bytes, &result)
	require.NoError(t, err, "deserialize into PersonV2")

	assert.Equal(t, "John Doe", result.FullName,
		"migration should concatenate firstName+lastName into fullName")
}
