package muxy

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"
)

// Matcher registers patterns as routes, matches requests and builds URLs.
type Matcher interface {
	Route(pattern string) (*Route, error)
	Match(r *http.Request) (http.Handler, map[string]string)
	URL(r *Route, values map[string]string) (*url.URL, error)
}

// -----------------------------------------------------------------------------

// MatcherOption defines a custom matcher to be used by the router.
// The returned value must be passed as an argument to New():
//
//     r := muxy.New(MatcherOption(CustomMatcher))
//
// It can only be used on New() and panics if the matcher was already set.
func MatcherOption(m Matcher) func(*Router) {
	return func(r *Router) {
		if r.matcher != nil {
			panic("muxy: matcher is already set")
		}
		r.matcher = m
	}
}

// PanicHandlerOption defines the panic handler to be used by the router.
// This handler is called if a panic occurs during ServeHTTP.
// The returned value must be passed as an argument to New():
//
//     r := muxy.New(PanicHandlerOption(CustomHandler))
func PanicHandlerOption(h func(http.ResponseWriter, *http.Request, interface{})) func(*Router) {
	return func(r *Router) {
		r.panicHandler = h
	}
}

// -----------------------------------------------------------------------------

// New creates a new Router.
//
// The optional options arguments can define a custom matcher and/or
// panic handler to be used by the router:
//
//     r := muxy.New(MatcherOption(CustomMatcher), PanicHandlerOption(CustomHandler))
func New(options ...func(*Router)) *Router {
	r := &Router{
		routes:      map[*Route]string{},
		namedRoutes: map[string]*Route{},
	}
	r.router = r
	for _, o := range options {
		o(r)
	}
	if r.matcher == nil {
		r.matcher = NewPathMatcher()
	}
	return r
}

// Router matches the URL of incoming requests against
// registered routes and calls the appropriate handler.
type Router struct {
	// matcher holds the Matcher implementation used by this router.
	matcher Matcher
	// panicHandler holds the handler used in case of panic during ServeHTTP.
	panicHandler func(http.ResponseWriter, *http.Request, interface{})
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
	defer func() {
		if r.panicHandler != nil {
			if v := recover(); v != nil {
				r.panicHandler(w, req, v)
			}
		}
		deleteVars(req)
	}()
	if h, v := r.router.matcher.Match(req); h != nil {
		setVars(req, v)
		h.ServeHTTP(w, req)
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
	Handlers map[string]http.Handler
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
func (r *Route) Handle(h http.Handler, methods ...string) *Route {
	if r.Handlers == nil {
		r.Handlers = map[string]http.Handler{}
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

// Below are convenience methods that map HTTP verbs to http.Handler, equivalent
// to call r.Handle(http.HandlerFunc(h), "METHOD-NAME").

// Connect sets the given handler to be served for the request method CONNECT.
func (r *Route) Connect(h func(http.ResponseWriter, *http.Request)) *Route {
	return r.Handle(http.HandlerFunc(h), "CONNECT")
}

// Delete sets the given handler to be served for the request method DELETE.
func (r *Route) Delete(h func(http.ResponseWriter, *http.Request)) *Route {
	return r.Handle(http.HandlerFunc(h), "DELETE")
}

// Get sets the given handler to be served for the request method GET.
func (r *Route) Get(h func(http.ResponseWriter, *http.Request)) *Route {
	return r.Handle(http.HandlerFunc(h), "GET")
}

// Head sets the given handler to be served for the request method HEAD.
func (r *Route) Head(h func(http.ResponseWriter, *http.Request)) *Route {
	return r.Handle(http.HandlerFunc(h), "HEAD")
}

// Options sets the given handler to be served for the request method OPTIONS.
func (r *Route) Options(h func(http.ResponseWriter, *http.Request)) *Route {
	return r.Handle(http.HandlerFunc(h), "OPTIONS")
}

// PATCH sets the given handler to be served for the request method PATCH.
func (r *Route) Patch(h func(http.ResponseWriter, *http.Request)) *Route {
	return r.Handle(http.HandlerFunc(h), "PATCH")
}

// POST sets the given handler to be served for the request method POST.
func (r *Route) Post(h func(http.ResponseWriter, *http.Request)) *Route {
	return r.Handle(http.HandlerFunc(h), "POST")
}

// Put sets the given handler to be served for the request method PUT.
func (r *Route) Put(h func(http.ResponseWriter, *http.Request)) *Route {
	return r.Handle(http.HandlerFunc(h), "PUT")
}

// Trace sets the given handler to be served for the request method TRACE.
func (r *Route) Trace(h func(http.ResponseWriter, *http.Request)) *Route {
	return r.Handle(http.HandlerFunc(h), "TRACE")
}

// -----------------------------------------------------------------------------

// Until net/context is part of the standard library, we'll use a map/lock
// setup to store the route variables for a request. This is to avoid defining
// our own handler signature, which would not be The Way To Go(tm).
var (
	mutex sync.RWMutex
	vars  = map[*http.Request]map[string]string{}
)

// Vars returns the route variables stored for a given request.
func Vars(r *http.Request) map[string]string {
	mutex.RLock()
	v := vars[r]
	mutex.RUnlock()
	return v
}

// setVars stores the route variables for a given request.
func setVars(r *http.Request, v map[string]string) {
	mutex.Lock()
	vars[r] = v
	mutex.Unlock()
}

// deleteVars deletes the route variables for a given request.
func deleteVars(r *http.Request) {
	mutex.Lock()
	delete(vars, r)
	mutex.Unlock()
}
