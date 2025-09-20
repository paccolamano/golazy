// Package recover provides an HTTP middleware that safely recovers from panics
// in downstream handlers. It logs the panic in a structured way and optionally
// includes the stack trace. It also provides a default JSON 500 response or
// allows a custom callback for custom recovery behavior.
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
//		"github.com/paccolamano/golazy/handlers/recover"
//	)
//
//	func main() {
//		mux := http.NewServeMux()
//
//		// Example handler that may panic
//		myHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//			panic("something went wrong")
//		})
//
//		// Default recovery handler
//		mux.Handle("/api", recover.New()(myHandler))
//
//		// Custom recovery: include stack trace and debug-level logging
//		mux.Handle("/dev", recover.New(
//			recover.WithIncludeStack(true),
//			recover.WithLogLevel(slog.LevelDebug),
//			recover.WithCallback(func(w http.ResponseWriter, r *http.Request, rec any, stack []byte) {
//				w.Header().Set("Content-Type", "text/plain")
//				w.WriteHeader(http.StatusInternalServerError)
//				fmt.Fprintf(w, "panic: %v\n", rec)
//				if len(stack) > 0 {
//					w.Write([]byte("\nstack:\n"))
//					w.Write(stack)
//				}
//			}),
//		)(myHandler))
//
//		log.Fatal(http.ListenAndServe(":8080", mux))
//	}
package recover

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Logger is a minimal structured-logger interface used by New.
// It mirrors slog.Logger.LogAttrs so callers can log with structured attributes.
type Logger interface {
	// LogAttrs logs a message at a given level with structured attributes.
	LogAttrs(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr)
}

// config holds configuration for the recover handler.
type config struct {
	// Logger is used for structured logging. Defaults to slog.Default().
	Logger Logger

	// Level is the log level used when emitting the recovered panic.
	// Defaults to slog.LevelError.
	Level slog.Level

	// IncludeStack controls whether the stack trace is captured and included
	// in the log and in the callback. Capturing the stack is relatively costly,
	// so the default is false.
	IncludeStack bool

	// Message is the log message used when logging recovered panics.
	// Defaults to "recovered from panic".
	Message string

	// StatusCode is the HTTP status code the default callback will use.
	// Defaults to http.StatusInternalServerError (500).
	StatusCode int

	// Callback is invoked after a panic is recovered. It receives the ResponseWriter,
	// the Request, the recovered value (any), and the stack trace (which may be nil).
	// If nil, a default JSON 500 response is written.
	Callback func(w http.ResponseWriter, r *http.Request, recovered any, stack []byte)
}

// Option mutates Options.
type Option func(*config)

// WithLogger sets a structured Logger for the recover handler.
func WithLogger(l Logger) Option {
	return func(c *config) {
		c.Logger = l
	}
}

// WithLogLevel sets the logging level used for recovered panics.
func WithLogLevel(level slog.Level) Option {
	return func(c *config) {
		c.Level = level
	}
}

// WithIncludeStack toggles whether to capture the stack trace.
func WithIncludeStack(include bool) Option {
	return func(c *config) {
		c.IncludeStack = include
	}
}

// WithMessage sets the log message used when a panic is recovered.
func WithMessage(msg string) Option {
	return func(c *config) {
		c.Message = msg
	}
}

// WithStatusCode sets the HTTP status code used by the default callback.
func WithStatusCode(code int) Option {
	return func(c *config) {
		c.StatusCode = code
	}
}

// WithCallback sets a custom callback invoked after recovery. The callback
// receives the recovered value and the stack trace (which may be nil if IncludeStack=false).
func WithCallback(f func(w http.ResponseWriter, r *http.Request, recovered any, stack []byte)) Option {
	return func(c *config) {
		c.Callback = f
	}
}

// New returns a handler that recovers from panics in handlers.
//
// Behavior & defaults:
//   - structured logging via Logger (defaults to slog.Default()).
//   - default log level: slog.LevelError.
//   - by default the stack trace is NOT captured (IncludeStack=false) to avoid overhead.
//   - default callback writes a JSON 500: {"error":"Internal Server Error"}.
//
// Example:
//
//	// default handler
//	mux.Handle("/api", recover.New()(myHandler))
//
//	// custom: include stack and debug-level log
//	mux.Handle("/dev", recover.New(
//	    recover.WithIncludeStack(true),
//	    recover.WithLogLevel(slog.LevelDebug),
//	    recover.WithCallback(func(w http.ResponseWriter, r *http.Request, rec any, stack []byte) {
//	        // custom response
//	        w.Header().Set("Content-Type", "text/plain")
//	        w.WriteHeader(http.StatusInternalServerError)
//	        fmt.Fprintf(w, "panic: %v\n", rec)
//	        if len(stack) > 0 {
//	            w.Write([]byte("\nstack:\n"))
//	            w.Write(stack)
//	        }
//	    }),
//	)(myHandler))
func New(opts ...Option) func(http.Handler) http.Handler {
	c := &config{
		Logger:       slog.Default(),
		Level:        slog.LevelError,
		IncludeStack: false,
		Message:      "recovered from panic",
		StatusCode:   http.StatusInternalServerError,
	}

	c.Callback = func(w http.ResponseWriter, r *http.Request, _ any, _ []byte) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(c.StatusCode)
		err := json.NewEncoder(w).Encode(map[string]string{
			"error": http.StatusText(c.StatusCode),
		})
		if err != nil {
			c.Logger.LogAttrs(r.Context(), c.Level,
				"failed to send recovery response",
				slog.String("error", err.Error()))
		}
	}

	for _, opt := range opts {
		opt(c)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func(ctx context.Context) {
				if rec := recover(); rec != nil {
					var errMsg string
					switch e := rec.(type) {
					case error:
						errMsg = e.Error()
					default:
						errMsg = fmt.Sprint(e)
					}

					var stack []byte
					if c.IncludeStack {
						stack = debug.Stack()
					}

					attrs := []slog.Attr{slog.String("error", errMsg)}
					if c.IncludeStack {
						attrs = append(attrs, slog.String("stack", string(stack)))
					}

					c.Logger.LogAttrs(ctx, c.Level, c.Message, attrs...)

					if c.Callback != nil {
						c.Callback(w, r, rec, stack)
					}
				}
			}(r.Context())

			next.ServeHTTP(w, r)
		})
	}
}
