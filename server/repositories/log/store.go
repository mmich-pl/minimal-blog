package logrepo

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/gocql/gocql"
	"github.com/jonboulle/clockwork"
)

type ConnConfig struct {
	Consistency gocql.Consistency
	Keyspace    string
	Hosts       []string
}

type Store struct {
	log       *slog.Logger
	session   *gocql.Session
	batchSize int
	interval  time.Duration
	batch     *gocql.Batch
	mu        sync.Mutex
	flushCh   chan struct{}
	clock     clockwork.Clock
}

// StoreOption defines a type for modifying Store configurations.
type StoreOption func(*Store)

// WithClock sets the clock for the Store.
func WithClock(clock clockwork.Clock) StoreOption {
	return func(s *Store) {
		s.clock = clock
	}
}

// WithBatchSize sets the batch size for the Store.
func WithBatchSize(batchSize int) StoreOption {
	return func(s *Store) {
		s.batchSize = batchSize // Adjust this logic based on how batch size is handled.
	}
}

// WithInterval sets the interval for batch flushing.
func WithInterval(interval time.Duration) StoreOption {
	return func(s *Store) {
		s.interval = interval
	}
}

// WithLogger sets the logger for store.
func WithLogger(log *slog.Logger) StoreOption {
	return func(s *Store) {
		s.log = log
	}
}

// NewStore initializes a new ScyllaDB client with options.
func NewStore(ctx context.Context, cfg *ConnConfig, opts ...StoreOption) (*Store, error) {
	cluster := createCluster(cfg.Consistency, cfg.Keyspace, cfg.Hosts...)
	session, err := gocql.NewSession(*cluster)
	if err != nil {
		return nil, err
	}

	store := &Store{
		session:   session,
		batch:     session.NewBatch(gocql.UnloggedBatch),
		flushCh:   make(chan struct{}),
		clock:     clockwork.NewRealClock(),
		batchSize: 5,
		interval:  1 * time.Second,
	}

	// Apply options
	for _, opt := range opts {
		opt(store)
	}

	go store.batchWorker(ctx)

	return store, nil
}

// Insert adds a request to the batch.
func (c *Store) Insert(ctx context.Context, query string, values ...interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.log.DebugContext(ctx, "Add new query to batch", "query", query, "values", values)
	c.batch.Query(query, values...)
	if len(c.batch.Entries) >= c.batchSize {
		// If batch size reaches limit, flush the batch.
		c.flushCh <- struct{}{}
	}
}

// batchWorker runs a loop that flushes the batch either at 5-second intervals or when batch size reaches 100.
func (c *Store) batchWorker(ctx context.Context) {
	ticker := c.clock.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.Chan():
			// Time-based flush
			c.flushBatch(ctx)
		case <-c.flushCh:
			// Size-based flush
			c.flushBatch(ctx)
		}
	}
}

// flushBatch executes the batch and clears it.
func (c *Store) flushBatch(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.batch.Entries) == 0 {
		return
	}

	// Execute the batch
	if err := c.session.ExecuteBatch(c.batch); err != nil {
		c.log.ErrorContext(ctx, "Failed to execute batch", slog.Any("error", err))
	} else {
		c.log.InfoContext(ctx, "Batch executed successfully", slog.Int("entries", len(c.batch.Entries)))
	}

	// Clear the batch
	c.batch = c.session.NewBatch(gocql.UnloggedBatch)
}

// Close closes the ScyllaDB session.
func (c *Store) Close(ctx context.Context) {
	c.flushBatch(ctx)
	c.session.Close()
}

func createCluster(
	consistency gocql.Consistency,
	keyspace string,
	hosts ...string,
) *gocql.ClusterConfig {
	retryPolicy := &gocql.ExponentialBackoffRetryPolicy{
		Min:        time.Second,
		Max:        10 * time.Second,
		NumRetries: 5,
	}
	cluster := gocql.NewCluster(hosts...)
	cluster.Keyspace = keyspace
	cluster.Timeout = 5 * time.Second
	cluster.RetryPolicy = retryPolicy
	cluster.Consistency = consistency
	cluster.PoolConfig.HostSelectionPolicy = gocql.TokenAwareHostPolicy(gocql.RoundRobinHostPolicy())
	return cluster
}
