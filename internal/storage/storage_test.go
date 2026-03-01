package storage

import "testing"

func TestParseSchemaType(t *testing.T) {
	tests := []struct {
		input   string
		want    SchemaType
		wantOK  bool
	}{
		{"", SchemaTypeAvro, true},
		{"AVRO", SchemaTypeAvro, true},
		{"PROTOBUF", SchemaTypeProtobuf, true},
		{"JSON", SchemaTypeJSON, true},
		{"avro", "", false},
		{"json", "", false},
		{"protobuf", "", false},
		{"xml", "", false},
		{"Avro", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, ok := ParseSchemaType(tt.input)
			if ok != tt.wantOK {
				t.Fatalf("ParseSchemaType(%q) ok = %v, want %v", tt.input, ok, tt.wantOK)
			}
			if got != tt.want {
				t.Fatalf("ParseSchemaType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
