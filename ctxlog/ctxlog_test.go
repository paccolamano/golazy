package ctxlog

import (
	"context"
	"log/slog"
	"testing"

	"gotest.tools/v3/assert"
)

type testHandler struct {
	records []slog.Record
}

func (h *testHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (h *testHandler) Handle(_ context.Context, rec slog.Record) error {
	h.records = append(h.records, rec)
	return nil
}

func (h *testHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return h
}

func (h *testHandler) WithGroup(_ string) slog.Handler {
	return h
}

func TestContextHandlerExtractAttrs(t *testing.T) {
	type userContextKey string

	th := &testHandler{}
	handler := NewContextHandler(
		WithBaseHandler(th),
		WithExtractor(func(ctx context.Context) []slog.Attr {
			if v := ctx.Value(userContextKey("userID")); v != nil {
				return []slog.Attr{slog.String("userID", v.(string))}
			}
			return nil
		}),
	)

	logger := slog.New(handler)

	ctx := context.WithValue(context.Background(), userContextKey("userID"), "user-123")
	logger.InfoContext(ctx, "Test message")

	assert.Equal(t, len(th.records), 1)

	rec := th.records[0]
	found := false
	rec.Attrs(func(attr slog.Attr) bool {
		if attr.Key == "userID" && attr.Value.String() == "user-123" {
			found = true
			return false
		}
		return true
	})

	assert.Assert(t, found, "userID attribute should be present")
}

func TestContextHandlerMultipleExtractors(t *testing.T) {
	th := &testHandler{}
	handler := NewContextHandler(
		WithBaseHandler(th),
		WithExtractor(func(_ context.Context) []slog.Attr {
			return []slog.Attr{slog.String("attr1", "val1")}
		}),
		WithExtractor(func(_ context.Context) []slog.Attr {
			return []slog.Attr{slog.String("attr2", "val2")}
		}),
	)

	logger := slog.New(handler)
	logger.InfoContext(context.Background(), "Test multiple extractors")

	assert.Equal(t, len(th.records), 1)

	keys := map[string]string{}
	th.records[0].Attrs(func(attr slog.Attr) bool {
		keys[attr.Key] = attr.Value.String()
		return true
	})

	assert.DeepEqual(t, keys, map[string]string{"attr1": "val1", "attr2": "val2"})
}

func TestContextHandlerWithAttrsWithGroup(t *testing.T) {
	th := &testHandler{}
	handler := NewContextHandler(WithBaseHandler(th))

	h1 := handler.WithAttrs([]slog.Attr{slog.String("key", "value")})
	logger1 := slog.New(h1)
	logger1.InfoContext(context.Background(), "Test WithAttrs")
	assert.Equal(t, len(th.records), 1)

	h2 := handler.WithGroup("mygroup")
	logger2 := slog.New(h2)
	logger2.InfoContext(context.Background(), "Test WithGroup")
	assert.Equal(t, len(th.records), 2)
}
