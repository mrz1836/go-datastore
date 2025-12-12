package datastore

import (
	"context"
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
	ID   string `json:"id"`
	Name string `json:"name"`
}

type UserPartialWithAge struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type UserFullPartial struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Age      int    `json:"age"`
}

// BaseInterface simulates a common interface pattern for testing unwrapInterface
type BaseInterface interface {
	GetModelName() string
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
		t.Logf("Tables: %v", tables)
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

	t.Run("GetModelPartial with partial struct", func(t *testing.T) {
		var partial UserPartial
		// Find Alice
		conditions := map[string]any{"name": "Alice"}
		err := c.GetModelPartial(context.Background(), &UserModel{}, &partial, conditions, 5*time.Second, false)
		require.NoError(t, err)
		assert.Equal(t, "Alice", partial.Name)
		assert.Equal(t, "alice@example.com", partial.Email)
	})

	t.Run("GetModelPartial with partial struct and ID", func(t *testing.T) {
		var partial UserPartialWithID
		conditions := map[string]any{"name": "Alice"}
		err := c.GetModelPartial(context.Background(), &UserModel{}, &partial, conditions, 5*time.Second, false)
		require.NoError(t, err)
		assert.Equal(t, "1", partial.ID)
		assert.Equal(t, "Alice", partial.Name)
	})

	t.Run("GetModelPartial with []string field names", func(t *testing.T) {
		var user UserModel
		conditions := map[string]any{"name": "Alice"}
		fields := []string{"id", "name", "email"}
		err := c.GetModelPartial(context.Background(), &user, fields, conditions, 5*time.Second, false)
		require.NoError(t, err)
		assert.Equal(t, "1", user.ID)
		assert.Equal(t, "Alice", user.Name)
		assert.Equal(t, "alice@example.com", user.Email)
		// Password should be empty since it wasn't selected
		assert.Empty(t, user.Password)
	})

	t.Run("GetModelPartial with nil fieldResult selects all", func(t *testing.T) {
		var user UserModel
		conditions := map[string]any{"name": "Alice"}
		err := c.GetModelPartial(context.Background(), &user, nil, conditions, 5*time.Second, false)
		require.NoError(t, err)
		assert.Equal(t, "1", user.ID)
		assert.Equal(t, "Alice", user.Name)
		assert.Equal(t, "alice@example.com", user.Email)
		assert.Equal(t, "secret", user.Password)
		assert.Equal(t, 30, user.Age)
	})

	t.Run("GetModelPartial with no conditions", func(t *testing.T) {
		var partial UserPartial
		// No conditions - should return first record
		err := c.GetModelPartial(context.Background(), &UserModel{}, &partial, nil, 5*time.Second, false)
		require.NoError(t, err)
		assert.NotEmpty(t, partial.Name)
	})

	t.Run("GetModelPartial with forceWriteDB", func(t *testing.T) {
		var partial UserPartial
		conditions := map[string]any{"name": "Bob"}
		err := c.GetModelPartial(context.Background(), &UserModel{}, &partial, conditions, 5*time.Second, true)
		require.NoError(t, err)
		assert.Equal(t, "Bob", partial.Name)
		assert.Equal(t, "bob@example.com", partial.Email)
	})

	t.Run("GetModelPartial with age field", func(t *testing.T) {
		var partial UserPartialWithAge
		conditions := map[string]any{"name": "Alice"}
		err := c.GetModelPartial(context.Background(), &UserModel{}, &partial, conditions, 5*time.Second, false)
		require.NoError(t, err)
		assert.Equal(t, "Alice", partial.Name)
		assert.Equal(t, 30, partial.Age)
	})

	// GetModelsPartial tests
	t.Run("GetModelsPartial with partial struct", func(t *testing.T) {
		var partials []UserPartial
		err := c.GetModelsPartial(context.Background(), &[]UserModel{}, &partials, nil, 5*time.Second)
		require.NoError(t, err)
		assert.Len(t, partials, 2)

		names := make(map[string]string)
		for _, u := range partials {
			names[u.Name] = u.Email
		}
		assert.Equal(t, "alice@example.com", names["Alice"])
		assert.Equal(t, "bob@example.com", names["Bob"])
	})

	t.Run("GetModelsPartial with partial struct and ID", func(t *testing.T) {
		var partials []UserPartialWithID
		err := c.GetModelsPartial(context.Background(), &[]UserModel{}, &partials, nil, 5*time.Second)
		require.NoError(t, err)
		assert.Len(t, partials, 2)

		ids := make(map[string]string)
		for _, u := range partials {
			ids[u.Name] = u.ID
		}
		assert.Equal(t, "1", ids["Alice"])
		assert.Equal(t, "2", ids["Bob"])
	})

	t.Run("GetModelsPartial with []string field names", func(t *testing.T) {
		var users []UserModel
		fields := []string{"id", "name"}
		err := c.GetModelsPartial(context.Background(), &users, fields, nil, 5*time.Second)
		require.NoError(t, err)
		assert.Len(t, users, 2)

		for _, u := range users {
			assert.NotEmpty(t, u.ID)
			assert.NotEmpty(t, u.Name)
			// Email and Password should be empty since they weren't selected
			assert.Empty(t, u.Email)
			assert.Empty(t, u.Password)
		}
	})

	t.Run("GetModelsPartial with nil fieldResults selects all", func(t *testing.T) {
		var users []UserModel
		err := c.GetModelsPartial(context.Background(), &users, nil, nil, 5*time.Second)
		require.NoError(t, err)
		assert.Len(t, users, 2)

		for _, u := range users {
			assert.NotEmpty(t, u.ID)
			assert.NotEmpty(t, u.Name)
			assert.NotEmpty(t, u.Email)
			assert.NotEmpty(t, u.Password)
		}
	})

	t.Run("GetModelsPartial with conditions", func(t *testing.T) {
		var partials []UserPartial
		conditions := map[string]any{"age": 30}
		err := c.GetModelsPartial(context.Background(), &[]UserModel{}, &partials, conditions, 5*time.Second)
		require.NoError(t, err)
		assert.Len(t, partials, 1)
		assert.Equal(t, "Alice", partials[0].Name)
	})

	t.Run("GetModelsPartial with conditions and []string fields", func(t *testing.T) {
		var users []UserModel
		conditions := map[string]any{"name": "Bob"}
		fields := []string{"id", "email"}
		err := c.GetModelsPartial(context.Background(), &users, fields, conditions, 5*time.Second)
		require.NoError(t, err)
		assert.Len(t, users, 1)
		assert.Equal(t, "2", users[0].ID)
		assert.Equal(t, "bob@example.com", users[0].Email)
		assert.Empty(t, users[0].Name) // Not selected
	})

	t.Run("GetModelsPartial with multiple conditions", func(t *testing.T) {
		var partials []UserPartialWithAge
		conditions := map[string]any{
			"age": map[string]any{
				"$gte": 25,
			},
		}
		err := c.GetModelsPartial(context.Background(), &[]UserModel{}, &partials, conditions, 5*time.Second)
		require.NoError(t, err)
		assert.Len(t, partials, 2)
	})

	t.Run("GetModelsPartial with no matching conditions", func(t *testing.T) {
		var partials []UserPartial
		conditions := map[string]any{"name": "NonExistent"}
		err := c.GetModelsPartial(context.Background(), &[]UserModel{}, &partials, conditions, 5*time.Second)
		// Should return error for no results
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNoResults)
	})

	t.Run("GetModelsPartial with full partial struct", func(t *testing.T) {
		var partials []UserFullPartial
		err := c.GetModelsPartial(context.Background(), &[]UserModel{}, &partials, nil, 5*time.Second)
		require.NoError(t, err)
		assert.Len(t, partials, 2)

		for _, u := range partials {
			assert.NotEmpty(t, u.ID)
			assert.NotEmpty(t, u.Name)
			assert.NotEmpty(t, u.Email)
			assert.NotEmpty(t, u.Password)
			assert.NotZero(t, u.Age)
		}
	})
}

