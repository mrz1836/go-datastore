package datastore

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type UserModel struct {
	ID        string `json:"id" gorm:"primaryKey"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	Age       int    `json:"age"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (u *UserModel) GetModelName() string {
	return "users"
}

func (u *UserModel) GetModelTableName() string {
	return "users"
}

func (u *UserModel) TableName() string {
	return "users"
}

type UserPartial struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type UserPartialWithID struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
}

func TestGetModelsWithSelection(t *testing.T) {
	// Setup SQLite
	c, err := NewClient(context.Background(), WithSQLite(&SQLiteConfig{
		DatabasePath: "test_selection.db",
		Shared:       true,
	}))
	require.NoError(t, err)
	require.NotNil(t, c)
	defer func() {
		_ = c.Close(context.Background())
		_ = os.Remove("test_selection.db")
	}()

	c.Debug(true)

	// Manually create table to avoid AutoMigrate issues in test env
	db := c.Execute("CREATE TABLE users (id text primary key, name text, email text, password text, age integer, created_at datetime, updated_at datetime)")
	if db.Error != nil {
		t.Logf("Create table error: %v", db.Error)
	}

	// Check tables
	var tables []string
	db = c.Raw("SELECT name FROM sqlite_master WHERE type='table'")
	if db != nil {
		db.Scan(&tables)
		fmt.Printf("Tables: %v\n", tables)
	}

	// Insert data
	users := []UserModel{
		{ID: "1", Name: "Alice", Email: "alice@example.com", Password: "secret", Age: 30},
		{ID: "2", Name: "Bob", Email: "bob@example.com", Password: "secret", Age: 25},
	}
	err = c.CreateInBatches(context.Background(), &users, 2)
	require.NoError(t, err)

	t.Run("GetModels with partial struct", func(t *testing.T) {
		var partials []UserPartial
		err := c.GetModels(context.Background(), &[]UserModel{}, nil, nil, &partials, 5*time.Second)
		require.NoError(t, err)
		assert.Len(t, partials, 2)

		var alice UserPartial
		for _, u := range partials {
			if u.Name == "Alice" {
				alice = u
				break
			}
		}
		assert.Equal(t, "Alice", alice.Name)
		assert.Equal(t, "alice@example.com", alice.Email)
	})

	t.Run("GetModels with partial struct and ID", func(t *testing.T) {
		var partials []UserPartialWithID
		err := c.GetModels(context.Background(), &[]UserModel{}, nil, nil, &partials, 5*time.Second)
		require.NoError(t, err)
		assert.Len(t, partials, 2)

		var alice UserPartialWithID
		for _, u := range partials {
			if u.Name == "Alice" {
				alice = u
				break
			}
		}
		assert.Equal(t, "1", alice.ID)
		assert.Equal(t, "Alice", alice.Name)
	})

	t.Run("GetModelSelect with partial struct", func(t *testing.T) {
		var partial UserPartial
		// Find Alice
		conditions := map[string]interface{}{"name": "Alice"}
		err := c.GetModelSelect(context.Background(), &UserModel{}, &partial, conditions, 5*time.Second, false)
		require.NoError(t, err)
		assert.Equal(t, "Alice", partial.Name)
		assert.Equal(t, "alice@example.com", partial.Email)
	})

	t.Run("GetModelSelect with partial struct and ID", func(t *testing.T) {
		var partial UserPartialWithID
		conditions := map[string]interface{}{"name": "Alice"}
		err := c.GetModelSelect(context.Background(), &UserModel{}, &partial, conditions, 5*time.Second, false)
		require.NoError(t, err)
		assert.Equal(t, "1", partial.ID)
		assert.Equal(t, "Alice", partial.Name)
	})
}
