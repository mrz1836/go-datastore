package datastore

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// TestModel is a simple model for testing
type TestModel struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
	Name      string
	Value     int
}

// GetModelName will return a model name
func (t *TestModel) GetModelName() string {
	return "test_models"
}

func setupTestClient(t *testing.T) ClientInterface {
	// Unique database name for each test run to avoid locking
	dbName := "file:memdb" + t.Name() + "?mode=memory&cache=shared"
	c, err := NewClient(context.Background(),
		WithSQLite(&SQLiteConfig{
			Shared:       true,
			DatabasePath: dbName,
		}),
	)
	require.NoError(t, err)
	require.NotNil(t, c)

	// Enable auto migration so AutoMigrateDatabase actually works
	c.(*Client).options.autoMigrate = true

	err = c.AutoMigrateDatabase(context.Background(), &TestModel{})
	require.NoError(t, err)

	return c
}

func TestNewTx(t *testing.T) {
	t.Run("basic transaction commit", func(t *testing.T) {
		c := setupTestClient(t)
		defer func() { _ = c.Close(context.Background()) }()

		err := c.NewTx(context.Background(), func(tx *Transaction) error {
			model := &TestModel{Name: "test1", Value: 10}
			// Don't commit in SaveModel, let NewTx handle it?
			// Actually NewTx DOES NOT commit automatically unless we tell it to?
			// The fn passed to NewTx does operations.
			// BUT `NewTx` implementation does:
			// return fn(&Transaction{ sqlTx: sessionDb.Begin() })
			// It returns the error of fn. It does NOT commit or rollback automatically based on error.
			// Wait, let me check transaction.go again.

			// NewTx just creates a transaction and passes it to fn.
			// It is the responsibility of fn (or functions called within) to commit.
			// SaveModel has a commitTx boolean.

			err := c.SaveModel(context.Background(), model, tx, true, false) // Don't commit yet?
			if err != nil {
				return err
			}
			return tx.Commit()
		})
		require.NoError(t, err)

		var model TestModel
		err = c.GetModel(context.Background(), &model, map[string]any{"name": "test1"}, time.Second, false)
		require.NoError(t, err)
		assert.Equal(t, "test1", model.Name)
	})

	t.Run("transaction rollback error", func(t *testing.T) {
		c := setupTestClient(t)
		defer func() { _ = c.Close(context.Background()) }()

		err := c.NewTx(context.Background(), func(tx *Transaction) error {
			model := &TestModel{Name: "test2", Value: 20}
			err := c.SaveModel(context.Background(), model, tx, true, false)
			if err != nil {
				return err
			}
			// Manually rollback or just return error (caller should handle rollback if NewTx doesn't)
			// transaction.go NewTx doesn't handle rollback on error!
			// This seems like a flaw or design choice in NewTx.
			// Checking NewTx source:
			/*
				func (c *Client) NewTx(ctx context.Context, fn func(*Transaction) error) error {
					if c.options.db != nil {
						sessionDb := c.options.db.Session(...)
						return fn(&Transaction{
							sqlTx: sessionDb.Begin(),
						})
					}
					...
				}
			*/
			// It just returns fn(...). So if fn returns error, the transaction is left open/uncommitted/unrolled back?
			// If sessionDb.Begin() is called, a transaction is started.
			// If fn returns error, who rolls back?
			// It seems the user is responsible for rollback in fn.

			_ = tx.Rollback()
			return assert.AnError // Simulate error
		})
		require.Error(t, err)

		var model TestModel
		err = c.GetModel(context.Background(), &model, map[string]any{"name": "test2"}, time.Second, false)
		require.Error(t, err) // Should not exist
		assert.True(t, errors.Is(err, ErrNoResults) || errors.Is(err, gorm.ErrRecordNotFound))
	})
}

func TestNewRawTx(t *testing.T) {
	t.Run("basic raw transaction", func(t *testing.T) {
		c := setupTestClient(t)
		defer func() { _ = c.Close(context.Background()) }()

		tx, err := c.NewRawTx()
		require.NoError(t, err)
		require.NotNil(t, tx)

		model := &TestModel{Name: "test3", Value: 30}
		err = c.SaveModel(context.Background(), model, tx, true, false)
		require.NoError(t, err)

		err = tx.Commit()
		require.NoError(t, err)

		var result TestModel
		err = c.GetModel(context.Background(), &result, map[string]any{"name": "test3"}, time.Second, false)
		require.NoError(t, err)
		assert.Equal(t, "test3", result.Name)
	})

	t.Run("raw transaction rollback", func(t *testing.T) {
		c := setupTestClient(t)
		defer func() { _ = c.Close(context.Background()) }()

		tx, err := c.NewRawTx()
		require.NoError(t, err)
		require.NotNil(t, tx)

		model := &TestModel{Name: "test4", Value: 40}
		err = c.SaveModel(context.Background(), model, tx, true, false)
		require.NoError(t, err)

		err = tx.Rollback()
		require.NoError(t, err)

		var result TestModel
		err = c.GetModel(context.Background(), &result, map[string]any{"name": "test4"}, time.Second, false)
		require.Error(t, err)
	})
}

func TestTransaction_CanCommit(t *testing.T) {
	t.Run("can commit", func(t *testing.T) {
		c := setupTestClient(t)
		defer func() { _ = c.Close(context.Background()) }()

		tx, err := c.NewRawTx()
		require.NoError(t, err)

		assert.True(t, tx.CanCommit())

		err = tx.Commit()
		require.NoError(t, err)

		assert.False(t, tx.CanCommit())
	})
}

func TestTransaction_CommitErrorWithRollback(t *testing.T) {
	t.Run("commit failure triggers rollback", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		mock.MatchExpectationsInOrder(true)

		mock.ExpectBegin()
		mock.ExpectCommit().WillReturnError(assert.AnError)

		gormDB, err := gorm.Open(mysql.New(mysql.Config{Conn: db, SkipInitializeWithVersion: true}))
		require.NoError(t, err)

		tx := &Transaction{sqlTx: gormDB.Begin()}

		err = tx.Commit()
		require.Error(t, err)
		assert.Contains(t, err.Error(), assert.AnError.Error())
		assert.False(t, tx.committed)
		assert.Zero(t, tx.rowsAffected)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("commit ignores nil transactions", func(t *testing.T) {
		tx := &Transaction{}
		err := tx.Commit()
		require.NoError(t, err)
		assert.False(t, tx.committed)
		assert.Zero(t, tx.rowsAffected)
	})
}

func TestTransaction_RollbackSafe(t *testing.T) {
	t.Run("nil transaction rolls back without error", func(t *testing.T) {
		tx := &Transaction{}
		require.NoError(t, tx.Rollback())
	})
}
