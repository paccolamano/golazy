package tracer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"gotest.tools/v3/assert"
)

func TestWithContextKey(t *testing.T) {
	t.Parallel()

	opts := config{}
	f := WithContextKey("reqID")
	f(&opts)

	assert.Equal(t, opts.contextKey, "reqID")
}

func TestWithHeaderKey(t *testing.T) {
	t.Parallel()

	opts := config{}
	f := WithHeaderKey("X-Trace-ID")
	f(&opts)

	assert.Equal(t, opts.headerKey, "X-Trace-ID")
}

func TestNew(t *testing.T) {
	t.Parallel()

	var traceIDInContext string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceIDVal := r.Context().Value("traceID")
		traceIDInContext, _ = traceIDVal.(string)

		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/tracer", nil)
	w := httptest.NewRecorder()

	New(WithContextKey("traceID"))(handler).ServeHTTP(w, req)

	resp := w.Result()
	requestID := resp.Header.Get("X-Trace-ID")

	assert.Assert(t, requestID != "", "X-Trace-ID header should be set")
	_, err := uuid.Parse(requestID)
	assert.NilError(t, err, "X-Trace-ID should be a valid UUID")

	assert.Equal(t, requestID, traceIDInContext, "Trace ID in context should match X-Trace-ID header")
}

func TestGetTraceID(t *testing.T) {
	t.Parallel()

	t.Run("GetTraceID() should return nil due to empty context", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.FailNow()
		}

		s := GetTraceID(req)

		assert.Assert(t, s == nil)
	})

	t.Run("GetTraceID() should return nil due to wrong type in context", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.FailNow()
		}

		ctx := context.WithValue(req.Context(), traceIDKey("traceID"), "not a uuid")
		req = req.WithContext(ctx)
		s := GetTraceID(req)

		assert.Assert(t, s == nil)
	})

	t.Run("GetTraceID() should return trace id", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.FailNow()
		}

		id, err := uuid.NewUUID()
		if err != nil {
			t.FailNow()
		}

		ctx := context.WithValue(req.Context(), traceIDKey("traceID"), id.String())
		req = req.WithContext(ctx)
		traceID := GetTraceID(req)

		assert.Equal(t, traceID.String(), id.String())
	})
}

func TestGetTraceIDWithKey(t *testing.T) {
	t.Parallel()

	t.Run("GetTraceIDWithKey() should return nil due to empty context", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.FailNow()
		}

		s := GetTraceIDWithKey(req, "traceID")

		assert.Assert(t, s == nil)
	})

	t.Run("GetTraceIDWithKey() should return nil due to wrong type in context", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.FailNow()
		}

		type newTraceIDKey string

		ctx := context.WithValue(req.Context(), newTraceIDKey("trace"), "not a uuid")
		req = req.WithContext(ctx)
		s := GetTraceIDWithKey(req, "traceID")

		assert.Assert(t, s == nil)
	})

	t.Run("GetTraceIDWithKey() should return trace id", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.FailNow()
		}

		id, err := uuid.NewUUID()
		if err != nil {
			t.FailNow()
		}

		type newTraceIDKey string

		ctx := context.WithValue(req.Context(), newTraceIDKey("trace"), id.String())
		req = req.WithContext(ctx)
		traceID := GetTraceIDWithKey(req, newTraceIDKey("trace"))

		assert.Equal(t, traceID.String(), id.String())
	})
}