// TestUnwrapInterface tests the unwrapInterface function
func TestUnwrapInterface(t *testing.T) {
	t.Parallel()

	t.Run("direct pointer", func(t *testing.T) {
		user := &UserModel{ID: "1", Name: "Test"}
		result := unwrapInterface(user)
		assert.Equal(t, user, result)
	})

	t.Run("single interface wrapper", func(t *testing.T) {
		user := &UserModel{ID: "1", Name: "Test"}
		var iface any = user
		result := unwrapInterface(iface)
		assert.Equal(t, user, result)
	})

	t.Run("nested interface wrappers", func(t *testing.T) {
		user := &UserModel{ID: "1", Name: "Test"}
		var baseIface BaseInterface = user
		var anyIface any = baseIface
		result := unwrapInterface(anyIface)
		// Should unwrap to the concrete *UserModel
		resultUser, ok := result.(*UserModel)
		require.True(t, ok, "expected *UserModel, got %T", result)
		assert.Equal(t, "1", resultUser.ID)
		assert.Equal(t, "Test", resultUser.Name)
	})

	t.Run("non-interface value (int)", func(t *testing.T) {
		val := 42
		result := unwrapInterface(val)
		assert.Equal(t, 42, result)
	})

	t.Run("non-interface value (string)", func(t *testing.T) {
		val := "hello"
		result := unwrapInterface(val)
		assert.Equal(t, "hello", result)
	})

	t.Run("slice value", func(t *testing.T) {
		users := []UserModel{{ID: "1"}, {ID: "2"}}
		result := unwrapInterface(users)
		resultSlice, ok := result.([]UserModel)
		require.True(t, ok)
		assert.Len(t, resultSlice, 2)
	})

	t.Run("pointer to slice", func(t *testing.T) {
		users := &[]UserModel{{ID: "1"}, {ID: "2"}}
		result := unwrapInterface(users)
		resultSlice, ok := result.(*[]UserModel)
		require.True(t, ok)
		assert.Len(t, *resultSlice, 2)
	})

	t.Run("interface containing slice pointer", func(t *testing.T) {
		users := &[]UserModel{{ID: "1"}, {ID: "2"}}
		var iface any = users
		result := unwrapInterface(iface)
		resultSlice, ok := result.(*[]UserModel)
		require.True(t, ok)
		assert.Len(t, *resultSlice, 2)
	})

	t.Run("double nested interface", func(t *testing.T) {
		user := &UserModel{ID: "1", Name: "DoubleNested"}
		var inner any = user
		outer := inner
		result := unwrapInterface(outer)
		resultUser, ok := result.(*UserModel)
		require.True(t, ok)
		assert.Equal(t, "DoubleNested", resultUser.Name)
	})

	t.Run("nil value", func(t *testing.T) {
		var user *UserModel
		result := unwrapInterface(user)
		assert.Nil(t, result)
	})

	t.Run("interface containing nil", func(t *testing.T) {
		var user *UserModel
		var iface any = user
		result := unwrapInterface(iface)
		assert.Nil(t, result)
	})
}

