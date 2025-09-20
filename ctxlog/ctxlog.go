// Package ctxlog provides a configurable slog.Handler that automatically
// extracts attributes from a context.Context and includes them in log records.
//
// Users can define custom extractors to add any contextual information they need.
//
// Example usage:
//
//	package main
//
//	import (
//		"context"
//		"log/slog"
//		"os"
//
//		"github.com/paccolamano/golazy/ctxlog"
//	)
//
//	func main() {
//		type traceIDContextKey string
//
//		ctxhandler := ctxlog.NewContextHandler(
//			ctxlog.WithBaseHandler(slog.NewJSONHandler(os.Stdout, nil)),
//			ctxlog.WithExtractor(func(ctx context.Context) []slog.Attr {
//				if v := ctx.Value(traceIDContextKey("traceID")); v != nil {
//					return []slog.Attr{slog.String("traceID", v.(string))}
//				}
//				return nil
//			}),
//		)
//
//		logger := slog.New(ctxhandler)
//
//		ctx := context.Background()
//		ctx = context.WithValue(ctx, traceIDContextKey("traceID"),
//			"0ff767b1-fc87-445e-b629-374fe93d10b2")
//
//		logger.InfoContext(ctx, "Starting process")
//	}
package ctxlog

import (
	"context"
	"log/slog"
	"os"
)

// AttrExtractor defines a function that extracts one or more slog.Attr
// from a context.Context. These attributes will be added to log records
// automatically by ContextHandler.
type AttrExtractor func(ctx context.Context) []slog.Attr

// config holds configuration options for ContextHandler.
type config struct {
	baseHandler slog.Handler
	extractors  []AttrExtractor
}

// Option defines a functional option used to configure a ContextHandler.
type Option func(*config)

// WithBaseHandler sets a custom base slog.Handler that ContextHandler will
// delegate log output to. By default, it uses slog.NewTextHandler(os.Stdout).
func WithBaseHandler(h slog.Handler) Option {
	return func(c *config) {
		c.baseHandler = h
	}
}

// WithExtractor adds an AttrExtractor to ContextHandler. Multiple extractors
// can be added and will all be applied to each log record.
func WithExtractor(ex AttrExtractor) Option {
	return func(c *config) {
		c.extractors = append(c.extractors, ex)
	}
}

// ContextHandler is a slog.Handler that wraps another base handler and
// automatically enriches log records with attributes extracted from a context.Context.
type ContextHandler struct {
	base       slog.Handler
	extractors []AttrExtractor
}

// NewContextHandler creates a new ContextHandler with optional configuration
// via functional options (base handler, extractors, etc.).
func NewContextHandler(opts ...Option) *ContextHandler {
	c := &config{
		baseHandler: slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
	}

	for _, opt := range opts {
		opt(c)
	}

	return &ContextHandler{
		base:       c.baseHandler,
		extractors: c.extractors,
	}
}

// Enabled reports whether a log at the given level would be handled by the base handler.
// Delegates to the underlying base handler.
func (h *ContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.base.Enabled(ctx, level)
}

// Handle enriches the given slog.Record with attributes extracted from the context
// and passes it to the base handler.
func (h *ContextHandler) Handle(ctx context.Context, rec slog.Record) error {
	attrs := h.extractAttrs(ctx)
	newRec := rec
	newRec.AddAttrs(attrs...)

	return h.base.Handle(ctx, newRec)
}

// WithAttrs returns a new ContextHandler that adds the provided attributes
// to every log record. The extractors remain unchanged.
func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{
		base:       h.base.WithAttrs(attrs),
		extractors: h.extractors,
	}
}

// WithGroup returns a new ContextHandler that groups all log attributes under
// the specified group name. The extractors remain unchanged.
func (h *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{
		base:       h.base.WithGroup(name),
		extractors: h.extractors,
	}
}

// extractAttrs applies all registered extractors to the given context and
// returns the combined list of slog.Attr.
func (h *ContextHandler) extractAttrs(ctx context.Context) []slog.Attr {
	var result []slog.Attr
	for _, ex := range h.extractors {
		result = append(result, ex(ctx)...)
	}
	return result
}
