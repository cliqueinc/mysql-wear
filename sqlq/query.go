// Package sqlq deals with building of sql query.
package sqlq

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

// comparison operators
const (
	lt   = "<"
	lte  = "<="
	eq   = "="
	neq  = "!="
	gt   = ">"
	gte  = ">="
	like = "LIKE"
)

// order operators
const (
	ASC  = "ASC"
	DESC = "DESC"
)

// these flags describe whether query option is allowed for a specific db operation.
const (
	OpSelect = "select"
	OpInsert = "insert"
	OpUpdate = "update"
	OpDelete = "delete"
)

// these flags describe whether query option is allowed for a specific db operation.
const (
	typeQuery = iota + 1
	typePagination
	typeOrder
	typeHaving
	typeColumns
	typeJoin
	// typeQueryAll enforses quering all data (like where 1=1),
	// used to prevent unintentional update or delete of all rows in table
	typeQueryAll
)

// DefaultSelectLimit sets the default limit for select if no limit specified.
const DefaultSelectLimit = 1000

// Query keeps query data during it's building.
type Query struct {
	limit, offset int
	order         []string
	group         []string
	queryType     string

	Args       []interface{}
	Columns    []string
	Query      string
	Having     string
	IsQueryAll bool
	Joins      []JoinConfig
}

// JoinConfig describes join config.
type JoinConfig struct {
	Condition string
	StructPtr interface{}
	Columns   []string
	TableName string
}

// Option describes common function for building query.
type Option func(q *Query) (query string, queryType int, err error)

// Equal adds where field = value construction to query.
func Equal(field string, value interface{}) Option {
	return where(field, eq, value)
}

// NotEqual adds where field != value construction to query.
func NotEqual(field string, value interface{}) Option {
	return where(field, neq, value)
}

// LessThan adds where field < value construction to query.
func LessThan(field string, value interface{}) Option {
	return where(field, lt, value)
}

// LessOrEqual adds where field <= value construction to query.
func LessOrEqual(field string, value interface{}) Option {
	return where(field, lte, value)
}

// GreaterThan adds where field > value construction to query.
func GreaterThan(field string, value interface{}) Option {
	return where(field, gt, value)
}

// GreaterOrEqual adds where field >= value construction to query.
func GreaterOrEqual(field string, value interface{}) Option {
	return where(field, gte, value)
}

// Like adds where field LIKE pattern construction to query.
func Like(field string, pattern string) Option {
	return where(field, like, pattern)
}

// where adds where construction to query, supporting comparison operators.
// See comparison operators in mw const as a samples.
func where(field string, cmp string, value interface{}) Option {
	return func(q *Query) (string, int, error) {
		if field == "" {
			return "", 0, errors.New("field cannot be empty")
		}

		q.Args = append(q.Args, value)
		if !strings.Contains(field, "(") && !(strings.HasPrefix(field, "`") && strings.HasSuffix(field, "`")) {
			field = "`" + field + "`"
		}

		return fmt.Sprintf("%s %s ?", field, string(cmp)), typeQuery, nil
	}
}

// Raw adds raw where query. Arguments in query expected to be marked as '?'.
// Example: query = "name = ? or status = ?", args = ["John", "active"]
func Raw(query string, args ...interface{}) Option {
	return func(q *Query) (string, int, error) {
		if query == "" {
			return "", 0, errors.New("query cannot be empty")
		}

		buf := bytes.NewBuffer(make([]byte, 0, len(query)+len(args)))
		for i := range query {
			buf.WriteByte(query[i])
		}
		q.Args = append(q.Args, args...)

		return buf.String(), typeQuery, nil
	}
}

// All enforses quering all data (like where 1=1)
// used to prevent unintentional update or delete of all rows in table.
func All() Option {
	return func(q *Query) (string, int, error) {
		return "", typeQueryAll, nil
	}
}

// IN adds IN construction to query.
func IN(field string, values ...string) Option {
	return func(q *Query) (string, int, error) {
		if field == "" {
			return "", 0, errors.New("field cannot be empty")
		}
		if len(values) == 0 {
			return "", 0, errors.New("IN values cannot be empty")
		}

		queryArgs := make([]string, 0, len(values))
		for _, val := range values {
			queryArgs = append(queryArgs, "?")
			q.Args = append(q.Args, val)
		}

		return fmt.Sprintf("`%s` IN ("+strings.Join(queryArgs, ",")+")", field), typeQuery, nil
	}
}

// OR combines multiple where options with OR condition.
func OR(opts ...Option) Option {
	return func(q *Query) (string, int, error) {
		queries := make([]string, 0, len(opts))
		for _, opt := range opts {
			optQuery, optType, err := opt(q)
			if err != nil {
				return "", 0, err
			}
			// not where option
			if optType != typeQuery {
				return "", 0, fmt.Errorf("cannot pass not a search query to OR confition")
			}

			queries = append(queries, optQuery)
		}
		if len(queries) == 0 {
			return "", 0, nil
		}
		return "(" + strings.Join(queries, " OR ") + ")", typeQuery, nil
	}
}

// AND combines multiple where options with AND condition.
func AND(opts ...Option) Option {
	return func(q *Query) (string, int, error) {
		queries := make([]string, 0, len(opts))
		for _, opt := range opts {
			optQuery, optType, err := opt(q)
			if err != nil {
				return "", 0, err
			}
			// not where option
			if optType != typeQuery {
				return "", 0, fmt.Errorf("cannot pass not a search query to AND condition")
			}

			queries = append(queries, optQuery)
		}
		if len(queries) == 0 {
			return "", 0, nil
		}
		return "(" + strings.Join(queries, " AND ") + ")", typeQuery, nil
	}
}

