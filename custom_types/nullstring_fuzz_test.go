package customtypes

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

// FuzzNullStringUnmarshalJSON tests the UnmarshalJSON method with various JSON inputs
func FuzzNullStringUnmarshalJSON(f *testing.F) {
	// Seed with common JSON string patterns
	f.Add(`"test"`)
	f.Add(`""`)
	f.Add(`"hello world"`)
	f.Add(`"unicode: αβγ"`)
	f.Add(`"emoji: 🚀"`)
	f.Add(`"with \"quotes\""`)
	f.Add(`"with 'apostrophes'"`)
	f.Add(`"with \n newlines"`)
	f.Add(`"with \t tabs"`)
	f.Add(`"with \r carriage returns"`)
	f.Add(`"with \\ backslashes"`)
	f.Add(`"JSON injection: {\"key\": \"value\"}"`)
	f.Add(`"null"`)
	f.Add(`"true"`)
	f.Add(`"false"`)
	f.Add(`"123"`)
	f.Add(`"0"`)
	f.Add(`"-1"`)
	f.Add(`"special chars: !@#$%^&*()"`)
	f.Add(`"\u0000"`)
	f.Add(`"\u001f"`)
	f.Add(`"\uffff"`)

	f.Fuzz(func(t *testing.T, jsonData string) {
		var ns NullString

		// Test unmarshaling
		err := ns.UnmarshalJSON([]byte(jsonData))
		// Check for consistent behavior
		if err != nil {
			// If there's an error, the NullString should be invalid
			if ns.Valid {
				t.Errorf("NullString should be invalid when UnmarshalJSON returns error")
			}
			return
		}

		// If no error, NullString should be valid
		if !ns.Valid {
			t.Errorf("NullString should be valid when UnmarshalJSON succeeds")
		}

		// Test roundtrip: marshal and unmarshal should give same result
		marshaled, marshalErr := ns.MarshalJSON()
		if marshalErr != nil {
			t.Errorf("MarshalJSON failed for valid NullString: %v", marshalErr)
			return
		}

		var ns2 NullString
		unmarshalErr := ns2.UnmarshalJSON(marshaled)
		if unmarshalErr != nil {
			t.Errorf("Roundtrip unmarshal failed: %v", unmarshalErr)
			return
		}

		if ns.Valid != ns2.Valid || (ns.Valid && ns.String != ns2.String) {
			t.Errorf("Roundtrip failed: original=%+v, roundtrip=%+v", ns, ns2)
		}
	})
}

// FuzzNullStringUnmarshalBSONValue tests the UnmarshalBSONValue method with various BSON inputs
func FuzzNullStringUnmarshalBSONValue(f *testing.F) {
	// Seed with various string values that will be encoded as BSON
	f.Add("test")
	f.Add("")
	f.Add("hello world")
	f.Add("unicode: αβγ")
	f.Add("emoji: 🚀")
	f.Add("with \"quotes\"")
	f.Add("with 'apostrophes'")
	f.Add("with \n newlines")
	f.Add("with \t tabs")
	f.Add("JSON: {\"key\": \"value\"}")
	f.Add("null")
	f.Add("true")
	f.Add("false")
	f.Add("123")
	f.Add("special chars: !@#$%^&*()")
	f.Add("\x00\x01\x02")

	f.Fuzz(func(t *testing.T, input string) {
		// Create BSON data for the input string
		bsonData, err := bson.Marshal(bson.D{{Key: "value", Value: input}})
		if err != nil {
			t.Skipf("Failed to marshal BSON: %v", err)
		}

		// Extract the value bytes from the BSON document
		var doc bson.D
		if unmarshalErr := bson.Unmarshal(bsonData, &doc); unmarshalErr != nil {
			t.Skipf("Failed to unmarshal BSON doc: %v", unmarshalErr)
		}

		if len(doc) == 0 {
			t.Skip("Empty BSON document")
		}

		// Get the raw value
		rawValue := bson.RawValue{Type: bson.TypeString}
		valueBytes, err := bson.Marshal(doc[0].Value)
		if err != nil {
			t.Skipf("Failed to marshal value: %v", err)
		}
		rawValue.Value = valueBytes

		var ns NullString

		// Test unmarshaling
		err = ns.UnmarshalBSONValue(rawValue.Type, rawValue.Value)
		if err != nil {
			// Some inputs might be invalid BSON
			if ns.Valid {
				t.Errorf("NullString should be invalid when UnmarshalBSONValue returns error")
			}
			return
		}

		// If no error, check validity
		if !ns.Valid {
			t.Errorf("NullString should be valid when UnmarshalBSONValue succeeds")
		}

		// The string value should match the input
		if ns.String != input {
			t.Errorf("String value mismatch: expected %q, got %q", input, ns.String)
		}

		// Test roundtrip: marshal and unmarshal should give same result
		bsonType, marshaled, marshalErr := ns.MarshalBSONValue()
		if marshalErr != nil {
			t.Errorf("MarshalBSONValue failed for valid NullString: %v", marshalErr)
			return
		}

		var ns2 NullString
		unmarshalErr := ns2.UnmarshalBSONValue(bsonType, marshaled)
		if unmarshalErr != nil {
			t.Errorf("Roundtrip unmarshal failed: %v", unmarshalErr)
			return
		}

		if ns.Valid != ns2.Valid || (ns.Valid && ns.String != ns2.String) {
			t.Errorf("Roundtrip failed: original=%+v, roundtrip=%+v", ns, ns2)
		}
	})
}

// FuzzNullStringNullValues tests behavior with null/nil inputs
func FuzzNullStringNullValues(f *testing.F) {
	f.Add([]byte("null"))
	f.Add([]byte(""))
	f.Add([]byte("\"\""))

	f.Fuzz(func(t *testing.T, data []byte) {
		var ns NullString

		// Test JSON null handling
		err := ns.UnmarshalJSON(data)
		if err != nil && string(data) == "null" {
			t.Errorf("UnmarshalJSON should handle null gracefully")
		}

		// Test nil data
		var ns2 NullString
		err2 := ns2.UnmarshalJSON(nil)
		if err2 != nil {
			t.Errorf("UnmarshalJSON should handle nil data gracefully: %v", err2)
		}
		if ns2.Valid {
			t.Errorf("NullString should be invalid for nil JSON data")
		}

		// Test BSON null handling
		var ns3 NullString
		err3 := ns3.UnmarshalBSONValue(bson.TypeNull, nil)
		if err3 != nil {
			t.Errorf("UnmarshalBSONValue should handle null type gracefully: %v", err3)
		}
		if ns3.Valid {
			t.Errorf("NullString should be invalid for BSON null type")
		}
	})
}
