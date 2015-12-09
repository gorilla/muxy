package muxy

import (
	"fmt"
	"net/http"
	"net/url"

	"golang.org/x/net/context"
)

// Handler is a context-aware version of net/http.Handler.
type Handler interface {
	ServeHTTPC(context.Context, http.ResponseWriter, *http.Request)
}

// HandlerFunc is an adapter to allow the use of ordinary functions as Handler.
type HandlerFunc func(context.Context, http.ResponseWriter, *http.Request)

// ServeHTTP implements net/http.Handler. It calls h(context.TODO(), w, r).
func (h HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h(context.TODO(), w, r)
}

// ServeHTTPC implements Handler. It calls h(c, w, r).
func (h HandlerFunc) ServeHTTPC(c context.Context, w http.ResponseWriter, r *http.Request) {
	h(c, w, r)
}

// -----------------------------------------------------------------------------

// Matcher registers patterns as routes, matches requests and builds URLs.
type Matcher interface {
	Route(pattern string) (*Route, error)
	Match(c context.Context, r *http.Request) (context.Context, Handler)
	URL(r *Route, values map[string]string) (*url.URL, error)
}

// -----------------------------------------------------------------------------

// New creates a new Router for the given matcher.
func New(m Matcher) *Router {
	r := &Router{
		matcher:     m,
		routes:      map[*Route]string{},
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
	// routes maps all routes to their correspondent patterns.
	routes map[*Route]string
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
//     r := muxy.New()
//     s := r.Sub("/admin").Name("admin:").Mount(admin.Router)
func (r *Router) Mount(router *Router) *Router {
	for v, _ := range router.router.routes {
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
	r.router.routes[route] = pattern
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
	r.ServeHTTPC(context.TODO(), w, req)
}

// ServeHTTPC is a context-aware version of ServeHTTP.
//
// It can be used by middleware to add extra context data before routing begins.
func (r *Router) ServeHTTPC(c context.Context, w http.ResponseWriter, req *http.Request) {
	if c, h := r.router.matcher.Match(c, req); h != nil {
		h.ServeHTTPC(c, w, req)
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
		panic(fmt.Sprintf("muxy: duplicated name %q", r.name))
	}
	r.router.namedRoutes[r.name] = r
	return r
}

// Handle sets the given handler to be served for the optional request methods.
func (r *Route) Handle(h Handler, methods ...string) *Route {
	if r.Handlers == nil {
		r.Handlers = make(map[string]Handler, len(methods))
	}
	if methods == nil {
		r.Handlers[""] = h
	} else {
		for _, m := range methods {
			r.Handlers[m] = h
		}
	}
	return r
}

// Below are convenience methods that map HTTP verbs to Handler, equivalent
// to call r.Handle(muxy.HandlerFunc(f), "METHOD-NAME").

// Connect sets the given function to be served for the request method CONNECT.
func (r *Route) Connect(f func(context.Context, http.ResponseWriter, *http.Request)) *Route {
	return r.Handle(HandlerFunc(f), "CONNECT")
}

// Delete sets the given function to be served for the request method DELETE.
func (r *Route) Delete(f func(context.Context, http.ResponseWriter, *http.Request)) *Route {
	return r.Handle(HandlerFunc(f), "DELETE")
}

// Get sets the given function to be served for the request method GET.
func (r *Route) Get(f func(context.Context, http.ResponseWriter, *http.Request)) *Route {
	return r.Handle(HandlerFunc(f), "GET")
}

// Head sets the given function to be served for the request method HEAD.
func (r *Route) Head(f func(context.Context, http.ResponseWriter, *http.Request)) *Route {
	return r.Handle(HandlerFunc(f), "HEAD")
}

// Options sets the given function to be served for the request method OPTIONS.
func (r *Route) Options(f func(context.Context, http.ResponseWriter, *http.Request)) *Route {
	return r.Handle(HandlerFunc(f), "OPTIONS")
}

// PATCH sets the given function to be served for the request method PATCH.
func (r *Route) Patch(f func(context.Context, http.ResponseWriter, *http.Request)) *Route {
	return r.Handle(HandlerFunc(f), "PATCH")
}

// POST sets the given function to be served for the request method POST.
func (r *Route) Post(f func(context.Context, http.ResponseWriter, *http.Request)) *Route {
	return r.Handle(HandlerFunc(f), "POST")
}

// Put sets the given function to be served for the request method PUT.
func (r *Route) Put(f func(context.Context, http.ResponseWriter, *http.Request)) *Route {
	return r.Handle(HandlerFunc(f), "PUT")
}

// Trace sets the given function to be served for the request method TRACE.
func (r *Route) Trace(f func(context.Context, http.ResponseWriter, *http.Request)) *Route {
	return r.Handle(HandlerFunc(f), "TRACE")
}

// -----------------------------------------------------------------------------

// Variable is a type used to set and retrieve route variables from the context.
type Variable string

// Var returns the route variable with the given name from the context.
//
// The returned value may be empty if the variable was never set.
func Var(c context.Context, name string) string {
	s, _ := c.Value(Variable(name)).(string)
	return s
}
