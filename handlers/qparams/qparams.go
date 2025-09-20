// Package qparams provides utilities to parse, validate, and extract
// structured search requests from HTTP query parameters. It supports
// complex filtering, ordering, pagination, and custom error handling.
//
// The package defines filters with relational operators (eq, ne, gt, lt, etc.),
// groups filters using logical operators (and, or), and allows sorting
// via order clauses. Middleware created with NewSearchHandler injects
// a parsed SearchRequest into the request context.
//
// Example usage:
//
//	package main
//
//	import (
//		"encoding/json"
//		"log"
//		"net/http"
//
//		"github.com/paccolamano/golazy/handlers/qparams"
//	)
//
//	func main() {
//		mux := http.NewServeMux()
//
//		// Set global defaults for all search handlers
//		qparams.SetDefaultQueryParam("s")
//		qparams.SetDefaultLimit(50)
//		qparams.SetDefaultFilterFields("id")
//		qparams.SetDefaultOrderFields("id")
//		qparams.SetDefaultErrorHandler(func(w http.ResponseWriter, _ *http.Request, err error) {
//			http.Error(w, err.Error(), http.StatusBadRequest)
//		})
//
//		// Create a search handler with custom options
//		search := qparams.NewSearchHandler(
//			qparams.WithExtraFilterFields("name", "email"),
//			qparams.WithOrderFields("created_at", "updated_at"),
//			qparams.WithLimit(10),
//		)
//
//		// Wrap your handler with the search handler middleware
//		mux.Handle("/api/v1/users", search(&usersHandler{}))
//
//		log.Fatal(http.ListenAndServe(":8080", mux))
//	}
//
//	// usersHandler demonstrates how to retrieve the parsed SearchRequest
//	type usersHandler struct{}
//
//	func (h usersHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
//		// Retrieve the search request from context
//		s := qparams.GetSearchRequest(r)
//		if s == nil {
//			log.Println("no search provided")
//			w.WriteHeader(http.StatusNoContent)
//			return
//		}
//
//		log.Printf("search request: %+v", *s)
//
//		w.Header().Add("Content-Type", "application/json")
//		err := json.NewEncoder(w).Encode(s)
//		if err != nil {
//			log.Println("failed to send response")
//		}
//	}
package qparams

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

// LogicalOperator defines how multiple filters or filter groups
// are combined in a search query (e.g., with "AND" or "OR").
type LogicalOperator string

// Symbol returns the SQL equivalent keyword for a given LogicalOperator.
// Defaults to "and" if the operator is not recognized.
func (o LogicalOperator) Symbol() string {
	switch o {
	case OrOperator:
		return string(OrOperator)
	default:
		return string(AndOperator)
	}
}

const (
	// AndOperator represents a logical AND between filters or groups.
	AndOperator LogicalOperator = "and"

	// OrOperator represents a logical OR between filters or groups.
	OrOperator LogicalOperator = "or"
)

var logicalOperators = map[LogicalOperator]struct{}{
	AndOperator: {},
	OrOperator:  {},
}

// RelationalOperator defines the set of supported comparison operators
// that can be used in query parameters to filter results.
type RelationalOperator string

// Symbol returns the SQL equivalent symbol for a given RelationalOperator.
// For example, "eq" maps to "=", "lt" maps to "<", "ilike" maps to "ilike", etc.
// If the operator is unknown, it defaults to "=".
func (o RelationalOperator) Symbol() string {
	switch o {
	case NotEqualsOperator:
		return "<>"
	case GreaterThanOperator:
		return ">"
	case GreaterThanEqualsOperator:
		return ">="
	case LowerThanOperator:
		return "<"
	case LowerThanEqualsOperator:
		return "<="
	case LikeOperator:
		return "like"
	case ILikeOperator:
		return "ilike"
	case InOperator:
		return "in"
	default:
		return "="
	}
}

