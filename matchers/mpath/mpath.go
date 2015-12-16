package mpath

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/muxy"
	"golang.org/x/net/context"
)

func New(options ...func(*matcher)) *muxy.Router {
	m := &matcher{
		patterns: map[*muxy.Route]pattern{},
	}
	for _, o := range options {
		o(m)
	}
	return muxy.New(m)
}

type matcher struct {
	// TODO...
	patterns map[*muxy.Route]pattern
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
	if p, ok := m.patterns[r]; ok {
		return p.build(vars...)
	}
	return "", fmt.Errorf("muxy: route not found: %v", r)
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

type pattern struct {
	parts []string
	keys  []muxy.Variable
}

// setVars sets the route variables in the context.
//
// Since the path matched already, we can make some assumptions: the path
// starts with a slash and there are no empty or dotted path segments.
//
// For the values:
//
//     p := pattern{
//         parts: []string{"/foo/bar/", "v1", "/baz/", "v2", "*"},
//         keys:  []muxy.Variable{"v1", "v2", "*"},
//     }
//     path := "/foo/bar/var1/baz/var2/x/y/z"
//
// The variables will be:
//
//     vars = []string{"var1", "var2", "x/y/z"}
func (p pattern) setVars(c context.Context, path string) context.Context {
	path, idx := path[1:], 0
	vars := make([]string, len(p.keys))
	for _, part := range p.parts {
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
	return &varsCtx{c, p.keys, vars}
}

func (p pattern) build(vars ...string) (string, error) {
	if len(p.keys)*2 != len(vars) {
		return "", fmt.Errorf("muxy: expected %d arguments, got %d: %v", len(p.keys)*2, len(vars), vars)
	}
	b := new(bytes.Buffer)
	for _, part := range p.parts {
		if part[0] == '/' {
			b.WriteString(part)
			continue
		}
		for i, s := 0, len(vars); i < s; i += 2 {
			if vars[i] == part {
				b.WriteString(vars[i+1])
			}
		}
	}
	return b.String(), nil
}

// -----------------------------------------------------------------------------

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
