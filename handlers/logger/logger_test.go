package logger

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

type mockLogger struct {
	entries []logEntry
}

type logEntry struct {
	level slog.Level
	msg   string
	attrs []slog.Attr
}

func (m *mockLogger) LogAttrs(_ context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	m.entries = append(m.entries, logEntry{level: level, msg: msg, attrs: attrs})
}

func TestNewLogsRequestAndResponse(t *testing.T) {
	logger := &mockLogger{}

	mw := New(
		WithLogger(logger),
		WithFieldsIn(FieldMethod, FieldPath, FieldIP),
		WithFieldsOut(FieldStatus, FieldDuration),
	)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test?x=1", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	w := httptest.NewRecorder()

	mw.ServeHTTP(w, req)

	assert.Equal(t, w.Code, http.StatusCreated)
	assert.Equal(t, len(logger.entries), 2)

	in := logger.entries[0]
	assert.Equal(t, in.msg, "incoming request")
	assert.Equal(t, in.level, slog.LevelInfo)
	assert.Assert(t, hasAttr(in.attrs, "method"))
	assert.Assert(t, hasAttr(in.attrs, "path"))
	assert.Assert(t, hasAttr(in.attrs, "ip"))

	out := logger.entries[1]
	assert.Equal(t, out.msg, "request completed")
	assert.Equal(t, out.level, slog.LevelInfo)
	assert.Assert(t, hasAttr(out.attrs, "status"))
	assert.Assert(t, hasAttr(out.attrs, "duration"))
}

func TestWithSkipPaths(t *testing.T) {
	logger := &mockLogger{}

	mw := New(
		WithLogger(logger),
		WithSkipPaths("/skip"),
	)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/skip/healthz", nil)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	assert.Equal(t, w.Code, http.StatusOK)
	assert.Equal(t, len(logger.entries), 0) // should be skipped
}

func TestWithSkipFunc(t *testing.T) {
	logger := &mockLogger{}

	mw := New(
		WithLogger(logger),
		WithSkipFunc(func(r *http.Request) bool {
			return strings.Contains(r.URL.Path, "internal")
		}),
	)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/internal/api", nil)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	assert.Equal(t, w.Code, http.StatusOK)
	assert.Equal(t, len(logger.entries), 0) // skipped
}

func TestCustomLevels(t *testing.T) {
	logger := &mockLogger{}

	mw := New(
		WithLogger(logger),
		WithRequestInLevel(slog.LevelDebug),
		WithRequestOutLevel(slog.LevelError),
	)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))

	req := httptest.NewRequest(http.MethodPost, "/custom", nil)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	assert.Equal(t, w.Code, http.StatusAccepted)
	assert.Equal(t, len(logger.entries), 2)

	assert.Equal(t, logger.entries[0].level, slog.LevelDebug)
	assert.Equal(t, logger.entries[1].level, slog.LevelError)
}

func TestBuildAttrsAllFields(t *testing.T) {
	rw := &responseWriter{statusCode: 200}
	r := httptest.NewRequest(http.MethodPut, "/all?foo=bar", nil)
	r.RemoteAddr = "192.168.1.1:5555"
	start := time.Now().Add(-time.Second)

	attrs := buildAttrs([]Field{
		FieldMethod,
		FieldPath,
		FieldQuery,
		FieldIP,
		FieldUserAgent,
		FieldContentLength,
		FieldStatus,
		FieldDuration,
	}, r, rw, "192.168.1.1", start)

	expected := []string{"method", "path", "query", "ip", "userAgent", "contentLength", "status", "duration"}
	for _, key := range expected {
		assert.Assert(t, hasAttr(attrs, key), "expected attr %s", key)
	}
}

func TestShouldSkip(t *testing.T) {
	opt := &config{
		SkipPaths: []string{"/skip"},
		SkipFunc: func(r *http.Request) bool {
			return r.URL.Path == "/exact"
		},
	}

	req1 := httptest.NewRequest(http.MethodGet, "/skip/foo", nil)
	assert.Assert(t, shouldSkip(req1, opt))

	req2 := httptest.NewRequest(http.MethodGet, "/exact", nil)
	assert.Assert(t, shouldSkip(req2, opt))

	req3 := httptest.NewRequest(http.MethodGet, "/other", nil)
	assert.Assert(t, !shouldSkip(req3, opt))
}

func hasAttr(attrs []slog.Attr, key string) bool {
	for _, a := range attrs {
		if a.Key == key {
			return true
		}
	}
	return false
}
