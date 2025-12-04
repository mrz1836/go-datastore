package datastore

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"

	customtypes "github.com/mrz1836/go-datastore/custom_types"
)

// FuzzEscapeDBString tests the escapeDBString function with various string inputs
func FuzzEscapeDBString(f *testing.F) {
	// Seed with common edge cases
	f.Add("")
	f.Add("'")
	f.Add("\"")
	f.Add("'\"")
	f.Add("\\")
	f.Add("test'string")
	f.Add("test\"string")
	f.Add("test\\string")
	f.Add("O'Reilly")
	f.Add("SELECT * FROM users WHERE name = 'admin'")
	f.Add("'; DROP TABLE users; --")
	f.Add("\x00\x01\x02")
	f.Add("unicode: αβγδ")
	f.Add("emoji: 🚀🔬")

	f.Fuzz(func(t *testing.T, input string) {
		result := escapeDBString(input)

		// The result should not contain unescaped single quotes
		for i, r := range result {
			if r == '\'' && (i == 0 || result[i-1] != '\\') {
				t.Errorf("Found unescaped single quote in result: %q", result)
			}
			if r == '"' && (i == 0 || result[i-1] != '\\') {
				t.Errorf("Found unescaped double quote in result: %q", result)
			}
		}

		// The escaped result should be longer or equal to original when escaping occurs
		if input != result && len(result) < len(input) {
			t.Errorf("Escaped string is shorter than original: %q -> %q", input, result)
		}
	})
}

// FuzzFormatCondition tests the formatCondition function with various types and engines
func FuzzFormatCondition(f *testing.F) {
	// Seed with various time formats and edge cases
	f.Add("2006-01-02T15:04:05Z")
	f.Add("2023-12-25T23:59:59Z")
	f.Add("1970-01-01T00:00:00Z")
	f.Add("2038-01-19T03:14:07Z")
	f.Add("string value")
	f.Add("")
	f.Add("1234567890")
	f.Add("true")
	f.Add("false")

	f.Fuzz(func(t *testing.T, timeStr string) {
		// Test with different engines for NullTime
		parsedTime, err := time.Parse(time.RFC3339, timeStr)
		if err != nil {
			// For invalid time strings, test with string condition
			result := formatCondition(timeStr, MySQL)
			if result != timeStr {
				t.Errorf("String condition should be unchanged: %v -> %v", timeStr, result)
			}
			return
		}

		nullTime := customtypes.NullTime{
			NullTime: sql.NullTime{
				Time:  parsedTime,
				Valid: true,
			},
		}

		// Test all database engines
		engines := []Engine{MySQL, PostgreSQL, SQLite}
		for _, engine := range engines {
			result := formatCondition(nullTime, engine)
			if result == nil {
				t.Errorf("Valid NullTime should not return nil for engine %v", engine)
			}

			resultStr, ok := result.(string)
			if !ok {
				t.Errorf("formatCondition should return string for valid NullTime, got %T", result)
			}

			if len(resultStr) == 0 {
				t.Errorf("formatCondition should not return empty string for valid time")
			}

			// Basic format validation for each engine
			switch engine {
			case MySQL:
				// MySQL format: "2006-01-02 15:04:05"
				if _, err := time.Parse("2006-01-02 15:04:05", resultStr); err != nil {
					t.Errorf("MySQL format invalid: %s", resultStr)
				}
			case PostgreSQL:
				// PostgreSQL format: "2006-01-02T15:04:05Z07:00"
				if _, err := time.Parse("2006-01-02T15:04:05Z07:00", resultStr); err != nil {
					t.Errorf("PostgreSQL format invalid: %s", resultStr)
				}
			default: // SQLite
				// SQLite format: "2006-01-02T15:04:05.000Z"
				if _, err := time.Parse("2006-01-02T15:04:05.000Z", resultStr); err != nil {
					t.Errorf("SQLite format invalid: %s", resultStr)
				}
			}
		}

		// Test invalid NullTime
		invalidNullTime := customtypes.NullTime{
			NullTime: sql.NullTime{Valid: false},
		}
		result := formatCondition(invalidNullTime, MySQL)
		if result != nil {
			t.Errorf("Invalid NullTime should return nil")
		}
	})
}

// mockClient implements ClientInterface for testing
type mockClient struct{}

// GetterInterface methods
func (m *mockClient) GetDatabaseName() string                                  { return "test" }
func (m *mockClient) GetMongoCollection(_ string) *mongo.Collection            { return nil }
func (m *mockClient) GetMongoCollectionByTableName(_ string) *mongo.Collection { return nil }
func (m *mockClient) GetMongoConditionProcessor() func(conditions *map[string]interface{}) {
	return nil
}
func (m *mockClient) GetMongoIndexer() func() map[string][]mongo.IndexModel { return nil }
func (m *mockClient) GetTableName(modelName string) string                  { return modelName }

// StorageService methods
func (m *mockClient) AutoMigrateDatabase(_ context.Context, _ ...interface{}) error {
	return nil
}

func (m *mockClient) CreateInBatches(_ context.Context, _ interface{}, _ int) error {
	return nil
}

