// Package logger provides an HTTP logging handler that integrates with Go's
// structured logging via log/slog. It allows configurable logging of
// request and response fields, supports skipping certain paths, and can
// plug into custom loggers.
//
// The handler logs both incoming requests and completed responses, including
// attributes such as method, path, query, IP, user-agent, status code, and
// duration. Users can define which fields to log, the log levels, and
// conditions to skip logging for specific requests.
//
// Example usage:
//
//	package main
//
//	import (
//		"log/slog"
//		"net/http"
//
//		"github.com/paccolamano/golazy/handlers/logger"
//	)
//
//	func main() {
//		mux := http.NewServeMux()
//		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
//			w.Write([]byte("hello"))
//		})
//
//		// Wrap the mux with the logging handler
//		loggedMux := logger.New(
//			logger.WithRequestInLevel(slog.LevelDebug),
//			logger.WithSkipPaths("/healthz", "/metrics"),
//		)(mux)
//
//		http.ListenAndServe(":8080", loggedMux)
//	}
package logger

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"
)

// Logger defines the minimal logging interface required by this handler.
// It matches log/slog.Logger's LogAttrs method, but allows plugging in custom loggers.
type Logger interface {
	LogAttrs(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr)
}

// Field represents a request/response attribute that can be logged.
type Field string

const (
	// FieldMethod logs the HTTP request method (GET, POST, etc).
	FieldMethod Field = "method"
	// FieldPath logs the request URL path.
	FieldPath Field = "path"
	// FieldQuery logs the raw query string from the URL.
	FieldQuery Field = "query"
	// FieldIP logs the client IP address (from X-Real-IP or RemoteAddr).
	FieldIP Field = "ip"
	// FieldUserAgent logs the User-Agent header.
	FieldUserAgent Field = "userAgent"
	// FieldContentLength logs the request Content-Length header value.
	FieldContentLength Field = "contentLength"
	// FieldStatus logs the HTTP response status code.
	FieldStatus Field = "status"
	// FieldDuration logs the time it took to serve the request.
	FieldDuration Field = "duration"
)

// config defines configuration for the logging handler.
type config struct {
	// Logger to use for structured logging. Defaults to slog.Default().
	Logger Logger
	// LevelRequestIn defines the log level for incoming requests.
	LevelRequestIn slog.Level
	// LevelRequestOut defines the log level for completed requests.
	LevelRequestOut slog.Level
	// FieldsIn is the list of request attributes to log at request start.
	FieldsIn []Field
	// FieldsOut is the list of response attributes to log after request completion.
	FieldsOut []Field
	// SkipPaths defines path prefixes that should not be logged.
	SkipPaths []string
	// SkipFunc is an optional function to skip logging for certain requests.
	SkipFunc func(r *http.Request) bool
}

// Option represents a functional option for configuring logger handler.
type Option func(*config)

// WithLogger sets a custom Logger implementation.
func WithLogger(l Logger) Option {
	return func(c *config) {
		c.Logger = l
	}
}

// WithRequestInLevel sets the log level for incoming requests.
func WithRequestInLevel(level slog.Level) Option {
	return func(c *config) {
		c.LevelRequestIn = level
	}
}

// WithRequestOutLevel sets the log level for completed requests.
func WithRequestOutLevel(level slog.Level) Option {
	return func(c *config) {
		c.LevelRequestOut = level
	}
}

// WithFieldsIn specifies which request fields to log at request start.
func WithFieldsIn(fields ...Field) Option {
	return func(c *config) {
		c.FieldsIn = fields
	}
}

// WithFieldsOut specifies which response fields to log after request completion.
func WithFieldsOut(fields ...Field) Option {
	return func(c *config) {
		c.FieldsOut = fields
	}
}

// WithSkipPaths configures path prefixes to exclude from logging.
func WithSkipPaths(paths ...string) Option {
	return func(c *config) {
		c.SkipPaths = append(c.SkipPaths, paths...)
	}
}

// WithSkipFunc sets a custom function to decide whether a request should be skipped.
func WithSkipFunc(fn func(r *http.Request) bool) Option {
	return func(c *config) {
		c.SkipFunc = fn
	}
}

// responseWriter is a wrapper around http.ResponseWriter
// that captures the HTTP status code written.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// New creates a new logging handler with the given options.
// It returns a function that wraps an http.Handler and logs request/response details.
//
// Example:
//
//	mux := http.NewServeMux()
//	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
//		w.Write([]byte("hello"))
//	})
//
//	loggedMux := logger.New(
//		logger.WithRequestInLevel(slog.LevelDebug),
//		logger.WithSkipPaths("/healthz", "/metrics"),
//	)(mux)
//
//	http.ListenAndServe(":8080", loggedMux)
func New(opts ...Option) func(http.Handler) http.Handler {
	c := &config{
		Logger:          slog.Default(),
		LevelRequestIn:  slog.LevelInfo,
		LevelRequestOut: slog.LevelInfo,
		FieldsIn: []Field{
			FieldMethod, FieldPath, FieldQuery, FieldIP, FieldUserAgent, FieldContentLength,
		},
		FieldsOut: []Field{
			FieldMethod, FieldPath, FieldStatus, FieldDuration,
		},
		SkipPaths: nil,
		SkipFunc:  nil,
	}

	for _, opt := range opts {
		opt(c)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if shouldSkip(r, c) {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			ip := r.Header.Get("X-Real-IP")
			if ip == "" {
				ip, _, _ = net.SplitHostPort(r.RemoteAddr)
			}

			c.Logger.LogAttrs(r.Context(), c.LevelRequestIn, "incoming request",
				buildAttrs(c.FieldsIn, r, rw, ip, start)...,
			)

			next.ServeHTTP(rw, r)

			c.Logger.LogAttrs(r.Context(), c.LevelRequestOut, "request completed",
				buildAttrs(c.FieldsOut, r, rw, ip, start)...,
			)
		})
	}
}

func shouldSkip(r *http.Request, opt *config) bool {
	if opt.SkipFunc != nil && opt.SkipFunc(r) {
		return true
	}

	for _, prefix := range opt.SkipPaths {
		if strings.HasPrefix(r.URL.Path, prefix) {
			return true
		}
	}

	return false
}

func buildAttrs(fields []Field, r *http.Request, rw *responseWriter, ip string, start time.Time) []slog.Attr {
	attrs := make([]slog.Attr, 0, len(fields))

	for _, f := range fields {
		switch f {
		case FieldMethod:
			attrs = append(attrs, slog.String("method", r.Method))
		case FieldPath:
			attrs = append(attrs, slog.String("path", r.URL.Path))
		case FieldQuery:
			attrs = append(attrs, slog.String("query", r.URL.RawQuery))
		case FieldIP:
			attrs = append(attrs, slog.String("ip", ip))
		case FieldUserAgent:
			attrs = append(attrs, slog.String("userAgent", r.UserAgent()))
		case FieldContentLength:
			attrs = append(attrs, slog.Int64("contentLength", r.ContentLength))
		case FieldStatus:
			attrs = append(attrs, slog.Int("status", rw.statusCode))
		case FieldDuration:
			attrs = append(attrs, slog.Duration("duration", time.Since(start)))
		}
	}

	return attrs
}
