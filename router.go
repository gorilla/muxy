package muxy

import (
	"net/http"

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

// Matcher registers patterns as routes and matches requests.
type Matcher interface {
	// Route returns a Route for the given pattern.
	Route(pattern string) (*Route, error)
	// Match matches registered routes against the incoming request,
	// sets URL variables in the context and returns a context and Handler.
	Match(c context.Context, r *http.Request) (context.Context, Handler)
	// Build returns a URL string for the given route and variables.
	Build(r *Route, vars ...string) (string, error)
}

// -----------------------------------------------------------------------------

// Variable is a type used to set and retrieve route variables from the context.
type Variable string

// Var returns the route variable with the given name from the context.
//
// The returned value may be empty if the variable wasn't set.
func Var(c context.Context, name string) string {
	v, _ := c.Value(Variable(name)).(string)
	return v
}

// -----------------------------------------------------------------------------

// New creates a new Router for the given matcher.
func New(m Matcher) *Router {
	r := &Router{
		matcher:     m,
		Routes:      map[*Route]string{},
		NamedRoutes: map[string]*Route{},
	}
	r.Router = r
	return r
}

// Router matches the URL of incoming requests against
// registered routes and calls the appropriate handler.
type Router struct {
	// matcher holds the Matcher implementation used by this router.
	matcher Matcher
	// Router holds the main router referenced by subrouters.
	Router *Router
	// Pattern holds the pattern prefix used to create new routes.
	Pattern string
	// Noun holds the name prefix used to create new routes.
	Noun string
	// Middleware holds the middleware to apply in new routes.
	Middleware []func(Handler) Handler
	// Routes maps all routes to their correspondent patterns.
	Routes map[*Route]string
	// NamedRoutes maps route names to their correspondent routes.
	NamedRoutes map[string]*Route
}

// Use appends the given middleware to this router.
func (r *Router) Use(middleware ...func(Handler) Handler) *Router {
	r.Middleware = append(r.Middleware, middleware...)
	return r
}

// Group creates a group for the given pattern prefix. All routes registered in
// the resulting router will prepend the prefix to its pattern. For example:
//
//     // Create a new router.
//     r := muxy.New(matcher)
//     // Create a group for the routes that share pattern prefix "/admin".
//     g := r.Group("/admin")
//     // Register a route in the admin group, and add handlers for two HTTP
//     // methods. These handlers will be served for the path "/admin/products".
//     g.Route("/products").Get(listProducts).Post(updateProducts)
func (r *Router) Group(pattern string) *Router {
	return &Router{
		Router:     r.Router,
		Pattern:    r.Pattern + pattern,
		Noun:       r.Noun,
		Middleware: r.Middleware,
	}
}

// Name sets the name prefix used for new routes. All routes registered in
// the resulting router will prepend the prefix to its name.
func (r *Router) Name(name string) *Router {
	r.Noun = r.Noun + name
	return r
}

// Mount imports all routes from the given router into this one.
//
// Combined with Group() and Name(), it is possible to submount a router
// defined in a different package using pattern and name prefixes.
// For example:
//
//     // Create a new router.
//     r := muxy.New(matcher)
//     // Create a group for the routes starting with the pattern "/admin",
//     // set the name prefix as "admin:" and register all routes from the
//     // external router.
//     g := r.Group("/admin").Name("admin:").Mount(admin.Router)
func (r *Router) Mount(src *Router) *Router {
	for k, _ := range src.Router.Routes {
		route := r.Route(k.Pattern).Name(k.Noun)
		for method, handler := range k.Handlers {
			route.Handle(handler, method)
		}
	}
	return r
}

// Route creates a new Route for the given pattern.
func (r *Router) Route(pattern string) *Route {
	route, err := r.Router.matcher.Route(r.Pattern + pattern)
	if err != nil {
		panic(err)
	}
	route.Router = r
	route.Pattern = r.Pattern + pattern
	route.Noun = r.Noun
	r.Router.Routes[route] = r.Pattern + pattern
	return route
}

// URL returns a URL for the given route name and variables.
func (r *Router) URL(name string, vars ...string) string {
	if route, ok := r.Router.NamedRoutes[name]; ok {
		u, err := r.Router.matcher.Build(route, vars...)
		if err != nil {
			panic(err)
		}
		return u
	}
	return ""
}

// ServeHTTP dispatches to the handler whose pattern matches the request.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.ServeHTTPC(context.TODO(), w, req)
}

