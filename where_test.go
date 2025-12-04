package datastore

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"

	customtypes "github.com/mrz1836/go-datastore/custom_types"
)

// Test_whereObject test the SQL where selector
func Test_whereSlice(t *testing.T) {
	t.Parallel()

	t.Run("MySQL", func(t *testing.T) {
		query := whereSlice(MySQL, fieldInIDs, "id_1")
		expected := `JSON_CONTAINS(` + fieldInIDs + `, CAST('["id_1"]' AS JSON))`
		assert.Equal(t, expected, query)
	})

	t.Run("Postgres", func(t *testing.T) {
		query := whereSlice(PostgreSQL, fieldInIDs, "id_1")
		expected := fieldInIDs + `::jsonb @> '["id_1"]'`
		assert.Equal(t, expected, query)
	})

	t.Run("SQLite", func(t *testing.T) {
		query := whereSlice(SQLite, fieldInIDs, "id_1")
		expected := `EXISTS (SELECT 1 FROM json_each(` + fieldInIDs + `) WHERE value = "id_1")`
		assert.Equal(t, expected, query)
	})
}

// Test_processConditions test the SQL where selectors
func Test_processConditions(t *testing.T) {
	t.Parallel()

	dateField := dateCreatedAt
	uniqueField := "unique_field_name"
	inField := "in_field_name"

	conditions := map[string]interface{}{
		dateField: map[string]interface{}{
			conditionGreaterThan: customtypes.NullTime{NullTime: sql.NullTime{
				Valid: true,
				Time:  time.Date(2022, 4, 4, 15, 12, 37, 651387237, time.UTC),
			}},
		},
		uniqueField: map[string]interface{}{
			conditionExists: true,
		},
		inField: map[string]interface{}{
			conditionIn: []interface{}{"value1", "value2", "value3"},
		},
	}

	checkWhereClauses := func(t *testing.T, actual []interface{}, expected []string) {
		for _, clause := range expected {
			matched := false
			for _, actualClause := range actual {
				re := regexp.MustCompile(`@var\d+`)
				if re.ReplaceAllString(clause, "@var") == re.ReplaceAllString(actualClause.(string), "@var") {
					matched = true
					break
				}
			}
			assert.True(t, matched, "Expected clause %s not found in actual clauses %v", clause, actual)
		}
	}

	checkVars := func(t *testing.T, actual map[string]interface{}, expected []interface{}) {
		for _, val := range expected {
			found := false
			for _, actualVal := range actual {
				if actualVal == val {
					found = true
					break
				}
			}
			assert.True(t, found, "Expected value %v not found in actual vars %v", val, actual)
		}
	}

	t.Run("MySQL", func(t *testing.T) {
		client, deferFunc := testClient(context.Background(), t)
		defer deferFunc()
		tx := &mockSQLCtx{
			WhereClauses: make([]interface{}, 0),
			Vars:         make(map[string]interface{}),
		}
		varNum := 0
		_ = processConditions(client, tx, conditions, MySQL, &varNum, nil)

		expectedWhereClauses := []string{
			dateField + " > @var0",
			uniqueField + " IS NOT NULL",
			inField + " IN (@var1,@var2,@var3)",
		}
		expectedVars := []interface{}{
			"2022-04-04 15:12:37",
			"value1",
			"value2",
			"value3",
		}

		// Add logging for debugging
		t.Logf("Actual WhereClauses: %v", tx.WhereClauses)
		t.Logf("Expected WhereClauses: %v", expectedWhereClauses)
		t.Logf("Actual Vars: %v", tx.Vars)
		t.Logf("Expected Vars: %v", expectedVars)

		checkWhereClauses(t, tx.WhereClauses, expectedWhereClauses)
		checkVars(t, tx.Vars, expectedVars)
	})

	t.Run("Postgres", func(t *testing.T) {
		client, deferFunc := testClient(context.Background(), t)
		defer deferFunc()
		tx := &mockSQLCtx{
			WhereClauses: make([]interface{}, 0),
			Vars:         make(map[string]interface{}),
		}
		varNum := 0
		_ = processConditions(client, tx, conditions, Postgres, &varNum, nil)

		expectedWhereClauses := []string{
			dateField + " > @var0",
			uniqueField + " IS NOT NULL",
			inField + " IN (@var1,@var2,@var3)",
		}
		expectedVars := []interface{}{
			"2022-04-04T15:12:37.651Z",
			"value1",
			"value2",
			"value3",
		}

		// Add logging for debugging
		t.Logf("Actual WhereClauses: %v", tx.WhereClauses)
		t.Logf("Expected WhereClauses: %v", expectedWhereClauses)
		t.Logf("Actual Vars: %v", tx.Vars)
		t.Logf("Expected Vars: %v", expectedVars)

		checkWhereClauses(t, tx.WhereClauses, expectedWhereClauses)
		checkVars(t, tx.Vars, expectedVars)
	})

	t.Run("SQLite", func(t *testing.T) {
		client, deferFunc := testClient(context.Background(), t)
		defer deferFunc()
		tx := &mockSQLCtx{
			WhereClauses: make([]interface{}, 0),
			Vars:         make(map[string]interface{}),
		}
		varNum := 0
		_ = processConditions(client, tx, conditions, SQLite, &varNum, nil)

		expectedWhereClauses := []string{
			dateField + " > @var0",
			uniqueField + " IS NOT NULL",
			inField + " IN (@var1,@var2,@var3)",
		}
		expectedVars := []interface{}{
			"2022-04-04T15:12:37.651Z",
			"value1",
			"value2",
			"value3",
		}

		// Add logging for debugging
		t.Logf("Actual WhereClauses: %v", tx.WhereClauses)
		t.Logf("Expected WhereClauses: %v", expectedWhereClauses)
		t.Logf("Actual Vars: %v", tx.Vars)
		t.Logf("Expected Vars: %v", expectedVars)

		checkWhereClauses(t, tx.WhereClauses, expectedWhereClauses)
		checkVars(t, tx.Vars, expectedVars)
	})
}

