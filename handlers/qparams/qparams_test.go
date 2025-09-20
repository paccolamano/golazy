package qparams

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/paccolamano/golazy/utility"
	"gotest.tools/v3/assert"
)

func TestLogicalOperatorSymbol(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		operator LogicalOperator
		expected string
	}{
		{
			name:     `Symbol() should return "and"`,
			operator: AndOperator,
			expected: "and",
		},
		{
			name:     `Symbol() should return "or"`,
			operator: OrOperator,
			expected: "or",
		},
		{
			name:     `Given wrong operator, Symbol() should return "and"`,
			operator: LogicalOperator("foo"),
			expected: "and",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.operator.Symbol(), tt.expected)
		})
	}
}

func TestRelationalOperatorSymbol(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		operator RelationalOperator
		expected string
	}{
		{
			name:     `Symbol() should return "="`,
			operator: EqualsOperator,
			expected: "=",
		},
		{
			name:     `Symbol() should return "<>"`,
			operator: NotEqualsOperator,
			expected: "<>",
		},
		{
			name:     `Symbol() should return ">"`,
			operator: GreaterThanOperator,
			expected: ">",
		},
		{
			name:     `Symbol() should return ">="`,
			operator: GreaterThanEqualsOperator,
			expected: ">=",
		},
		{
			name:     `Symbol() should return "<"`,
			operator: LowerThanOperator,
			expected: "<",
		},
		{
			name:     `Symbol() should return "<="`,
			operator: LowerThanEqualsOperator,
			expected: "<=",
		},
		{
			name:     `Symbol() should return "like"`,
			operator: LikeOperator,
			expected: "like",
		},
		{
			name:     `Symbol() should return "ilike"`,
			operator: ILikeOperator,
			expected: "ilike",
		},
		{
			name:     `Symbol() should return "in"`,
			operator: InOperator,
			expected: "in",
		},
		{
			name:     `Given wrong operator, Symbol() should return "="`,
			operator: RelationalOperator("foo"),
			expected: "=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.operator.Symbol(), tt.expected)
		})
	}
}

func TestOrderDirectionSymbol(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		direction OrderDirection
		expected  string
	}{
		{
			name:      `Symbol() should return "asc"`,
			direction: OrderAsc,
			expected:  "asc",
		},
		{
			name:      `Symbol() should return "desc"`,
			direction: OrderDesc,
			expected:  "desc",
		},
		{
			name:      `Given wrong direction, Symbol() should return "asc"`,
			direction: OrderDirection("foo"),
			expected:  "asc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.direction.Symbol(), tt.expected)
		})
	}
}

func TestSetDefaultQueryParam(t *testing.T) {
	original := defaultQueryParam
	defer func() {
		defaultQueryParam = original
	}()

	SetDefaultQueryParam("search")
	assert.Equal(t, defaultQueryParam, "search")
}

func TestSetDefaultSearchMandatory(t *testing.T) {
	original := defaultSearchMandatory
	defer func() {
		defaultSearchMandatory = original
	}()

	SetDefaultSearchMandatory(false)
	assert.Equal(t, defaultSearchMandatory, false)
}

func TestSetDefaultLogicalOperators(t *testing.T) {
	original := defaultLogicalOperators
	defer func() {
		defaultLogicalOperators = original
	}()

	SetDefaultLogicalOperators(AndOperator)
	assert.DeepEqual(t, defaultLogicalOperators, map[LogicalOperator]struct{}{AndOperator: {}})
}

func TestSetDefaultRelationalOperators(t *testing.T) {
	original := defaultRelationalOperators
	defer func() {
		defaultRelationalOperators = original
	}()

	SetDefaultRelationalOperators(EqualsOperator, NotEqualsOperator)
	assert.DeepEqual(t, defaultRelationalOperators, map[RelationalOperator]struct{}{EqualsOperator: {}, NotEqualsOperator: {}})
}

func TestSetDefaultLimit(t *testing.T) {
	original := defaultLimit
	defer func() {
		defaultLimit = original
	}()

	SetDefaultLimit(50)
	assert.Equal(t, *defaultLimit, 50)
}

func TestSetDefaultErrorHandler(t *testing.T) {
	original := defaultErrorHandler
	defer func() {
		defaultErrorHandler = original
	}()

	called := false
	errHandler := func(w http.ResponseWriter, _ *http.Request, _ error) {
		called = true
		w.WriteHeader(http.StatusTeapot)
	}

	SetDefaultErrorHandler(errHandler)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	defaultErrorHandler(rr, req, errors.New("something went wrong"))

	assert.Equal(t, called, true)
	assert.Equal(t, rr.Code, http.StatusTeapot)
}

func TestSetDefaultFilterFields(t *testing.T) {
	original := defaultFilterFields
	defer func() {
		defaultFilterFields = original
	}()

	SetDefaultFilterFields("id")

	assert.DeepEqual(t, defaultFilterFields, map[string]struct{}{"id": {}})
}

