package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"
	slogchi "github.com/samber/slog-chi"
	httpSwagger "github.com/swaggo/http-swagger"

	s3client "ndb/clients/aws"
	"ndb/config"
	"ndb/repositories/posts"
	"ndb/services/post"
)

type Server struct {
	*config.HTTPServer
	log    *slog.Logger
	router *chi.Mux

	uploadService *post.Service
}

func NewServer(ctx context.Context, logger *slog.Logger, cfg *config.Config) (*Server, error) {
	s3Client, err := s3client.New(ctx, logger, &cfg.S3)
	if err != nil {
		return nil, err
	}

	mongo, err := posts.NewStore(ctx, logger, &cfg.Neo4j)
	if err != nil {
		return nil, err
	}

	srv := &Server{
		HTTPServer:    &cfg.HTTPServer,
		log:           logger,
		router:        chi.NewRouter(),
		uploadService: post.NewService(s3Client, mongo, logger),
	}

	srv.router.Use(slogchi.NewWithConfig(logger, slogchi.Config{
		DefaultLevel:     slog.LevelInfo,
		ClientErrorLevel: slog.LevelError,
		ServerErrorLevel: slog.LevelError,
		WithUserAgent:    true,
		WithRequestID:    false,
	}))
	srv.routes()

	return srv, err
}

func (s *Server) Start(ctx context.Context) {
	server := http.Server{
		Addr:         fmt.Sprintf(":%d", s.HTTPServer.Port),
		Handler:      s.router,
		IdleTimeout:  s.HTTPServer.IdleTimeout,
		ReadTimeout:  s.HTTPServer.ReadTimeout,
		WriteTimeout: s.HTTPServer.WriteTimeout,
	}

	shutdownComplete := handleShutdown(func() {
		if err := server.Shutdown(ctx); err != nil {
			s.log.ErrorContext(ctx, "Server shutdown failed", slog.Any("error", err))
			return
		}
	})

	if err := server.ListenAndServe(); errors.Is(err, http.ErrServerClosed) {
		s.log.InfoContext(ctx, "Server started successfully", slog.Any("address", server.Addr))
		<-shutdownComplete
	} else {
		s.log.ErrorContext(ctx, "Server listen and serve failed", slog.Any("error", err))
		return
	}

	s.log.InfoContext(ctx, "Shutdown gracefully")
}

func handleShutdown(onShutdownSignal func()) <-chan struct{} {
	shutdown := make(chan struct{})

	go func() {
		shutdownSignal := make(chan os.Signal, 1)
		signal.Notify(shutdownSignal, os.Interrupt, syscall.SIGTERM)

		<-shutdownSignal

		onShutdownSignal()
		close(shutdown)
	}()

	return shutdown
}

func (s *Server) routes() {
	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
	}))

	s.router.Use(render.SetContentType(render.ContentTypeJSON))
	s.router.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/swagger/doc.json"), // The url pointing to API definition
	))

	s.router.Get("/health", s.handleGetHealth)

	s.router.Route("/api/v1/posts", func(r chi.Router) {
		r.Post("/", s.CreatePost)
	})
}
