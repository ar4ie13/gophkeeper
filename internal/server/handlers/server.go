package handlers

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

// StartServer starts http and gRPC (if enabled) server
func (h *Handlers) StartServer() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-sigChan
		cancel()
	}()

	g, ctx := errgroup.WithContext(ctx)

	// HTTP server
	srv := &http.Server{
		Addr:              h.cfg.ServerAddr,
		Handler:           h.registerRoutes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	g.Go(func() error {
		h.zlog.Info().Msgf("listening on %v", h.cfg.ServerAddr)

		if err := srv.ListenAndServeTLS(h.cfg.TLSCertPath, h.cfg.TLSKeyPath); !errors.Is(err, http.ErrServerClosed) {
			return err
		}

		return nil
	})

	// HTTP graceful shutdown
	g.Go(func() error {
		<-ctx.Done()
		h.zlog.Info().Msg("Shutting down HTTP server...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		return srv.Shutdown(shutdownCtx)
	})

	// Waiting for all goroutines to stop
	if err := g.Wait(); err != nil {
		h.zlog.Error().Msgf("Server shutdown with error: %v", err)
	}

	h.zlog.Info().Msgf("GophKeeper server shutdown gracefully")
	return nil
}