const (
	// EqualsOperator represents equality comparison (=).
	EqualsOperator RelationalOperator = "eq"

	// NotEqualsOperator represents inequality comparison (<>).
	NotEqualsOperator RelationalOperator = "ne"

	// GreaterThanOperator represents greater-than comparison (>).
	GreaterThanOperator RelationalOperator = "gt"

	// GreaterThanEqualsOperator represents greater-than-or-equal comparison (>=).
	GreaterThanEqualsOperator RelationalOperator = "gte"

	// LowerThanOperator represents less-than comparison (<).
	LowerThanOperator RelationalOperator = "lt"

	// LowerThanEqualsOperator represents less-than-or-equal comparison (<=).
	LowerThanEqualsOperator RelationalOperator = "lte"

	// LikeOperator represents a case-sensitive pattern match (LIKE).
	LikeOperator RelationalOperator = "like"

	// ILikeOperator represents a case-insensitive pattern match (ILIKE).
	ILikeOperator RelationalOperator = "ilike"

	// InOperator represents an inclusion check (IN).
	InOperator RelationalOperator = "in"
)

var relationalOperators = map[RelationalOperator]struct{}{
	EqualsOperator:            {},
	NotEqualsOperator:         {},
	GreaterThanOperator:       {},
	GreaterThanEqualsOperator: {},
	LowerThanOperator:         {},
	LowerThanEqualsOperator:   {},
	LikeOperator:              {},
	ILikeOperator:             {},
	InOperator:                {},
}

// Filter represents a single filtering condition in a query.
// It targets a specific field, applies a relational operator,
// and compares against the given value.
//
// Example:
//
//	{ "field": "name", "op": "eq", "value": "Alice" }
type Filter struct {
	// Field is the name of the column or attribute being filtered.
	Field string `json:"field"`

	// Op is the relational operator to apply (e.g., eq, lt, in).
	Op RelationalOperator `json:"op"`

	// Value is the comparison value used with the operator.
	Value string `json:"value"`
}

// FilterGroup represents a collection of filters combined together
// with a logical operator (AND/OR). FilterGroups can be nested,
// enabling the construction of complex, tree-like query conditions.
//
// Example:
//
//	{ "op": "and", "filters": [...], "groups": [...] }
type FilterGroup struct {
	// Op determines how Filters and Groups inside this group are combined.
	// Supported values: "and", "or".
	Op LogicalOperator `json:"op"`

	// Filters is the list of individual filtering conditions in this group.
	Filters []Filter `json:"filters,omitempty"`

	// Groups allows nesting of additional filter groups for more complex queries.
	Groups []FilterGroup `json:"groups,omitempty"`
}

// OrderDirection represents the direction of a sort clause in a query.
// Supported values are "asc" (ascending) and "desc" (descending).
type OrderDirection string

// Symbol returns the SQL keyword corresponding to the order direction.
// Defaults to "asc" if the direction is not explicitly "desc".
func (d OrderDirection) Symbol() string {
	switch d {
	case OrderDesc:
		return string(OrderDesc)
	default:
		return string(OrderAsc)
	}
}

const (
	// OrderAsc sorts results in ascending order (default).
	OrderAsc OrderDirection = "asc"

	// OrderDesc sorts results in descending order.
	OrderDesc OrderDirection = "desc"
)

// OrderClause represents a single ORDER BY clause in a query.
// It specifies the field to sort on and the direction of sorting.
//
// Example:
//
//	{ "field": "created_at", "direction": "desc" }
type OrderClause struct {
	// Field is the column or attribute to sort by.
	Field string `json:"field"`

	// Direction is the order direction ("asc" or "desc").
	Direction OrderDirection `json:"direction"`
}

