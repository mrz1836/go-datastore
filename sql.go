package datastore

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mrz1836/go-datastore/nrgorm"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"gorm.io/plugin/dbresolver"
)

/*
// Load the NewRelic capable drivers
// _ "github.com/newrelic/go-agent/v3/integrations/nrmysql"
// _ "github.com/newrelic/go-agent/v3/integrations/nrpgx"
// _ "github.com/newrelic/go-agent/v3/integrations/nrsqlite3"
*/

// SQL related default settings
// todo: make this configurable for the end-user?
const (
	defaultDatetimePrecision            = true            // disable datetime precision, which not supported before MySQL 5.6
	defaultDontSupportRenameColumn      = true            // `change` when rename column, rename column not supported before MySQL 8, MariaDB
	defaultDontSupportRenameIndex       = true            // drop and create when rename index, rename index not supported before MySQL 5.7, MariaDB
	defaultFieldStringSize         uint = 256             // default size for string fields
	dsnDefault                          = "file::memory:" // DSN for connection (file or memory, default is memory)
	defaultPreparedStatements           = false           // Flag for prepared statements for SQL
)

// openSQLDatabase will open a new SQL database connection using the provided configurations.
// It supports MySQL and PostgreSQL drivers and sets up a connection pool with optional read replicas.
// The function also registers NewRelic callbacks for monitoring and performance tracking.
//
// Parameters:
// - optionalLogger: An optional logger interface for GORM logging.
// - configs: A variadic parameter of SQLConfig pointers, where the first config is the source and the rest are optional replicas.
//
// Returns:
// - db: A pointer to the opened gorm.DB instance.
// - err: An error if the database connection fails.
//
// The function performs the following steps:
// 1. Retrieves the source database configuration from the provided configs.
// 2. Validates the driver type (MySQL or PostgreSQL) and creates the corresponding GORM dialector.
// 3. Opens a new GORM database connection using the source configuration.
// 4. Configures the dbresolver for read replicas if additional configs are provided.
// 5. Sets connection pool parameters such as max idle connections, max open connections, and connection lifetimes.
// 6. Registers NewRelic callbacks for monitoring.
// 7. Returns the opened database connection or an error if the process fails.
func openSQLDatabase(optionalLogger glogger.Interface, configs ...*SQLConfig) (db *gorm.DB, err error) {

	// Try to find a source
	var sourceConfig *SQLConfig
	if sourceConfig, configs = getSourceDatabase(configs); sourceConfig == nil {
		return nil, ErrNoSourceFound
	}

	// Not a valid driver?
	if sourceConfig.Driver != MySQL.String() && sourceConfig.Driver != PostgreSQL.String() {
		return nil, ErrUnsupportedDriver
	}

	// Switch on a driver
	sourceDialector := getDialector(sourceConfig)

	// Create a new source connection
	// todo: make this configurable? (PrepareStmt)
	if db, err = gorm.Open(
		sourceDialector, getGormConfig(
			sourceConfig.TablePrefix, defaultPreparedStatements,
			sourceConfig.Debug, optionalLogger,
		),
	); err != nil {
		return
	}

	// Start the resolver (default is a source, and replica is the same)
	resolverConfig := dbresolver.Config{
		Policy:   dbresolver.RandomPolicy{},
		Replicas: []gorm.Dialector{sourceDialector},
		Sources:  []gorm.Dialector{sourceDialector},
	}

	// Do we have additional?
	if len(configs) > 0 {

		// Clear the existing replica
		resolverConfig.Replicas = nil

		// Loop configs
		for _, config := range configs {

			// Get the dialector
			dialector := getDialector(config)

			// Set based on replica
			if config.Replica {
				resolverConfig.Replicas = append(resolverConfig.Replicas, dialector)
			} else {
				resolverConfig.Sources = append(resolverConfig.Sources, dialector)
			}
		}

		// No replica?
		if len(resolverConfig.Replicas) == 0 {
			resolverConfig.Replicas = append(resolverConfig.Replicas, sourceDialector)
		}
	}

	// Create the register and set the configuration
	//
	// See: https://gorm.io/docs/dbresolver.html
	// var register *dbresolver.DBResolver
	register := new(dbresolver.DBResolver)
	register.Register(resolverConfig)
	if sourceConfig.MaxConnectionIdleTime.String() != emptyTimeDuration {
		register = register.SetConnMaxIdleTime(sourceConfig.MaxConnectionIdleTime)
	}
	if sourceConfig.MaxConnectionTime.String() != emptyTimeDuration {
		register = register.SetConnMaxLifetime(sourceConfig.MaxConnectionTime)
	}
	if sourceConfig.MaxOpenConnections > 0 {
		register = register.SetMaxOpenConns(sourceConfig.MaxOpenConnections)
	}
	if sourceConfig.MaxIdleConnections > 0 {
		register = register.SetMaxIdleConns(sourceConfig.MaxIdleConnections)
	}

	// Use the register
	if err = db.Use(register); err != nil {
		return
	}

	// Register the callbacks with NewRelic
	nrgorm.AddGormCallbacks(db)

	// Return the connection
	return
}

