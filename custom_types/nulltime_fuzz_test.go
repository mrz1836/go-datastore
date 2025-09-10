package customtypes

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

// FuzzNullTimeUnmarshalJSON tests the UnmarshalJSON method with various JSON time inputs
func FuzzNullTimeUnmarshalJSON(f *testing.F) {
	// Seed with various RFC3339 time formats and edge cases
	f.Add(`"2006-01-02T15:04:05Z"`)
	f.Add(`"2023-12-25T23:59:59Z"`)
	f.Add(`"1970-01-01T00:00:00Z"`)
	f.Add(`"2038-01-19T03:14:07Z"`)
	f.Add(`"2000-02-29T12:00:00Z"`) // Leap year
	f.Add(`"1999-12-31T23:59:59Z"`) // Y2K
	f.Add(`"2006-01-02T15:04:05.000Z"`)
	f.Add(`"2006-01-02T15:04:05+07:00"`)
	f.Add(`"2006-01-02T15:04:05-05:00"`)
	f.Add(`"2006-01-02T15:04:05.123456789Z"`)
	f.Add(`""`)                        // Empty string
	f.Add(`"invalid"`)                 // Invalid format
	f.Add(`"2006-13-02T15:04:05Z"`)    // Invalid month
	f.Add(`"2006-01-32T15:04:05Z"`)    // Invalid day
	f.Add(`"2006-01-02T25:04:05Z"`)    // Invalid hour
	f.Add(`"2006-01-02T15:70:05Z"`)    // Invalid minute
	f.Add(`"2006-01-02T15:04:70Z"`)    // Invalid second
	f.Add(`"2006/01/02 15:04:05"`)     // Wrong format
	f.Add(`"Mon Jan 2 15:04:05 2006"`) // Different format

	f.Fuzz(func(t *testing.T, jsonData string) {
		var nt NullTime

		// Test unmarshaling
		err := nt.UnmarshalJSON([]byte(jsonData))
		if err != nil {
			// If there's an error, the NullTime should be invalid
			if nt.Valid {
				t.Errorf("NullTime should be invalid when UnmarshalJSON returns error")
			}
			return
		}

		// Empty string should result in invalid NullTime
		if jsonData == `""` {
			if nt.Valid {
				t.Errorf("Empty string should result in invalid NullTime")
			}
			return
		}

		// If no error and not empty string, NullTime should be valid
		if !nt.Valid {
			t.Errorf("NullTime should be valid when UnmarshalJSON succeeds with non-empty string")
		}

		// The time should be parseable as RFC3339
		timeStr := jsonData
		if len(timeStr) >= 2 && timeStr[0] == '"' && timeStr[len(timeStr)-1] == '"' {
			timeStr = timeStr[1 : len(timeStr)-1]
		}

		expectedTime, parseErr := time.Parse(time.RFC3339, timeStr)
		if parseErr != nil {
			t.Errorf("Time should be parseable as RFC3339: %v", parseErr)
			return
		}

		// The parsed time should match
		if !nt.Time.Equal(expectedTime) {
			t.Errorf("Time mismatch: expected %v, got %v", expectedTime, nt.Time)
		}

		// Test roundtrip: marshal and unmarshal should give same result
		marshaled, marshalErr := nt.MarshalJSON()
		if marshalErr != nil {
			t.Errorf("MarshalJSON failed for valid NullTime: %v", marshalErr)
			return
		}

		var nt2 NullTime
		unmarshalErr := nt2.UnmarshalJSON(marshaled)
		if unmarshalErr != nil {
			t.Errorf("Roundtrip unmarshal failed: %v", unmarshalErr)
			return
		}

		if nt.Valid != nt2.Valid || (nt.Valid && !nt.Time.Equal(nt2.Time)) {
			t.Errorf("Roundtrip failed: original=%+v, roundtrip=%+v", nt, nt2)
		}
	})
}

