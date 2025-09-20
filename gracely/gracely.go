// Package gracely provides a simple function for running long-lived services
// with graceful shutdown support. It allows configuration of logger, shutdown
// timeout, and OS signals to handle termination.
//
// Example usage:
//
//	package main
//
//	import (
//		"context"
//		"errors"
//		"log/slog"
//		"net/http"
//		"os"
//		"time"
//
//		"github.com/paccolamano/golazy/gracely"
//	)
//
//	// HTTPService wraps an http.Server to implement gracely.Service
//	type HTTPService struct {
//		name   string
//		logger *slog.Logger
//		server *http.Server
//	}
//
//	// Run starts the HTTP server and blocks until context is canceled
//	func (s *HTTPService) Run(ctx context.Context) {
//		s.logger.Info("Starting service", "name", s.name, "addr", s.server.Addr)
//		if err := s.server.ListenAndServe(); err != nil {
//			if !errors.Is(err, http.ErrServerClosed) {
//				s.logger.Error("failed to start service %q: %v", s.name, err)
//			}
//		}
//	}
//
//	// Shutdown gracefully shuts down the HTTP server
//	func (s *HTTPService) Shutdown(ctx context.Context) {
//		s.logger.Info("Stopping service", "name", s.name)
//		if err := s.server.Shutdown(ctx); err != nil {
//			s.logger.Error("failed to stop service %q: %v", s.name, err)
//		}
//	}
//
//	func main() {
//		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
//
//		apiserver := &HTTPService{
//			name:   "apiserver",
//			logger: logger,
//			server: &http.Server{
//				Addr: ":8080",
//				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//					_, _ = w.Write([]byte("Hello from gracely HTTP service!\n"))
//				}),
//			},
//		}
//
//		// Start the service with graceful shutdown
//		gracely.Start([]gracely.Service{apiserver},
//			gracely.WithLogger(logger),
//			gracely.WithTimeout(5*time.Second),
//		)
//
//		logger.Info("Main function exiting")
//	}
package gracely

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Logger defines the minimal logging interface required by gracely services.
// It matches the log/slog.Logger LogAttrs method, allowing custom loggers to be plugged in.
type Logger interface {
	// LogAttrs logs a message with the given level, message, and attributes.
	LogAttrs(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr)
}

// noopLogger is a default logger that discards all log messages.
type noopLogger struct{}

// LogAttrs for noopLogger does nothing.
func (noopLogger) LogAttrs(_ context.Context, _ slog.Level, _ string, _ ...slog.Attr) {}

// Service represents a long-running service with graceful shutdown support.
type Service interface {
	// Run starts the service. It should block until the service is stopped or the context is cancelled.
	Run(ctx context.Context)

	// Shutdown is called when a shutdown signal is received.
	// It should clean up resources and terminate gracefully.
	Shutdown(ctx context.Context)
}

// config holds the configuration for Start and is modified by Options.
type config struct {
	logger  Logger
	timeout time.Duration
	signals []os.Signal
}

// Option defines a functional option for configuring gracely.
type Option func(*config)

// WithLogger sets a custom Logger to be used by Start.
// Default is a no-op logger that discards messages.
func WithLogger(l Logger) Option {
	return func(c *config) {
		c.logger = l
	}
}

// WithTimeout sets the shutdown timeout for all services.
// Default is 10 seconds.
func WithTimeout(t time.Duration) Option {
	return func(c *config) {
		c.timeout = t
	}
}

// WithSignals sets which OS signals trigger shutdown.
// Default is syscall.SIGINT and syscall.SIGTERM.
func WithSignals(s ...os.Signal) Option {
	return func(c *config) {
		c.signals = s
	}
}

// Start launches the given services concurrently and handles graceful shutdown.
//
// It listens for OS signals (SIGINT, SIGTERM by default), cancels the context for all
// services when a signal is received, and calls Shutdown on each service with a
// configurable timeout.
//
// Usage example:
//
//	services := []gracely.Service{&MyService{}}
//	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
//	gracely.Start(services, gracely.WithLogger(logger), gracely.WithTimeout(5*time.Second))
func Start(services []Service, opts ...Option) {
	c := &config{
		logger:  noopLogger{},
		timeout: 10 * time.Second,
		signals: []os.Signal{syscall.SIGINT, syscall.SIGTERM},
	}

	for _, opt := range opts {
		opt(c)
	}

	ctx, stop := signal.NotifyContext(context.Background(), c.signals...)
	defer stop()

	var wg sync.WaitGroup

	for _, svc := range services {
		wait(&wg, func() {
			svc.Run(ctx)
		})
	}

	<-ctx.Done()
	c.logger.LogAttrs(ctx, slog.LevelInfo, "shutdown signal received", slog.Duration("timeout", c.timeout))

	shutdownCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	for _, svc := range services {
		wait(&wg, func() {
			svc.Shutdown(shutdownCtx)
		})
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		c.logger.LogAttrs(context.Background(), slog.LevelInfo, "graceful shutdown completed")
	case <-time.After(c.timeout):
		c.logger.LogAttrs(context.Background(), slog.LevelWarn, "forced shutdown: timeout reached")
	}
}

func wait(wg *sync.WaitGroup, f func()) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		f()
	}()
}