// Test_processConditions_NotIn tests the SQL where selectors for the NOT IN operator
func Test_processConditions_NotIn(t *testing.T) {
	t.Parallel()

	dateField := dateCreatedAt
	uniqueField := "unique_field_name"
	notInField := "not_in_field_name"

	conditions := map[string]interface{}{
		dateField: map[string]interface{}{
			conditionGreaterThan: customtypes.NullTime{NullTime: sql.NullTime{
				Valid: true,
				Time:  time.Date(2022, 4, 4, 15, 12, 37, 651387237, time.UTC),
			}},
		},
		uniqueField: map[string]interface{}{
			conditionExists: true,
		},
		notInField: map[string]interface{}{
			conditionNotIn: []interface{}{"value1", "value2", "value3"},
		},
	}

	checkWhereClauses := func(t *testing.T, actual []interface{}, expected []string) {
		for _, clause := range expected {
			matched := false
			for _, actualClause := range actual {
				re := regexp.MustCompile(`@var\d+`)
				if re.ReplaceAllString(clause, "@var") == re.ReplaceAllString(actualClause.(string), "@var") {
					matched = true
					break
				}
			}
			assert.True(t, matched, "Expected clause %s not found in actual clauses %v", clause, actual)
		}
	}

	checkVars := func(t *testing.T, actual map[string]interface{}, expected []interface{}) {
		for _, val := range expected {
			found := false
			for _, actualVal := range actual {
				if actualVal == val {
					found = true
					break
				}
			}
			assert.True(t, found, "Expected value %v not found in actual vars %v", val, actual)
		}
	}

	tests := []struct {
		name                 string
		driver               Engine
		expectedWhereClauses []string
		expectedVars         []interface{}
	}{
		{
			name:   "MySQL",
			driver: MySQL,
			expectedWhereClauses: []string{
				dateField + " > @var0",
				uniqueField + " IS NOT NULL",
				notInField + " NOT IN (@var1,@var2,@var3)",
			},
			expectedVars: []interface{}{
				"2022-04-04 15:12:37",
				"value1",
				"value2",
				"value3",
			},
		},
		{
			name:   "Postgres",
			driver: Postgres,
			expectedWhereClauses: []string{
				dateField + " > @var0",
				uniqueField + " IS NOT NULL",
				notInField + " NOT IN (@var1,@var2,@var3)",
			},
			expectedVars: []interface{}{
				"2022-04-04T15:12:37.651Z",
				"value1",
				"value2",
				"value3",
			},
		},
		{
			name:   "SQLite",
			driver: SQLite,
			expectedWhereClauses: []string{
				dateField + " > @var0",
				uniqueField + " IS NOT NULL",
				notInField + " NOT IN (@var1,@var2,@var3)",
			},
			expectedVars: []interface{}{
				"2022-04-04T15:12:37.651Z",
				"value1",
				"value2",
				"value3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, deferFunc := testClient(context.Background(), t)
			defer deferFunc()

			tx := &mockSQLCtx{
				WhereClauses: make([]interface{}, 0),
				Vars:         make(map[string]interface{}),
			}

			var varNum int
			_ = processConditions(client, tx, conditions, tt.driver, &varNum, nil)

			// Helpful debugging output
			t.Logf("Actual   WhereClauses: %v", tx.WhereClauses)
			t.Logf("Expected WhereClauses: %v", tt.expectedWhereClauses)
			t.Logf("Actual   Vars: %v", tx.Vars)
			t.Logf("Expected Vars: %v", tt.expectedVars)

			checkWhereClauses(t, tx.WhereClauses, tt.expectedWhereClauses)
			checkVars(t, tx.Vars, tt.expectedVars)
		})
	}
}

