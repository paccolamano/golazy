package recover

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

type mockLogger struct {
	entries []string
	level   slog.Level
}

func (m *mockLogger) LogAttrs(_ context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	m.level = level
	sb := &strings.Builder{}
	sb.WriteString(msg)
	for _, a := range attrs {
		sb.WriteString(" ")
		sb.WriteString(a.String())
	}
	m.entries = append(m.entries, sb.String())
}

type failingWriter struct {
	header http.Header
}

func (fw *failingWriter) Header() http.Header {
	return fw.header
}

func (fw *failingWriter) WriteHeader(_ int) {}

func (fw *failingWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestDefaultRecoveryWritesJSON500(t *testing.T) {
	logger := &mockLogger{}
	h := New(WithLogger(logger))(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("boom")
	}))

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)

	var body map[string]string
	err := json.NewDecoder(rr.Body).Decode(&body)
	assert.NilError(t, err)
	assert.Equal(t, "Internal Server Error", body["error"])

	assert.Assert(t, len(logger.entries) > 0)
	assert.Equal(t, logger.level, slog.LevelError)
	assert.Assert(t, strings.Contains(logger.entries[0], "boom"))
}

func TestRecoveryWithCustomCallback(t *testing.T) {
	logger := &mockLogger{}
	var called bool

	cb := func(w http.ResponseWriter, _ *http.Request, _ any, _ []byte) {
		called = true
		w.WriteHeader(http.StatusTeapot)
		_, err := w.Write([]byte("custom response"))
		if err != nil {
			t.FailNow()
		}
	}

	h := New(
		WithLogger(logger),
		WithCallback(cb),
	)(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic(errors.New("kaboom"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Assert(t, called)
	assert.Equal(t, http.StatusTeapot, rr.Code)
	assert.Equal(t, "custom response", rr.Body.String())
	assert.Assert(t, strings.Contains(logger.entries[0], "kaboom"))
}

func TestRecoveryWithIncludeStack(t *testing.T) {
	logger := &mockLogger{}
	h := New(
		WithLogger(logger),
		WithIncludeStack(true),
	)(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("with stack")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Assert(t, strings.Contains(logger.entries[0], "with stack"))
	assert.Assert(t, strings.Contains(logger.entries[0], "goroutine"))
}

func TestRecoveryWithCustomStatusCode(t *testing.T) {
	logger := &mockLogger{}
	h := New(
		WithLogger(logger),
		WithStatusCode(http.StatusBadGateway),
	)(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("bad gateway panic")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadGateway, rr.Code)
	var body map[string]string
	_ = json.NewDecoder(rr.Body).Decode(&body)
	assert.Equal(t, "Bad Gateway", body["error"])
}

func TestRecoveryWithCustomMessageAndLevel(t *testing.T) {
	logger := &mockLogger{}
	h := New(
		WithLogger(logger),
		WithMessage("panic occurred"),
		WithLogLevel(slog.LevelWarn),
	)(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("oops")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Assert(t, strings.Contains(logger.entries[0], "panic occurred"))
	assert.Equal(t, logger.level, slog.LevelWarn)
}

func TestNoPanicPassesThrough(t *testing.T) {
	logger := &mockLogger{}
	h := New(WithLogger(logger))(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("ok"))
		if err != nil {
			t.FailNow()
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "ok", rr.Body.String())
	assert.Equal(t, 0, len(logger.entries))
}

func TestDefaultCallbackJSONEncodingError(t *testing.T) {
	// Break JSON encoding by replacing ResponseWriter with one that fails
	logger := &mockLogger{}
	brokenWriter := &failingWriter{header: make(http.Header)}

	h := New(WithLogger(logger))(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("broken writer panic")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(brokenWriter, req)

	assert.Assert(t, len(logger.entries) > 0)
	assert.Assert(t, strings.Contains(logger.entries[len(logger.entries)-1], "failed to send recovery response"))
}
