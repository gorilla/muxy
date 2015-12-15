package mpath

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/muxy"
	"golang.org/x/net/context"
)

type route struct {
	parts []string
	keys  []muxy.Variable
}

// -----------------------------------------------------------------------------

func New(options ...func(*matcher)) *muxy.Router {
	m := &matcher{
		routes: map[*muxy.Route]route{},
	}
	for _, o := range options {
		o(m)
	}
	return muxy.New(m)
}

type matcher struct {
	// TODO...
	routes map[*muxy.Route]route
}

func (m *matcher) Route(pattern string) (*muxy.Route, error) {
	// TODO...
	return nil, nil
}

func (m *matcher) Match(c context.Context, r *http.Request) (context.Context, muxy.Handler) {
	// TODO...
	return nil, nil
}

func (m *matcher) Build(r *muxy.Route, vars ...string) (string, error) {
	route, ok := m.routes[r]
	if !ok {
		return "", fmt.Errorf("muxy: route not found: %v", r)
	}
	size := len(vars)
	if size%2 != 0 {
		return "", fmt.Errorf("muxy: expected even variadic arguments, got %v: %v", len(vars), vars)
	}
	b, count := new(bytes.Buffer), 0
	for _, part := range route.parts {
		if part[0] == '/' {
			b.WriteString(part)
			continue
		}
		for i := 0; i < size; i += 2 {
			if vars[i] == part {
				b.WriteString(vars[i+1])
				count++
			}
		}
	}
	if size/2 != count {
		return "", fmt.Errorf("muxy: unexpected variadic arguments: %v", vars)
	}
	return b.String(), nil
}

// -----------------------------------------------------------------------------

// methodHandler returns the handler registered for the given HTTP method.
func methodHandler(handlers map[string]muxy.Handler, method string) muxy.Handler {
	if handlers == nil || len(handlers) == 0 {
		return nil
	}
	if h, ok := handlers[method]; ok {
		return h
	}
	switch method {
	case "OPTIONS":
		return allowHandler(handlers, 200)
	case "HEAD":
		if h, ok := handlers["GET"]; ok {
			return h
		}
		fallthrough
	default:
		if h, ok := handlers[""]; ok {
			return h
		}
	}
	return allowHandler(handlers, 405)
}

// allowHandler returns a handler that sets a header with the given
// status code and allowed methods.
func allowHandler(handlers map[string]muxy.Handler, code int) muxy.Handler {
	allowed := make([]string, len(handlers)+1)
	allowed[0] = "OPTIONS"
	i := 1
	for m, _ := range handlers {
		if m != "" && m != "OPTIONS" {
			allowed[i] = m
			i++
		}
	}
	return muxy.HandlerFunc(func(c context.Context, w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Allow", strings.Join(allowed[:i], ", "))
		w.WriteHeader(code)
		fmt.Fprintln(w, code, http.StatusText(code))
	})
}

// -----------------------------------------------------------------------------

// setVars sets the route variables in the context.
//
// Since the path matched already, we can make some assumptions: the path
// starts with a slash and there are no empty or dotted path segments.
//
// For the values:
//
//     path := "/foo/bar/var1/baz/var2/x/y/z"
//     parts := []string{"/foo/bar/", "v1", "/baz/", "v2", "*"}
//     keys := []muxy.Variable{"v1", "v2", "*"}
//
// The variables will be:
//
//     vars = []string{"var1", "var2", "x/y/z"}
func setVars(c context.Context, path string, parts []string, keys []muxy.Variable) context.Context {
	path, idx := path[1:], 0
	vars := make([]string, len(keys))
	for _, part := range parts {
		switch part[0] {
		case '/':
			path = path[len(part)-1:]
		case '*':
			vars[idx] = path
			break
		default:
			if i := strings.IndexByte(path, '/'); i < 0 {
				vars[idx] = path
				break
			} else {
				vars[idx] = path[:i]
				path = path[i+1:]
				idx++
			}
		}
	}
	return &varsCtx{c, keys, vars}
}

// varsCtx carries a key-variables mapping. It implements Context.Value() and
// delegates all other calls to the embedded Context.
type varsCtx struct {
	context.Context
	keys []muxy.Variable
	vars []string
}

func (c *varsCtx) Value(key interface{}) interface{} {
	for k, v := range c.keys {
		if v == key {
			return c.vars[k]
		}
	}
	return c.Context.Value(key)
}
