package datastore

import (
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// errTestBoom is a static error used for testing error scenarios.
var errTestBoom = errors.New("boom")

// TestIndexExists verifies the public IndexExists dispatcher covers unsupported engines.
func TestIndexExists(t *testing.T) {
	t.Parallel()

	client := &Client{options: &clientOptions{engine: PostgreSQL}}
	exists, err := client.IndexExists("table", "idx")
	require.ErrorIs(t, err, ErrUnknownSQL)
	assert.False(t, exists)
}

// TestIndexExistsMySQL exercises success and failure paths of the MySQL index probe.
func TestIndexExistsMySQL(t *testing.T) {
	t.Parallel()

	query := `SELECT 1
                        FROM INFORMATION_SCHEMA.STATISTICS
                        WHERE TABLE_SCHEMA = 'test_db'
                          AND TABLE_NAME = 'table'
                          AND INDEX_NAME = 'idx'`

	t.Run("index exists", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		mock.ExpectQuery(regexp.QuoteMeta(query)).WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))

		gormDB, err := gorm.Open(mysql.New(mysql.Config{Conn: db, SkipInitializeWithVersion: true}), &gorm.Config{})
		require.NoError(t, err)

		client := &Client{options: &clientOptions{
			engine: MySQL,
			db:     gormDB,
			sqlConfigs: []*SQLConfig{{
				Name: "test_db",
			}},
		}}

		exists, err := client.indexExistsMySQL("table", "idx")
		require.NoError(t, err)
		assert.True(t, exists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		mock.ExpectQuery(regexp.QuoteMeta(query)).WillReturnError(errTestBoom)

		gormDB, err := gorm.Open(mysql.New(mysql.Config{Conn: db, SkipInitializeWithVersion: true}), &gorm.Config{})
		require.NoError(t, err)

		client := &Client{options: &clientOptions{engine: MySQL, db: gormDB, sqlConfigs: []*SQLConfig{{Name: "test_db"}}}}

		exists, err := client.indexExistsMySQL("table", "idx")
		require.Error(t, err)
		assert.False(t, exists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("scan failure", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		mock.ExpectQuery(regexp.QuoteMeta(query)).WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow("bad"))

		gormDB, err := gorm.Open(mysql.New(mysql.Config{Conn: db, SkipInitializeWithVersion: true}), &gorm.Config{})
		require.NoError(t, err)

		client := &Client{options: &clientOptions{engine: MySQL, db: gormDB, sqlConfigs: []*SQLConfig{{Name: "test_db"}}}}

		exists, err := client.indexExistsMySQL("table", "idx")
		require.Error(t, err)
		assert.False(t, exists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