func (m *mockClient) CustomWhere(_ CustomWhereInterface, _ map[string]interface{}, _ Engine) interface{} {
	return nil
}
func (m *mockClient) Execute(_ string) *gorm.DB { return nil }
func (m *mockClient) GetModel(_ context.Context, _ interface{}, _ map[string]interface{}, _ time.Duration, _ bool) error {
	return nil
}

func (m *mockClient) GetModels(_ context.Context, _ interface{}, _ map[string]interface{}, _ *QueryParams, _ interface{}, _ time.Duration) error {
	return nil
}

func (m *mockClient) GetModelCount(_ context.Context, _ interface{}, _ map[string]interface{}, _ time.Duration) (int64, error) {
	return 0, nil
}

func (m *mockClient) GetModelsAggregate(_ context.Context, _ interface{}, _ map[string]interface{}, _ string, _ time.Duration) (map[string]interface{}, error) {
	return make(map[string]interface{}), nil
}
func (m *mockClient) HasMigratedModel(_ string) bool { return true }
func (m *mockClient) IncrementModel(_ context.Context, _ interface{}, _ string, _ int64) (newValue int64, err error) {
	return 0, nil
}
func (m *mockClient) IndexExists(_, _ string) (bool, error)                     { return false, nil }
func (m *mockClient) IndexMetadata(_, _ string) error                           { return nil }
func (m *mockClient) NewTx(_ context.Context, _ func(*Transaction) error) error { return nil }
func (m *mockClient) NewRawTx() (*Transaction, error)                           { return &Transaction{}, nil }
func (m *mockClient) Raw(_ string) *gorm.DB                                     { return nil }
func (m *mockClient) SaveModel(_ context.Context, _ interface{}, _ *Transaction, _, _ bool) error {
	return nil
}

// ClientInterface methods
func (m *mockClient) Close(_ context.Context) error        { return nil }
func (m *mockClient) Debug(_ bool)                         {}
func (m *mockClient) DebugLog(_ context.Context, _ string) {}
func (m *mockClient) Engine() Engine                       { return MySQL }
func (m *mockClient) IsAutoMigrate() bool                  { return false }
func (m *mockClient) IsDebug() bool                        { return false }
func (m *mockClient) IsNewRelicEnabled() bool              { return false }

func (m *mockClient) GetArrayFields() []string {
	return []string{"tags", "categories"}
}

func (m *mockClient) GetObjectFields() []string {
	return []string{"metadata", "settings"}
}

// mockTx implements CustomWhereInterface for testing
type mockTx struct {
	whereClauses []string
	vars         []map[string]interface{}
}

func (m *mockTx) Where(query interface{}, args ...interface{}) {
	if query != nil {
		m.whereClauses = append(m.whereClauses, query.(string))
	}
	if len(args) > 0 {
		for _, arg := range args {
			if arg != nil {
				m.vars = append(m.vars, arg.(map[string]interface{}))
			}
		}
	}
}

func (m *mockTx) getGormTx() *gorm.DB {
	return nil
}

// FuzzProcessConditions tests the processConditions function with various condition maps
func FuzzProcessConditions(f *testing.F) {
	// Seed with valid JSON condition structures
	f.Add(`{"name": "test"}`)
	f.Add(`{"age": 25}`)
	f.Add(`{"$gt": 10}`)
	f.Add(`{"$lt": 100}`)
	f.Add(`{"$in": ["a", "b", "c"]}`)
	f.Add(`{"$and": [{"name": "test"}, {"age": 25}]}`)
	f.Add(`{"$or": [{"name": "test"}, {"age": 25}]}`)
	f.Add(`{"metadata": {"key": "value"}}`)
	f.Add(`{"tags": "important"}`)
	f.Add(`{"$exists": true}`)
	f.Add(`{"$ne": "admin"}`)
	f.Add(`{}`)

	f.Fuzz(func(t *testing.T, jsonStr string) {
		// Skip empty strings
		if len(jsonStr) == 0 {
			return
		}

		// Try to parse as valid JSON map
		var conditions map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &conditions); err != nil {
			// Skip invalid JSON
			return
		}

		// Create mock instances
		client := &mockClient{}
		tx := &mockTx{
			whereClauses: make([]string, 0),
			vars:         make([]map[string]interface{}, 0),
		}

		// Test with different engines
		engines := []Engine{MySQL, PostgreSQL, SQLite}
		for _, engine := range engines {
			// Reset transaction state
			tx.whereClauses = make([]string, 0)
			tx.vars = make([]map[string]interface{}, 0)

			varNum := 0

			// This should not panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("processConditions panicked with engine %v: %v", engine, r)
				}
			}()

			result := processConditions(client, tx, conditions, engine, &varNum, nil)

			// Result should be the same as input
			if result == nil && conditions != nil {
				t.Errorf("processConditions returned nil for non-nil input")
			}

			// Variable counter should not be negative
			if varNum < 0 {
				t.Errorf("Variable counter became negative: %d", varNum)
			}

			// Where clauses should be strings if they exist
			for _, clause := range tx.whereClauses {
				// Empty clauses are acceptable for empty conditions
				if len(clause) > 0 && strings.TrimSpace(clause) == "" {
					t.Errorf("Where clause should not be whitespace-only: %q", clause)
				}
			}

			// Variables should be valid
			for _, vars := range tx.vars {
				if vars == nil {
					t.Errorf("Nil variables map generated")
				}
			}
		}
	})
}
