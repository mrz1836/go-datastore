package datastore

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveModel(t *testing.T) {
	t.Run("create new model", func(t *testing.T) {
		c := setupTestClient(t)
		defer func() { _ = c.Close(context.Background()) }()

		tx, err := c.NewRawTx()
		require.NoError(t, err)

		model := &TestModel{Name: "save_new", Value: 100}
		err = c.SaveModel(context.Background(), model, tx, true, true)
		require.NoError(t, err)

		var result TestModel
		err = c.GetModel(context.Background(), &result, map[string]any{"name": "save_new"}, time.Second, false)
		require.NoError(t, err)
		assert.Equal(t, "save_new", result.Name)
		assert.Equal(t, 100, result.Value)
		assert.NotZero(t, result.ID)
	})

	t.Run("update existing model", func(t *testing.T) {
		c := setupTestClient(t)
		defer func() { _ = c.Close(context.Background()) }()

		// Create first
		tx, err := c.NewRawTx()
		require.NoError(t, err)
		model := &TestModel{Name: "update_me", Value: 200}
		err = c.SaveModel(context.Background(), model, tx, true, true)
		require.NoError(t, err)

		// Update
		tx, err = c.NewRawTx()
		require.NoError(t, err)
		model.Value = 201
		err = c.SaveModel(context.Background(), model, tx, false, true)
		require.NoError(t, err)

		var result TestModel
		err = c.GetModel(context.Background(), &result, map[string]any{"name": "update_me"}, time.Second, false)
		require.NoError(t, err)
		assert.Equal(t, 201, result.Value)
	})
}

func TestIncrementModel(t *testing.T) {
	t.Run("increment value", func(t *testing.T) {
		c := setupTestClient(t)
		defer func() { _ = c.Close(context.Background()) }()

		// Create
		tx, err := c.NewRawTx()
		require.NoError(t, err)
		model := &TestModel{Name: "increment_me", Value: 10}
		err = c.SaveModel(context.Background(), model, tx, true, true)
		require.NoError(t, err)

		// Reload model to ensure GORM sees the field value?
		// Actually the issue is that in SQLite, "Value" might not be returned as expected type or something?
		// Or maybe the field name "Value" (snake_case "value") is matching, but `result[fieldName]` is returning nil or 0?
		// The error message "actual: 5" means it treated current value as 0. 0 + 5 = 5.
		// So `result[fieldName]` was likely nil or 0, even though we saved it as 10.

		// Debug: check what is in DB
		var check TestModel
		err = c.GetModel(context.Background(), &check, map[string]any{"name": "increment_me"}, time.Second, false)
		require.NoError(t, err)
		assert.Equal(t, 10, check.Value)

		// Increment - "value" is the column name in DB, "Value" is struct field
		// result is map[string]any from First(&result).
		// In Gorm/SQLite map scan, keys are column names (usually snake_case).
		// So result will have "value", NOT "Value".
		// But if we pass "Value", result["Value"] is nil -> 0.
		// Then update("Value", ...) Gorm maps it to column "value".

		// So we must pass the COLUMN name, or the code needs to handle snake_case conversion?
		// models.go:
		// newValue = convertToInt64(result[fieldName]) + increment
		// tx.Model(&model)...Update(fieldName, newValue)

		// If I pass "value" (lowercase):
		newVal, err := c.IncrementModel(context.Background(), model, "value", 5)
		require.NoError(t, err)
		assert.Equal(t, int64(15), newVal)

		// Verify
		var result TestModel
		err = c.GetModel(context.Background(), &result, map[string]any{"name": "increment_me"}, time.Second, false)
		require.NoError(t, err)
		assert.Equal(t, 15, result.Value)
	})
}

func TestGetModels(t *testing.T) {
	t.Run("get models with conditions", func(t *testing.T) {
		c := setupTestClient(t)
		defer func() { _ = c.Close(context.Background()) }()

		// Create multiple
		tx, err := c.NewRawTx()
		require.NoError(t, err)
		for i := 0; i < 5; i++ {
			model := &TestModel{Name: "group_a", Value: i}
			err = c.SaveModel(context.Background(), model, tx, true, false)
			require.NoError(t, err)
		}
		err = tx.Commit()
		require.NoError(t, err)

		var models []TestModel
		err = c.GetModels(context.Background(), &models, map[string]any{"name": "group_a"}, nil, nil, time.Second)
		require.NoError(t, err)
		assert.Len(t, models, 5)
	})

	t.Run("pagination", func(t *testing.T) {
		c := setupTestClient(t)
		defer func() { _ = c.Close(context.Background()) }()

		tx, err := c.NewRawTx()
		require.NoError(t, err)
		for i := 0; i < 10; i++ {
			model := &TestModel{Name: "paged", Value: i}
			err = c.SaveModel(context.Background(), model, tx, true, false)
			require.NoError(t, err)
		}
		err = tx.Commit()
		require.NoError(t, err)

		var models []TestModel
		queryParams := &QueryParams{Page: 1, PageSize: 3}
		err = c.GetModels(context.Background(), &models, map[string]any{"name": "paged"}, queryParams, nil, time.Second)
		require.NoError(t, err)
		assert.Len(t, models, 3)

		queryParams = &QueryParams{Page: 2, PageSize: 3}
		err = c.GetModels(context.Background(), &models, map[string]any{"name": "paged"}, queryParams, nil, time.Second)
		require.NoError(t, err)
		assert.Len(t, models, 3)
	})
}