// ServeHTTPC is a context-aware version of ServeHTTP.
//
// It can be used to add extra context data before routing begins.
func (r *Router) ServeHTTPC(c context.Context, w http.ResponseWriter, req *http.Request) {
	if c, h := r.Router.matcher.Match(c, req); h != nil {
		h.ServeHTTPC(c, w, req)
		return
	}
	http.NotFound(w, req)
}

// -----------------------------------------------------------------------------

// Route stores a URL pattern to be matched and the handler to be served
// in case of a match, optionally mapping HTTP methods to different handlers.
type Route struct {
	// Router holds the router that registered this route.
	Router *Router
	// Pattern holds the route pattern.
	Pattern string
	// Noun holds the route name.
	Noun string
	// Handlers maps request methods to the handlers that will handle them.
	Handlers map[string]Handler
}

// Name defines the route name used for URL building.
func (r *Route) Name(name string) *Route {
	r.Noun = r.Noun + name
	if _, ok := r.Router.Router.NamedRoutes[r.Noun]; ok {
		panic("muxy: duplicated name: " + r.Noun)
	}
	r.Router.Router.NamedRoutes[r.Noun] = r
	return r
}

// Handle sets the given handler to be served for the optional request methods.
func (r *Route) Handle(h Handler, methods ...string) *Route {
	for i := len(r.Router.Middleware) - 1; i >= 0; i-- {
		h = r.Router.Middleware[i](h)
	}
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
// to call r.Handle(h, "METHOD-NAME").

// Delete sets the given handler to be served for the request method DELETE.
//
// h must be Handler or func(context.Context, http.ResponseWriter, *http.Request).
func (r *Route) Delete(h interface{}) *Route {
	return r.Handle(toHandler(h), "DELETE")
}

// Get sets the given handler to be served for the request method GET.
//
// h must be Handler or func(context.Context, http.ResponseWriter, *http.Request).
func (r *Route) Get(h interface{}) *Route {
	return r.Handle(toHandler(h), "GET")
}

// Head sets the given handler to be served for the request method HEAD.
//
// h must be Handler or func(context.Context, http.ResponseWriter, *http.Request).
func (r *Route) Head(h interface{}) *Route {
	return r.Handle(toHandler(h), "HEAD")
}

// Options sets the given handler to be served for the request method OPTIONS.
//
// h must be Handler or func(context.Context, http.ResponseWriter, *http.Request).
func (r *Route) Options(h interface{}) *Route {
	return r.Handle(toHandler(h), "OPTIONS")
}

// PATCH sets the given handler to be served for the request method PATCH.
//
// h must be Handler or func(context.Context, http.ResponseWriter, *http.Request).
func (r *Route) Patch(h interface{}) *Route {
	return r.Handle(toHandler(h), "PATCH")
}

// POST sets the given handler to be served for the request method POST.
//
// h must be Handler or func(context.Context, http.ResponseWriter, *http.Request).
func (r *Route) Post(h interface{}) *Route {
	return r.Handle(toHandler(h), "POST")
}

// Put sets the given handler to be served for the request method PUT.
//
// h must be Handler or func(context.Context, http.ResponseWriter, *http.Request).
func (r *Route) Put(h interface{}) *Route {
	return r.Handle(toHandler(h), "PUT")
}

// toHandler converts the argument passed to convenience methods to a Handler.
//
// h must be Handler or func(context.Context, http.ResponseWriter, *http.Request).
func toHandler(h interface{}) Handler {
	switch h := h.(type) {
	case Handler:
		return h
	case func(context.Context, http.ResponseWriter, *http.Request):
		return HandlerFunc(h)
	}
	panic("muxy: handler must be muxy.Handler or func(context.Context, http.ResponseWriter, *http.Request)")
}