// TestGetModelPartialEdgeCases tests edge cases for GetModelPartial
func TestGetModelPartialEdgeCases(t *testing.T) {
	// Setup SQLite
	c, err := NewClient(context.Background(), WithSQLite(&SQLiteConfig{
		DatabasePath: "test_partial_edge.db",
		Shared:       true,
	}))
	require.NoError(t, err)
	require.NotNil(t, c)
	defer func() {
		_ = c.Close(context.Background())
		_ = os.Remove("test_partial_edge.db")
	}()

	// Create table and insert data
	db := c.Execute("CREATE TABLE users (id text primary key, name text, email text, password text, age integer, created_at datetime, updated_at datetime)")
	require.NoError(t, db.Error)

	users := []UserModel{
		{ID: "1", Name: "Alice", Email: "alice@example.com", Password: "secret1", Age: 30},
		{ID: "2", Name: "Bob", Email: "bob@example.com", Password: "secret2", Age: 25},
		{ID: "3", Name: "Charlie", Email: "charlie@example.com", Password: "secret3", Age: 35},
	}
	err = c.CreateInBatches(context.Background(), &users, 3)
	require.NoError(t, err)

	t.Run("GetModelPartial non-existent record returns error", func(t *testing.T) {
		var partial UserPartial
		conditions := map[string]any{"id": "999"}
		err := c.GetModelPartial(context.Background(), &UserModel{}, &partial, conditions, 5*time.Second, false)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNoResults)
	})

	t.Run("GetModelPartial with empty []string fields selects nothing meaningful", func(t *testing.T) {
		var user UserModel
		conditions := map[string]any{"name": "Alice"}
		fields := []string{}
		err := c.GetModelPartial(context.Background(), &user, fields, conditions, 5*time.Second, false)
		// Empty field selection behavior depends on GORM - may select all or none
		require.NoError(t, err)
	})

	t.Run("GetModelPartial with single field", func(t *testing.T) {
		var user UserModel
		conditions := map[string]any{"name": "Charlie"}
		fields := []string{"name"}
		err := c.GetModelPartial(context.Background(), &user, fields, conditions, 5*time.Second, false)
		require.NoError(t, err)
		assert.Equal(t, "Charlie", user.Name)
		assert.Empty(t, user.ID)
		assert.Empty(t, user.Email)
	})

	t.Run("GetModelPartial with IN condition", func(t *testing.T) {
		var partial UserPartial
		conditions := map[string]any{
			"name": map[string]any{
				"$in": []string{"Alice", "Bob"},
			},
		}
		err := c.GetModelPartial(context.Background(), &UserModel{}, &partial, conditions, 5*time.Second, false)
		require.NoError(t, err)
		assert.True(t, partial.Name == "Alice" || partial.Name == "Bob")
	})

	t.Run("GetModelPartial with complex conditions", func(t *testing.T) {
		var partial UserPartialWithAge
		conditions := map[string]any{
			"age": map[string]any{
				"$gt": 25,
				"$lt": 35,
			},
		}
		err := c.GetModelPartial(context.Background(), &UserModel{}, &partial, conditions, 5*time.Second, false)
		require.NoError(t, err)
		assert.Equal(t, "Alice", partial.Name)
		assert.Equal(t, 30, partial.Age)
	})
}

