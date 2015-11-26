package muxy

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// New creates a new Router.
func New() *Router {
	r := &Router{
		matcher:         newPathMatcher(),
		namedRoutes:     map[string]*Route{},
		NotFoundHandler: http.NotFound,
	}
	r.router = r
	return r
}

// Router matches the URL of incoming requests against
// registered routes and calls the appropriate handler.
type Router struct {
	router          *Router
	pattern         string
	name            string
	middleware      []func(http.Handler) http.Handler
	matcher         matcher
	namedRoutes     map[string]*Route
	NotFoundHandler func(http.ResponseWriter, *http.Request)
	// ...
}

// Name sets the name prefix used for new routes.
func (r *Router) Name(name string) *Router {
	r.name = r.name + name
	return r
}

// Use sets the middleware used for new routes.
func (r *Router) Use(middleware ...func(http.Handler) http.Handler) *Router {
	r.middleware = append(r.middleware, middleware...)
	return r
}

// Route creates a new Route for the given pattern.
func (r *Router) Route(pattern string) *Route {
	route := r.router.route(r.pattern + pattern)
	route.NamePrefix = r.name
	route.Middleware = r.middleware
	return route
}

// route creates a new Route for the given pattern.
func (r *Router) route(pattern string) *Route {
	m, pm := newPattern(r.matcher, pattern)
	r.matcher = m
	if pm.leaf == nil {
		pm.leaf = newRoute(r, pattern)
	}
	if pm.leaf.Pattern != pattern {
		panic(fmt.Sprintf("mux: pattern %q has a registered equivalent %q", pattern, pm.leaf.Pattern))
	}
	return pm.leaf
}

// Sub creates a subrouter for the given pattern prefix.
func (r *Router) Sub(pattern string) *Router {
	return &Router{
		router:     r.router,
		pattern:    r.pattern + pattern,
		name:       r.name,
		middleware: r.middleware,
	}
}

// Vars returns the matched variables for the given request.
func (r *Router) Vars(req *http.Request) url.Values {
	// oooh... what a muxy trick!
	if route := r.router.match(req); route != nil {
		return route.vars(req)
	}
	return nil
}

// URL returns a URL segment for the given route name and variables.
func (r *Router) URL(name string, vars url.Values) string {
	if route, ok := r.router.namedRoutes[name]; ok {
		return route.url(vars)
	}
	return ""
}

// ServeHTTP dispatches to the handler whose pattern matches the request.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// not implemented...
	if route := r.router.match(req); route != nil {
		route.handler(req).ServeHTTP(w, req)
		return
	}
	r.NotFoundHandler(w, req)
}

// match returns the matched route for the given request.
func (r *Router) match(req *http.Request) *Route {
	u := req.URL
	if pm := r.matcher.match(u.Scheme, u.Host, u.Path[1:]); pm != nil {
		return pm.leaf
	}
	return nil
}

// -----------------------------------------------------------------------------

// newRoute creates a new Route.
func newRoute(r *Router, pattern string) *Route {
	return &Route{
		Router:   r,
		Pattern:  pattern,
		Handlers: map[string]func(http.ResponseWriter, *http.Request){},
	}
}

// Route stores a URL pattern to be matched and the handler to be served
// in case of a match, optionally mapping HTTP methods to different handlers.
type Route struct {
	Router     *Router
	Pattern    string
	NamePrefix string
	Middleware []func(http.Handler) http.Handler
	Handlers   map[string]func(http.ResponseWriter, *http.Request)
}

// Name defines the route name used for URL building.
func (r *Route) Name(name string) *Route {
	name = r.NamePrefix + name
	if _, ok := r.Router.namedRoutes[name]; ok {
		panic(fmt.Sprintf("mux: duplicated name %q", name))
	}
	r.Router.namedRoutes[name] = r
	return r
}

// Handle sets the given handler to be served for the optional request methods.
func (r *Route) Handle(handler func(http.ResponseWriter, *http.Request), methods ...string) *Route {
	if methods == nil {
		r.Handlers[""] = handler
	} else {
		for _, m := range methods {
			r.Handlers[m] = handler
		}
	}
	return r
}

// Below are convenience methods that map HTTP verbs to handlers, equivalent
// to call r.Handle(handler, "METHOD-NAME").

// Connect sets the given handler to be served for the request method CONNECT.
func (r *Route) Connect(handler func(http.ResponseWriter, *http.Request)) *Route {
	return r.Handle(handler, "CONNECT")
}

// Delete sets the given handler to be served for the request method DELETE.
func (r *Route) Delete(handler func(http.ResponseWriter, *http.Request)) *Route {
	return r.Handle(handler, "DELETE")
}

// Get sets the given handler to be served for the request method GET.
func (r *Route) Get(handler func(http.ResponseWriter, *http.Request)) *Route {
	return r.Handle(handler, "GET")
}

// Head sets the given handler to be served for the request method HEAD.
func (r *Route) Head(handler func(http.ResponseWriter, *http.Request)) *Route {
	return r.Handle(handler, "HEAD")
}

// Options sets the given handler to be served for the request method OPTIONS.
func (r *Route) Options(handler func(http.ResponseWriter, *http.Request)) *Route {
	return r.Handle(handler, "OPTIONS")
}

// PATCH sets the given handler to be served for the request method PATCH.
func (r *Route) Patch(handler func(http.ResponseWriter, *http.Request)) *Route {
	return r.Handle(handler, "PATCH")
}

// POST sets the given handler to be served for the request method POST.
func (r *Route) Post(handler func(http.ResponseWriter, *http.Request)) *Route {
	return r.Handle(handler, "POST")
}

// Put sets the given handler to be served for the request method PUT.
func (r *Route) Put(handler func(http.ResponseWriter, *http.Request)) *Route {
	return r.Handle(handler, "PUT")
}

// Trace sets the given handler to be served for the request method TRACE.
func (r *Route) Trace(handler func(http.ResponseWriter, *http.Request)) *Route {
	return r.Handle(handler, "TRACE")
}

// handler returns a handler for the request, applying registered middleware.
func (r *Route) handler(req *http.Request) http.Handler {
	var h http.Handler = http.HandlerFunc(r.methodHandler(req.Method))
	for i := len(r.Middleware) - 1; i >= 0; i-- {
		h = r.Middleware[i](h)
	}
	return h
}

// methodHandler returns the handler registered for the given HTTP method.
func (r *Route) methodHandler(method string) func(http.ResponseWriter, *http.Request) {
	if h, ok := r.Handlers[method]; ok {
		return h
	}
	switch method {
	case "OPTIONS":
		return r.allowHandler(200)
	case "HEAD":
		if h, ok := r.Handlers["GET"]; ok {
			return h
		}
		fallthrough
	default:
		if h, ok := r.Handlers[""]; ok {
			return h
		}
	}
	return r.allowHandler(405)
}

// allowHandler returns a handler that sets a header with the given
// status code and allowed methods.
func (r *Route) allowHandler(code int) func(http.ResponseWriter, *http.Request) {
	allowed := []string{"OPTIONS"}
	for m, _ := range r.Handlers {
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

// url returns a URL segment for the given variables.
func (r *Route) url(vars url.Values) string {
	// not implemented...
	return ""
}

// vars returns the matched variables for the given request.
func (r *Route) vars(req *http.Request) url.Values {
	// not implemented...
	return nil
}