// Having adds having clause to select.
func Having(opts ...Option) Option {
	return func(q *Query) (string, int, error) {
		if q.queryType != OpSelect {
			return "", 0, fmt.Errorf("cannot use having in (%s)", q.queryType)
		}
		optQuery, _, err := AND(opts...)(q)
		if err != nil {
			return "", 0, err
		}

		q.Having = optQuery
		return "", typeHaving, nil
	}
}

// GroupBy adds group by construction to query. Example: mw.GroupBy("user_id"), or mw.GroupBy("user_id", "price")
func GroupBy(columns ...string) Option {
	return func(q *Query) (string, int, error) {
		if q.queryType != OpSelect {
			return "", 0, fmt.Errorf("cannot use group by in (%s)", q.queryType)
		}
		if len(columns) == 0 {
			return "", 0, fmt.Errorf("no columns specified for group by")
		}

		q.group = columns
		return "", typeOrder, nil
	}
}

// Limit adds limit to query. If multiple limits specified, the last one will be set.
func Limit(limit int) Option {
	return func(q *Query) (string, int, error) {
		if q.queryType != OpSelect {
			return "", 0, fmt.Errorf("cannot use limit in (%s)", q.queryType)
		}
		if limit < 0 {
			return "", 0, errors.New("limit cannot be less than 0")
		}

		q.limit = limit
		return "", typePagination, nil
	}
}

// Offset adds limit offset to query. If multiple offsets specified, the last one will be set.
func Offset(offset int) Option {
	return func(q *Query) (string, int, error) {
		if q.queryType != OpSelect {
			return "", 0, fmt.Errorf("cannot use offset in (%s)", q.queryType)
		}
		if offset < 0 {
			return "", 0, errors.New("offset cannot be less than 0")
		}

		q.offset = offset
		return "", typePagination, nil
	}
}

// Order adds order to query. If multiple orders specified, each will be added to query.
// For example mw.Order("id", sqlq.ASC), mw.Order("updated", mwq.DESC) will produce ORDER BY "id" ASC, "updated" DESC.
func Order(field string, orderBy string) Option {
	return func(q *Query) (string, int, error) {
		if q.queryType != OpSelect {
			return "", 0, fmt.Errorf("cannot use order in (%s)", q.queryType)
		}
		if strings.ToLower(orderBy) != "asc" && strings.ToLower(orderBy) != "desc" {
			return "", 0, fmt.Errorf("unknown order %s", orderBy)
		}
		if !strings.HasPrefix(field, "`") {
			field = "`" + field + "`"
		}

		q.order = append(q.order, fmt.Sprintf("%s %s", field, orderBy))
		return "", typeOrder, nil
	}
}

// Columns specifies columns that needs to be fetched. By default all columns are fetched.
func Columns(columns ...string) Option {
	return func(q *Query) (string, int, error) {
		if q.queryType != OpSelect {
			return "", 0, fmt.Errorf("cannot use columns in (%s)", q.queryType)
		}

		q.Columns = columns
		return "", typeColumns, nil
	}
}

// Join adds join to query.
/*
	1. define joined struct
	2. search for field with mw:"join" tag
	3. compare struct type, if same, add field address to fields slice
	4. when scan, check whether model row already exist, if yes, try to join rows
	5. TODO: how to join rows with multiple join case?????????????????????
	6. probably duplicate address to keep raw data in order to easy compare it
*/
func Join(structPtr interface{}, condition string, columns ...string) Option {
	return func(q *Query) (string, int, error) {
		if q.queryType != OpSelect {
			return "", 0, fmt.Errorf("cannot use join in (%s)", q.queryType)
		}
		if structPtr == nil {
			return "", 0, errors.New("struct pointer cannot be nil")
		}
		if condition == "" {
			return "", 0, errors.New("join condition cannot be empty")
		}

		q.Joins = append(q.Joins, JoinConfig{
			Columns:   columns,
			Condition: condition,
			StructPtr: structPtr,
		})
		return "", typeJoin, nil
	}
}

// Build builds sql query from given query option
func Build(opts []Option, queryType string, existingArgs ...interface{}) (*Query, error) {
	var (
		stmt       = new(Query)
		whereOpts  []string
		isQueryAll bool
	)
	stmt.queryType = queryType
	stmt.Args = existingArgs

	for _, opt := range opts {
		optQuery, optType, err := opt(stmt)
		if err != nil {
			return nil, err
		}
		if optType == typeQueryAll {
			isQueryAll = true
			continue
		}
		if optType != typeQuery {
			continue
		}

		if whereOpts == nil {
			whereOpts = make([]string, 0, 4)
		}
		whereOpts = append(whereOpts, optQuery)
	}

	var query string
	if len(whereOpts) != 0 {
		query = "WHERE " + strings.Join(whereOpts, " AND ")
	}
	if len(stmt.group) != 0 {
		query += " GROUP BY " + strings.Join(stmt.group, ",") + " "
	}
	if stmt.Having != "" {
		query += " HAVING " + stmt.Having
	}
	if len(stmt.order) != 0 {
		query += " ORDER BY " + strings.Join(stmt.order, ", ")
	}
	if stmt.limit == 0 && queryType == OpSelect && !isQueryAll {
		stmt.limit = DefaultSelectLimit
	}
	if stmt.limit != 0 {
		query += fmt.Sprintf(" LIMIT %d", stmt.limit)
	}
	if stmt.offset != 0 {
		query += fmt.Sprintf(" OFFSET %d", stmt.offset)
	}

	stmt.Query = query
	stmt.IsQueryAll = isQueryAll

	return stmt, nil
}