// SearchRequest represents a structured query definition parsed from request parameters.
// It combines filtering (via FilterGroups), ordering, and pagination options.
//
// This object is typically extracted from query params (e.g. `?q=<json>`)
// and then used to build database queries or other filtering logic.
//
// Example:
//
//	{
//	  "groups": {
//	    "op": "and",
//	    "filters": [
//	      { "field": "status", "op": "eq", "value": "active" }
//	    ],
//	    "groups": [
//	      {
//	        "op": "or",
//	        "filters": [
//	          { "field": "role", "op": "eq", "value": "admin" },
//	          { "field": "role", "op": "eq", "value": "editor" }
//	        ]
//	      }
//	    ]
//	  },
//	  "order_by": [
//	    { "field": "created_at", "direction": "desc" }
//	  ],
//	  "limit": 20,
//	  "offset": 0
//	}
type SearchRequest struct {
	// Groups represents the root filter group, which can contain
	// multiple filters and nested groups combined with logical operators.
	Groups *FilterGroup `json:"groups,omitempty"`

	// OrderBy defines the sorting rules to apply to the result set.
	OrderBy []OrderClause `json:"order_by,omitempty"`

	// Limit restricts the maximum number of items returned.
	// If nil, no explicit limit is applied.
	Limit *int `json:"limit,omitempty"`

	// Offset specifies how many items to skip before starting to return results.
	// Useful for pagination in combination with Limit.
	Offset *int `json:"offset,omitempty"`
}

// contextKey is a custom type used to avoid collisions when
// storing values in request contexts.
type contextKey string

// searchKey is the context key under which parsed SearchRequest
// objects are stored.
const searchKey = contextKey("search")

// ErrorHandler defines the signature of a function responsible
// for handling request errors. It receives the HTTP response writer,
// the request, and the encountered error.
type ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)

var (
	// defaultQueryParam holds the default query parameter name used to
	// retrieve the search payload.
	defaultQueryParam = "q"

	// defaultSearchMandatory indicates whether a search parameter is
	// required by default.
	defaultSearchMandatory = true

	// defaultLogicalOperators defines the default set of logical
	// operators allowed in filters.
	defaultLogicalOperators = logicalOperators

	// defaultRelationalOperators defines the default set of relational
	// operators allowed in filters.
	defaultRelationalOperators = relationalOperators

	// defaultLimit defines the default maximum limit applied to
	// search requests. Nil means "no limit".
	defaultLimit *int

	// defaultErrorHandler is the fallback handler used when no custom
	// error handler is configured. It writes a error response with
	// HTTP 400 status code.
	defaultErrorHandler ErrorHandler = func(w http.ResponseWriter, r *http.Request, _ error) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte(http.StatusText(http.StatusBadRequest)))
		if err != nil {
			slog.Default().ErrorContext(r.Context(), "failed to send response", slog.String("err", err.Error()))
		}
	}

	// defaultFilterFields defines the default set of fields
	// allowed in filters.
	defaultFilterFields = map[string]struct{}{}

	// defaultOrderFields defines the default set of fields
	// allowed in order by.
	defaultOrderFields = map[string]struct{}{}
)

// SetDefaultQueryParam sets the default query parameter name
// used to extract search payloads.
func SetDefaultQueryParam(queryParam string) {
	defaultQueryParam = queryParam
}

// SetDefaultSearchMandatory sets the global default for requiring
// the search query parameter.
func SetDefaultSearchMandatory(isMandatory bool) {
	defaultSearchMandatory = isMandatory
}

// SetDefaultLogicalOperators replaces the default set of allowed
// logical operators with the provided ones.
func SetDefaultLogicalOperators(operators ...LogicalOperator) {
	clear(defaultLogicalOperators)
	for _, v := range operators {
		defaultLogicalOperators[v] = struct{}{}
	}
}

// SetDefaultRelationalOperators replaces the default set of allowed
// relational operators with the provided ones.
func SetDefaultRelationalOperators(operators ...RelationalOperator) {
	clear(defaultRelationalOperators)
	for _, v := range operators {
		defaultRelationalOperators[v] = struct{}{}
	}
}

