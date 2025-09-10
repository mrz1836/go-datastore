package datastore

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/mrz1836/go-datastore/nrgorm"
	"github.com/newrelic/go-agent/v3/newrelic"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/plugin/dbresolver"
)

// SaveModel will handle creating or updating a model based on its primary key, abstracting the database operations.
// It supports both SQL and MongoDB engines. For MongoDB, it uses a session context for transaction support if available.
// For SQL databases, it uses GORM to create or update the table schema.
//
// Parameters:
// - ctx: The context for the save operation, used for logging and tracing.
// - model: A pointer to the model to be saved.
// - tx: The transaction object to be used for the save operation.
// - newRecord: A boolean indicating whether the model is a new record (true) or an existing record (false).
// - commitTx: A boolean indicating whether to commit the transaction after saving the model.
//
// Returns:
// - An error if the save operation fails.
//
// The function performs the following steps:
// 1. Checks the database engine and handles MongoDB separately as it does not support transactions.
// 2. Sets the NewRelic transaction to the GORM database if using SQL.
// 3. Captures any panics during the save operation and rolls back the transaction if a panic occurs.
// 4. For new records, it creates the model in the database, omitting associations.
// 5. For existing records, it updates the model in the database, omitting associations.
// 6. Commits the transaction if commitTx is true.
// 7. Returns any errors encountered during the save operation.
func (c *Client) SaveModel(
	ctx context.Context,
	model interface{},
	tx *Transaction,
	newRecord, commitTx bool,
) error {
	// MongoDB (does not support transactions at this time)
	if c.Engine() == MongoDB {
		sessionContext := ctx //nolint:contextcheck // we need to overwrite the ctx for transaction support
		if tx.mongoTx != nil {
			// set the context to the session context -> mongo transaction
			sessionContext = *tx.mongoTx
		}
		return c.saveWithMongo(sessionContext, model, newRecord)
	} else if !IsSQLEngine(c.Engine()) {
		return ErrUnsupportedEngine
	}

	// Set the NewRelic txn
	c.options.db = nrgorm.SetTxnToGorm(newrelic.FromContext(ctx), c.options.db)

	// Capture any panics
	defer func() {
		if r := recover(); r != nil {
			c.DebugLog(context.Background(), fmt.Sprintf("panic recovered: %v", r))
			_ = tx.Rollback()
		}
	}()
	if err := tx.sqlTx.Error; err != nil {
		return err
	}

	// Create vs Update
	if newRecord {
		if err := tx.sqlTx.Omit(clause.Associations).Create(model).Error; err != nil {
			_ = tx.Rollback()
			// todo add duplicate key check for MySQL, Postgres and SQLite
			return err
		}
	} else {
		if err := tx.sqlTx.Omit(clause.Associations).Save(model).Error; err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	// Commit & check for errors
	if commitTx {
		if err := tx.Commit(); err != nil {
			return err
		}
	}

	// Return the tx
	return nil
}

// IncrementModel will increment the given field atomically in the database and return the new value.
// It supports both SQL and MongoDB engines. For MongoDB, it uses a session context for transaction support if available.
// For SQL databases, it uses GORM to perform the increment operation within a transaction.
//
// Parameters:
// - ctx: The context for the increment operation, used for logging and tracing.
// - model: A pointer to the model to be incremented.
// - fieldName: The name of the field to be incremented.
// - increment: The value by which to increment the field.
//
// Returns:
// - newValue: The new value of the incremented field.
// - err: An error if the increment operation fails.
//
// The function performs the following steps:
// 1. Checks the database engine and handles MongoDB separately as it does not support transactions.
// 2. Sets the NewRelic transaction to the GORM database if using SQL.
// 3. Creates a new transaction and locks the row for update to ensure atomicity.
// 4. Retrieves the current value of the field and increments it by the specified amount.
// 5. Updates the field with the new value in the database.
// 6. Returns the new value and any errors encountered during the increment operation.
func (c *Client) IncrementModel(
	ctx context.Context,
	model interface{},
	fieldName string,
	increment int64,
) (newValue int64, err error) {
	if c.Engine() == MongoDB {
		return c.incrementWithMongo(ctx, model, fieldName, increment)
	} else if !IsSQLEngine(c.Engine()) {
		return 0, ErrUnsupportedEngine
	}

	// Set the NewRelic txn
	c.options.db = nrgorm.SetTxnToGorm(newrelic.FromContext(ctx), c.options.db)

	// Create a new transaction
	if err = c.options.db.Transaction(func(tx *gorm.DB) error {
		// Get the id of the model
		id := GetModelStringAttribute(model, sqlIDFieldProper)
		if id == nil {
			return errors.New("model is missing an " + sqlIDFieldProper + " field")
		}

		// Get model if exist
		var result map[string]interface{}
		if err = tx.Model(&model).Clauses(clause.Locking{Strength: "UPDATE"}).Where(sqlIDField+" = ?", id).First(&result).Error; err != nil {
			return err
		}

		if result == nil {
			newValue = increment
			return nil
		}

		// Increment Counter
		newValue = convertToInt64(result[fieldName]) + increment
		return tx.Model(&model).Where(sqlIDField+" = ?", id).Update(fieldName, newValue).Error
	}); err != nil {
		return
	}

	return
}

// CreateInBatches creates all the models given in batches, supporting both SQL and MongoDB engines.
// For MongoDB, it uses a session context for transaction support if available. For SQL databases,
// it uses GORM to perform the batch creation.
//
// Parameters:
// - ctx: The context for the batch creation operation, used for logging and tracing.
// - models: A slice of models to be created in batches.
// - batchSize: The number of models to include in each batch.
//
// Returns:
// - An error if the batch creation operation fails.
//
// The function performs the following steps:
// 1. Checks the database engine and handles MongoDB separately as it does not support transactions.
// 2. For SQL databases, it uses GORM's CreateInBatches method to insert the models in batches.
// 3. Returns any errors encountered during the batch creation operation.
func (c *Client) CreateInBatches(
	ctx context.Context,
	models interface{},
	batchSize int,
) error {
	if c.Engine() == MongoDB {
		return c.CreateInBatchesMongo(ctx, models, batchSize)
	}

	tx := c.options.db.CreateInBatches(models, batchSize)
	return tx.Error
}

// convertToInt64 will convert an interface to an int64
func convertToInt64(i interface{}) int64 {
	switch v := i.(type) {
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case uint32:
		return int64(v)
	case uint64:
		return int64(v)
	}

	return i.(int64)
}

type gormWhere struct {
	tx *gorm.DB
}

// Where will help fire the tx.Where method
func (g *gormWhere) Where(query interface{}, args ...interface{}) {
	g.tx.Where(query, args...)
}

// getGormTx returns the GORM db tx
func (g *gormWhere) getGormTx() *gorm.DB {
	return g.tx
}

// GetModel will retrieve a single model from the datastore based on the provided conditions.
// It supports both SQL and MongoDB engines. For MongoDB, it uses a session context for transaction support if available.
// For SQL databases, it uses GORM to perform the query.
//
// Parameters:
// - ctx: The context for the retrieval operation, used for logging and tracing.
// - model: A pointer to the model to be retrieved.
// - conditions: A map of conditions to filter the query.
// - timeout: The duration to wait before timing out the query.
// - forceWriteDB: A boolean indicating whether to force the query to use the "write database" (only applicable for MySQL and PostgreSQL).
//
// Returns:
// - An error if the retrieval operation fails or if no results are found.
//
// The function performs the following steps:
// 1. Checks the database engine and handles MongoDB separately as it does not support transactions.
// 2. Sets the NewRelic transaction to the GORM database if using SQL.
// 3. Creates a new context and database transaction with the specified timeout.
// 4. Constructs the query based on the provided conditions and executes it.
// 5. If forceWriteDB is true, it uses the "write database" for the query (only for MySQL and PostgreSQL).
// 6. Returns any errors encountered during the retrieval operation or if no results are found.
func (c *Client) GetModel(
	ctx context.Context,
	model interface{},
	conditions map[string]interface{},
	timeout time.Duration,
	forceWriteDB bool,
) error {
	// Switch on the datastore engines
	if c.Engine() == MongoDB { // Get using Mongo
		return c.getWithMongo(ctx, model, conditions, nil, nil)
	} else if !IsSQLEngine(c.Engine()) {
		return ErrUnsupportedEngine
	}

	// Set the NewRelic txn
	c.options.db = nrgorm.SetTxnToGorm(newrelic.FromContext(ctx), c.options.db)

	// Create a new context and new db tx
	ctxDB, cancel := createCtx(ctx, c.options.db, timeout, c.IsDebug(), c.options.loggerDB)
	defer cancel()

	// Get the model data using a select
	// todo: optimize by specific fields
	var tx *gorm.DB
	if forceWriteDB { // Use the "write" database for this query (Only MySQL and Postgres)
		if c.Engine() == MySQL || c.Engine() == PostgreSQL {
			tx = ctxDB.Clauses(dbresolver.Write).Select("*")
		} else {
			tx = ctxDB.Select("*")
		}
	} else { // Use a replica if found
		tx = ctxDB.Select("*")
	}

	// Add conditions
	if len(conditions) > 0 {
		gtx := gormWhere{tx: tx}
		return checkResult(c.CustomWhere(&gtx, conditions, c.Engine()).(*gorm.DB).Find(model))
	}

	return checkResult(tx.Find(model))
}

// GetModels will return a slice of models based on the given conditions and query parameters.
// It supports both SQL and MongoDB engines. For MongoDB, it uses a session context for transaction support if available.
// For SQL databases, it uses GORM to perform the query.
//
// Parameters:
// - ctx: The context for the retrieval operation, used for logging and tracing.
// - models: A pointer to a slice of models to be retrieved.
// - conditions: A map of conditions to filter the query.
// - queryParams: A pointer to QueryParams struct containing pagination and sorting information.
// - fieldResults: A pointer to a slice where the results will be stored if not nil.
// - timeout: The duration to wait before timing out the query.
//
// Returns:
// - An error if the retrieval operation fails or if no results are found.
//
// The function performs the following steps:
// 1. Initializes default values for queryParams if not provided.
// 2. Checks the database engine and handles MongoDB separately as it does not support transactions.
// 3. Sets the NewRelic transaction to the GORM database if using SQL.
// 4. Creates a new context and database transaction with the specified timeout.
// 5. Constructs the query based on the provided conditions, pagination, and sorting information.
// 6. Executes the query and stores the results in the provided models or fieldResults slice.
// 7. Returns any errors encountered during the retrieval operation or if no results are found.
func (c *Client) GetModels(
	ctx context.Context,
	models interface{},
	conditions map[string]interface{},
	queryParams *QueryParams,
	fieldResults interface{},
	timeout time.Duration,
) error {
	if queryParams == nil {
		// init a new empty object for the default queryParams
		queryParams = &QueryParams{}
	}
	// Set default page size
	if queryParams.Page > 0 && queryParams.PageSize < 1 {
		queryParams.PageSize = defaultPageSize
	}

	// lower case the sort direction (asc / desc)
	queryParams.SortDirection = strings.ToLower(queryParams.SortDirection)

	// Switch on the datastore engines
	if c.Engine() == MongoDB { // Get using Mongo
		return c.getWithMongo(ctx, models, conditions, fieldResults, queryParams)
	} else if !IsSQLEngine(c.Engine()) {
		return ErrUnsupportedEngine
	}
	return c.find(ctx, models, conditions, queryParams, fieldResults, timeout)
}

// GetModelCount will return a count of the models matching the provided conditions.
// It supports both SQL and MongoDB engines. For MongoDB, it uses a session context for transaction support if available.
// For SQL databases, it uses GORM to perform the count operation.
//
// Parameters:
// - ctx: The context for the count operation, used for logging and tracing.
// - model: A pointer to the model type for which the count is to be retrieved.
// - conditions: A map of conditions to filter the count query.
// - timeout: The duration to wait before timing out the query.
//
// Returns:
// - count: The number of models matching the provided conditions.
// - err: An error if the count operation fails.
//
// The function performs the following steps:
// 1. Checks the database engine and handles MongoDB separately as it does not support transactions.
// 2. Sets the NewRelic transaction to the GORM database if using SQL.
// 3. Creates a new context and database transaction with the specified timeout.
// 4. Constructs the count query based on the provided conditions and executes it.
// 5. Returns the count of models and any errors encountered during the count operation.
func (c *Client) GetModelCount(
	ctx context.Context,
	model interface{},
	conditions map[string]interface{},
	timeout time.Duration,
) (int64, error) {
	// Switch on the datastore engines
	if c.Engine() == MongoDB {
		return c.countWithMongo(ctx, model, conditions)
	} else if !IsSQLEngine(c.Engine()) {
		return 0, ErrUnsupportedEngine
	}

	return c.count(ctx, model, conditions, timeout)
}

// GetModelsAggregate will return an aggregate count of the model matching conditions.
// It supports both SQL and MongoDB engines. For MongoDB, it uses a session context for transaction support if available.
// For SQL databases, it uses GORM to perform the aggregate operation.
//
// Parameters:
// - ctx: The context for the aggregate operation, used for logging and tracing.
// - models: A pointer to a slice of models to be aggregated.
// - conditions: A map of conditions to filter the aggregate query.
// - aggregateColumn: The name of the column to aggregate.
// - timeout: The duration to wait before timing out the query.
//
// Returns:
// - result: A map where the keys are the aggregated column values and the values are the counts of models matching the conditions.
// - err: An error if the aggregate operation fails.
//
// The function performs the following steps:
// 1. Checks the database engine and handles MongoDB separately as it does not support transactions.
// 2. Sets the NewRelic transaction to the GORM database if using SQL.
// 3. Creates a new context and database transaction with the specified timeout.
// 4. Constructs the aggregate query based on the provided conditions and executes it.
// 5. For date fields, formats the date according to the database engine.
// 6. Returns the aggregate result and any errors encountered during the aggregate operation.
func (c *Client) GetModelsAggregate(ctx context.Context, models interface{},
	conditions map[string]interface{}, aggregateColumn string, timeout time.Duration,
) (map[string]interface{}, error) {
	// Switch on the datastore engines
	if c.Engine() == MongoDB {
		return c.aggregateWithMongo(ctx, models, conditions, aggregateColumn, timeout)
	} else if !IsSQLEngine(c.Engine()) {
		return nil, ErrUnsupportedEngine
	}

	return c.aggregate(ctx, models, conditions, aggregateColumn, timeout)
}

// find will get records and return
func (c *Client) find(ctx context.Context, result interface{}, conditions map[string]interface{},
	queryParams *QueryParams, fieldResults interface{}, timeout time.Duration,
) error {
	// Find the type
	if reflect.TypeOf(result).Elem().Kind() != reflect.Slice {
		return errors.New("field: result is not a slice, found: " + reflect.TypeOf(result).Kind().String())
	}

	// Set the NewRelic txn
	c.options.db = nrgorm.SetTxnToGorm(newrelic.FromContext(ctx), c.options.db)

	// Create a new context and new db tx
	ctxDB, cancel := createCtx(ctx, c.options.db, timeout, c.IsDebug(), c.options.loggerDB)
	defer cancel()

	tx := ctxDB.Model(result)

	// Create the offset
	offset := (queryParams.Page - 1) * queryParams.PageSize

	// Use the limit and offset
	if queryParams.Page > 0 && queryParams.PageSize > 0 {
		tx = tx.Limit(queryParams.PageSize).Offset(offset)
	}

	// Use an order field/sort
	if len(queryParams.OrderByField) > 0 {
		tx = tx.Order(clause.OrderByColumn{
			Column: clause.Column{
				Name: queryParams.OrderByField,
			},
			Desc: strings.ToLower(queryParams.SortDirection) == SortDesc,
		})
	}

	// Check for errors or no records found
	if len(conditions) > 0 {
		gtx := gormWhere{tx: tx}
		if fieldResults != nil {
			return checkResult(c.CustomWhere(&gtx, conditions, c.Engine()).(*gorm.DB).Find(fieldResults))
		}
		return checkResult(c.CustomWhere(&gtx, conditions, c.Engine()).(*gorm.DB).Find(result))
	}

	// Skip the conditions
	if fieldResults != nil {
		return checkResult(tx.Find(fieldResults))
	}
	return checkResult(tx.Find(result))
}

// find will get records and return
func (c *Client) count(ctx context.Context, model interface{}, conditions map[string]interface{},
	timeout time.Duration,
) (int64, error) {
	// Set the NewRelic txn
	c.options.db = nrgorm.SetTxnToGorm(newrelic.FromContext(ctx), c.options.db)

	// Create a new context and new db tx
	ctxDB, cancel := createCtx(ctx, c.options.db, timeout, c.IsDebug(), c.options.loggerDB)
	defer cancel()

	tx := ctxDB.Model(model)

	// Check for errors or no records found
	if len(conditions) > 0 {
		gtx := gormWhere{tx: tx}
		var count int64
		err := checkResult(c.CustomWhere(&gtx, conditions, c.Engine()).(*gorm.DB).Model(model).Count(&count))
		return count, err
	}
	var count int64
	err := checkResult(tx.Count(&count))

	return count, err
}

// find will get records and return
func (c *Client) aggregate(ctx context.Context, model interface{}, conditions map[string]interface{},
	aggregateColumn string, timeout time.Duration,
) (map[string]interface{}, error) {
	// Find the type
	if reflect.TypeOf(model).Elem().Kind() != reflect.Slice {
		return nil, errors.New("field: result is not a slice, found: " + reflect.TypeOf(model).Kind().String())
	}

	// Set the NewRelic txn
	c.options.db = nrgorm.SetTxnToGorm(newrelic.FromContext(ctx), c.options.db)

	// Create a new context and new db tx
	ctxDB, cancel := createCtx(ctx, c.options.db, timeout, c.IsDebug(), c.options.loggerDB)
	defer cancel()

	// Get the tx
	tx := ctxDB.Model(model)

	// Check for errors or no records found
	var aggregate []map[string]interface{}
	if len(conditions) > 0 {
		gtx := gormWhere{tx: tx}
		err := checkResult(c.CustomWhere(&gtx, conditions, c.Engine()).(*gorm.DB).Model(model).Group(aggregateColumn).Scan(&aggregate))
		if err != nil {
			return nil, err
		}
	} else {
		aggregateCol := aggregateColumn

		// Check for a known date field
		if StringInSlice(aggregateCol, DateFields) {
			if c.Engine() == MySQL {
				aggregateCol = "DATE_FORMAT(" + aggregateCol + ", '%Y%m%d')"
			} else if c.Engine() == Postgres {
				aggregateCol = "to_char(" + aggregateCol + ", 'YYYYMMDD')"
			} else {
				aggregateCol = "strftime('%Y%m%d', " + aggregateCol + ")"
			}
		}
		err := checkResult(tx.Select(aggregateCol + " as _id, COUNT(id) AS count").Group(aggregateCol).Scan(&aggregate))
		if err != nil {
			return nil, err
		}
	}

	// Create the result
	aggregateResult := make(map[string]interface{})
	for _, item := range aggregate {
		key := item[mongoIDField].(string)
		aggregateResult[key] = item[accumulationCountField]
	}

	return aggregateResult, nil
}

// Execute a SQL query
func (c *Client) Execute(query string) *gorm.DB {
	if IsSQLEngine(c.Engine()) {
		return c.options.db.Exec(query)
	}

	return nil
}

// Raw a raw SQL query
func (c *Client) Raw(query string) *gorm.DB {
	if IsSQLEngine(c.Engine()) {
		return c.options.db.Raw(query)
	}

	return nil
}

// checkResult will check for records or error
func checkResult(result *gorm.DB) error {
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return ErrNoResults
		}
		return result.Error
	}

	// We should actually have some rows according to GORM
	if result.RowsAffected == 0 {
		return ErrNoResults
	}
	return nil
}

// createCtx will make a new DB context
func createCtx(ctx context.Context, db *gorm.DB, timeout time.Duration, debug bool,
	optionalLogger logger.Interface,
) (*gorm.DB, context.CancelFunc) {
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, timeout)
	return db.Session(getGormSessionConfig(db.PrepareStmt, debug, optionalLogger)).WithContext(ctx), cancel
}
