package datastore

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockClient is a mock implementation of the Client interface
type MockClient struct {
	mock.Mock
}

// IndexExists is a mock implementation of the IndexExists method
func (m *MockClient) IndexExists(tableName, indexName string) (bool, error) {
	args := m.Called(tableName, indexName)
	return args.Bool(0), args.Error(1)
}

// Raw is a mock implementation of the Raw method
func (m *MockClient) Raw(query string, args ...any) *MockTx {
	args = append([]any{query}, args...)
	return m.Called(args...).Get(0).(*MockTx)
}

// GetDatabaseName is a mock implementation of the GetDatabaseName method
func (m *MockClient) GetDatabaseName() string {
	return m.Called().String(0)
}

// Execute is a mock implementation of the Execute method
func (m *MockClient) Execute(query string, args ...any) *MockTx {
	args = append([]any{query}, args...)
	return m.Called(args...).Get(0).(*MockTx)
}

/*
// indexExistsMySQL is a mock implementation of the indexExistsMySQL method
func (m *MockClient) indexExistsMySQL(tableName, indexName string) (bool, error) {
	args := m.Called(tableName, indexName)
	return args.Bool(0), args.Error(1)
}
*/

// IndexMetadata is a mock implementation of the IndexMetadata method
func (m *MockClient) IndexMetadata(tableName, metadata string) error {
	args := m.Called(tableName, metadata)
	return args.Error(0)
}

// MockTx is a mock implementation of the transaction
type MockTx struct {
	mock.Mock

	Error error
}

// Scan is a mock implementation of the Scan method
func (tx *MockTx) Scan(dest any) *MockTx {
	args := tx.Called(dest)
	return args.Get(0).(*MockTx)
}

// TestClient_IndexExists tests the IndexExists method
func TestClient_IndexExists(t *testing.T) {
	mockClient := new(MockClient)

	// Define the expected behavior
	mockClient.On("IndexExists", "test_table", "test_index").Return(true, nil)

	// Call the method
	exists, err := mockClient.IndexExists("test_table", "test_index")

	// Assert the results
	require.NoError(t, err)
	assert.True(t, exists)

	// Assert that the expectations were met
	mockClient.AssertExpectations(t)
}