// openSQLiteDatabase will open an SQLite database connection using the provided configuration.
// It supports both file-based and in-memory databases and can use an existing connection if provided.
// The function also registers NewRelic callbacks for monitoring and performance tracking.
//
// Parameters:
// - optionalLogger: An optional logger interface for GORM logging.
// - config: A pointer to the SQLiteConfig struct containing the database configuration.
//
// Returns:
// - db: A pointer to the opened gorm.DB instance.
// - err: An error if the database connection fails.
//
// The function performs the following steps:
// 1. Checks if an existing connection is provided in the configuration.
// 2. If an existing connection is provided, uses it to create the GORM dialector.
// 3. If no existing connection is provided, constructs the DSN for a file-based or in-memory database.
// 4. Opens a new GORM database connection using the constructed dialector and configuration.
// 5. Registers NewRelic callbacks for monitoring.
// 6. Returns the opened database connection or an error if the process fails.
func openSQLiteDatabase(optionalLogger glogger.Interface, config *SQLiteConfig) (db *gorm.DB, err error) {

	// Check for an existing connection
	var dialector gorm.Dialector
	if config.ExistingConnection != nil {
		dialector = sqlite.Dialector{Conn: config.ExistingConnection}
	} else {
		dialector = sqlite.Open(getDNS(config.DatabasePath, config.Shared))
	}

	/*
		// todo: implement this functionality (name spaced in-memory tables)
		NOTE: https://www.sqlite.org/inmemorydb.html
		If two or more distinct but shareable in-memory databases are needed in a single process, then the mode=memory
		query parameter can be used with a URI filename to create a named in-memory database:
		rc = sqlite3_open("file:memdb1?mode=memory&cache=shared", &db);
	*/

	// Create a new connection
	if db, err = gorm.Open(
		dialector, getGormConfig(
			config.TablePrefix, defaultPreparedStatements,
			config.Debug, optionalLogger,
		),
	); err != nil {
		return
	}

	// @mrz: turned off, unsure if it's really necessary or not
	// Get the SQL DB
	// var sqlDB *sql.DB
	// sqlDB, err = db.DB()
	// sqlDB.SetMaxIdleConns(config.MaxIdleConnections)
	// sqlDB.SetMaxOpenConns(config.MaxOpenConnections)
	// sqlDB.SetConnMaxLifetime(config.MaxConnectionTime)
	// sqlDB.SetConnMaxIdleTime(config.MaxConnectionIdleTime)

	// Register the callbacks with NewRelic
	nrgorm.AddGormCallbacks(db)

	// Return the connection
	return
}