// FuzzNullTimeUnmarshalBSONValue tests the UnmarshalBSONValue method with various BSON time inputs
func FuzzNullTimeUnmarshalBSONValue(f *testing.F) {
	// Seed with various time values for BSON encoding
	baseTime := time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC)
	times := []time.Time{
		baseTime,
		time.Unix(0, 0).UTC(), // Unix epoch
		time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),      // Epoch
		time.Date(2038, 1, 19, 3, 14, 7, 0, time.UTC),    // Y2038
		time.Date(2000, 2, 29, 12, 0, 0, 0, time.UTC),    // Leap year
		time.Date(1999, 12, 31, 23, 59, 59, 0, time.UTC), // Y2K
		time.Now().UTC(),
		time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC), // Minimum time
	}

	for _, t := range times {
		f.Add(t.Unix())
	}

	f.Fuzz(func(t *testing.T, unixTime int64) {
		// Create a time from the unix timestamp
		inputTime := time.Unix(unixTime, 0).UTC()

		// Create BSON data for the time
		bsonData, err := bson.Marshal(bson.D{{Key: "time", Value: inputTime}})
		if err != nil {
			t.Skipf("Failed to marshal BSON: %v", err)
		}

		// Extract the time bytes from the BSON document
		var doc bson.D
		if unmarshalErr := bson.Unmarshal(bsonData, &doc); unmarshalErr != nil {
			t.Skipf("Failed to unmarshal BSON doc: %v", err)
		}

		if len(doc) == 0 {
			t.Skip("Empty BSON document")
		}

		// Get the raw value
		rawValue := bson.RawValue{Type: bson.TypeDateTime}
		valueBytes, err := bson.Marshal(doc[0].Value)
		if err != nil {
			t.Skipf("Failed to marshal value: %v", err)
		}
		rawValue.Value = valueBytes

		var nt NullTime

		// Test unmarshaling
		err = nt.UnmarshalBSONValue(rawValue.Type, rawValue.Value)
		if err != nil {
			// Some inputs might be invalid BSON
			if nt.Valid {
				t.Errorf("NullTime should be invalid when UnmarshalBSONValue returns error")
			}
			return
		}

		// If no error, check validity
		if !nt.Valid {
			t.Errorf("NullTime should be valid when UnmarshalBSONValue succeeds")
		}

		// The time should be reasonably close to the input (within 1 second due to precision)
		if nt.Time.Unix() != inputTime.Unix() {
			t.Errorf("Time mismatch: expected %v (%d), got %v (%d)",
				inputTime, inputTime.Unix(), nt.Time, nt.Time.Unix())
		}

		// Test roundtrip: marshal and unmarshal should give same result
		bsonType, marshaled, marshalErr := nt.MarshalBSONValue()
		if marshalErr != nil {
			t.Errorf("MarshalBSONValue failed for valid NullTime: %v", marshalErr)
			return
		}

		var nt2 NullTime
		unmarshalErr := nt2.UnmarshalBSONValue(bsonType, marshaled)
		if unmarshalErr != nil {
			t.Errorf("Roundtrip unmarshal failed: %v", unmarshalErr)
			return
		}

		if nt.Valid != nt2.Valid || (nt.Valid && nt.Time.Unix() != nt2.Time.Unix()) {
			t.Errorf("Roundtrip failed: original=%+v, roundtrip=%+v", nt, nt2)
		}
	})
}

// FuzzNullTimeNullValues tests behavior with null/nil inputs
func FuzzNullTimeNullValues(f *testing.F) {
	f.Add([]byte("null"))
	f.Add([]byte(""))
	f.Add([]byte(`""`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var nt NullTime

		// Test JSON null handling
		err := nt.UnmarshalJSON(data)
		if err != nil && string(data) == "null" {
			t.Errorf("UnmarshalJSON should handle null gracefully")
		}

		// Test nil data
		var nt2 NullTime
		err2 := nt2.UnmarshalJSON(nil)
		if err2 != nil {
			t.Errorf("UnmarshalJSON should handle nil data gracefully: %v", err2)
		}
		if nt2.Valid {
			t.Errorf("NullTime should be invalid for nil JSON data")
		}

		// Test BSON null handling
		var nt3 NullTime
		err3 := nt3.UnmarshalBSONValue(bson.TypeNull, nil)
		if err3 != nil {
			t.Errorf("UnmarshalBSONValue should handle null type gracefully: %v", err3)
		}
		if nt3.Valid {
			t.Errorf("NullTime should be invalid for BSON null type")
		}
	})
}

// FuzzNullTimeEdgeCases tests edge cases in time parsing
func FuzzNullTimeEdgeCases(f *testing.F) {
	// Seed with edge case time strings
	f.Add(`"2006-01-02T15:04:05Z"`)
	f.Add(`"2006-01-02T15:04:05.000Z"`)
	f.Add(`"2006-01-02T15:04:05.123Z"`)
	f.Add(`"2006-01-02T15:04:05.123456Z"`)
	f.Add(`"2006-01-02T15:04:05.123456789Z"`)
	f.Add(`"0001-01-01T00:00:00Z"`)
	f.Add(`"9999-12-31T23:59:59Z"`)

	f.Fuzz(func(t *testing.T, jsonData string) {
		var nt NullTime

		// Test unmarshaling edge cases
		err := nt.UnmarshalJSON([]byte(jsonData))

		if err == nil && nt.Valid {
			// Verify that the time is within reasonable bounds
			if nt.Time.Year() < 1 || nt.Time.Year() > 9999 {
				t.Errorf("Time year out of reasonable bounds: %d", nt.Time.Year())
			}

			// Verify that we can marshal it back
			marshaled, marshalErr := nt.MarshalJSON()
			if marshalErr != nil {
				t.Errorf("Failed to marshal valid time: %v", marshalErr)
			}

			// Verify the marshaled result is valid JSON
			if len(marshaled) == 0 {
				t.Errorf("Marshaled time should not be empty")
			}
		}
	})
}
