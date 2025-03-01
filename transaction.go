package datastore

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"
)

// NewTx will start a new datastore transaction based on the configured database options.
// It supports both GORM-based SQL databases and MongoDB, handling the transaction lifecycle accordingly.
//
// Parameters:
// - ctx: The context for the transaction, used for managing request-scoped values, cancelation signals, and deadlines.
// - fn: A function that takes a pointer to a Transaction and returns an error. This function contains the operations to be performed within the transaction.
//
// Returns:
// - error: An error if the transaction initialization or the provided function fails.
//
// The function performs the following steps:
// 1. Checks if a GORM database is configured. If so, it starts a new GORM session and begins a transaction.
// 2. If MongoDB transactions are enabled, it starts a new MongoDB session and transaction.
// 3. If no database is configured, it executes the provided function with an empty transaction.
// 4. The provided function is executed within the context of the started transaction.
func (c *Client) NewTx(ctx context.Context, fn func(*Transaction) error) error {

	// All GORM databases
	if c.options.db != nil {
		sessionDb := c.options.db.Session(getGormSessionConfig(c.options.db.PrepareStmt, c.IsDebug(), c.options.loggerDB))
		return fn(&Transaction{
			sqlTx: sessionDb.Begin(),
		})
	}

	// For MongoDB
	if c.options.mongoDBConfig.Transactions {
		return c.options.mongoDB.Client().UseSession(ctx, func(sessionContext mongo.SessionContext) error {
			if err := sessionContext.StartTransaction(); err != nil {
				return err
			}
			return fn(&Transaction{
				sqlTx:   nil,
				mongoTx: &sessionContext,
			})
		})
	}

	// Empty transaction
	return fn(&Transaction{})
}

// NewRawTx will start a new datastore transaction based on the configured database options.
// It supports both GORM-based SQL databases and MongoDB, handling the transaction lifecycle accordingly.
//
// Returns:
// - Transaction: A pointer to the started Transaction struct.
// - error: An error if the transaction initialization fails.
//
// The function performs the following steps:
// 1. Checks if a GORM database is configured. If so, it starts a new GORM session and begins a transaction.
// 2. If MongoDB transactions are enabled, it returns an error as MongoDB transactions require a callback function.
// 3. If no database is configured, it returns an empty Transaction struct.
func (c *Client) NewRawTx() (*Transaction, error) {

	// All GORM databases
	if c.options.db != nil {
		sessionDb := c.options.db.Session(getGormSessionConfig(c.options.db.PrepareStmt, c.IsDebug(), c.options.loggerDB))
		return &Transaction{
			sqlTx: sessionDb.Begin(),
		}, nil
	}

	// For MongoDB
	// todo: implement - but the issue is Mongo uses a callback
	if c.options.mongoDBConfig.Transactions {
		return nil, ErrNotImplemented
	}

	// Empty transaction
	return &Transaction{}, nil
}

// Transaction is the internal datastore transaction
type Transaction struct {
	committed    bool
	mongoTx      *mongo.SessionContext
	rowsAffected int64
	sqlTx        *gorm.DB
}

// CanCommit will return true if it can commit
func (tx *Transaction) CanCommit() bool {
	return !tx.committed && (tx.sqlTx != nil || tx.mongoTx != nil)
}

// Rollback the transaction
func (tx *Transaction) Rollback() error {
	if tx.sqlTx != nil {
		tx.sqlTx.Rollback()
	}

	if tx.mongoTx != nil {
		return (*tx.mongoTx).AbortTransaction(*tx.mongoTx)
	}

	return nil
}

// Commit will commit the transaction
func (tx *Transaction) Commit() error {

	// Have we already committed?
	if tx.committed {
		return nil
	} else if tx.sqlTx == nil &&
		tx.mongoTx == nil {
		return nil
	}

	// Finally commit
	if tx.sqlTx != nil {
		result := tx.sqlTx.Commit()
		if result.Error != nil {
			_ = result.Rollback()
			return result.Error
		}
		tx.committed = true
		tx.rowsAffected = result.RowsAffected
	}

	if tx.mongoTx != nil {
		if err := (*tx.mongoTx).CommitTransaction(*tx.mongoTx); err != nil {
			return err
		}
		tx.committed = true
		tx.rowsAffected = 1 // todo: can we get all rows affected ?
	}

	return nil
}
