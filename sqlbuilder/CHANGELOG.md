# Changelog

## [0.1.0] - 2026-04-23

### Added
- `Builder` — fluent SQL accumulator for a single table
- `New(table) *Builder`
- Filter methods: `Where`, `WhereOp`, `WhereNotEmpty`, `WhereNotNil`,
  `WhereBool`, `WhereLike`, `WhereMultiLike`
- `Select(columns)` overrides default "*"
- `OrderBy(clause)` (raw — do NOT pass user input)
- `Clone()` returns an independent deep copy
- `BuildCountQuery()` and `BuildListQuery(cols, limit, page)` render
  the final SQL + bound args
- Default limit 50 when caller passes `0`; `page` is 1-based
- 9 tests + 1 runnable example (Output-verified)
- Zero third-party deps
