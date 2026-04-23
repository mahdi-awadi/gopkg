package jsonx

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type user struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func makeReq(body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	return r
}

func TestDecode_OK(t *testing.T) {
	var u user
	if err := Decode(makeReq(`{"name":"alice","age":30}`), &u, DecodeOptions{}); err != nil {
		t.Fatal(err)
	}
	if u.Name != "alice" || u.Age != 30 {
		t.Fatalf("got %+v", u)
	}
}

func TestDecode_TooLarge(t *testing.T) {
	big := strings.Repeat("x", 100)
	body := `{"name":"` + big + `"}`
	var u user
	err := Decode(makeReq(body), &u, DecodeOptions{MaxBodySize: 50})
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("expected ErrTooLarge, got %v", err)
	}
}

func TestDecode_DisallowUnknownFields(t *testing.T) {
	var u user
	err := Decode(makeReq(`{"name":"a","age":1,"extra":"x"}`), &u, DecodeOptions{DisallowUnknownFields: true})
	if err == nil {
		t.Fatal("expected error for unknown field")
	}
}

func TestDecode_AllowsUnknownByDefault(t *testing.T) {
	var u user
	if err := Decode(makeReq(`{"name":"a","age":1,"extra":"x"}`), &u, DecodeOptions{}); err != nil {
		t.Fatalf("default should allow unknown, got %v", err)
	}
}

func TestDecode_TrailingContentRejected(t *testing.T) {
	var u user
	err := Decode(makeReq(`{"name":"a","age":1}{"extra":"y"}`), &u, DecodeOptions{})
	if err == nil {
		t.Fatal("expected error for trailing JSON")
	}
}

func TestWrite_StatusAndContentType(t *testing.T) {
	w := httptest.NewRecorder()
	if err := Write(w, http.StatusAccepted, user{Name: "bob", Age: 25}); err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusAccepted {
		t.Fatalf("status=%d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("content-type=%q", ct)
	}
	var u user
	if err := json.NewDecoder(w.Body).Decode(&u); err != nil {
		t.Fatal(err)
	}
	if u.Name != "bob" {
		t.Fatalf("got %+v", u)
	}
}

func TestError_Shape(t *testing.T) {
	w := httptest.NewRecorder()
	Error(w, http.StatusBadRequest, "bad thing")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", w.Code)
	}
	body, _ := io.ReadAll(w.Body)
	var m map[string]string
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatal(err)
	}
	if m["error"] != "bad thing" {
		t.Fatalf("got %v", m)
	}
}

func TestWrite_DoesNotEscapeHTML(t *testing.T) {
	w := httptest.NewRecorder()
	_ = Write(w, 200, map[string]string{"html": "<b>hi</b>"})
	if !bytes.Contains(w.Body.Bytes(), []byte("<b>hi</b>")) {
		t.Fatalf("expected unescaped HTML, got %s", w.Body.String())
	}
}
