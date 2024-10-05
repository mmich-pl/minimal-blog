package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/gocql/gocql"
	"github.com/jonboulle/clockwork"

	"ndb/app/api"
	"ndb/config"
	_ "ndb/docs"
	"ndb/logging"
	logrepo "ndb/repositories/log"
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

	cfg, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}

	server, err := api.NewServer(ctx, log, cfg)
	if err != nil {
		panic(err)
	}

	server.Start(ctx)
}