// getDNS will return the Data Source Name (DSN) string for an SQLite database connection.
// It supports both file-based and in-memory databases with an optional shared cache mode.
//
// Parameters:
// - databasePath: The path to the SQLite database file. If empty, an in-memory database is used.
// - shared: A boolean flag indicating whether to use a shared cache mode for the SQLite database.
//
// Returns:
// - dsn: The constructed DSN string for the SQLite database connection.
//
// The function performs the following steps:
// 1. Checks if a file-based path is provided. If so, use it as the DSN.
// 2. If no file-based path is provided, defaults to an in-memory database DSN.
// 3. Appends the shared cache mode parameter to the DSN if the shared flag is true.
func getDNS(databasePath string, shared bool) (dsn string) {

	// Use a file-based path?
	if len(databasePath) > 0 {
		dsn = databasePath
	} else { // Default is in-memory
		dsn = dsnDefault
	}

	// Shared?
	if shared {
		dsn += "?cache=shared"
	}
	return
}

// getDialector will return a new gorm.Dialector based on a driver
func getDialector(config *SQLConfig) gorm.Dialector {
	if config.Driver == MySQL.String() {
		return mySQLDialector(config)
	}
	return postgreSQLDialector(config)
}

// mySQLDialector will return a gorm.Dialector
func mySQLDialector(config *SQLConfig) gorm.Dialector {

	// Create the default MySQL configuration
	cfg := mysql.Config{
		// DriverName: "nrmysql",
		// todo: make all params customizable via config
		DSN: config.User + ":" + config.Password +
			"@tcp(" + config.Host + ":" + config.Port + ")/" +
			config.Name + "?charset=utf8&parseTime=True&loc=Local", // data source name (connection string)
		DefaultStringSize:         defaultFieldStringSize,           // default size for string fields
		DisableDatetimePrecision:  defaultDatetimePrecision,         // disable datetime precision, which not supported before MySQL 5.6
		DontSupportRenameIndex:    defaultDontSupportRenameIndex,    // drop and create when rename index, rename index not supported before MySQL 5.7, MariaDB
		DontSupportRenameColumn:   defaultDontSupportRenameColumn,   // `change` when rename column, rename column not supported before MySQL 8, MariaDB
		SkipInitializeWithVersion: config.SkipInitializeWithVersion, // autoconfigure based on a currently MySQL version
	}

	// Do we have an existing connection?
	if config.ExistingConnection != nil {
		cfg.DSN = ""
		cfg.Conn = config.ExistingConnection
	}

	return mysql.New(cfg)
}

// postgreSQLDialector will return a gorm.Dialector
func postgreSQLDialector(config *SQLConfig) gorm.Dialector {

	// Create the default PostgreSQL configuration
	cfg := postgres.Config{
		// DriverName: "nrpgx",
		PreferSimpleProtocol: true, // turn to TRUE to disable implicit prepared statement usage
		WithoutReturning:     false,
	}

	// Do we have an existing connection?
	if config.ExistingConnection != nil {
		cfg.Conn = config.ExistingConnection
	} else {
		cfg.DSN = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
			config.Host, config.User, config.Password, config.Name, config.Port, config.SslMode, config.TimeZone)
	}

	return postgres.New(cfg)
}

// getSourceDatabase will loop all configs and get the first source
//
// todo: this will grab ANY source (create a better way to seed the source database)
func getSourceDatabase(configs []*SQLConfig) (*SQLConfig, []*SQLConfig) {

	for index, config := range configs {
		if !config.Replica {
			if len(configs) > 1 {
				var processed []*SQLConfig
				for i, c := range configs {
					if i != index {
						processed = append(processed, c)
					}
				}
				return configs[index], processed
			}
			return configs[index], nil
		}
	}
	return nil, configs
}