func TestGetModelCount(t *testing.T) {
	t.Run("count models", func(t *testing.T) {
		c := setupTestClient(t)
		defer func() { _ = c.Close(context.Background()) }()

		tx, err := c.NewRawTx()
		require.NoError(t, err)
		for i := 0; i < 7; i++ {
			model := &TestModel{Name: "count_me", Value: i}
			err = c.SaveModel(context.Background(), model, tx, true, false)
			require.NoError(t, err)
		}
		err = tx.Commit()
		require.NoError(t, err)

		count, err := c.GetModelCount(context.Background(), &TestModel{}, map[string]any{"name": "count_me"}, time.Second)
		require.NoError(t, err)
		assert.Equal(t, int64(7), count)
	})
}

func TestGetModelPartial(t *testing.T) {
	t.Run("get model partial", func(t *testing.T) {
		c := setupTestClient(t)
		defer func() { _ = c.Close(context.Background()) }()

		tx, err := c.NewRawTx()
		require.NoError(t, err)
		model := &TestModel{Name: "partial_me", Value: 300}
		err = c.SaveModel(context.Background(), model, tx, true, true)
		require.NoError(t, err)

		var result TestModel
		// Select only ID and Name
		err = c.GetModelPartial(context.Background(), &result, []string{"id", "name"}, map[string]any{"name": "partial_me"}, time.Second, false)
		require.NoError(t, err)
		assert.Equal(t, "partial_me", result.Name)
		assert.Equal(t, 0, result.Value) // Value should be 0 because it wasn't selected
	})
}

func TestGetModelsPartial(t *testing.T) {
	t.Run("get models partial", func(t *testing.T) {
		c := setupTestClient(t)
		defer func() { _ = c.Close(context.Background()) }()

		tx, err := c.NewRawTx()
		require.NoError(t, err)
		for i := 0; i < 3; i++ {
			model := &TestModel{Name: "partial_models", Value: i}
			err = c.SaveModel(context.Background(), model, tx, true, false)
			require.NoError(t, err)
		}
		err = tx.Commit()
		require.NoError(t, err)

		var models []TestModel
		err = c.GetModelsPartial(context.Background(), &models, []string{"id", "value"}, map[string]any{"name": "partial_models"}, time.Second)
		require.NoError(t, err)
		assert.Len(t, models, 3)
		for _, m := range models {
			assert.Empty(t, m.Name) // Name should be empty
			assert.GreaterOrEqual(t, m.Value, 0)
		}
	})
}

func TestExecuteAndRaw(t *testing.T) {
	t.Run("execute and raw sql", func(t *testing.T) {
		c := setupTestClient(t)
		defer func() { _ = c.Close(context.Background()) }()

		tx, err := c.NewRawTx()
		require.NoError(t, err)
		model := &TestModel{Name: "sql_exec", Value: 500}
		err = c.SaveModel(context.Background(), model, tx, true, true)
		require.NoError(t, err)

		// Raw
		// The Raw/Execute methods take only a query string?
		// models.go:
		// func (c *Client) Raw(query string) *gorm.DB {
		// 	if IsSQLEngine(c.Engine()) {
		// 		return c.options.db.Raw(query)
		// 	}
		// 	return nil
		// }
		// It seems it does not take args in the current implementation!
		// It just passes the query string to db.Raw(query).
		// Wait, gorm.Raw takes (sql string, values ...interface{}).
		// But the interface in `models.go` is `Raw(query string)`.

		// So I must format the string myself or fix the interface?
		// The tool `read_file` of `models.go` showed:
		// func (c *Client) Execute(query string) *gorm.DB
		// func (c *Client) Raw(query string) *gorm.DB

		// So passing args is not supported by the wrapper.
		// I will format the string.

		var count int64
		db := c.Raw("SELECT count(*) FROM test_models WHERE name = 'sql_exec'")
		require.NoError(t, db.Error)
		db.Scan(&count)
		assert.Equal(t, int64(1), count)

		// Execute
		db = c.Execute("UPDATE test_models SET value = 501 WHERE name = 'sql_exec'")
		require.NoError(t, db.Error)

		var result TestModel
		err = c.GetModel(context.Background(), &result, map[string]any{"name": "sql_exec"}, time.Second, false)
		require.NoError(t, err)
		assert.Equal(t, 501, result.Value)
	})
}