// SetDefaultLimit sets the global default limit to apply to
// search requests. Negative means "no limit".
func SetDefaultLimit(limit int) {
	if limit < 0 {
		defaultLimit = nil
	} else {
		defaultLimit = &limit
	}
}

// SetDefaultErrorHandler replaces the default global error handler
// with the provided function.
func SetDefaultErrorHandler(handler ErrorHandler) {
	defaultErrorHandler = handler
}

// SetDefaultFilterFields replaces the default set of allowed
// filter fields with the provided ones.
func SetDefaultFilterFields(fields ...string) {
	clear(defaultFilterFields)
	for _, v := range fields {
		defaultFilterFields[v] = struct{}{}
	}
}

// SetDefaultOrderFields replaces the default set of allowed
// order fields with the provided ones.
func SetDefaultOrderFields(fields ...string) {
	clear(defaultOrderFields)
	for _, v := range fields {
		defaultOrderFields[v] = struct{}{}
	}
}

// config stores the configuration for a search handler,
// including query parameter names, validation rules,
// allowed operators, limits, and error handling.
type config struct {
	queryParam                 string
	isSearchMandatory          bool
	allowedLogicalOperators    map[LogicalOperator]struct{}
	allowedRelationalOperators map[RelationalOperator]struct{}
	limit                      *int
	errorHandler               ErrorHandler
	allowedFilterFields        map[string]struct{}
	allowedOrderFields         map[string]struct{}
}

// Option is a functional option type used to configure Options
// when creating a new search handler.
type Option func(*config)

// WithQueryParam sets a custom query parameter name for extracting
// search payloads.
func WithQueryParam(queryParam string) Option {
	return func(c *config) {
		c.queryParam = queryParam
	}
}

// WithSearchMandatory configures whether the query parameter
// containing the search payload is required.
func WithSearchMandatory(isMandatory bool) Option {
	return func(c *config) {
		c.isSearchMandatory = isMandatory
	}
}

// WithLogicalOperators restricts the set of logical operators
// allowed in filters.
func WithLogicalOperators(operators ...LogicalOperator) Option {
	return func(c *config) {
		clear(c.allowedLogicalOperators)
		for _, v := range operators {
			c.allowedLogicalOperators[v] = struct{}{}
		}
	}
}

// WithRelationalOperators restricts the set of relational operators
// allowed in filters.
func WithRelationalOperators(operators ...RelationalOperator) Option {
	return func(c *config) {
		clear(c.allowedRelationalOperators)
		for _, v := range operators {
			c.allowedRelationalOperators[v] = struct{}{}
		}
	}
}

// WithLimit sets a maximum number of results for search requests.
// Negative values mean "no limit".
func WithLimit(limit int) Option {
	return func(c *config) {
		if limit < 0 {
			c.limit = nil
		} else {
			c.limit = &limit
		}
	}
}

// WithErrorHandler overrides the error handler used by the search handler.
func WithErrorHandler(handler ErrorHandler) Option {
	return func(c *config) {
		c.errorHandler = handler
	}
}

// WithFilterFields restricts the fields that can be used
// in filter conditions. It replace the fields set by
// SetDefaultFilterFields.
func WithFilterFields(fields ...string) Option {
	return func(c *config) {
		clear(c.allowedFilterFields)
		for _, v := range fields {
			c.allowedFilterFields[v] = struct{}{}
		}
	}
}

// WithExtraFilterFields restricts the fields that can be used
// in filter conditions.
func WithExtraFilterFields(fields ...string) Option {
	return func(c *config) {
		for _, v := range fields {
			c.allowedFilterFields[v] = struct{}{}
		}
	}
}

// WithOrderFields restricts the fields that can be used
// in order by clauses. It replace the fields set by
// SetDefaultOrderFields.
func WithOrderFields(fields ...string) Option {
	return func(c *config) {
		clear(c.allowedOrderFields)
		for _, v := range fields {
			c.allowedOrderFields[v] = struct{}{}
		}
	}
}

