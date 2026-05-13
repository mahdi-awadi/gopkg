package conversation

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestCache(t *testing.T) (Cache, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return NewRedisCache(client), mr
}

func TestKeyForUser(t *testing.T) {
	if k := KeyForUser("p1", "u1"); k != "ai:conversation:p1:u1" {
		t.Fatalf("unexpected key: %s", k)
	}
}

func TestAppend_AndGet(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()
	key := KeyForUser("p1", "u1")
	if err := c.Append(ctx, key, Message{Role: "user", Channel: "voice", Content: "hello"}); err != nil {
		t.Fatal(err)
	}
	msgs, err := c.Get(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Content != "hello" {
		t.Fatalf("unexpected content: %s", msgs[0].Content)
	}
}

func TestAppend_WindowsTo20(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()
	key := KeyForUser("p1", "u1")
	for i := 0; i < 25; i++ {
		if err := c.Append(ctx, key, Message{Role: "user", Content: "x"}); err != nil {
			t.Fatal(err)
		}
	}
	msgs, _ := c.Get(ctx, key)
	if len(msgs) != ConversationMaxItems {
		t.Fatalf("expected window of %d, got %d", ConversationMaxItems, len(msgs))
	}
}

func TestRecent(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()
	key := KeyForUser("p1", "u1")
	for _, s := range []string{"a", "b", "c", "d", "e"} {
		if err := c.Append(ctx, key, Message{Content: s}); err != nil {
			t.Fatal(err)
		}
	}
	recent, err := c.Recent(ctx, key, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(recent) != 2 || recent[0].Content != "d" || recent[1].Content != "e" {
		t.Fatalf("unexpected recent: %+v", recent)
	}
}

func TestTTL_Honored(t *testing.T) {
	c, mr := newTestCache(t)
	ctx := context.Background()
	key := KeyForUser("p1", "u1")
	if err := c.Append(ctx, key, Message{Content: "x"}); err != nil {
		t.Fatal(err)
	}
	mr.FastForward(ConversationTTL + time.Second)
	msgs, err := c.Get(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 0 {
		t.Fatalf("expected expired, got %d", len(msgs))
	}
}

func TestClear(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()
	key := KeyForUser("p1", "u1")
	_ = c.Append(ctx, key, Message{Content: "x"})
	_ = c.Clear(ctx, key)
	msgs, _ := c.Get(ctx, key)
	if len(msgs) != 0 {
		t.Fatalf("expected empty, got %d", len(msgs))
	}
}

func TestGet_Empty_ReturnsNilNil(t *testing.T) {
	c, _ := newTestCache(t)
	msgs, err := c.Get(context.Background(), KeyForUser("p1", "u1"))
	if err != nil {
		t.Fatalf("expected no error on miss, got %v", err)
	}
	if msgs != nil {
		t.Fatalf("expected nil msgs on miss, got %+v", msgs)
	}
}
