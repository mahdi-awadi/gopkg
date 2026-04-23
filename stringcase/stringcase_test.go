package stringcase

import (
	"testing"
)

func TestSnake(t *testing.T) {
	cases := map[string]string{
		"userID":        "user_id",
		"UserID":        "user_id",
		"OrderItem":     "order_item",
		"order-item":    "order_item",
		"order item":    "order_item",
		"getOrderById":  "get_order_by_id",
		"API":           "api",
		"HTTPRequest":   "http_request",
	}
	for in, want := range cases {
		if got := Snake(in); got != want {
			t.Errorf("Snake(%q)=%q, want %q", in, got, want)
		}
	}
}

func TestKebab(t *testing.T) {
	cases := map[string]string{
		"userID":       "user-id",
		"HTTPRequest":  "http-request",
		"order_item":   "order-item",
	}
	for in, want := range cases {
		if got := Kebab(in); got != want {
			t.Errorf("Kebab(%q)=%q, want %q", in, got, want)
		}
	}
}

func TestScreamingSnake(t *testing.T) {
	cases := map[string]string{
		"userID":      "USER_ID",
		"HTTPRequest": "HTTP_REQUEST",
	}
	for in, want := range cases {
		if got := ScreamingSnake(in); got != want {
			t.Errorf("ScreamingSnake(%q)=%q, want %q", in, got, want)
		}
	}
}

func TestCamel(t *testing.T) {
	cases := map[string]string{
		"user_id":       "userId",
		"user-id":       "userId",
		"user id":       "userId",
		"HTTP_REQUEST":  "httpRequest",
		"OrderItem":     "orderItem",
	}
	for in, want := range cases {
		if got := Camel(in); got != want {
			t.Errorf("Camel(%q)=%q, want %q", in, got, want)
		}
	}
}

func TestPascal(t *testing.T) {
	cases := map[string]string{
		"user_id":      "UserId",
		"user-id":      "UserId",
		"HTTP_REQUEST": "HttpRequest",
		"orderItem":    "OrderItem",
	}
	for in, want := range cases {
		if got := Pascal(in); got != want {
			t.Errorf("Pascal(%q)=%q, want %q", in, got, want)
		}
	}
}

func TestSplit_EmptyAndSeparatorOnly(t *testing.T) {
	if got := Split(""); len(got) != 0 {
		t.Fatalf("Split('')=%v, want empty", got)
	}
	if got := Split("___"); len(got) != 0 {
		t.Fatalf("Split('___')=%v, want empty", got)
	}
}
