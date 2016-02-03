package mpath

import (
	"bytes"
	"fmt"
	"net/http"
	"path"
	"strings"
	"unicode"

	"github.com/gorilla/muxy"
	"golang.org/x/net/context"
)

func NotFoundHandler(h muxy.Handler) func(*matcher) {
	return func(m *matcher) {
		m.notFoundHandler = h
	}
}

func New(options ...func(*matcher)) *muxy.Router {
	m := &matcher{
		root:            &node{edges: map[string]*node{}},
		patterns:        map[*muxy.Route]*pattern{},
		notFoundHandler: muxy.HandlerFunc(notFound),
	}
	for _, o := range options {
		o(m)
	}
	return muxy.New(m)
}

type matcher struct {
	root            *node
	patterns        map[*muxy.Route]*pattern
	notFoundHandler muxy.Handler
}

func (m *matcher) Route(pattern string) (*muxy.Route, error) {
	segs, err := parse(pattern)
	if err != nil {
		return nil, err
	}
	e := m.root.edge(segs)
	if r := e.leaf; r != nil {
		return nil, fmt.Errorf("muxy: a route for the pattern %q or equivalent already exists: %q", pattern, r.Pattern)
	}
	e.leaf = &muxy.Route{}
	m.patterns[e.leaf] = newPattern(segs)
	return e.leaf, nil
}

func (m *matcher) Match(c context.Context, r *http.Request) (context.Context, muxy.Handler) {
	var h muxy.Handler
	// TODO: use a backward-compatible alternative to URL.RawPath here.
	path := cleanPath(r.URL.Path)
	e := m.root.match(path)
	if e != nil && e.leaf != nil {
		h = methodHandler(e.leaf.Handlers, r.Method)
	}
	if h == nil {
		h = m.notFoundHandler
	} else {
		c = m.patterns[e.leaf].setVars(c, path)
	}
	return c, h
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

type node struct {
	edges map[string]*node // static edges, if any
	vEdge *node            // variable edge, if any
	wEdge *node            // wildcard edge, if any
	leaf  *muxy.Route      // leaf value, if any
}

// edge returns the edge for the given path segments, creating it if needed.
func (n *node) edge(segs []string) *node {
	for _, seg := range segs {
		switch seg[0] {
		case ':':
			if n.vEdge == nil {
				n.vEdge = &node{edges: map[string]*node{}}
			}
			n = n.vEdge
		case '*':
			if n.wEdge == nil {
				n.wEdge = &node{}
			}
			return n.wEdge
		default:
			if n.edges[seg] == nil {
				n.edges[seg] = &node{edges: map[string]*node{}}
			}
			n = n.edges[seg]
		}
	}
	return n
}

func (n *node) match(path string) *node {
	part, path := "", path[1:]
	for len(path) > 0 {
		if i := strings.IndexByte(path, '/'); i < 0 {
			part, path = path, ""
		} else {
			part, path = path[:i], path[i+1:]
		}
		if e, ok := n.edges[part]; ok {
			n = e
			continue
		}
		if e := n.vEdge; e != nil {
			n = e
			continue
		}
		return n.wEdge
	}
	return n
}

// -----------------------------------------------------------------------------

// newPattern returns a pattern for the given path segments.
func newPattern(segs []string) *pattern {
	p := pattern{}
	b := new(bytes.Buffer)
	for _, s := range segs {
		switch s[0] {
		case ':':
			s = s[1:]
		case '*':
		default:
			b.WriteByte('/')
			b.WriteString(s)
			s = ""
		}
		if s != "" {
			p.parts = append(p.parts, b.String()+"/")
			b.Reset()
			p.parts = append(p.parts, s)
			p.keys = append(p.keys, muxy.Variable(s))
		}
	}
	if b.Len() != 0 {
		p.parts = append(p.parts, b.String())
	}
	return &p
}

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
func (p *pattern) setVars(c context.Context, path string) context.Context {
	if len(p.keys) == 0 {
		return c
	}
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

func (p *pattern) build(vars ...string) (string, error) {
	if len(p.keys)*2 != len(vars) {
		return "", fmt.Errorf("muxy: expected %d arguments, got %d: %v", len(p.keys)*2, len(vars), vars)
	}
	b := new(bytes.Buffer)
Loop:
	for _, part := range p.parts {
		if part[0] == '/' {
			b.WriteString(part)
			continue
		}
		for i, s := 0, len(vars); i < s; i += 2 {
			if vars[i] == part {
				b.WriteString(vars[i+1])
				vars[i] = ""
				continue Loop
			}
		}
		return "", fmt.Errorf("muxy: missing argument for variable %q", part)
	}
	return b.String(), nil
}

// -----------------------------------------------------------------------------

func parse(pattern string) ([]string, error) {
	pattern = cleanPath(pattern)
	count := 0
	for _, r := range pattern {
		if r == '/' {
			count++
		}
	}
	segs := make([]string, count)
	idx := 0
	part, path := "", pattern[1:]
	for len(path) > 0 {
		if i := strings.IndexByte(path, '/'); i < 0 {
			part, path = path, ""
		} else {
			part, path = path[:i], path[i+1:]
		}
		switch part[0] {
		case ':':
			if len(part) == 1 {
				return nil, fmt.Errorf("empty variable name")
			}
			for k, r := range part[1:] {
				if k == 0 {
					if r != '_' && !unicode.IsLetter(r) {
						return nil, fmt.Errorf("expected underscore or letter starting a variable name; got %q", r)
					}
				} else if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
					return nil, fmt.Errorf("unexpected %q in variable name", r)
				}
			}
		case '*':
			if len(part) != 1 {
				return nil, fmt.Errorf("unexpected wildcard: %q", part)
			}
			if len(path) != 0 {
				return nil, fmt.Errorf("wildcard must be at the end of a pattern; got: .../*/%v", path)
			}
		}
		segs[idx] = part
		idx++
	}
	return segs, nil
}

// cleanPath returns the canonical path for p, eliminating . and .. elements.
//
// Borrowed from net/http.
func cleanPath(p string) string {
	if p == "" {
		return "/"
	}
	if p[0] != '/' {
		p = "/" + p
	}
	np := path.Clean(p)
	// path.Clean removes trailing slash except for root;
	// put the trailing slash back if necessary.
	if p[len(p)-1] == '/' && np != "/" {
		np += "/"
	}
	return np
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

// -----------------------------------------------------------------------------

// notFound replies to the request with an HTTP 404 not found error.
func notFound(c context.Context, w http.ResponseWriter, r *http.Request) {
	http.Error(w, "404 page not found", http.StatusNotFound)
}
