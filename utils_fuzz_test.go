package datastore

import (
	"database/sql"
	"reflect"
	"testing"
	"time"

	customtypes "github.com/mrz1836/go-datastore/custom_types"
)

// TestModelFuzz represents a model for testing GetModelUnset
type TestModelFuzz struct {
	ID       string                 `bson:"_id"`
	Name     customtypes.NullString `bson:"name"`
	Email    customtypes.NullString `bson:"email"`
	Title    customtypes.NullString `bson:"title"`
	Created  customtypes.NullTime   `bson:"created_at"`
	Updated  customtypes.NullTime   `bson:"updated_at"`
	LastSeen customtypes.NullTime   `bson:"last_seen"`
	Private  customtypes.NullString `bson:"-"`
	Regular  string                 `bson:"regular_field"`
	Number   int                    `bson:"number_field"`
}

// EmbeddedModel represents a model with embedded fields
type EmbeddedModel struct {
	TestModelFuzz

	ExtraField customtypes.NullString `bson:"extra"`
}

// CustomTagModel represents a model with custom BSON tags
type CustomTagModel struct {
	Field1 customtypes.NullString `bson:"custom_name,omitempty"`
	Field2 customtypes.NullTime   `bson:"another_name"`
	Field3 customtypes.NullString `bson:"field3,required"`
}

// FuzzGetModelUnset tests the GetModelUnset function with various model configurations
func FuzzGetModelUnset(f *testing.F) {
	// Seed with different valid/invalid combinations
	f.Add(true, true, true)    // All valid
	f.Add(false, false, false) // All invalid
	f.Add(true, false, true)   // Mixed
	f.Add(false, true, false)  // Mixed

	f.Fuzz(func(t *testing.T, nameValid, emailValid, createdValid bool) {
		// Create test model with fuzzed validity states
		model := &TestModelFuzz{
			ID:      "test-id",
			Regular: "test",
			Number:  42,
		}

		// Set NullString fields based on fuzz input
		model.Name = customtypes.NullString{
			NullString: sql.NullString{
				String: "test-name",
				Valid:  nameValid,
			},
		}

		model.Email = customtypes.NullString{
			NullString: sql.NullString{
				String: "test@example.com",
				Valid:  emailValid,
			},
		}

		// Set NullTime field based on fuzz input
		model.Created = customtypes.NullTime{
			NullTime: sql.NullTime{
				Time:  time.Now(),
				Valid: createdValid,
			},
		}

		// Test with pointer to struct
		unset := GetModelUnset(model)

		// Verify the function doesn't panic
		if unset == nil {
			t.Errorf("GetModelUnset should never return nil")
		}

		// Check that invalid fields are correctly identified
		if !nameValid {
			if !unset["name"] {
				t.Errorf("Invalid name field should be in unset map")
			}
		} else {
			if unset["name"] {
				t.Errorf("Valid name field should not be in unset map")
			}
		}

		if !emailValid {
			if !unset["email"] {
				t.Errorf("Invalid email field should be in unset map")
			}
		} else {
			if unset["email"] {
				t.Errorf("Valid email field should not be in unset map")
			}
		}

		// Private fields (bson:"-") should not appear in unset map
		if unset["private"] {
			t.Errorf("Private field should not appear in unset map")
		}

		// Regular non-null fields should not appear in unset map
		if unset["regular_field"] || unset["number_field"] || unset["_id"] {
			t.Errorf("Regular fields should not appear in unset map")
		}

		// Test with struct value (not pointer)
		unset2 := GetModelUnset(*model)
		if len(unset2) != len(unset) {
			t.Errorf("Struct and pointer should produce same results")
		}

		// Test with custom tag model
		customModel := &CustomTagModel{
			Field1: customtypes.NullString{
				NullString: sql.NullString{Valid: nameValid},
			},
		}

		unsetCustom := GetModelUnset(customModel)
		if !nameValid && !unsetCustom["custom_name"] {
			t.Errorf("Custom BSON tag should be used in field name")
		}
	})
}

// FuzzGetModelUnsetEdgeCases tests edge cases for GetModelUnset
func FuzzGetModelUnsetEdgeCases(f *testing.F) {
	f.Add(0)
	f.Add(1)
	f.Add(100)

	f.Fuzz(func(t *testing.T, _ int) {
		// Test with nil input
		unset := GetModelUnset(nil)
		if len(unset) != 0 {
			t.Errorf("Nil input should return empty map, got %v", unset)
		}

		// Test with non-struct types
		unsetInt := GetModelUnset(42)
		if len(unsetInt) != 0 {
			t.Errorf("Non-struct input should return empty map")
		}

		unsetString := GetModelUnset("test")
		if len(unsetString) != 0 {
			t.Errorf("String input should return empty map")
		}

		unsetSlice := GetModelUnset([]int{1, 2, 3})
		if len(unsetSlice) != 0 {
			t.Errorf("Slice input should return empty map")
		}

		// Test with pointer to non-struct
		value := 42
		unsetPtr := GetModelUnset(&value)
		if len(unsetPtr) != 0 {
			t.Errorf("Pointer to non-struct should return empty map")
		}

		// Test with embedded model
		embedded := &EmbeddedModel{
			TestModelFuzz: TestModelFuzz{
				Name: customtypes.NullString{
					NullString: sql.NullString{Valid: false},
				},
			},
			ExtraField: customtypes.NullString{
				NullString: sql.NullString{Valid: false},
			},
		}

		unsetEmbedded := GetModelUnset(embedded)

		// Should handle embedded fields correctly (only base level fields)
		if !unsetEmbedded["extra"] {
			t.Errorf("Extra field should be unset")
		}

		// Test with struct containing only non-null fields
		type SimpleModel struct {
			Name string `bson:"name"`
			Age  int    `bson:"age"`
		}

		simple := SimpleModel{Name: "test", Age: 25}
		unsetSimple := GetModelUnset(simple)
		if len(unsetSimple) != 0 {
			t.Errorf("Struct with no null fields should return empty map")
		}
	})
}

// FuzzGetModelUnsetReflectionSafety tests reflection safety
func FuzzGetModelUnsetReflectionSafety(f *testing.F) {
	f.Add("")
	f.Add("test")
	f.Add("field1")

	f.Fuzz(func(t *testing.T, fieldValue string) {
		// Test that the function handles reflection errors gracefully
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("GetModelUnset should not panic: %v", r)
			}
		}()

		// Create a model with dynamic field assignment
		model := &TestModelFuzz{}

		// Use reflection to set field values dynamically
		v := reflect.ValueOf(model).Elem()
		nameField := v.FieldByName("Name")
		if nameField.IsValid() && nameField.CanSet() {
			// Create a NullString with fuzzed validity
			nullStr := customtypes.NullString{
				NullString: sql.NullString{
					String: fieldValue,
					Valid:  len(fieldValue) > 0,
				},
			}
			nameField.Set(reflect.ValueOf(nullStr))
		}

		// Test GetModelUnset with dynamically modified struct
		unset := GetModelUnset(model)

		// Should not panic and should return valid map
		if unset == nil {
			t.Errorf("Should return valid map even with reflection modifications")
		}

		// Check field was processed correctly
		if len(fieldValue) == 0 && !unset["name"] {
			t.Errorf("Empty field should be marked as unset")
		}
		if len(fieldValue) > 0 && unset["name"] {
			t.Errorf("Non-empty field should not be marked as unset")
		}
	})
}
