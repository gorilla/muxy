package muxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

var (
	getDefaultRoute = newRoute(nil, "").
			Get(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "get") }).
			Handle(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "default") })
	postRoute = newRoute(nil, "").
			Post(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "post") })
)

var methodLookupTests = []struct {
	route  *Route
	method string
	body   string
}{
	{
		route:  getDefaultRoute,
		method: "GET",
		body:   "get",
	},
	{
		route:  getDefaultRoute,
		method: "HEAD",
		body:   "get",
	},
	{
		route:  getDefaultRoute,
		method: "PUT",
		body:   "default",
	},
	{
		route:  postRoute,
		method: "POST",
		body:   "post",
	},
	{
		route:  postRoute,
		method: "GET",
		body:   "405 Method Not Allowed\n",
	},
	{
		route:  postRoute,
		method: "HEAD",
		body:   "405 Method Not Allowed\n",
	},
	{
		route:  postRoute,
		method: "OPTIONS",
		body:   "200 OK\n",
	},
}

func TestMethodLookup(t *testing.T) {
	for _, tt := range methodLookupTests {
		w := httptest.NewRecorder()
		tt.route.methodHandler(tt.method)(w, nil)
		if w.Body.String() != tt.body {
			t.Errorf("%s: got body %q, want %q", tt.method, w.Body.String(), tt.body)
		}
	}
}
