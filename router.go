package muxy

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// New creates a new Router.
func New() *Router {
	// not implemented...
	return &Router{}
}

// Router matches the URL of incoming requests against
// registered routes and calls the appropriate handler.
type Router struct {
	// ...
}

// Route creates a new Route for the given pattern.
func (r *Router) Route(pattern string) *Route {
	// not implemented...
	return newRoute(r, pattern)
}

// Sub creates a subrouter for the given pattern prefix.
func (r *Router) Sub(pattern string) *Subrouter {
	return &Subrouter{router: r, prefix: pattern}
}

// Vars returns the matched variables for the given request.
func (r *Router) Vars(req *http.Request) url.Values {
	// not implemented...
	return nil
}

// URL returns a URL segment for the given route name and variables.
func (r *Router) URL(name string, vars url.Values) string {
	// not implemented...
	return ""
}

// ServeHTTP dispatches to the handler whose pattern matches the request.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// not implemented...
}

// -----------------------------------------------------------------------------

// Subrouter creates routes with a predefined pattern prefix.
type Subrouter struct {
	router *Router
	prefix string
}

// Route creates a new Route for the given pattern.
func (s *Subrouter) Route(pattern string) *Route {
	return s.router.Route(s.prefix + pattern)
}

// Sub creates a subrouter for the given pattern prefix.
func (s *Subrouter) Sub(pattern string) *Subrouter {
	return s.router.Sub(s.prefix + pattern)
}

// -----------------------------------------------------------------------------

// newRoute creates a new Route.
func newRoute(r *Router, pattern string) *Route {
	// not implemented...
	return &Route{router: r, methods: map[string]func(http.ResponseWriter, *http.Request){}}
}

// Route stores a URL pattern to be matched and the handler to be served
// in case of a match, optionally mapping HTTP methods to different handlers.
type Route struct {
	router  *Router
	methods map[string]func(http.ResponseWriter, *http.Request)
	// ...
}

// Name defines the route name used for URL building.
func (r *Route) Name(name string) *Route {
	// not implemented...
	return r
}

// Handler sets the given handler to be served for the optional request methods.
func (r *Route) Handler(handler func(http.ResponseWriter, *http.Request), methods ...string) *Route {
	if methods == nil {
		r.methods[""] = handler
	} else {
		for _, m := range methods {
			r.methods[m] = handler
		}
	}
	return r
}

// Below are convenience methods that map HTTP verbs to handlers, equivalent
// to call r.Handler(handler, "METHOD-NAME").

// Connect sets the given handler to be served for the request method CONNECT.
func (r *Route) Connect(handler func(http.ResponseWriter, *http.Request)) *Route {
	return r.Handler(handler, "CONNECT")
}

// Delete sets the given handler to be served for the request method DELETE.
func (r *Route) Delete(handler func(http.ResponseWriter, *http.Request)) *Route {
	return r.Handler(handler, "DELETE")
}

// Get sets the given handler to be served for the request method GET.
func (r *Route) Get(handler func(http.ResponseWriter, *http.Request)) *Route {
	return r.Handler(handler, "GET")
}

// Head sets the given handler to be served for the request method HEAD.
func (r *Route) Head(handler func(http.ResponseWriter, *http.Request)) *Route {
	return r.Handler(handler, "HEAD")
}

// Options sets the given handler to be served for the request method OPTIONS.
func (r *Route) Options(handler func(http.ResponseWriter, *http.Request)) *Route {
	return r.Handler(handler, "OPTIONS")
}

// PATCH sets the given handler to be served for the request method PATCH.
func (r *Route) Patch(handler func(http.ResponseWriter, *http.Request)) *Route {
	return r.Handler(handler, "PATCH")
}

// POST sets the given handler to be served for the request method POST.
func (r *Route) Post(handler func(http.ResponseWriter, *http.Request)) *Route {
	return r.Handler(handler, "POST")
}

// Put sets the given handler to be served for the request method PUT.
func (r *Route) Put(handler func(http.ResponseWriter, *http.Request)) *Route {
	return r.Handler(handler, "PUT")
}

// Trace sets the given handler to be served for the request method TRACE.
func (r *Route) Trace(handler func(http.ResponseWriter, *http.Request)) *Route {
	return r.Handler(handler, "TRACE")
}

// methodHandler returns the handler registered for the given HTTP method.
func (r *Route) methodHandler(method string) func(http.ResponseWriter, *http.Request) {
	if h, ok := r.methods[method]; ok {
		return h
	}
	if method == "HEAD" {
		if h, ok := r.methods["GET"]; ok {
			return h
		}
	}
	if h, ok := r.methods[""]; ok {
		return h
	}
	if method == "OPTIONS" {
		return r.allowHandler(200)
	}
	return r.allowHandler(405)
}

// allowHandler returns a handler that sets a header with the given
// status code and allowed methods.
func (r *Route) allowHandler(code int) func(http.ResponseWriter, *http.Request) {
	allowed := []string{"OPTIONS"}
	for m, _ := range r.methods {
		if m != "" && m != "OPTIONS" {
			allowed = append(allowed, m)
		}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Allow", strings.Join(allowed, ", "))
		w.WriteHeader(code)
		fmt.Fprintln(w, code, http.StatusText(code))
	}
}
