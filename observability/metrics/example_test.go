package metrics_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/mahdi-awadi/gopkg/observability/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

func Example() {
	e := metrics.New(metrics.DefaultConfig(9090))

	reqs := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "demo_requests_total",
		Help: "Demo counter",
	})
	e.Registry().MustRegister(reqs)
	reqs.Add(7)

	r := httptest.NewRecorder()
	e.Handler().ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	// Output prints part of the scraped metric line.
	fmt.Println(strings.Contains(r.Body.String(), "demo_requests_total 7"))
	// Output: true
}