// Test_whereObject test the SQL where selector
func Test_whereObject(t *testing.T) {
	t.Parallel()

	t.Run("MySQL", func(t *testing.T) {
		metadata := map[string]interface{}{
			"test_key": "test-value",
		}
		query := whereObject(MySQL, metadataField, metadata)
		expected := "JSON_EXTRACT(" + metadataField + ", '$.test_key') = \"test-value\""
		assert.Equal(t, expected, query)

		metadata = map[string]interface{}{
			"test_key": "test-'value'",
		}
		query = whereObject(MySQL, metadataField, metadata)
		expected = "JSON_EXTRACT(" + metadataField + ", '$.test_key') = \"test-\\'value\\'\""
		assert.Equal(t, expected, query)

		metadata = map[string]interface{}{
			"test_key1": "test-value",
			"test_key2": "test-value2",
		}
		query = whereObject(MySQL, metadataField, metadata)

		assert.Contains(t, []string{
			"(JSON_EXTRACT(" + metadataField + ", '$.test_key1') = \"test-value\" AND JSON_EXTRACT(" + metadataField + ", '$.test_key2') = \"test-value2\")",
			"(JSON_EXTRACT(" + metadataField + ", '$.test_key2') = \"test-value2\" AND JSON_EXTRACT(" + metadataField + ", '$.test_key1') = \"test-value\")",
		}, query)

		// The order of the items can change, hence the query order can change
		// assert.Equal(t, expected, query)

		objectMetadata := map[string]interface{}{
			"testId": map[string]interface{}{
				"test_key1": "test-value",
				"test_key2": "test-value2",
			},
		}
		query = whereObject(MySQL, "object_metadata", objectMetadata)

		assert.Contains(t, []string{
			"(JSON_EXTRACT(object_metadata, '$.testId.test_key1') = \"test-value\" AND JSON_EXTRACT(object_metadata, '$.testId.test_key2') = \"test-value2\")",
			"(JSON_EXTRACT(object_metadata, '$.testId.test_key2') = \"test-value2\" AND JSON_EXTRACT(object_metadata, '$.testId.test_key1') = \"test-value\")",
		}, query)

		// The order of the items can change, hence the query order can change
		// assert.Equal(t, expected, query)
	})

	t.Run("Postgres", func(t *testing.T) {
		metadata := map[string]interface{}{
			"test_key": "test-value",
		}
		query := whereObject(PostgreSQL, metadataField, metadata)
		expected := metadataField + "::jsonb @> '{\"test_key\":\"test-value\"}'::jsonb"
		assert.Equal(t, expected, query)

		metadata = map[string]interface{}{
			"test_key": "test-'value'",
		}
		query = whereObject(PostgreSQL, metadataField, metadata)
		expected = metadataField + "::jsonb @> '{\"test_key\":\"test-\\'value\\'\"}'::jsonb"
		assert.Equal(t, expected, query)

		metadata = map[string]interface{}{
			"test_key1": "test-value",
			"test_key2": "test-value2",
		}
		query = whereObject(PostgreSQL, metadataField, metadata)

		assert.Contains(t, []string{
			"(" + metadataField + "::jsonb @> '{\"test_key1\":\"test-value\"}'::jsonb AND " + metadataField + "::jsonb @> '{\"test_key2\":\"test-value2\"}'::jsonb)",
			"(" + metadataField + "::jsonb @> '{\"test_key2\":\"test-value2\"}'::jsonb AND " + metadataField + "::jsonb @> '{\"test_key1\":\"test-value\"}'::jsonb)",
		}, query)

		// The order of the items can change, hence the query order can change
		// assert.Equal(t, expected, query)

		objectMetadata := map[string]interface{}{
			"testId": map[string]interface{}{
				"test_key1": "test-value",
				"test_key2": "test-value2",
			},
		}
		query = whereObject(PostgreSQL, "object_metadata", objectMetadata)
		assert.Contains(t, []string{
			"object_metadata::jsonb @> '{\"testId\":{\"test_key1\":\"test-value\",\"test_key2\":\"test-value2\"}}'::jsonb",
			"object_metadata::jsonb @> '{\"testId\":{\"test_key2\":\"test-value2\",\"test_key1\":\"test-value\"}}'::jsonb",
		}, query)

		// The order of the items can change, hence the query order can change
		// assert.Equal(t, expected, query)
	})

	t.Run("SQLite", func(t *testing.T) {
		metadata := map[string]interface{}{
			"test_key": "test-value",
		}
		query := whereObject(SQLite, metadataField, metadata)
		expected := "JSON_EXTRACT(" + metadataField + ", '$.test_key') = \"test-value\""
		assert.Equal(t, expected, query)

		metadata = map[string]interface{}{
			"test_key": "test-'value'",
		}
		query = whereObject(SQLite, metadataField, metadata)
		expected = "JSON_EXTRACT(" + metadataField + ", '$.test_key') = \"test-\\'value\\'\""
		assert.Equal(t, expected, query)

		metadata = map[string]interface{}{
			"test_key1": "test-value",
			"test_key2": "test-value2",
		}
		query = whereObject(SQLite, metadataField, metadata)
		assert.Contains(t, []string{
			"(JSON_EXTRACT(" + metadataField + ", '$.test_key1') = \"test-value\" AND JSON_EXTRACT(" + metadataField + ", '$.test_key2') = \"test-value2\")",
			"(JSON_EXTRACT(" + metadataField + ", '$.test_key2') = \"test-value2\" AND JSON_EXTRACT(" + metadataField + ", '$.test_key1') = \"test-value\")",
		}, query)

		// The order of the items can change, hence the query order can change
		// assert.Equal(t, expected, query)

		objectMetadata := map[string]interface{}{
			"testId": map[string]interface{}{
				"test_key1": "test-value",
				"test_key2": "test-value2",
			},
		}
		query = whereObject(SQLite, "object_metadata", objectMetadata)
		assert.Contains(t, []string{
			"(JSON_EXTRACT(object_metadata, '$.testId.test_key1') = \"test-value\" AND JSON_EXTRACT(object_metadata, '$.testId.test_key2') = \"test-value2\")",
			"(JSON_EXTRACT(object_metadata, '$.testId.test_key2') = \"test-value2\" AND JSON_EXTRACT(object_metadata, '$.testId.test_key1') = \"test-value\")",
		}, query)
		// The order of the items can change, hence the query order can change
		// assert.Equal(t, expected, query)
	})
}