// TestGetModelsPartialEdgeCases tests edge cases for GetModelsPartial
func TestGetModelsPartialEdgeCases(t *testing.T) {
	// Setup SQLite
	c, err := NewClient(context.Background(), WithSQLite(&SQLiteConfig{
		DatabasePath: "test_models_partial_edge.db",
		Shared:       true,
	}))
	require.NoError(t, err)
	require.NotNil(t, c)
	defer func() {
		_ = c.Close(context.Background())
		_ = os.Remove("test_models_partial_edge.db")
	}()

	// Create table and insert data
	db := c.Execute("CREATE TABLE users (id text primary key, name text, email text, password text, age integer, created_at datetime, updated_at datetime)")
	require.NoError(t, db.Error)

	users := []UserModel{
		{ID: "1", Name: "Alice", Email: "alice@example.com", Password: "secret1", Age: 30},
		{ID: "2", Name: "Bob", Email: "bob@example.com", Password: "secret2", Age: 25},
		{ID: "3", Name: "Charlie", Email: "charlie@example.com", Password: "secret3", Age: 35},
		{ID: "4", Name: "Diana", Email: "diana@example.com", Password: "secret4", Age: 28},
	}
	err = c.CreateInBatches(context.Background(), &users, 4)
	require.NoError(t, err)

	t.Run("GetModelsPartial with all fields via []string", func(t *testing.T) {
		var users []UserModel
		fields := []string{"id", "name", "email", "password", "age"}
		err := c.GetModelsPartial(context.Background(), &users, fields, nil, 5*time.Second)
		require.NoError(t, err)
		assert.Len(t, users, 4)
		for _, u := range users {
			assert.NotEmpty(t, u.ID)
			assert.NotEmpty(t, u.Name)
			assert.NotEmpty(t, u.Email)
			assert.NotEmpty(t, u.Password)
		}
	})

	t.Run("GetModelsPartial with range conditions", func(t *testing.T) {
		var partials []UserPartialWithAge
		conditions := map[string]any{
			"age": map[string]any{
				"$gte": 28,
				"$lte": 35,
			},
		}
		err := c.GetModelsPartial(context.Background(), &[]UserModel{}, &partials, conditions, 5*time.Second)
		require.NoError(t, err)
		assert.Len(t, partials, 3) // Alice (30), Charlie (35), Diana (28)
	})

	t.Run("GetModelsPartial with equality condition", func(t *testing.T) {
		var partials []UserPartial
		conditions := map[string]any{"name": "Alice"}
		err := c.GetModelsPartial(context.Background(), &[]UserModel{}, &partials, conditions, 5*time.Second)
		require.NoError(t, err)
		assert.Len(t, partials, 1)
		assert.Equal(t, "Alice", partials[0].Name)
	})

	t.Run("GetModelsPartial with multiple equality conditions", func(t *testing.T) {
		var partials []UserPartial
		conditions := map[string]any{
			"name": "Bob",
			"age":  25,
		}
		err := c.GetModelsPartial(context.Background(), &[]UserModel{}, &partials, conditions, 5*time.Second)
		require.NoError(t, err)
		assert.Len(t, partials, 1)
		assert.Equal(t, "Bob", partials[0].Name)
	})

	t.Run("GetModelsPartial destination is same as models", func(t *testing.T) {
		var users []UserModel
		// Use models slice as both source and destination
		err := c.GetModelsPartial(context.Background(), &users, nil, nil, 5*time.Second)
		require.NoError(t, err)
		assert.Len(t, users, 4)
	})

	t.Run("GetModelsPartial with empty result set", func(t *testing.T) {
		var partials []UserPartial
		conditions := map[string]any{"age": 100}
		err := c.GetModelsPartial(context.Background(), &[]UserModel{}, &partials, conditions, 5*time.Second)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNoResults)
	})

	t.Run("GetModelsPartial with single field []string", func(t *testing.T) {
		var users []UserModel
		fields := []string{"email"}
		err := c.GetModelsPartial(context.Background(), &users, fields, nil, 5*time.Second)
		require.NoError(t, err)
		assert.Len(t, users, 4)
		for _, u := range users {
			assert.NotEmpty(t, u.Email)
			assert.Empty(t, u.Name) // Not selected
			assert.Empty(t, u.ID)   // Not selected
		}
	})

	t.Run("GetModelsPartial with greater than condition", func(t *testing.T) {
		var partials []UserPartialWithAge
		conditions := map[string]any{
			"age": map[string]any{
				"$gt": 28,
			},
		}
		err := c.GetModelsPartial(context.Background(), &[]UserModel{}, &partials, conditions, 5*time.Second)
		require.NoError(t, err)
		assert.Len(t, partials, 2) // Alice (30), Charlie (35)
	})

	t.Run("GetModelsPartial error on non-slice models", func(t *testing.T) {
		var user UserModel
		err := c.GetModelsPartial(context.Background(), &user, nil, nil, 5*time.Second)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a slice")
	})
}
