package sqlbuilder_test

import (
	"fmt"

	"github.com/mahdi-awadi/gopkg/sqlbuilder"
)

func Example() {
	qb := sqlbuilder.New("users").
		WhereNotEmpty("name", "alice").
		WhereOp("created_at", ">=", "2026-01-01").
		OrderBy("created_at DESC")

	// Derive count + list queries without re-typing filters.
	countQ, countArgs := qb.Clone().BuildCountQuery()
	listQ, listArgs := qb.BuildListQuery("id, name, email", 20, 1)

	fmt.Println(countQ)
	fmt.Println(len(countArgs))
	fmt.Println(listQ)
	fmt.Println(len(listArgs))
	// Output:
	// SELECT COUNT(*) FROM users WHERE name = $1 AND created_at >= $2
	// 2
	// SELECT id, name, email FROM users WHERE name = $1 AND created_at >= $2 ORDER BY created_at DESC LIMIT 20 OFFSET 0
	// 2
}