// mockSQLCtx is used to mock the SQL
type mockSQLCtx struct {
	WhereClauses []interface{}
	Vars         map[string]interface{}
}

// Where will append the where clause
func (f *mockSQLCtx) Where(query interface{}, args ...interface{}) {
	f.WhereClauses = append(f.WhereClauses, query)
	if len(args) > 0 {
		for _, variables := range args {
			for key, value := range variables.(map[string]interface{}) {
				f.Vars[key] = value
			}
		}
	}
}

// getGormTx will return the GORM transaction
func (f *mockSQLCtx) getGormTx() *gorm.DB {
	return nil
}

// TestCustomWhere will test the method CustomWhere()
func TestCustomWhere(t *testing.T) {
	t.Parallel()

	t.Run("SQLite empty select", func(t *testing.T) {
		client, deferFunc := testClient(context.Background(), t)
		defer deferFunc()
		tx := mockSQLCtx{
			WhereClauses: make([]interface{}, 0),
			Vars:         make(map[string]interface{}),
		}
		conditions := map[string]interface{}{}
		_ = client.CustomWhere(&tx, conditions, SQLite)
		assert.Equal(t, []interface{}{}, tx.WhereClauses)
	})

	t.Run("SQLite simple select", func(t *testing.T) {
		client, deferFunc := testClient(context.Background(), t)
		defer deferFunc()
		tx := mockSQLCtx{
			WhereClauses: make([]interface{}, 0),
			Vars:         make(map[string]interface{}),
		}
		conditions := map[string]interface{}{
			sqlIDFieldProper: "testID",
		}
		_ = client.CustomWhere(&tx, conditions, SQLite)
		assert.Len(t, tx.WhereClauses, 1)
		assert.Equal(t, sqlIDFieldProper+" = @var0", tx.WhereClauses[0])
		assert.Equal(t, "testID", tx.Vars["var0"])
	})

	t.Run("SQLite "+conditionOr, func(t *testing.T) {
		arrayField1 := fieldInIDs
		arrayField2 := fieldOutIDs

		client, deferFunc := testClient(context.Background(), t, WithCustomFields([]string{arrayField1, arrayField2}, nil))
		defer deferFunc()
		tx := mockSQLCtx{
			WhereClauses: make([]interface{}, 0),
			Vars:         make(map[string]interface{}),
		}
		conditions := map[string]interface{}{
			conditionOr: []map[string]interface{}{{
				arrayField1: "value_id",
			}, {
				arrayField2: "value_id",
			}},
		}
		_ = client.CustomWhere(&tx, conditions, SQLite)
		assert.Len(t, tx.WhereClauses, 1)
		assert.Equal(t, " ( (EXISTS (SELECT 1 FROM json_each("+arrayField1+") WHERE value = \"value_id\")) OR (EXISTS (SELECT 1 FROM json_each("+arrayField2+") WHERE value = \"value_id\")) ) ", tx.WhereClauses[0])
	})

	t.Run("MySQL "+conditionOr, func(t *testing.T) {
		arrayField1 := fieldInIDs
		arrayField2 := fieldOutIDs

		client, deferFunc := testClient(context.Background(), t, WithCustomFields([]string{arrayField1, arrayField2}, nil))
		defer deferFunc()
		tx := mockSQLCtx{
			WhereClauses: make([]interface{}, 0),
			Vars:         make(map[string]interface{}),
		}
		conditions := map[string]interface{}{
			conditionOr: []map[string]interface{}{{
				arrayField1: "value_id",
			}, {
				arrayField2: "value_id",
			}},
		}
		_ = client.CustomWhere(&tx, conditions, MySQL)
		assert.Len(t, tx.WhereClauses, 1)
		assert.Equal(t, " ( (JSON_CONTAINS("+arrayField1+", CAST('[\"value_id\"]' AS JSON))) OR (JSON_CONTAINS("+arrayField2+", CAST('[\"value_id\"]' AS JSON))) ) ", tx.WhereClauses[0])
	})

	t.Run("PostgreSQL "+conditionOr, func(t *testing.T) {
		arrayField1 := fieldInIDs
		arrayField2 := fieldOutIDs

		client, deferFunc := testClient(context.Background(), t, WithCustomFields([]string{arrayField1, arrayField2}, nil))
		defer deferFunc()
		tx := mockSQLCtx{
			WhereClauses: make([]interface{}, 0),
			Vars:         make(map[string]interface{}),
		}
		conditions := map[string]interface{}{
			conditionOr: []map[string]interface{}{{
				arrayField1: "value_id",
			}, {
				arrayField2: "value_id",
			}},
		}
		_ = client.CustomWhere(&tx, conditions, PostgreSQL)
		assert.Len(t, tx.WhereClauses, 1)
		assert.Equal(t, " ( ("+arrayField1+"::jsonb @> '[\"value_id\"]') OR ("+arrayField2+"::jsonb @> '[\"value_id\"]') ) ", tx.WhereClauses[0])
	})

	t.Run("SQLite "+metadataField, func(t *testing.T) {
		client, deferFunc := testClient(context.Background(), t)
		defer deferFunc()
		tx := mockSQLCtx{
			WhereClauses: make([]interface{}, 0),
			Vars:         make(map[string]interface{}),
		}
		conditions := map[string]interface{}{
			metadataField: map[string]interface{}{
				"field_name": "field_value",
			},
		}
		_ = client.CustomWhere(&tx, conditions, SQLite)
		assert.Len(t, tx.WhereClauses, 1)
		assert.Equal(t, "JSON_EXTRACT("+metadataField+", '$.field_name') = \"field_value\"", tx.WhereClauses[0])
	})

	t.Run("MySQL "+metadataField, func(t *testing.T) {
		client, deferFunc := testClient(context.Background(), t)
		defer deferFunc()
		tx := mockSQLCtx{
			WhereClauses: make([]interface{}, 0),
			Vars:         make(map[string]interface{}),
		}
		conditions := map[string]interface{}{
			metadataField: map[string]interface{}{
				"field_name": "field_value",
			},
		}
		_ = client.CustomWhere(&tx, conditions, MySQL)
		assert.Len(t, tx.WhereClauses, 1)
		assert.Equal(t, "JSON_EXTRACT("+metadataField+", '$.field_name') = \"field_value\"", tx.WhereClauses[0])
	})

	t.Run("PostgreSQL "+metadataField, func(t *testing.T) {
		client, deferFunc := testClient(context.Background(), t)
		defer deferFunc()
		tx := mockSQLCtx{
			WhereClauses: make([]interface{}, 0),
			Vars:         make(map[string]interface{}),
		}
		conditions := map[string]interface{}{
			metadataField: map[string]interface{}{
				"field_name": "field_value",
			},
		}
		_ = client.CustomWhere(&tx, conditions, PostgreSQL)
		assert.Len(t, tx.WhereClauses, 1)
		assert.Equal(t, metadataField+"::jsonb @> '{\"field_name\":\"field_value\"}'::jsonb", tx.WhereClauses[0])
	})

	t.Run("SQLite "+conditionAnd, func(t *testing.T) {
		arrayField1 := fieldInIDs
		arrayField2 := fieldOutIDs

		client, deferFunc := testClient(context.Background(), t, WithCustomFields([]string{arrayField1, arrayField2}, nil))
		defer deferFunc()
		tx := mockSQLCtx{
			WhereClauses: make([]interface{}, 0),
			Vars:         make(map[string]interface{}),
		}
		conditions := map[string]interface{}{
			conditionAnd: []map[string]interface{}{{
				"reference_id": "reference",
			}, {
				"number": 12,
			}, {
				conditionOr: []map[string]interface{}{{
					arrayField1: "value_id",
				}, {
					arrayField2: "value_id",
				}},
			}},
		}
		_ = client.CustomWhere(&tx, conditions, SQLite)
		assert.Len(t, tx.WhereClauses, 1)
		assert.Equal(t, " ( reference_id = @var0 AND number = @var1 AND  ( (EXISTS (SELECT 1 FROM json_each("+arrayField1+") WHERE value = \"value_id\")) OR (EXISTS (SELECT 1 FROM json_each("+arrayField2+") WHERE value = \"value_id\")) )  ) ", tx.WhereClauses[0])
		assert.Equal(t, "reference", tx.Vars["var0"])
		assert.Equal(t, 12, tx.Vars["var1"])
	})

	t.Run("MySQL "+conditionAnd, func(t *testing.T) {
		arrayField1 := fieldInIDs
		arrayField2 := fieldOutIDs

		client, deferFunc := testClient(context.Background(), t, WithCustomFields([]string{arrayField1, arrayField2}, nil))
		defer deferFunc()
		tx := mockSQLCtx{
			WhereClauses: make([]interface{}, 0),
			Vars:         make(map[string]interface{}),
		}
		conditions := map[string]interface{}{
			conditionAnd: []map[string]interface{}{{
				"reference_id": "reference",
			}, {
				"number": 12,
			}, {
				conditionOr: []map[string]interface{}{{
					arrayField1: "value_id",
				}, {
					arrayField2: "value_id",
				}},
			}},
		}
		_ = client.CustomWhere(&tx, conditions, MySQL)
		assert.Len(t, tx.WhereClauses, 1)
		assert.Equal(t, " ( reference_id = @var0 AND number = @var1 AND  ( (JSON_CONTAINS("+arrayField1+", CAST('[\"value_id\"]' AS JSON))) OR (JSON_CONTAINS("+arrayField2+", CAST('[\"value_id\"]' AS JSON))) )  ) ", tx.WhereClauses[0])
		assert.Equal(t, "reference", tx.Vars["var0"])
		assert.Equal(t, 12, tx.Vars["var1"])
	})

	t.Run("PostgreSQL "+conditionAnd, func(t *testing.T) {
		arrayField1 := fieldInIDs
		arrayField2 := fieldOutIDs

		client, deferFunc := testClient(context.Background(), t, WithCustomFields([]string{arrayField1, arrayField2}, nil))
		defer deferFunc()
		tx := mockSQLCtx{
			WhereClauses: make([]interface{}, 0),
			Vars:         make(map[string]interface{}),
		}
		conditions := map[string]interface{}{
			conditionAnd: []map[string]interface{}{{
				"reference_id": "reference",
			}, {
				"number": 12,
			}, {
				conditionOr: []map[string]interface{}{{
					arrayField1: "value_id",
				}, {
					arrayField2: "value_id",
				}},
			}},
		}
		_ = client.CustomWhere(&tx, conditions, PostgreSQL)
		assert.Len(t, tx.WhereClauses, 1)
		assert.Equal(t, " ( reference_id = @var0 AND number = @var1 AND  ( ("+arrayField1+"::jsonb @> '[\"value_id\"]') OR ("+arrayField2+"::jsonb @> '[\"value_id\"]') )  ) ", tx.WhereClauses[0])
		assert.Equal(t, "reference", tx.Vars["var0"])
		assert.Equal(t, 12, tx.Vars["var1"])
	})

	t.Run("Where "+conditionGreaterThan, func(t *testing.T) {
		client, deferFunc := testClient(context.Background(), t)
		defer deferFunc()
		tx := mockSQLCtx{
			WhereClauses: make([]interface{}, 0),
			Vars:         make(map[string]interface{}),
		}
		conditions := map[string]interface{}{
			"amount": map[string]interface{}{
				conditionGreaterThan: 502,
			},
		}
		_ = client.CustomWhere(&tx, conditions, PostgreSQL) // all the same
		assert.Len(t, tx.WhereClauses, 1)
		assert.Equal(t, "amount > @var0", tx.WhereClauses[0])
		assert.Equal(t, 502, tx.Vars["var0"])
	})

	t.Run("Where "+conditionGreaterThan+" "+conditionLessThan, func(t *testing.T) {
		client, deferFunc := testClient(context.Background(), t)
		defer deferFunc()
		tx := mockSQLCtx{
			WhereClauses: make([]interface{}, 0),
			Vars:         make(map[string]interface{}),
		}
		conditions := map[string]interface{}{
			conditionAnd: []map[string]interface{}{{
				"amount": map[string]interface{}{
					conditionLessThan: 503,
				},
			}, {
				"amount": map[string]interface{}{
					conditionGreaterThan: 203,
				},
			}},
		}
		_ = client.CustomWhere(&tx, conditions, PostgreSQL) // all the same
		assert.Len(t, tx.WhereClauses, 1)
		assert.Equal(t, " ( amount < @var0 AND amount > @var1 ) ", tx.WhereClauses[0])
		assert.Equal(t, 503, tx.Vars["var0"])
		assert.Equal(t, 203, tx.Vars["var1"])
	})

	t.Run("Where "+conditionGreaterThanOrEqual+" "+conditionLessThanOrEqual, func(t *testing.T) {
		client, deferFunc := testClient(context.Background(), t)
		defer deferFunc()
		tx := mockSQLCtx{
			WhereClauses: make([]interface{}, 0),
			Vars:         make(map[string]interface{}),
		}
		conditions := map[string]interface{}{
			conditionOr: []map[string]interface{}{{
				"amount": map[string]interface{}{
					conditionLessThanOrEqual: 203,
				},
			}, {
				"amount": map[string]interface{}{
					conditionGreaterThanOrEqual: 1203,
				},
			}},
		}
		_ = client.CustomWhere(&tx, conditions, PostgreSQL) // all the same
		assert.Len(t, tx.WhereClauses, 1)
		assert.Equal(t, " ( (amount <= @var0) OR (amount >= @var1) ) ", tx.WhereClauses[0])
		assert.Equal(t, 203, tx.Vars["var0"])
		assert.Equal(t, 1203, tx.Vars["var1"])
	})

	t.Run("Where "+conditionOr+" "+conditionAnd+" "+conditionOr+" "+conditionGreaterThanOrEqual+" "+conditionLessThanOrEqual, func(t *testing.T) {
		client, deferFunc := testClient(context.Background(), t)
		defer deferFunc()
		tx := mockSQLCtx{
			WhereClauses: make([]interface{}, 0),
			Vars:         make(map[string]interface{}),
		}
		conditions := map[string]interface{}{
			conditionOr: []map[string]interface{}{{
				conditionAnd: []map[string]interface{}{{
					"amount": map[string]interface{}{
						conditionLessThanOrEqual: 203,
					},
				}, {
					conditionOr: []map[string]interface{}{{
						"amount": map[string]interface{}{
							conditionGreaterThanOrEqual: 1203,
						},
					}, {
						"value": map[string]interface{}{
							conditionGreaterThanOrEqual: 2203,
						},
					}},
				}},
			}, {
				conditionAnd: []map[string]interface{}{{
					"amount": map[string]interface{}{
						conditionGreaterThanOrEqual: 3203,
					},
				}, {
					"value": map[string]interface{}{
						conditionGreaterThanOrEqual: 4203,
					},
				}},
			}},
		}
		_ = client.CustomWhere(&tx, conditions, PostgreSQL) // all the same
		assert.Len(t, tx.WhereClauses, 1)
		assert.Equal(t, " ( ( ( amount <= @var0 AND  ( (amount >= @var1) OR (value >= @var2) )  ) ) OR ( ( amount >= @var3 AND value >= @var4 ) ) ) ", tx.WhereClauses[0])
		assert.Equal(t, 203, tx.Vars["var0"])
		assert.Equal(t, 1203, tx.Vars["var1"])
		assert.Equal(t, 2203, tx.Vars["var2"])
		assert.Equal(t, 3203, tx.Vars["var3"])
		assert.Equal(t, 4203, tx.Vars["var4"])
	})

	t.Run("Where "+conditionAnd+" "+conditionOr+" "+conditionOr+" "+conditionGreaterThanOrEqual+" "+conditionLessThanOrEqual, func(t *testing.T) {
		client, deferFunc := testClient(context.Background(), t)
		defer deferFunc()
		tx := mockSQLCtx{
			WhereClauses: make([]interface{}, 0),
			Vars:         make(map[string]interface{}),
		}
		conditions := map[string]interface{}{
			conditionAnd: []map[string]interface{}{{
				conditionAnd: []map[string]interface{}{{
					"amount": map[string]interface{}{
						conditionLessThanOrEqual:    203,
						conditionGreaterThanOrEqual: 103,
					},
				}, {
					conditionOr: []map[string]interface{}{{
						"amount": map[string]interface{}{
							conditionGreaterThanOrEqual: 1203,
						},
					}, {
						"value": map[string]interface{}{
							conditionGreaterThanOrEqual: 2203,
						},
					}},
				}},
			}, {
				conditionOr: []map[string]interface{}{{
					"amount": map[string]interface{}{
						conditionGreaterThanOrEqual: 3203,
					},
				}, {
					"value": map[string]interface{}{
						conditionGreaterThanOrEqual: 4203,
					},
				}},
			}},
		}
		_ = client.CustomWhere(&tx, conditions, PostgreSQL) // all the same
		assert.Len(t, tx.WhereClauses, 1)
		assert.Contains(t, []string{
			" (  ( amount <= @var0 AND amount >= @var1 AND  ( (amount >= @var2) OR (value >= @var3) )  )  AND  ( (amount >= @var4) OR (value >= @var5) )  ) ",
			" (  ( amount >= @var0 AND amount <= @var1 AND  ( (amount >= @var2) OR (value >= @var3) )  )  AND  ( (amount >= @var4) OR (value >= @var5) )  ) ",
		}, tx.WhereClauses[0])

		// assert.Equal(t, " (  ( amount <= @var0 AND amount >= @var1 AND  ( (amount >= @var2) OR (value >= @var3) )  )  AND  ( (amount >= @var4) OR (value >= @var5) )  ) ", tx.WhereClauses[0])

		assert.Contains(t, []int{203, 103}, tx.Vars["var0"])
		assert.Contains(t, []int{203, 103}, tx.Vars["var1"])
		// assert.Equal(t, 203, tx.Vars["var0"])
		// assert.Equal(t, 103, tx.Vars["var1"])
		assert.Equal(t, 1203, tx.Vars["var2"])
		assert.Equal(t, 2203, tx.Vars["var3"])
		assert.Equal(t, 3203, tx.Vars["var4"])
		assert.Equal(t, 4203, tx.Vars["var5"])
	})
}

