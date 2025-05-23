package datastore

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"
)

// StorageService is the storage-related methods
type StorageService interface {
	AutoMigrateDatabase(ctx context.Context, models ...interface{}) error
	CreateInBatches(ctx context.Context, models interface{}, batchSize int) error
	CustomWhere(tx CustomWhereInterface, conditions map[string]interface{}, engine Engine) interface{}
	Execute(query string) *gorm.DB
	GetModel(ctx context.Context, model interface{}, conditions map[string]interface{},
		timeout time.Duration, forceWriteDB bool) error
	GetModels(ctx context.Context, models interface{}, conditions map[string]interface{}, queryParams *QueryParams,
		fieldResults interface{}, timeout time.Duration) error
	GetModelCount(ctx context.Context, model interface{}, conditions map[string]interface{},
		timeout time.Duration) (int64, error)
	GetModelsAggregate(ctx context.Context, models interface{}, conditions map[string]interface{},
		aggregateColumn string, timeout time.Duration) (map[string]interface{}, error)
	HasMigratedModel(modelType string) bool
	IncrementModel(ctx context.Context, model interface{},
		fieldName string, increment int64) (newValue int64, err error)
	IndexExists(tableName, indexName string) (bool, error)
	IndexMetadata(tableName, field string) error
	NewTx(ctx context.Context, fn func(*Transaction) error) error
	NewRawTx() (*Transaction, error)
	Raw(query string) *gorm.DB
	SaveModel(ctx context.Context, model interface{}, tx *Transaction, newRecord, commitTx bool) error
}

// GetterInterface is the getter methods
type GetterInterface interface {
	GetArrayFields() []string
	GetDatabaseName() string
	GetMongoCollection(collectionName string) *mongo.Collection
	GetMongoCollectionByTableName(tableName string) *mongo.Collection
	GetMongoConditionProcessor() func(conditions *map[string]interface{})
	GetMongoIndexer() func() map[string][]mongo.IndexModel
	GetObjectFields() []string
	GetTableName(modelName string) string
}

// ClientInterface is the Datastore client interface
type ClientInterface interface {
	GetterInterface
	StorageService
	Close(ctx context.Context) error
	Debug(on bool)
	DebugLog(ctx context.Context, text string)
	Engine() Engine
	IsAutoMigrate() bool
	IsDebug() bool
	IsNewRelicEnabled() bool
}
