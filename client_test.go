package datastore

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testClient will generate a test client
func testClient(ctx context.Context, t *testing.T, opts ...ClientOps) (ClientInterface, func()) {
	client, err := NewClient(ctx, opts...)
	require.NoError(t, err)
	require.NotNil(t, client)
	return client, func() {
		_ = client.Close(ctx)
	}
}

// TestClient_IsDebug will test the method IsDebug()
func TestClient_IsDebug(t *testing.T) {
	t.Run("toggle debug", func(t *testing.T) {
		c, err := NewClient(context.Background(), WithDebugging())
		require.NotNil(t, c)
		require.NoError(t, err)

		assert.True(t, c.IsDebug())

		c.Debug(false)

		assert.False(t, c.IsDebug())
	})

	// Attempt to remove a file created during the test
	t.Cleanup(func() {
		_ = os.Remove("datastore.db")
	})
}

// TestClient_Debug will test the method Debug()
func TestClient_Debug(t *testing.T) {
	t.Run("turn debug on", func(t *testing.T) {
		c, err := NewClient(context.Background())
		require.NotNil(t, c)
		require.NoError(t, err)

		assert.False(t, c.IsDebug())

		c.Debug(true)

		assert.True(t, c.IsDebug())
	})

	// Attempt to remove a file created during the test
	t.Cleanup(func() {
		_ = os.Remove("datastore.db")
	})
}

// TestClient_DebugLog will test the method DebugLog()
func TestClient_DebugLog(t *testing.T) {
	t.Run("write debug log", func(t *testing.T) {
		c, err := NewClient(context.Background(), WithDebugging())
		require.NotNil(t, c)
		require.NoError(t, err)

		c.DebugLog(context.Background(), "test message")
	})

	// Attempt to remove a file created during the test
	t.Cleanup(func() {
		_ = os.Remove("datastore.db")
	})
}

// TestClient_Engine will test the method Engine()
func TestClient_Engine(t *testing.T) {
	t.Run("[sqlite] - get engine", func(t *testing.T) {
		c, err := NewClient(context.Background(), WithSQLite(&SQLiteConfig{
			DatabasePath: "",
			Shared:       true,
		}))
		assert.NotNil(t, c)
		require.NoError(t, err)
		assert.Equal(t, SQLite, c.Engine())
	})

	t.Run("[mongo] - failed to load", func(t *testing.T) {
		c, err := NewClient(context.Background(), WithMongo(&MongoDBConfig{
			DatabaseName: "test",
			Transactions: false,
			URI:          "",
		}))
		assert.Nil(t, c)
		require.Error(t, err)
	})

	// todo: add MySQL, Postgresql and MongoDB
}

// TestClient_GetTableName will test the method GetTableName()
func TestClient_GetTableName(t *testing.T) {
	t.Run("table prefix", func(t *testing.T) {
		c, err := NewClient(context.Background(), WithDebugging(), WithSQLite(&SQLiteConfig{
			CommonConfig: CommonConfig{
				TablePrefix: testTablePrefix,
			},
			DatabasePath: "",
			Shared:       true,
		}))
		require.NotNil(t, c)
		require.NoError(t, err)

		tableName := c.GetTableName(testModelName)
		assert.Equal(t, testTablePrefix+"_"+testModelName, tableName)
	})

	t.Run("no table prefix", func(t *testing.T) {
		c, err := NewClient(context.Background(), WithDebugging(), WithSQLite(&SQLiteConfig{
			CommonConfig: CommonConfig{
				TablePrefix: "",
			},
			DatabasePath: "",
			Shared:       true,
		}))
		require.NotNil(t, c)
		require.NoError(t, err)

		tableName := c.GetTableName(testModelName)
		assert.Equal(t, testModelName, tableName)
	})

	// Attempt to remove a file created during the test
	t.Cleanup(func() {
		_ = os.Remove("datastore.db")
	})
}

// TestClient_GetDatabaseName will test the method GetDatabaseName()
func TestClient_GetDatabaseName(t *testing.T) {
	t.Skip("these do not fully work since they try to connect to the database")

	t.Run("MySQL database name", func(t *testing.T) {
		c, err := NewClient(context.Background(), WithSQL(MySQL, []*SQLConfig{{Name: "test_db"}}))
		require.Error(t, err)
		require.Nil(t, c)
		assert.Equal(t, "test_db", c.GetDatabaseName())
	})

	t.Run("MongoDB database name", func(t *testing.T) {
		c, err := NewClient(context.Background(), WithMongo(&MongoDBConfig{DatabaseName: "test_db", URI: "mongodb://localhost:27017"}))
		require.Error(t, err)
		require.Nil(t, c)
		assert.Equal(t, "test_db", c.GetDatabaseName())
	})
}

// TestClient_GetArrayFields will test the method GetArrayFields()
func TestClient_GetArrayFields(t *testing.T) {
	t.Run("array fields", func(t *testing.T) {
		c, err := NewClient(context.Background(), WithCustomFields([]string{"field1", "field2"}, nil))
		require.NoError(t, err)
		assert.Equal(t, []string{"field1", "field2"}, c.GetArrayFields())
	})
}

// TestClient_GetObjectFields will test the method GetObjectFields()
func TestClient_GetObjectFields(t *testing.T) {
	t.Run("object fields", func(t *testing.T) {
		c, err := NewClient(context.Background(), WithCustomFields(nil, []string{"field1", "field2"}))
		require.NoError(t, err)
		assert.Contains(t, c.GetObjectFields(), "metadata")
		assert.Contains(t, c.GetObjectFields(), "field1")
		assert.Contains(t, c.GetObjectFields(), "field2")
	})
}
