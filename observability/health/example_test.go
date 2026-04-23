package health_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/mahdi-awadi/gopkg/observability/health"
)

func ExampleChecker() {
	c := health.NewChecker("orders", "v1.2.3")
	c.Add("db", func() error { return nil })                       // passing
	c.Add("cache", func() error { return errors.New("connection") }) // failing

	// In a real server: http.Handle("/health", c.Handler())
	r := httptest.NewRecorder()
	c.Handler().ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/health", nil))
	fmt.Println(r.Code) // 503 because one check is failing
	// Output: 503
}
