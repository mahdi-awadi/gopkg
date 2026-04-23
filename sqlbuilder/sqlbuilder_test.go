package sqlbuilder

import (
	"strings"
	"testing"
)

func TestWhere_AddsEqualityCondition(t *testing.T) {
	q, args := New("users").Where("id", 42).BuildCountQuery()
	if q != "SELECT COUNT(*) FROM users WHERE id = $1" {
		t.Fatalf("got %q", q)
	}
	if len(args) != 1 || args[0] != 42 {
		t.Fatalf("args = %v", args)
	}
}

func TestWhereNotEmpty_SkipsEmpty(t *testing.T) {
	q, args := New("users").WhereNotEmpty("name", "").BuildCountQuery()
	if strings.Contains(q, "WHERE") {
		t.Fatalf("should have no WHERE: %q", q)
	}
	if len(args) != 0 {
		t.Fatalf("expected no args, got %v", args)
	}
}

func TestWhereNotEmpty_IncludesWhenSet(t *testing.T) {
	q, args := New("users").WhereNotEmpty("name", "alice").BuildCountQuery()
	if q != "SELECT COUNT(*) FROM users WHERE name = $1" {
		t.Fatalf("got %q", q)
	}
	if args[0] != "alice" {
		t.Fatalf("args=%v", args)
	}
}

func TestWhereOp(t *testing.T) {
	q, _ := New("orders").WhereOp("amount", ">=", 100).BuildCountQuery()
	if q != "SELECT COUNT(*) FROM orders WHERE amount >= $1" {
		t.Fatalf("got %q", q)
	}
}

func TestMultipleConditionsChained(t *testing.T) {
	q, args := New("users").
		Where("status", "active").
		WhereOp("created_at", ">=", "2026-01-01").
		BuildCountQuery()
	want := "SELECT COUNT(*) FROM users WHERE status = $1 AND created_at >= $2"
	if q != want {
		t.Fatalf("got %q", q)
	}
	if len(args) != 2 {
		t.Fatalf("got %v", args)
	}
}

func TestWhereMultiLike(t *testing.T) {
	q, args := New("users").
		WhereMultiLike("%al%", "name", "email", "username").
		BuildCountQuery()
	want := "SELECT COUNT(*) FROM users WHERE (name ILIKE $1 OR email ILIKE $1 OR username ILIKE $1)"
	if q != want {
		t.Fatalf("got %q", q)
	}
	if len(args) != 1 || args[0] != "%al%" {
		t.Fatalf("args=%v", args)
	}
}

func TestClone_IndependentAfterMutation(t *testing.T) {
	base := New("users").Where("partner_id", 5).WhereNotEmpty("name", "alice")

	countQ, _ := base.Clone().BuildCountQuery()
	if !strings.Contains(countQ, "partner_id = $1") || !strings.Contains(countQ, "name = $2") {
		t.Fatalf("count has wrong WHERE: %q", countQ)
	}

	listQ, _ := base.OrderBy("created_at DESC").BuildListQuery("", 10, 2)
	if !strings.Contains(listQ, "LIMIT 10 OFFSET 10") {
		t.Fatalf("list has wrong pagination: %q", listQ)
	}
	if !strings.Contains(listQ, "ORDER BY created_at DESC") {
		t.Fatalf("list missing ORDER BY: %q", listQ)
	}
}

func TestBuildListQuery_DefaultLimit(t *testing.T) {
	q, _ := New("users").BuildListQuery("id, name", 0, 0)
	if !strings.Contains(q, "LIMIT 50 OFFSET 0") {
		t.Fatalf("default limit 50 / offset 0 expected: %q", q)
	}
}

func TestWhereBool_Gated(t *testing.T) {
	q1, _ := New("users").WhereBool("is_active", true, false).BuildCountQuery()
	if strings.Contains(q1, "is_active") {
		t.Fatalf("gate false should skip: %q", q1)
	}
	q2, _ := New("users").WhereBool("is_active", true, true).BuildCountQuery()
	if !strings.Contains(q2, "is_active = $1") {
		t.Fatalf("gate true should include: %q", q2)
	}
}
