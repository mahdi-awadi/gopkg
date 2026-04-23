// Package sqlbuilder is a tiny fluent builder for parameterized Postgres
// SELECT queries with optional filters and pagination.
//
// Designed to eliminate the common duplication between COUNT and LIST
// variants of list/search endpoints: build the filters once, derive both
// queries via Clone.
//
// Zero third-party deps.
package sqlbuilder

import (
	"fmt"
	"strings"
)

// Builder accumulates WHERE conditions and ORDER BY for a single table.
//
// NOT safe for concurrent use. Intended to be built on-demand per request.
type Builder struct {
	table     string
	columns   string
	where     []string
	args      []any
	orderBy   string
	paramSeq  int
}

// New starts a builder targeting a single table.
func New(table string) *Builder {
	return &Builder{table: table, columns: "*"}
}

// Select overrides the default "*" with a custom column list.
func (b *Builder) Select(columns string) *Builder {
	b.columns = columns
	return b
}

// Where appends "col = $N" unconditionally with value v.
func (b *Builder) Where(col string, v any) *Builder {
	b.paramSeq++
	b.where = append(b.where, fmt.Sprintf("%s = $%d", col, b.paramSeq))
	b.args = append(b.args, v)
	return b
}

// WhereOp appends "col <op> $N" (e.g. ">=", "!=").
func (b *Builder) WhereOp(col, op string, v any) *Builder {
	b.paramSeq++
	b.where = append(b.where, fmt.Sprintf("%s %s $%d", col, op, b.paramSeq))
	b.args = append(b.args, v)
	return b
}

// WhereNotEmpty appends only if s is not the empty string.
func (b *Builder) WhereNotEmpty(col, s string) *Builder {
	if s == "" {
		return b
	}
	return b.Where(col, s)
}

// WhereNotNil appends only if v is not nil.
func (b *Builder) WhereNotNil(col string, v any) *Builder {
	if v == nil {
		return b
	}
	return b.Where(col, v)
}

// WhereBool appends "col = $N" with v only when gate is true.
func (b *Builder) WhereBool(col string, v bool, gate bool) *Builder {
	if !gate {
		return b
	}
	return b.Where(col, v)
}

// WhereLike appends "col ILIKE $N" with the value (caller is responsible
// for wildcard placement).
func (b *Builder) WhereLike(col, pattern string) *Builder {
	if pattern == "" {
		return b
	}
	b.paramSeq++
	b.where = append(b.where, fmt.Sprintf("%s ILIKE $%d", col, b.paramSeq))
	b.args = append(b.args, pattern)
	return b
}

// WhereMultiLike appends a grouped OR across multiple columns, binding the
// same value once. Example: WhereMultiLike("%x%", "name", "email").
func (b *Builder) WhereMultiLike(pattern string, cols ...string) *Builder {
	if pattern == "" || len(cols) == 0 {
		return b
	}
	b.paramSeq++
	parts := make([]string, 0, len(cols))
	for _, c := range cols {
		parts = append(parts, fmt.Sprintf("%s ILIKE $%d", c, b.paramSeq))
	}
	b.where = append(b.where, "("+strings.Join(parts, " OR ")+")")
	b.args = append(b.args, pattern)
	return b
}

// OrderBy sets the ORDER BY clause (raw string).
// Caller is responsible for making sure the columns come from a fixed
// whitelist — do NOT pass user input.
func (b *Builder) OrderBy(clause string) *Builder {
	b.orderBy = clause
	return b
}

// Clone returns a deep copy of the builder. Use this to derive a COUNT
// query from a LIST query without duplicating the WHERE chain.
func (b *Builder) Clone() *Builder {
	c := &Builder{
		table:    b.table,
		columns:  b.columns,
		orderBy:  b.orderBy,
		paramSeq: b.paramSeq,
	}
	c.where = append(c.where, b.where...)
	c.args = append(c.args, b.args...)
	return c
}

// BuildCountQuery renders a SELECT COUNT(*) query and returns the SQL
// plus the bound args.
func (b *Builder) BuildCountQuery() (string, []any) {
	q := "SELECT COUNT(*) FROM " + b.table
	if len(b.where) > 0 {
		q += " WHERE " + strings.Join(b.where, " AND ")
	}
	return q, b.args
}

// BuildListQuery renders a SELECT with pagination.
// If columns is empty, the builder's configured columns (default "*") is used.
// page is 1-based; limit must be positive.
func (b *Builder) BuildListQuery(columns string, limit, page int) (string, []any) {
	cols := columns
	if cols == "" {
		cols = b.columns
	}
	if limit <= 0 {
		limit = 50
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	q := "SELECT " + cols + " FROM " + b.table
	if len(b.where) > 0 {
		q += " WHERE " + strings.Join(b.where, " AND ")
	}
	if b.orderBy != "" {
		q += " ORDER BY " + b.orderBy
	}
	q += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)
	return q, b.args
}
