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

var matchTests = []struct {
	pattern string
	url     string
}{
	// scheme
	{
		pattern: "myscheme:",
		url:     "myscheme://www.anything.com/anything",
	},
	// host
	{
		pattern: "//mydomain.com",
		url:     "anything://mydomain.com/anything",
	},
	// path
	{
		pattern: "/a/{b}/{*}",
		url:     "anything://www.anything.com/a/b/c",
	},
	{
		pattern: "/{*}",
		url:     "anything://www.anything.com/a",
	},
	// host + path
	{
		pattern: "//{sub}.domain.com/a/{b}/{*}",
		url:     "anything://my.domain.com/a/b/c",
	},
	// scheme + path
	{
		pattern: "https:///a/{b}/{*}",
		url:     "https://www.anything.com/a/b/c",
	},
	// scheme + host + path
	{
		pattern: "https://{sub}.domain.com/a/{b}/{*}",
		url:     "https://my.domain.com/a/b/c",
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

func TestMatch(t *testing.T) {
	r := New()
	for _, v := range matchTests {
		r.Route(v.pattern)
	}
	for _, v := range matchTests {
		req, _ := http.NewRequest("GET", v.url, nil)
		route := r.match(req)
		if route == nil {
			t.Errorf("%q: expected to match %q", v.url, v.pattern)
		} else if route.pattern != v.pattern {
			t.Errorf("%q: got pattern %q, want %q", v.url, route.pattern, v.pattern)
		}
	}
}
