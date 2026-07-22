package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/joho/godotenv"

	"posts-service/internal/config"
	graphqlhandler "posts-service/internal/handler/graphql"
	"posts-service/internal/handler/graphql/dataloader"
	"posts-service/internal/middleware"
	"posts-service/internal/pubsub"
	"posts-service/internal/repository"
	"posts-service/internal/repository/memory"
	"posts-service/internal/repository/postgres"
	"posts-service/internal/usecase"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("service stopped with error", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	postRepo, commentRepo, txManager, cleanup, err := buildRepositories(ctx, cfg)
	if err != nil {
		return err
	}
	defer cleanup()

	bus := pubsub.NewCommentBus()
	postUsecase := usecase.NewPostUsecase(postRepo)
	commentUsecase := usecase.NewCommentUsecase(commentRepo, postRepo, txManager, bus)

	resolver := graphqlhandler.NewResolver(postUsecase, commentUsecase, bus)
	gqlServer := graphqlhandler.NewServer(resolver)

	mux := http.NewServeMux()
	mux.Handle("/", playground.Handler("Posts and Comments", "/query"))
	mux.Handle("/query", dataloader.Middleware(commentRepo, gqlServer))

	var handler http.Handler = mux
	handler = middleware.Logging(logger, handler)
	handler = middleware.Recover(logger, handler)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("server started", "port", cfg.Port, "storage", cfg.Storage)
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}

func buildRepositories(ctx context.Context, cfg config.Config) (repository.PostRepository, repository.CommentRepository, repository.TxManager, func(), error) {
	switch cfg.Storage {
	case config.StoragePostgres:
		pool, err := postgres.Connect(ctx, cfg.PostgresDSN)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		return postgres.NewPostRepo(pool), postgres.NewCommentRepo(pool), postgres.NewTxManager(pool), pool.Close, nil
	default:
		return memory.NewPostRepo(), memory.NewCommentRepo(), memory.NewTxManager(), func() {}, nil
	}
}
