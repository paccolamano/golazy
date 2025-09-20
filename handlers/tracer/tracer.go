// Package tracer provides an HTTP middleware for generating and propagating
// unique request identifiers (UUIDs) for incoming HTTP requests.
//
// Each request is assigned a UUID that is:
//  1. Stored in the request context under a configurable key.
//  2. Added to the HTTP response headers under a configurable header key (default "X-Trace-ID").
//
// This is useful for request tracing, correlation in logs, and distributed system debugging.
//
// Example usage:
//
//	package main
//
//	import (
//		"fmt"
//		"log"
//		"net/http"
//
//		"github.com/google/uuid"
//		"github.com/paccolamano/golazy/handlers/tracer"
//	)
//
//	func main() {
//		mux := http.NewServeMux()
//
//		myHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//			traceID := tracer.GetTraceID(r)
//			if traceID != nil {
//				fmt.Fprintf(w, "Trace ID: %s\n", traceID.String())
//			} else {
//				fmt.Fprintln(w, "No trace ID found")
//			}
//		})
//
//		// Default tracer middleware
//		mux.Handle("/api", tracer.New()(myHandler))
//
//		// Custom context key and header key
//		mux.Handle("/custom", tracer.New(
//			tracer.WithContextKey("reqID"),
//			tracer.WithHeaderKey("X-Custom-Trace-ID"),
//		)(myHandler))
//
//		log.Fatal(http.ListenAndServe(":8080", mux))
//	}
package tracer

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type traceIDKey string

const (
	defaultTraceIDKey traceIDKey = "traceID"
)

// config holds configuration options for the Tracer handler.
type config struct {
	contextKey any
	headerKey  string
}

// Option represents a functional option for configuring Tracer handler.
type Option func(*config)

// WithContextKey sets a custom trace key for the Tracer handler.
func WithContextKey(key any) Option {
	return func(c *config) {
		c.contextKey = key
	}
}

// WithHeaderKey sets a custom header key for the Tracer handler.
func WithHeaderKey(key string) Option {
	return func(c *config) {
		c.headerKey = key
	}
}

// New returns a handler that generates a unique request ID (UUID) for each incoming HTTP request,
// attaches it to the response header (default as "X-Trace-ID"), and stores it in the request context using the provided context key.
//
// This is useful for request tracing, correlation across distributed systems, and contextual logging.
//
// Parameters:
//   - contextKey: the context key under which the generated UUID will be stored in the request context.
//   - headerKey: the header key under which the generated UUID will be stored in the response headers.
//
// Example usage:
//
//	http.Handle("/api", New(WithContextKey("reqID"))(yourHandler))
//
// You can then retrieve the trace ID later in the request lifecycle:
//
//	traceID := r.Context().Value("reqID").(string)
func New(opts ...Option) func(http.Handler) http.Handler {
	c := &config{
		contextKey: defaultTraceIDKey,
		headerKey:  "X-Trace-ID",
	}

	for _, opt := range opts {
		opt(c)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			uuid := uuid.New().String()
			w.Header().Set(c.headerKey, uuid)
			ctx := context.WithValue(r.Context(), c.contextKey, uuid)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetTraceID retrieves the id stored in the
// request context by New with the default key. If no id is stored or is not valid uuid, it
// returns nil.
func GetTraceID(r *http.Request) *uuid.UUID {
	return getTraceID(r, defaultTraceIDKey)
}

// GetTraceIDWithKey retrieves the id stored in the
// request context by New with the given key. If no id is stored or is not valid uuid, it
// returns nil.
func GetTraceIDWithKey(r *http.Request, key any) *uuid.UUID {
	return getTraceID(r, key)
}

func getTraceID(r *http.Request, key any) *uuid.UUID {
	if r == nil {
		return nil
	}

	v := r.Context().Value(key)
	if v == nil {
		return nil
	}

	s, ok := v.(string)
	if !ok {
		return nil
	}

	id, err := uuid.Parse(s)
	if err != nil {
		return nil
	}

	return &id
}
