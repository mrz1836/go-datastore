package datastore

import (
	"context"
	"testing"

	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutoMigrateDatabase(t *testing.T) {
	t.Run("auto migrate sqlite", func(t *testing.T) {
		// Unique DB name
		dbName := "file:memdb_migrate_" + t.Name() + "?mode=memory&cache=shared"
		c, err := NewClient(context.Background(),
			WithSQLite(&SQLiteConfig{
				Shared:       true,
				DatabasePath: dbName,
			}),
		)
		require.NoError(t, err)
		defer c.Close(context.Background())

		// TestModel2 definition
		type TestModel2 struct {
			ID   uint   `gorm:"primaryKey"`
			Name string `gorm:"uniqueIndex"`
		}

		c.(*Client).options.autoMigrate = true

		err = c.AutoMigrateDatabase(context.Background(), &TestModel2{})
		require.NoError(t, err)

		// Verify table exists by trying to insert
		tx, err := c.NewRawTx()
		require.NoError(t, err)

		model := &TestModel2{Name: "test_migrate"}
		err = c.SaveModel(context.Background(), model, tx, true, true)
		require.NoError(t, err)

		var result TestModel2
		// Short timeout was causing issues? Or 0 timeout means something else?
		// CreateCtx implementation: ctx, cancel = context.WithTimeout(ctx, timeout)
		// If timeout is 0, it might be instant timeout?
		// Let's use 1 second.
		err = c.GetModel(context.Background(), &result, map[string]any{"name": "test_migrate"}, 1 * time.Second, false)
		require.NoError(t, err)
		assert.Equal(t, "test_migrate", result.Name)
	})

	t.Run("auto migrate disabled", func(t *testing.T) {
		dbName := "file:memdb_migrate_disabled_" + t.Name() + "?mode=memory&cache=shared"
		c, err := NewClient(context.Background(),
			WithSQLite(&SQLiteConfig{
				Shared:       true,
				DatabasePath: dbName,
			}),
			// Auto migrate default is false, but let's be explicit via options if we could,
			// but defaultClientOptions has autoMigrate: false.
		)
		require.NoError(t, err)
		defer c.Close(context.Background())

		// TestModel3
		type TestModel3 struct {
			ID   uint
			Name string
		}

		// Should not migrate
		err = c.AutoMigrateDatabase(context.Background(), &TestModel3{})
		require.NoError(t, err) // It returns nil but does nothing

		// Verify table does NOT exist
		tx, err := c.NewRawTx()
		require.NoError(t, err)

		model := &TestModel3{Name: "test_migrate_fail"}
		err = c.SaveModel(context.Background(), model, tx, true, true)
		// Should fail because table doesn't exist
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no such table")
	})

	t.Run("duplicate migration check", func(t *testing.T) {
		dbName := "file:memdb_migrate_dup_" + t.Name() + "?mode=memory&cache=shared"
		c, err := NewClient(context.Background(),
			WithSQLite(&SQLiteConfig{
				Shared:       true,
				DatabasePath: dbName,
			}),
		)
		require.NoError(t, err)
		defer c.Close(context.Background())

		type TestModel4 struct {
			ID uint
		}

		c.(*Client).options.autoMigrate = true

		err = c.AutoMigrateDatabase(context.Background(), &TestModel4{})
		require.NoError(t, err)

		// Try again
		err = c.AutoMigrateDatabase(context.Background(), &TestModel4{})
		require.Error(t, err)
		assert.Equal(t, errModelAlreadyMigrated, err.(interface{ Unwrap() error }).Unwrap())
	})
}