// Test_escapeDBString will test the method escapeDBString()
func Test_escapeDBString(t *testing.T) {
	t.Parallel()

	str := escapeDBString(`SELECT * FROM 'table' WHERE 'field'=1;`)
	assert.Equal(t, `SELECT * FROM \'table\' WHERE \'field\'=1;`, str)
}

// Mock CustomWhereInterface
type MockCustomWhereInterface struct {
	mock.Mock
}

// Correct the Where method to match the interface
func (m *MockCustomWhereInterface) Where(query interface{}, args ...interface{}) {
	m.Called(query, args)
}

// Mock the getGormTx method (assuming it returns *gorm.DB)
func (m *MockCustomWhereInterface) getGormTx() *gorm.DB {
	// Return nil or mock *gorm.DB behavior if needed
	return nil
}

// Test the processConditions function
func TestProcessConditions(t *testing.T) {
	tests := []struct {
		name       string
		conditions map[string]interface{}
		expected   string
	}{
		{
			name: "Greater Than Condition",
			conditions: map[string]interface{}{
				"$gt": 100,
			},
			expected: "field > @var0",
		},
		{
			name: "Less Than Condition",
			conditions: map[string]interface{}{
				"$lt": 50,
			},
			expected: "field < @var0",
		},
		{
			name: "Greater Than or Equal Condition",
			conditions: map[string]interface{}{
				"$gte": 100,
			},
			expected: "field >= @var0",
		},
		{
			name: "Less Than or Equal Condition",
			conditions: map[string]interface{}{
				"$lte": 50,
			},
			expected: "field <= @var0",
		},
		{
			name: "Not Equals Condition",
			conditions: map[string]interface{}{
				"$ne": 10,
			},
			expected: "field != @var0",
		},
		{
			name: "Exists Condition - True",
			conditions: map[string]interface{}{
				"$exists": true,
			},
			expected: "field IS NOT NULL",
		},
		{
			name: "Exists Condition - False",
			conditions: map[string]interface{}{
				"$exists": false,
			},
			expected: "field IS NULL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, deferFunc := testClient(context.Background(), t)
			defer deferFunc()

			// Initialize mocks
			mockTx := &MockCustomWhereInterface{}

			// Define mock behavior for tx.Where
			mockTx.On("Where", mock.Anything, mock.Anything).Return(nil)

			// Initialize variables
			varNum := 0
			parentKey := "field"

			// Call the function being tested
			processConditions(client, mockTx, tt.conditions, SQLite, &varNum, &parentKey)

			// Assert that the correct SQL query was generated
			mockTx.AssertCalled(t, "Where", tt.expected, mock.Anything)
		})
	}
}
