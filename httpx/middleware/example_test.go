package middleware_test

import (
	"fmt"
	"net/http"

	"github.com/mahdi-awadi/gopkg/httpx/middleware"
)

func ExampleChain() {
	stack := middleware.Chain(
		middleware.Recover(nil),
		middleware.RequestID(""),
		middleware.Logger(func(e middleware.LogEntry) {
			fmt.Printf("%s %s → %d\n", e.Method, e.Path, e.Status)
		}),
	)
	_ = stack(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
}