// getGormSessionConfig returns the gorm session config
func getGormSessionConfig(preparedStatement, debug bool, optionalLogger glogger.Interface) *gorm.Session {

	config := &gorm.Session{
		AllowGlobalUpdate:        false,
		CreateBatchSize:          0,
		DisableNestedTransaction: false,
		DryRun:                   false,
		FullSaveAssociations:     false,
		Logger:                   optionalLogger,
		NewDB:                    false,
		NowFunc:                  nil,
		PrepareStmt:              preparedStatement,
		QueryFields:              false,
		SkipDefaultTransaction:   false,
		SkipHooks:                true,
	}

	// Optional logger vs basic
	if optionalLogger == nil {
		logLevel := glogger.Silent
		if debug {
			logLevel = glogger.Info
		}

		config.Logger = glogger.New(
			log.New(os.Stdout, "\r\n ", log.LstdFlags), // io writer
			glogger.Config{
				SlowThreshold:             5 * time.Second, // Slow SQL threshold
				LogLevel:                  logLevel,        // Log level
				IgnoreRecordNotFoundError: true,            // Ignore ErrRecordNotFound error for logger
				Colorful:                  false,           // Disable color
			},
		)
	}

	return config
}

// getGormConfig will return a valid gorm.Config
//
// See: https://gorm.io/docs/gorm_config.html
func getGormConfig(tablePrefix string, preparedStatement, debug bool, optionalLogger glogger.Interface) *gorm.Config {

	// Set the prefix
	if len(tablePrefix) > 0 {
		tablePrefix = tablePrefix + "_"
	}

	// Create the configuration
	config := &gorm.Config{
		AllowGlobalUpdate:                        false,
		ClauseBuilders:                           nil,
		ConnPool:                                 nil,
		CreateBatchSize:                          0,
		Dialector:                                nil,
		DisableAutomaticPing:                     false,
		DisableForeignKeyConstraintWhenMigrating: true,
		DisableNestedTransaction:                 false,
		DryRun:                                   false, // toggle for extreme debugging
		FullSaveAssociations:                     false,
		Logger:                                   optionalLogger,
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   tablePrefix, // table name prefix, table for `User` would be `t_users`
			SingularTable: false,       // use singular table name, table for `User` would be `user` with this option enabled
		},
		NowFunc:                nil,
		Plugins:                nil,
		PrepareStmt:            preparedStatement, // default is: false
		QueryFields:            false,
		SkipDefaultTransaction: false,
	}

	// Optional logger vs basic
	if optionalLogger == nil {
		logLevel := glogger.Silent
		if debug {
			logLevel = glogger.Info
		}

		config.Logger = glogger.New(
			log.New(os.Stdout, "\r\n ", log.LstdFlags), // io writer
			glogger.Config{
				SlowThreshold:             5 * time.Second, // Slow SQL threshold
				LogLevel:                  logLevel,        // Log level
				IgnoreRecordNotFoundError: true,            // Ignore ErrRecordNotFound error for logger
				Colorful:                  false,           // Disable color
			},
		)
	}

	return config
}

// closeSQLDatabase will close an SQL connection safely
func closeSQLDatabase(gormDB *gorm.DB) error {
	if gormDB == nil {
		return nil
	}
	sqlDB, err := gormDB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// sqlDefaults will set the default values if missing
func (s *SQLConfig) sqlDefaults(engine Engine) *SQLConfig {

	// Set the default(s)
	if s.TxTimeout.String() == emptyTimeDuration {
		s.TxTimeout = defaultDatabaseTxTimeout
	}
	if s.MaxConnectionTime.String() == emptyTimeDuration {
		s.MaxConnectionTime = defaultDatabaseMaxTimeout
	}
	if s.MaxConnectionIdleTime.String() == emptyTimeDuration {
		s.MaxConnectionIdleTime = defaultDatabaseMaxIdleTime
	}
	if len(s.Port) == 0 {
		if engine == MySQL {
			s.Port = defaultMySQLPort
		} else {
			s.Port = defaultPostgreSQLPort
		}
	}
	if len(s.Host) == 0 {
		if engine == MySQL {
			s.Host = defaultMySQLHost
		} else {
			s.Host = defaultPostgreSQLHost
		}
	}
	if len(s.TimeZone) == 0 {
		s.TimeZone = defaultTimeZone
	}
	if len(s.SslMode) == 0 {
		s.SslMode = defaultPostgreSQLSslMode
	}
	return s
}
