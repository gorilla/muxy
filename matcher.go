package muxy

import (
	"net/http"
	"net/url"
	"strings"
)

const (
	variableKey = "{v}"
	wildcardKey = "(*}"
)

type matcher interface {
	match(*Router, *http.Request) *Route
}

// -----------------------------------------------------------------------------

type schemeMatcher map[string]*hostMatcher

func (m schemeMatcher) match(r *Router, req *http.Request) *Route {
	if hm, ok := m[req.URL.Scheme]; ok {
		if route := hm.match(r, req); route != nil {
			return route
		}
	}
	if hm, ok := m[wildcardKey]; ok {
		if route := hm.match(r, req); route != nil {
			return route
		}
	}
	return nil
}

// -----------------------------------------------------------------------------

func newHostMatcher() *hostMatcher {
	return &hostMatcher{edges: map[string]*hostMatcher{}}
}

type hostMatcher struct {
	leaf  *pathMatcher            // leaf node, if any
	edges map[string]*hostMatcher // edge nodes, if any
}

func (m *hostMatcher) match(r *Router, req *http.Request) *Route {
	return m.matchHost(r, req, req.URL.Host)
}

// matchHost returns the route for the given request.
func (m *hostMatcher) matchHost(r *Router, req *http.Request, host string) *Route {
	next := ""
	if idx := strings.IndexByte(host, '.'); idx >= 0 {
		host, next = host[:idx], host[idx+1:]
	}
	if hm, ok := m.edges[host]; ok {
		if len(next) == 0 {
			if pm := hm.leaf; pm != nil {
				if route := pm.match(r, req); route != nil {
					return route
				}
			}
		} else if route := hm.matchHost(r, req, next); route != nil {
			return route
		}
	}
	if hm, ok := m.edges[variableKey]; ok {
		if len(next) == 0 {
			if pm := hm.leaf; pm != nil {
				if route := pm.match(r, req); route != nil {
					return route
				}
			}
		} else if route := hm.matchHost(r, req, next); route != nil {
			return route
		}
	}
	if hm, ok := m.edges[wildcardKey]; ok && hm.leaf != nil {
		return hm.leaf.match(r, req)
	}
	return nil
}

// newEdge returns the edge for the given parts, creating them if needed.
func (m *hostMatcher) newEdge(p parts) *hostMatcher {
	for _, v := range p {
		key := wildcardKey
		switch v.typ {
		case staticPart:
			key = v.val
		case variablePart:
			key = variableKey
		}
		e, ok := m.edges[key]
		if !ok {
			e = newHostMatcher()
			m.edges[key] = e
		}
		m = e
	}
	return m
}

func (m *hostMatcher) toSchemeMatcher() schemeMatcher {
	return schemeMatcher{wildcardKey: m}
}

// -----------------------------------------------------------------------------

func newPathMatcher() *pathMatcher {
	return &pathMatcher{edges: map[string]*pathMatcher{}}
}

type pathMatcher struct {
	leaf  *Route                  // leaf node, if any
	edges map[string]*pathMatcher // edge nodes, if any
}

func (m *pathMatcher) match(r *Router, req *http.Request) *Route {
	// TODO: following router configuration, return route that sets a
	// redirection for strict slashes.
	if e := m.edge(req.URL.Path[1:]); e != nil {
		return e.leaf
	}
	return nil
}

// edge returns the edge node for the given path.
func (m *pathMatcher) edge(path string) *pathMatcher {
	next := ""
	if idx := strings.IndexByte(path, '/'); idx >= 0 {
		path, next = path[:idx], path[idx+1:]
	}
	if pm, ok := m.edges[path]; ok {
		if len(next) == 0 {
			return pm
		} else if pm = pm.edge(next); pm != nil {
			return pm
		}
	}
	if pm, ok := m.edges[variableKey]; ok {
		if len(next) == 0 {
			return pm
		} else if pm = pm.edge(next); pm != nil {
			return pm
		}
	}
	if pm, ok := m.edges[wildcardKey]; ok {
		return pm
	}
	return nil
}

// newEdge returns the edge for the given parts, creating them if needed.
func (m *pathMatcher) newEdge(p parts) *pathMatcher {
	for _, v := range p {
		key := wildcardKey
		switch v.typ {
		case staticPart:
			key = v.val
		case variablePart:
			key = variableKey
		}
		e, ok := m.edges[key]
		if !ok {
			e = newPathMatcher()
			m.edges[key] = e
		}
		m = e
	}
	return m
}

func (m *pathMatcher) toSchemeMatcher() schemeMatcher {
	return m.toHostMatcher().toSchemeMatcher()
}

func (m *pathMatcher) toHostMatcher() *hostMatcher {
	hm := newHostMatcher()
	hm.edges[wildcardKey] = newHostMatcher()
	hm.edges[wildcardKey].leaf = m
	return hm
}

// -----------------------------------------------------------------------------

func newPattern(m matcher, pattern string) (matcher, *pathMatcher) {
	u, err := url.Parse(pattern)
	if err != nil {
		panic(err)
	}
	m2 := m
	// conversion needed?
	switch m := m.(type) {
	case *hostMatcher:
		if u.Scheme != "" {
			m2 = m.toSchemeMatcher()
		}
	case *pathMatcher:
		if u.Scheme != "" {
			m2 = m.toSchemeMatcher()
		} else if u.Host != "" {
			m2 = m.toHostMatcher()
		}
	}

	var pm *pathMatcher
	switch m2 := m2.(type) {
	case schemeMatcher:
		pm = matcherForScheme(u.Scheme, u.Host, u.Path, m2)
	case *hostMatcher:
		pm = matcherForHost(u.Host, u.Path, m2)
	case *pathMatcher:
		pm = matcherForPath(u.Path, m2)
	}
	return m2, pm
}

func matcherForScheme(scheme, host, path string, sm schemeMatcher) *pathMatcher {
	if scheme == "" {
		scheme = wildcardKey
	}
	hm, ok := sm[scheme]
	if !ok {
		hm = newHostMatcher()
		sm[scheme] = hm
	}
	return matcherForHost(host, path, hm)
}

func matcherForHost(host, path string, hm *hostMatcher) *pathMatcher {
	if host == "" {
		hm = hm.newEdge(parts{{typ: wildcardPart}})
	} else {
		parts, err := parse(host, '.')
		if err != nil {
			panic(err)
		}
		hm = hm.newEdge(parts)
	}
	if hm.leaf == nil {
		hm.leaf = newPathMatcher()
	}
	return matcherForPath(path, hm.leaf)
}

func matcherForPath(path string, pm *pathMatcher) *pathMatcher {
	if path == "" {
		return pm.newEdge(parts{{typ: wildcardPart}})
	}
	parts, err := parse(path, '/')
	if err != nil {
		panic(err)
	}
	return pm.newEdge(parts)
}