func TestSetDefaultOrderFields(t *testing.T) {
	original := defaultOrderFields
	defer func() {
		defaultOrderFields = original
	}()

	SetDefaultOrderFields("id")

	assert.DeepEqual(t, defaultOrderFields, map[string]struct{}{"id": {}})
}

func TestWithQueryParam(t *testing.T) {
	t.Parallel()

	opts := config{}
	f := WithQueryParam("search")
	f(&opts)

	assert.Equal(t, opts.queryParam, "search")
}

func TestWithSearchMandatory(t *testing.T) {
	t.Parallel()

	opts := config{}
	f := WithSearchMandatory(false)
	f(&opts)

	assert.Equal(t, opts.isSearchMandatory, false)
}

func TestWithLogicalOperators(t *testing.T) {
	t.Parallel()

	opts := config{
		allowedLogicalOperators: map[LogicalOperator]struct{}{},
	}
	f := WithLogicalOperators(AndOperator)
	f(&opts)

	assert.DeepEqual(t, opts.allowedLogicalOperators, map[LogicalOperator]struct{}{AndOperator: {}})
}

func TestWithRelationalOperators(t *testing.T) {
	t.Parallel()

	opts := config{
		allowedRelationalOperators: map[RelationalOperator]struct{}{},
	}
	f := WithRelationalOperators(EqualsOperator)
	f(&opts)

	assert.DeepEqual(t, opts.allowedRelationalOperators, map[RelationalOperator]struct{}{EqualsOperator: {}})
}

func TestWithLimit(t *testing.T) {
	t.Parallel()

	opts := config{}
	f := WithLimit(50)
	f(&opts)

	assert.Equal(t, *opts.limit, 50)
}

func TestWithErrorHandler(t *testing.T) {
	t.Parallel()

	opts := config{}

	called := false
	errHandler := func(w http.ResponseWriter, _ *http.Request, _ error) {
		called = true
		w.WriteHeader(http.StatusTeapot)
	}

	f := WithErrorHandler(errHandler)
	f(&opts)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	opts.errorHandler(rr, req, errors.New("something went wrong"))

	assert.Equal(t, called, true)
	assert.Equal(t, rr.Code, http.StatusTeapot)
}

func TestWithFilterFields(t *testing.T) {
	t.Parallel()

	opts := config{
		allowedFilterFields: map[string]struct{}{},
	}
	f := WithFilterFields("id")
	f(&opts)

	assert.DeepEqual(t, opts.allowedFilterFields, map[string]struct{}{"id": {}})
}

func TestWithExtraFilterFields(t *testing.T) {
	t.Parallel()

	opts := config{
		allowedFilterFields: map[string]struct{}{"id": {}},
	}
	f := WithExtraFilterFields("name")
	f(&opts)

	assert.DeepEqual(t, opts.allowedFilterFields, map[string]struct{}{"id": {}, "name": {}})
}

func TestWithOrderFields(t *testing.T) {
	t.Parallel()

	opts := config{
		allowedOrderFields: map[string]struct{}{},
	}
	f := WithOrderFields("id")
	f(&opts)

	assert.DeepEqual(t, opts.allowedOrderFields, map[string]struct{}{"id": {}})
}

func TestWithExtraOrderFields(t *testing.T) {
	t.Parallel()

	opts := config{
		allowedOrderFields: map[string]struct{}{"id": {}},
	}
	f := WithExtraOrderFields("name")
	f(&opts)

	assert.DeepEqual(t, opts.allowedOrderFields, map[string]struct{}{"id": {}, "name": {}})
}

func TestNewSearchHandler(t *testing.T) {
	t.Parallel()

	makeRequest := func(t *testing.T, handler http.Handler, url string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, url, nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		return rr
	}

	tests := []struct {
		name    string
		path    string
		handler http.Handler
		check   func(t *testing.T, res *httptest.ResponseRecorder)
	}{
		{
			name: "with mandatory search",
			path: "/search",
			handler: NewSearchHandler()(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				t.Errorf("next handler should not be called when param is missing")
			})),
			check: func(t *testing.T, res *httptest.ResponseRecorder) {
				assert.Equal(t, res.Code, http.StatusBadRequest)
			},
		},
		{
			name: "without mandatory search",
			path: "/search",
			handler: NewSearchHandler(WithSearchMandatory(false))(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})),
			check: func(t *testing.T, res *httptest.ResponseRecorder) {
				assert.Equal(t, res.Code, http.StatusOK)
			},
		},
		{
			name: "with invalid JSON",
			path: "/search?q={notvalidJSON}",
			handler: NewSearchHandler()(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				t.Errorf("next handler should not be called when JSON is invalid")
			})),
			check: func(t *testing.T, res *httptest.ResponseRecorder) {
				assert.Equal(t, res.Code, http.StatusBadRequest)
			},
		},
		{
			name: "with valid request",
			path: `/search?q={"limit":10,"offset":0}`,
			handler: NewSearchHandler()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				req := GetSearchRequest(r)
				if req == nil {
					t.Fatal("expected SearchRequest in context")
				}

				expected := SearchRequest{
					Limit:  utility.Ptr(10),
					Offset: utility.Ptr(0),
				}
				assert.DeepEqual(t, req, utility.Ptr(expected))

				w.WriteHeader(http.StatusOK)
			})),
			check: func(t *testing.T, res *httptest.ResponseRecorder) {
				assert.Equal(t, res.Code, http.StatusOK)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := makeRequest(t, tt.handler, tt.path)
			tt.check(t, rr)
		})
	}
}

func TestValidateSearchRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		search SearchRequest
		opts   config
		check  func(t *testing.T, err error)
	}{
		{
			name: "with negative limit",
			search: SearchRequest{
				Limit: utility.Ptr(-5),
			},
			opts: config{},
			check: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, `limit must be null or >= 0`)
			},
		},
		{
			name: "with null limit when mandatory",
			search: SearchRequest{
				Limit: nil,
			},
			opts: config{
				limit: utility.Ptr(10),
			},
			check: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, `limit is mandatory`)
			},
		},
		{
			name: "with too high limit",
			search: SearchRequest{
				Limit: utility.Ptr(1000),
			},
			opts: config{
				limit: utility.Ptr(10),
			},
			check: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, `limit must be between 0 and 10`)
			},
		},
		{
			name: "with negative offset",
			search: SearchRequest{
				Offset: utility.Ptr(-5),
			},
			opts: config{},
			check: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, `offset must be null or >= 0`)
			},
		},
		{
			name: "with not allowed order field",
			search: SearchRequest{
				OrderBy: []OrderClause{
					{Field: "notAllowedField", Direction: OrderAsc},
				},
			},
			opts: config{
				allowedOrderFields: map[string]struct{}{"name": {}},
			},
			check: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, `field "notAllowedField" not allowed in order by`)
			},
		},
		{
			name: "with not allowed filter field",
			search: SearchRequest{
				Groups: &FilterGroup{
					Op: AndOperator,
					Filters: []Filter{
						{Field: "notAllowedField", Op: EqualsOperator, Value: "foo"},
					},
				},
			},
			opts: config{
				allowedLogicalOperators:    logicalOperators,
				allowedRelationalOperators: relationalOperators,
				allowedFilterFields:        map[string]struct{}{"name": {}},
			},
			check: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, `field "notAllowedField" not allowed in filters`)
			},
		},
		{
			name: "with not allowed relational operator",
			search: SearchRequest{
				Groups: &FilterGroup{
					Op: AndOperator,
					Filters: []Filter{
						{Field: "name", Op: NotEqualsOperator, Value: "foo"},
					},
				},
			},
			opts: config{
				allowedLogicalOperators:    logicalOperators,
				allowedRelationalOperators: map[RelationalOperator]struct{}{EqualsOperator: {}},
				allowedFilterFields:        map[string]struct{}{"name": {}},
			},
			check: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, `relational operator "ne" not allowed for field "name"`)
			},
		},
		{
			name: "with not allowed logical operator",
			search: SearchRequest{
				Groups: &FilterGroup{
					Op: AndOperator,
					Filters: []Filter{
						{Field: "name", Op: NotEqualsOperator, Value: "foo"},
					},
				},
			},
			opts: config{
				allowedLogicalOperators:    map[LogicalOperator]struct{}{OrOperator: {}},
				allowedRelationalOperators: relationalOperators,
				allowedFilterFields:        map[string]struct{}{"name": {}},
			},
			check: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, `logical operator "and" not allowed`)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSearchRequest(&tt.search, &tt.opts)
			tt.check(t, err)
		})
	}
}

func TestGetSearchRequest(t *testing.T) {
	t.Parallel()

	t.Run("GetSearchRequest() should return nil due to empty context", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.FailNow()
		}

		s := GetSearchRequest(req)

		assert.Equal(t, s == nil, true)
	})

	t.Run("GetSearchRequest() should return nil due to wrong type in context", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.FailNow()
		}

		ctx := context.WithValue(req.Context(), searchKey, "not a SearchRequest")
		req = req.WithContext(ctx)
		s := GetSearchRequest(req)

		assert.Equal(t, s == nil, true)
	})

	t.Run("GetSearchRequest() should return search request", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.FailNow()
		}

		limit := 50
		expected := &SearchRequest{Limit: &limit}
		ctx := context.WithValue(req.Context(), searchKey, expected)
		req = req.WithContext(ctx)
		s := GetSearchRequest(req)

		assert.Equal(t, s, expected)
	})
}
