package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/sashabeketov/avito-go-labworks/internal/config"
	"github.com/sashabeketov/avito-go-labworks/internal/container"
)

func main() {
	os.Exit(exitCode())
}

func exitCode() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		slog.Error("trip service stopped", "error", err)
		return 1
	}

	return 0
}

func run(ctx context.Context) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	app, err := container.New(ctx, cfg)
	if err != nil {
		return fmt.Errorf("build application: %w", err)
	}

	server := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           app.Handler,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	}

	app.Logger.InfoContext(ctx, "service starting", "addr", cfg.HTTP.Addr)
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.ListenAndServe()
	}()

	var listenErr error
	stopHTTP := false
	select {
	case listenErr = <-serveErr:
	case <-ctx.Done():
		stopHTTP = true
	}

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.App.ShutdownTimeout)
	defer cancel()
	var shutdownErr error
	if stopHTTP {
		if err := server.Shutdown(shutdownCtx); err != nil {
			shutdownErr = errors.Join(fmt.Errorf("shutdown HTTP server: %w", err), server.Close())
		}
		listenErr = <-serveErr
	}
	if listenErr != nil && !errors.Is(listenErr, http.ErrServerClosed) {
		shutdownErr = errors.Join(shutdownErr, fmt.Errorf("HTTP server: %w", listenErr))
	}
	shutdownErr = errors.Join(shutdownErr, app.Close(shutdownCtx))
	if shutdownErr == nil {
		app.Logger.InfoContext(shutdownCtx, "service stopped")
	}
	return shutdownErr
}
