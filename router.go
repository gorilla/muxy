package muxy

import (
	"fmt"
	"net/http"
	"net/url"

	"golang.org/x/net/context"
)

// Handler is a context aware version of net/http's Handler interface.
//
// This interface will (or will not) change depending on how context is
// implemented in net/http. See:
//
//     https://github.com/golang/go/issues/13021
type Handler interface {
	ServeHTTP(context.Context, http.ResponseWriter, *http.Request)
}

// HandlerFunc is a context aware version of net/http's HandlerFunc type.
//
// See the docs from Handler for possible signatures changes in the future.
type HandlerFunc func(context.Context, http.ResponseWriter, *http.Request)

// ServeHTTP calls f(ctx, w, r).
func (f HandlerFunc) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	f(ctx, w, r)
}

// -----------------------------------------------------------------------------

// ctxKey is used to set and retrieve route variables from the context.
type ctxKey int

// varsKey is the key used to set and retrieve route variables from the context.
var varsKey ctxKey = 0

// Vars returns the route variables from the given context.
func Vars(ctx context.Context) map[string]string {
	if vars, ok := ctx.Value(varsKey).(map[string]string); ok {
		return vars
	}
	return map[string]string{}
}

// -----------------------------------------------------------------------------

// Matcher registers patterns as routes, matches requests and builds URLs.
type Matcher interface {
	Route(pattern string) (*Route, error)
	Match(r *http.Request) (Handler, map[string]string)
	URL(r *Route, values map[string]string) (*url.URL, error)
}

// -----------------------------------------------------------------------------

// New creates a new Router.
func New() *Router {
	// TODO: options variadic argument (to set matcher, panic handler etc).
	r := &Router{
		matcher:     NewPathMatcher(),
		routes:      map[string]*Route{},
		namedRoutes: map[string]*Route{},
	}
	r.router = r
	return r
}

// Router matches the URL of incoming requests against
// registered routes and calls the appropriate handler.
type Router struct {
	// matcher holds the Matcher implementation used by this router.
	matcher Matcher
	// routes maps all route patterns to their correspondent routes.
	routes map[string]*Route
	// namedRoutes maps route names to their correspondent routes.
	namedRoutes map[string]*Route
	// router holds the main router referenced by subrouters.
	router *Router
	// pattern holds the pattern prefix used to create new routes.
	pattern string
	// name holds the name prefix used to create new routes.
	name string
}

// Sub creates a subrouter for the given pattern prefix.
func (r *Router) Sub(pattern string) *Router {
	return &Router{
		router:  r.router,
		pattern: r.pattern + pattern,
		name:    r.name,
	}
}

// Name sets the name prefix used for new routes.
func (r *Router) Name(name string) *Router {
	r.name = r.name + name
	return r
}

// Mount imports all routes from the given router into this one.
//
// Combined with Sub() and Name(), it is possible to submount a router
// defined in a different package using pattern and name prefixes:
//
//     r := New()
//     s := r.Sub("/admin").Name("admin:").Mount(admin.Router)
func (r *Router) Mount(router *Router) *Router {
	for _, v := range router.router.routes {
		route := r.Route(v.pattern).Name(v.name)
		for method, handler := range v.Handlers {
			route.Handle(handler, method)
		}
	}
	return r
}

// Route creates a new Route for the given pattern.
func (r *Router) Route(pattern string) *Route {
	pattern = r.pattern + pattern
	route, err := r.router.matcher.Route(pattern)
	if err != nil {
		panic(err)
	}
	route.router = r.router
	route.pattern = pattern
	route.name = r.name
	r.router.routes[pattern] = route
	return route
}

// URL returns a URL for the given route name and variables.
func (r *Router) URL(name string, values map[string]string) *url.URL {
	if route, ok := r.router.namedRoutes[name]; ok {
		u, err := r.router.matcher.URL(route, values)
		if err != nil {
			panic(err)
		}
		return u
	}
	return nil
}

// ServeHTTP dispatches to the handler whose pattern matches the request.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if handler, vars := r.router.matcher.Match(req); handler != nil {
		ctx := context.WithValue(context.Background(), varsKey, vars)
		handler.ServeHTTP(ctx, w, req)
		return
	}
	http.NotFound(w, req)
}

// -----------------------------------------------------------------------------

// Route stores a URL pattern to be matched and the handler to be served
// in case of a match, optionally mapping HTTP methods to different handlers.
type Route struct {
	// router holds the router that registered this route.
	router *Router
	// pattern holds the route pattern.
	pattern string
	// name holds the route name.
	name string
	// Handlers maps request methods to the handlers that will handle them.
	Handlers map[string]Handler
}

// Name defines the route name used for URL building.
func (r *Route) Name(name string) *Route {
	r.name = r.name + name
	if _, ok := r.router.namedRoutes[r.name]; ok {
		panic(fmt.Sprintf("mux: duplicated name %q", r.name))
	}
	r.router.namedRoutes[r.name] = r
	return r
}

// Handle sets the given handler to be served for the optional request methods.
func (r *Route) Handle(h Handler, methods ...string) *Route {
	if methods == nil {
		r.Handlers[""] = h
	} else {
		for _, m := range methods {
			r.Handlers[m] = h
		}
	}
	return r
}

// Below are convenience methods that map HTTP verbs to handlers, equivalent
// to call r.Handle(h, "METHOD-NAME").

// Connect sets the given handler to be served for the request method CONNECT.
func (r *Route) Connect(h Handler) *Route {
	return r.Handle(h, "CONNECT")
}

// Delete sets the given handler to be served for the request method DELETE.
func (r *Route) Delete(h Handler) *Route {
	return r.Handle(h, "DELETE")
}

// Get sets the given handler to be served for the request method GET.
func (r *Route) Get(h Handler) *Route {
	return r.Handle(h, "GET")
}

// Head sets the given handler to be served for the request method HEAD.
func (r *Route) Head(h Handler) *Route {
	return r.Handle(h, "HEAD")
}

// Options sets the given handler to be served for the request method OPTIONS.
func (r *Route) Options(h Handler) *Route {
	return r.Handle(h, "OPTIONS")
}

// PATCH sets the given handler to be served for the request method PATCH.
func (r *Route) Patch(h Handler) *Route {
	return r.Handle(h, "PATCH")
}

// POST sets the given handler to be served for the request method POST.
func (r *Route) Post(h Handler) *Route {
	return r.Handle(h, "POST")
}

// Put sets the given handler to be served for the request method PUT.
func (r *Route) Put(h Handler) *Route {
	return r.Handle(h, "PUT")
}

// Trace sets the given handler to be served for the request method TRACE.
func (r *Route) Trace(h Handler) *Route {
	return r.Handle(h, "TRACE")
}
