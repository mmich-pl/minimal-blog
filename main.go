package main

import (
	"context"
	"github.com/gocql/gocql"
	"github.com/jonboulle/clockwork"
	"log/slog"
	"ndb/logging"
	logrepo "ndb/repositories/log"
	"os"
)

func main() {
	ctx := context.Background()
	defaultLogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	store, err := logrepo.NewStore(
		ctx,
		&logrepo.ConnConfig{
			Consistency: gocql.Quorum,
			Keyspace:    "log_storage",
			Hosts:       []string{"127.0.0.1"},
		},
		logrepo.WithClock(clockwork.NewRealClock()),
		logrepo.WithLogger(defaultLogger),
	)
	if err != nil {
		panic(err)
	}

	defer store.Close(ctx)

	persister := logging.NewPersister(&logging.Config{
		Level:    slog.LevelInfo,
		LogStore: store,
	})

	log := slog.New(
		logging.NewFanOut(
			slog.NewJSONHandler(os.Stdout, nil),
			persister,
		),
	)

	log.InfoContext(context.Background(), "first log", slog.Any("status", "success"))
}
