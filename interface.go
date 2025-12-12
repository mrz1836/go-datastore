package datastore

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"
)

// StorageService is the storage-related methods
type StorageService interface {
	AutoMigrateDatabase(ctx context.Context, models ...any) error
	CreateInBatches(ctx context.Context, models any, batchSize int) error
	CustomWhere(tx CustomWhereInterface, conditions map[string]any, engine Engine) any
	Execute(query string) *gorm.DB
	GetModel(ctx context.Context, model any, conditions map[string]any,
		timeout time.Duration, forceWriteDB bool) error
	GetModelPartial(ctx context.Context, model, fieldResult any, conditions map[string]any,
		timeout time.Duration, forceWriteDB bool) error
	GetModels(ctx context.Context, models any, conditions map[string]any, queryParams *QueryParams,
		fieldResults any, timeout time.Duration) error
	GetModelsPartial(ctx context.Context, models, fieldResults any, conditions map[string]any,
		timeout time.Duration) error
	GetModelCount(ctx context.Context, model any, conditions map[string]any,
		timeout time.Duration) (int64, error)
	GetModelsAggregate(ctx context.Context, models any, conditions map[string]any,
		aggregateColumn string, timeout time.Duration) (map[string]any, error)
	HasMigratedModel(modelType string) bool
	IncrementModel(ctx context.Context, model any,
		fieldName string, increment int64) (newValue int64, err error)
	IndexExists(tableName, indexName string) (bool, error)
	IndexMetadata(tableName, field string) error
	NewTx(ctx context.Context, fn func(*Transaction) error) error
	NewRawTx() (*Transaction, error)
	Raw(query string) *gorm.DB
	SaveModel(ctx context.Context, model any, tx *Transaction, newRecord, commitTx bool) error
}

// GetterInterface is the getter methods
type GetterInterface interface {
	GetArrayFields() []string
	GetDatabaseName() string
	GetMongoCollection(collectionName string) *mongo.Collection
	GetMongoCollectionByTableName(tableName string) *mongo.Collection
	GetMongoConditionProcessor() func(conditions *map[string]any)
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
