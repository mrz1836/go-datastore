# CLAUDE.md - go-datastore

## Overview

`go-datastore` is a unified data layer abstraction library that provides a common interface for multiple database backends using [GORM](https://gorm.io/). It supports **MySQL**, **PostgreSQL**, **SQLite**, and **MongoDB** with consistent APIs, transaction handling, and query building.

**Key Features:**
- Multi-database abstraction with unified interface
- Transaction support across all engines (including MongoDB sessions)
- Advanced query building with custom conditions
- Auto-migration and indexing capabilities
- NewRelic monitoring integration
- Custom field types (NullString, NullTime)
- Comprehensive testing with fuzz tests

## Architecture

### Core Interfaces

- **`ClientInterface`** - Main client combining storage and getter functionality
- **`StorageService`** - Database operations (CRUD, migrations, transactions)
- **`GetterInterface`** - Configuration and metadata access
- **`CustomWhereInterface`** - Query building interface

### Database Engines

```go
const (
    Empty      Engine = "empty"      // Default/unset
    MongoDB    Engine = "mongodb"    // MongoDB with mongo-driver
    MySQL      Engine = "mysql"      // MySQL with GORM
    PostgreSQL Engine = "postgresql" // PostgreSQL with GORM
    SQLite     Engine = "sqlite"     // SQLite with GORM
)
```

## Essential Commands

```bash
# Install dependencies and tools
magex update:install

# Run tests (fast)
magex test

# Run tests with race detection (slower)
magex test:race

# Run benchmarks
magex bench

# Format code
magex format:fix

# Lint code
magex lint

# Vet code
magex vet

# Tidy modules
magex tidy

# Update dependencies
magex deps:update
```

## Critical Files

| File | Purpose |
|------|---------|
| `interface.go` | Core interface definitions |
| `client.go` | Main client implementation and configuration |
| `client_options.go` | Client configuration options and functional options |
| `models.go` | Model CRUD operations and business logic |
| `transaction.go` | Transaction management for SQL and MongoDB |
| `where.go` | Query condition building and processing |
| `sql.go` | SQL database connection management |
| `mongodb.go` | MongoDB connection and operations |
| `definitions.go` | Constants, configurations, and field definitions |
| `engine.go` | Database engine types and utilities |
| `errors.go` | Custom error definitions |
| `custom_types/` | Custom field types (NullString, NullTime) |

## Client Initialization Patterns

### Basic SQLite (Default)
```go
client, err := datastore.NewClient(ctx)
```

### Explicit SQLite Configuration
```go
client, err := datastore.NewClient(ctx,
    datastore.WithSQLite(&datastore.SQLiteConfig{
        DatabasePath: "my_app.db",
        Shared: true,
    }),
)
```

### MySQL/PostgreSQL
```go
client, err := datastore.NewClient(ctx,
    datastore.WithSQL(datastore.MySQL, []*datastore.SQLConfig{
        {
            Host:     "localhost",
            Port:     "3306",
            Name:     "mydb",
            User:     "user",
            Password: "pass",
        },
    }),
)
```

### MongoDB
```go
client, err := datastore.NewClient(ctx,
    datastore.WithMongo(&datastore.MongoDBConfig{
        URI:          "mongodb://localhost:27017",
        DatabaseName: "mydb",
        Transactions: true,
    }),
)
```

### With Auto-Migration
```go
client, err := datastore.NewClient(ctx,
    datastore.WithAutoMigrate(&User{}, &Post{}),
    datastore.WithDebugging(),
)
```

## Common Operations

### Save Model
```go
err := client.SaveModel(ctx, &model, tx, newRecord, commitTx)
```

### Get Single Model
```go
err := client.GetModel(ctx, &model, conditions, timeout, forceWriteDB)
```

### Get Multiple Models
```go
err := client.GetModels(ctx, &models, conditions, queryParams, fieldResults, timeout)
```

### Transactions
```go
err := client.NewTx(ctx, func(tx *datastore.Transaction) error {
    // Your transactional operations
    return client.SaveModel(ctx, &model, tx, true, false)
})
```

## Query Conditions

The library uses MongoDB-style condition operators:

```go
conditions := map[string]interface{}{
    "name": "John",
    "age": map[string]interface{}{
        "$gte": 18,
        "$lt": 65,
    },
    "$and": []map[string]interface{}{
        {"status": "active"},
        {"verified": true},
    },
}
```

**Supported Operators:**
- `$and`, `$or` - Logical operators
- `$gt`, `$gte`, `$lt`, `$lte` - Comparison operators
- `$ne` - Not equals
- `$in`, `$nin` - In/not in arrays
- `$exists` - Field existence checks

## Model Conventions

### Required Fields
- **`id`** - Primary key (uses `_id` internally for MongoDB)
- **`metadata`** - Optional JSON field for key-value storage

### Date Fields
- `created_at` - Record creation timestamp
- `updated_at` - Record update timestamp
- `modified_at` - Alternative update timestamp

### Custom Field Types
```go
import "github.com/mrz1836/go-datastore/custom_types"

type User struct {
    ID       uint64                     `json:"id" gorm:"primaryKey"`
    Name     customtypes.NullString     `json:"name"`
    BirthDate customtypes.NullTime      `json:"birth_date"`
    Metadata datatypes.JSON             `json:"metadata"`
}
```

## Testing Patterns

### Test Client Creation
```go
func testClient(ctx context.Context, t *testing.T, opts ...ClientOps) (ClientInterface, func()) {
    client, err := NewClient(ctx, opts...)
    require.NoError(t, err)
    require.NotNil(t, client)
    return client, func() {
        _ = client.Close(ctx)
    }
}
```

### Cleanup SQLite Files
```go
t.Cleanup(func() {
    _ = os.Remove("datastore.db")
})
```

## Database-Specific Considerations

### MongoDB
- Uses sessions for transaction support
- Custom condition processors for complex queries
- Custom indexer functions for performance
- Collection name mapping from model names

### SQL Databases
- GORM-based implementation
- Support for read replicas with dbresolver
- Prepared statements (configurable)
- Connection pooling configuration

## Common Pitfalls

1. **Always close clients**: Use `defer client.Close(ctx)`
2. **Transaction management**: Commit transactions explicitly when needed
3. **Context usage**: Pass context for timeouts and cancellation
4. **Engine checks**: Use `IsSQLEngine()` when engine-specific logic is needed
5. **Field configuration**: Configure array and object fields with `WithCustomFields()`

## NewRelic Integration

Enable with `WithNewRelic()` option. The library automatically:
- Wraps database operations with segments
- Tracks query performance
- Reports errors and metrics

## Performance Considerations

- Use connection pooling settings in configurations
- Configure appropriate timeouts
- Use read replicas for read-heavy workloads
- Enable prepared statements for repeated queries
- Consider MongoDB indexing for complex queries

## Debugging

Enable with `WithDebugging()` or call `client.Debug(true)`. This provides:
- SQL query logging
- Performance metrics
- Error details
- Connection information
