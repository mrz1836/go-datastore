package datastore

import (
	"database/sql"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"
)

// Defaults for library functionality
const (
	defaultDatabaseCreateIndexTimeout = 20 * time.Second  // Default timeout for creating indexes
	defaultDatabaseMaxIdleTime        = 360 * time.Second // Default max idle open connection time
	defaultDatabaseMaxTimeout         = 60 * time.Second  // Default max timeout on a query
	defaultDatabaseTxTimeout          = 10 * time.Second  // Default transaction timeout
	defaultMySQLHost                  = "localhost"       // Default host for MySQL
	defaultMySQLPort                  = "3306"            // Default port for MySQL
	defaultPageSize                   = 20                // The default number of results to return
	defaultPostgreSQLHost             = "localhost"       // Default host for PostgreSQL
	defaultPostgreSQLPort             = "5432"            // Default port for PostgreSQL
	defaultPostgreSQLSslMode          = "disable"         // Default sslmode for PostgreSQL
	defaultSQLiteFileName             = "datastore.db"    // Default database filename
	defaultSQLiteSharing              = true              // Default value for "sharing" in loading an SQLite database
	defaultTablePrefix                = "x"               // Default database prefix for table names (x_model)
	defaultTimeZone                   = "UTC"             // Default is UTC (IE: America/New_York)
	emptyTimeDuration                 = "0s"              // Empty time duration for comparison
	maxIdleConnectionsSQLite          = 1                 // The max for SQLite (in-memory)

	// Fields and Field Names
	accumulationCountField = "count"       // The field for accumulating
	dateCreatedAt          = "created_at"  // Field for record-created time
	dateModifiedAt         = "modified_at" // Field for record-modified time
	dateUpdatedAt          = "updated_at"  // Field for record-updated time
	metadataField          = "metadata"    // The metadata field
	mongoIDField           = "_id"         // The ID field for mongo
	sqlIDField             = "id"          // The ID field for SQL
	sqlIDFieldProper       = "ID"          // The ID field for SQL (capitalized)

	// Field types and tags
	bsonTagName         = "bson"       // Tag name for BSON
	nullStringFieldType = "NullString" // Field type name for Null String
	nullTimeFieldType   = "NullTime"   // Field type name for Null Time

	// Conditions
	conditionAnd                = "$and"          // Condition for an AND statement
	conditionDateToString       = "$dateToString" // Condition for a Date to String command
	conditionExists             = "$exists"       // Condition for an EXISTS statement
	conditionGreaterThan        = "$gt"           // Condition for greater than (>)
	conditionGreaterThanOrEqual = "$gte"          // Condition for greater than or equal (>=)
	conditionGroup              = "$group"        // Condition for a GROUP command
	conditionIn                 = "$in"           // Condition for an IN statement
	conditionIncrement          = "$inc"          // Condition for an INCREMENT command
	conditionLessThan           = "$lt"           // Condition for less than ( < )
	conditionLessThanOrEqual    = "$lte"          // Condition for less than or equal (<=)
	conditionMatch              = "$match"        // Condition for a MATCH command
	conditionNotEquals          = "$ne"           // Condition for doesn't equal (!=)
	conditionNotIn              = "$nin"          // Condition for a NOT IN statement
	conditionOr                 = "$or"           // Condition for an OR statement
	conditionSet                = "$set"          // Condition for a SET command
	conditionSum                = "$sum"          // Condition for a SUM command
	conditionUnSet              = "$unset"        // Condition for an UNSET command

	// SortDesc will sort descending
	SortDesc = "desc"

	// SortAsc will sort ascending
	SortAsc = "asc"
)

var (
	// DateFields are standard known date fields
	DateFields = []string{dateCreatedAt, dateUpdatedAt, dateModifiedAt}
)

// CommonConfig is the common configuration fields between engines
type CommonConfig struct {
	Debug                 bool          `json:"debug" mapstructure:"debug"`                                       // flag for debugging sql queries in logs
	MaxConnectionIdleTime time.Duration `json:"max_connection_idle_time" mapstructure:"max_connection_idle_time"` // 360
	MaxConnectionTime     time.Duration `json:"max_connection_time" mapstructure:"max_connection_time"`           // 60
	MaxIdleConnections    int           `json:"max_idle_connections" mapstructure:"max_idle_connections"`         // 5
	MaxOpenConnections    int           `json:"max_open_connections" mapstructure:"max_open_connections"`         // 5
	TablePrefix           string        `json:"table_prefix" mapstructure:"table_prefix"`                         // pre_users (pre)
}

// SQLConfig is the configuration for each SQL connection (mysql or postgresql)
type SQLConfig struct {
	CommonConfig              `json:",inline" mapstructure:",squash"` // Common configuration
	Driver                    string                                  `json:"driver" mapstructure:"driver"`                                             // mysql or postgresql
	ExistingConnection        *sql.DB                                 `json:"-" mapstructure:"-"`                                                       // Used for existing database connection
	Host                      string                                  `json:"host" mapstructure:"host"`                                                 // database host IE: localhost
	Name                      string                                  `json:"name" mapstructure:"name"`                                                 // database-name
	Password                  string                                  `json:"password" mapstructure:"password" encrypted:"true"`                        // user-password
	Port                      string                                  `json:"port" mapstructure:"port"`                                                 // 3306
	Replica                   bool                                    `json:"replica" mapstructure:"replica"`                                           // True if it's a replica (Read-Only)
	SkipInitializeWithVersion bool                                    `json:"skip_initialize_with_version" mapstructure:"skip_initialize_with_version"` // Skip using MySQL in test mode
	TimeZone                  string                                  `json:"time_zone" mapstructure:"time_zone"`                                       // timezone (IE: Asia/Shanghai)
	TxTimeout                 time.Duration                           `json:"tx_timeout" mapstructure:"tx_timeout"`                                     // 5*time.Second
	User                      string                                  `json:"user" mapstructure:"user"`                                                 // database username
	SslMode                   string                                  `json:"ssl_mode" mapstructure:"ssl_mode"`                                         // ssl mode (for PostgreSQL) [disable|allow|prefer|require|verify-ca|verify-full]
}

// SQLiteConfig is the configuration for each SQLite connection
type SQLiteConfig struct {
	CommonConfig       `json:",inline" mapstructure:",squash"` // Common configuration
	DatabasePath       string                                  `json:"database_path" mapstructure:"database_path"` // Location of a permanent database file (if NOT set, uses temporary memory)
	ExistingConnection gorm.ConnPool                           `json:"-" mapstructure:"-"`                         // Used for existing database connection
	Shared             bool                                    `json:"shared" mapstructure:"shared"`               // Adds a shared param to the connection string
}

// MongoDBConfig is the configuration for each MongoDB connection
type MongoDBConfig struct {
	CommonConfig       `json:",inline" mapstructure:",squash"` // Common configuration
	DatabaseName       string                                  `json:"database_name" mapstructure:"database_name"` // The database name
	ExistingConnection *mongo.Database                         `json:"-" mapstructure:"-"`                         // Used for existing database connection
	Transactions       bool                                    `json:"transactions" mapstructure:"transactions"`   // If it has transactions
	URI                string                                  `json:"uri" mapstructure:"uri"`                     // The connection string URI
}