// WithExtraOrderFields restricts the fields that can be used
// in order by clauses.
func WithExtraOrderFields(fields ...string) Option {
	return func(c *config) {
		for _, v := range fields {
			c.allowedOrderFields[v] = struct{}{}
		}
	}
}

// NewSearchHandler creates a middleware that parses, validates,
// and injects a SearchRequest into the request context.
// It can be customized via Option functions, falling back to
// global defaults when not provided.
func NewSearchHandler(opts ...Option) func(http.Handler) http.Handler {
	c := &config{
		queryParam:                 defaultQueryParam,
		isSearchMandatory:          defaultSearchMandatory,
		allowedLogicalOperators:    defaultLogicalOperators,
		allowedRelationalOperators: defaultRelationalOperators,
		limit:                      defaultLimit,
		errorHandler:               defaultErrorHandler,
		allowedFilterFields:        defaultFilterFields,
		allowedOrderFields:         defaultOrderFields,
	}

	for _, opt := range opts {
		opt(c)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s := r.URL.Query().Get(c.queryParam)
			if s == "" {
				if !c.isSearchMandatory {
					next.ServeHTTP(w, r)
					return
				}

				c.errorHandler(w, r, fmt.Errorf("missing %q query parameter", c.queryParam))
				return
			}

			decoder := json.NewDecoder(strings.NewReader(s))
			decoder.DisallowUnknownFields()

			var search SearchRequest
			if err := decoder.Decode(&search); err != nil {
				c.errorHandler(w, r, err)
				return
			}

			if err := validateSearchRequest(&search, c); err != nil {
				c.errorHandler(w, r, err)
				return
			}

			ctx := context.WithValue(r.Context(), searchKey, &search)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func validateSearchRequest(s *SearchRequest, opts *config) error {
	// even though it is optional, if it is less than zero, it returns an error
	if s.Limit != nil && *s.Limit < 0 {
		return errors.New("limit must be null or >= 0")
	}

	if opts.limit != nil {
		if s.Limit == nil {
			return errors.New("limit is mandatory")
		}
		if *s.Limit > *opts.limit {
			return fmt.Errorf("limit must be between 0 and %d", *opts.limit)
		}
	}

	// even though it is optional, if it is less than zero, it returns an error
	if s.Offset != nil && *s.Offset < 0 {
		return errors.New("offset must be null or >= 0")
	}

	for _, o := range s.OrderBy {
		if _, ok := opts.allowedOrderFields[o.Field]; !ok {
			return fmt.Errorf("field %q not allowed in order by", o.Field)
		}
	}

	var validateGroup func(g *FilterGroup) error
	validateGroup = func(g *FilterGroup) error {
		if g == nil {
			return nil
		}

		if _, ok := opts.allowedLogicalOperators[g.Op]; !ok {
			return fmt.Errorf("logical operator %q not allowed", g.Op)
		}

		for _, f := range g.Filters {
			if _, ok := opts.allowedFilterFields[f.Field]; !ok {
				return fmt.Errorf("field %q not allowed in filters", f.Field)
			}

			if _, ok := opts.allowedRelationalOperators[f.Op]; !ok {
				return fmt.Errorf("relational operator %q not allowed for field %q", f.Op, f.Field)
			}
		}

		for _, sg := range g.Groups {
			if err := validateGroup(&sg); err != nil {
				return err
			}
		}

		return nil
	}

	if err := validateGroup(s.Groups); err != nil {
		return err
	}

	return nil
}

// GetSearchRequest retrieves the parsed SearchRequest stored in the
// request context by NewSearchHandler. If no request is stored, it
// returns nil.
func GetSearchRequest(r *http.Request) *SearchRequest {
	v := r.Context().Value(searchKey)
	if v == nil {
		return nil
	}

	s, ok := v.(*SearchRequest)
	if !ok {
		return nil
	}

	return s
}
