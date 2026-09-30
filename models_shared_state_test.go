package datastore

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newRelicSettingKey is the gorm setting the nrgorm package stores a New Relic transaction under
const newRelicSettingKey = "newrelicTransaction"

// errWrongResult is returned by a concurrent worker whose read did not see its own write
var errWrongResult = errors.New("a concurrent call returned another call's result")

// newRelicTestContext returns a context carrying a transaction from a New Relic application that never reports
func newRelicTestContext(t *testing.T) context.Context {
	t.Helper()
	app, err := newrelic.NewApplication(newrelic.ConfigAppName("go-datastore-test"), newrelic.ConfigEnabled(false))
	require.NoError(t, err)
	txn := app.StartTransaction("datastore-test")
	require.NotNil(t, txn)
	t.Cleanup(txn.End)
	return newrelic.NewContext(context.Background(), txn)
}

// traceRecorder records, for every statement a client runs, whether it carried a New Relic transaction
type traceRecorder struct {
	mu     sync.Mutex
	traced []bool
}

// recordTraces registers gorm callbacks on the client that record whether each statement carried a transaction
func recordTraces(t *testing.T, c ClientInterface) *traceRecorder {
	t.Helper()
	recorder := &traceRecorder{}
	record := func(db *gorm.DB) {
		_, traced := db.Get(newRelicSettingKey)
		recorder.mu.Lock()
		defer recorder.mu.Unlock()
		recorder.traced = append(recorder.traced, traced)
	}
	callbacks := c.(*Client).options.db.Callback()
	require.NoError(t, callbacks.Query().After("gorm:query").Register("test:record_query", record))
	require.NoError(t, callbacks.Create().After("gorm:create").Register("test:record_create", record))
	require.NoError(t, callbacks.Update().After("gorm:update").Register("test:record_update", record))
	require.NoError(t, callbacks.Row().After("gorm:row").Register("test:record_row", record))
	return recorder
}

// take returns what was recorded since the last call and starts over
func (r *traceRecorder) take() []bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	traced := r.traced
	r.traced = nil
	return traced
}

// TestClient_NewRelicTransactionStaysWithItsCall proves each call's New Relic transaction is carried by that call's own
// statements and nothing else: the client's shared database handle is never replaced or given a transaction, and a
// later call without one does not inherit it
func TestClient_NewRelicTransactionStaysWithItsCall(t *testing.T) {
	c := setupTestClient(t)
	defer func() { _ = c.Close(context.Background()) }()
	shared := c.(*Client).options.db
	recorder := recordTraces(t, c)
	ctx := newRelicTestContext(t)
	conditions := map[string]any{testFieldName: "traced"}

	calls := []struct {
		name string
		call func() error
	}{
		{"SaveModel", func() error {
			tx, err := c.NewRawTx()
			if err != nil {
				return err
			}
			return c.SaveModel(ctx, &TestModel{Name: "traced", Value: 1}, tx, true, true)
		}},
		{"GetModel", func() error {
			return c.GetModel(ctx, &TestModel{}, conditions, time.Second, false)
		}},
		{"GetModelPartial", func() error {
			return c.GetModelPartial(ctx, &TestModel{}, []string{"id"}, conditions, time.Second, false)
		}},
		{"GetModels", func() error {
			var models []TestModel
			return c.GetModels(ctx, &models, conditions, &QueryParams{}, nil, time.Second)
		}},
		{"GetModelsPartial", func() error {
			var models []TestModel
			return c.GetModelsPartial(ctx, &models, []string{"id"}, conditions, time.Second)
		}},
		{"GetModelCount", func() error {
			_, err := c.GetModelCount(ctx, &TestModel{}, conditions, time.Second)
			return err
		}},
		{"GetModelsAggregate", func() error {
			var models []TestModel
			_, err := c.GetModelsAggregate(ctx, &models, map[string]any{}, testFieldName, time.Second)
			return err
		}},
		{"IncrementModel", func() error {
			var model TestModel
			if err := c.GetModel(ctx, &model, conditions, time.Second, false); err != nil {
				return err
			}
			recorder.take() // Only the increment's own statements are checked
			_, err := c.IncrementModel(ctx, &model, "value", 1)
			return err
		}},
	}
	for _, tc := range calls {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, tc.call())
			traced := recorder.take()
			require.NotEmpty(t, traced, "the call must run at least one statement")
			for _, carried := range traced {
				assert.True(t, carried, "every statement of the call must carry its transaction")
			}
			assert.Same(t, shared, c.(*Client).options.db, "the shared handle must never be replaced")
			_, stored := shared.Get(newRelicSettingKey)
			assert.False(t, stored, "the shared handle must never hold a call's transaction")
		})
	}

	t.Run("a later call without a transaction inherits none", func(t *testing.T) {
		require.NoError(t, c.GetModel(context.Background(), &TestModel{}, conditions, time.Second, false))
		traced := recorder.take()
		require.NotEmpty(t, traced)
		for _, carried := range traced {
			assert.False(t, carried)
		}
	})
}

// TestClient_ConcurrentCallsShareNoState proves calls with and without a New Relic transaction can run on one client
// from many goroutines at once, next to raw statements and new transactions, and each still sees its own result. Run
// it with the race detector: a call that writes the client's shared state is reported here. The rows are written
// first, one at a time, because an in-memory SQLite database refuses concurrent writers outright.
func TestClient_ConcurrentCallsShareNoState(t *testing.T) {
	c := setupTestClient(t)
	defer func() { _ = c.Close(context.Background()) }()
	traced := newRelicTestContext(t)

	const workers = 8
	for i := range workers {
		tx, err := c.NewRawTx()
		require.NoError(t, err)
		require.NoError(t, c.SaveModel(context.Background(), &TestModel{Name: "concurrent_" + strconv.Itoa(i), Value: i},
			tx, true, true))
	}

	var wg sync.WaitGroup
	errs := make(chan error, workers*8)
	for i := range workers {
		ctx := context.Background()
		if i%2 == 0 {
			ctx = traced
		}
		name := "concurrent_" + strconv.Itoa(i)
		wg.Add(1)
		go func() {
			defer wg.Done()
			var found TestModel
			errs <- c.GetModel(ctx, &found, map[string]any{testFieldName: name}, 5*time.Second, false)
			if found.Name != name || found.Value != i {
				errs <- errWrongResult
			}
			count, err := c.GetModelCount(ctx, &TestModel{}, map[string]any{testFieldName: name}, 5*time.Second)
			errs <- err
			if count != 1 {
				errs <- errWrongResult
			}
			var raw int64
			errs <- c.Raw("SELECT count(*) FROM test_models WHERE name = '" + name + "'").Scan(&raw).Error
			if raw != 1 {
				errs <- errWrongResult
			}
			tx, err := c.NewRawTx()
			errs <- err
			if err == nil {
				errs <- tx.Rollback() //nolint:contextcheck // Rollback takes no context
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
}
