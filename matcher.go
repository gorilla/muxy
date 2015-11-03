package muxy

import (
	"net/url"
	"strings"
)

const (
	variableKey = "{v}"
	wildcardKey = "(*}"
)

type matcher interface {
	match(scheme, host, path string) *pathMatcher
}

// -----------------------------------------------------------------------------

type schemeMatcher map[string]*hostMatcher

func (m schemeMatcher) match(scheme, host, path string) *pathMatcher {
	if hm, ok := m[scheme]; ok {
		if pm := hm.match(scheme, host, path); pm != nil {
			return pm
		}
	}
	if hm, ok := m[wildcardKey]; ok {
		if pm := hm.match(scheme, host, path); pm != nil {
			return pm
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

func (m *hostMatcher) match(scheme, host, path string) *pathMatcher {
	next := ""
	idx := strings.IndexByte(host, '.')
	if idx >= 0 {
		host, next = host[:idx], host[idx+1:]
	}
	if hm, ok := m.edges[host]; ok {
		if idx < 0 {
			if pm := hm.leaf; pm != nil {
				if pm := pm.match(scheme, host, path); pm != nil {
					return pm
				}
			}
		} else if pm := hm.match(scheme, next, path); pm != nil {
			return pm
		}
	}
	if hm, ok := m.edges[variableKey]; ok {
		if idx < 0 {
			if pm := hm.leaf; pm != nil {
				if pm := pm.match(scheme, host, path); pm != nil {
					return pm
				}
			}
		} else if pm := hm.match(scheme, next, path); pm != nil {
			return pm
		}
	}
	if hm, ok := m.edges[wildcardKey]; ok && hm.leaf != nil {
		return hm.leaf.match(scheme, host, path)
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

func (m *pathMatcher) match(scheme, host, path string) *pathMatcher {
	next := ""
	idx := strings.IndexByte(path, '/')
	if idx >= 0 {
		path, next = path[:idx], path[idx+1:]
	}
	if pm, ok := m.edges[path]; ok {
		if idx < 0 {
			return pm
		} else if pm = pm.match(scheme, host, next); pm != nil {
			return pm
		}
	}
	if pm, ok := m.edges[variableKey]; ok {
		if idx < 0 {
			return pm
		} else if pm = pm.match(scheme, host, next); pm != nil {
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

	// conversion needed?
	m2 := m
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
		pm = matcherForScheme(m2, u.Scheme, u.Host, u.Path)
	case *hostMatcher:
		pm = matcherForHost(m2, u.Host, u.Path)
	case *pathMatcher:
		pm = matcherForPath(m2, u.Path)
	}
	return m2, pm
}

func matcherForScheme(sm schemeMatcher, scheme, host, path string) *pathMatcher {
	if scheme == "" {
		scheme = wildcardKey
	}
	if _, ok := sm[scheme]; !ok {
		sm[scheme] = newHostMatcher()
	}
	return matcherForHost(sm[scheme], host, path)
}

func matcherForHost(hm *hostMatcher, host, path string) *pathMatcher {
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
	return matcherForPath(hm.leaf, path)
}

func matcherForPath(pm *pathMatcher, path string) *pathMatcher {
	if path == "" {
		return pm.newEdge(parts{{typ: wildcardPart}})
	}
	parts, err := parse(path, '/')
	if err != nil {
		panic(err)
	}
	return pm.newEdge(parts)
}
