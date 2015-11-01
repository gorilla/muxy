package muxy

import (
	"strings"
)

const (
	variableKey = "{v}"
	wildcardKey = "(*}"
)

// newNode creates a new node.
func newNode() *node {
	return &node{edges: map[string]*node{}}
}

// node is a tree that stores keys to be matched and a possible value.
type node struct {
	leaf  interface{}      // leaf node, if any
	edges map[string]*node // edge nodes, if any
}

// newEdge returns the edge for the given parts, creating them if needed.
func (n *node) newEdge(p parts) *node {
	for _, v := range p {
		key := wildcardKey
		switch v.typ {
		case staticPart:
			key = v.val
		case variablePart:
			key = variableKey
		}
		e, ok := n.edges[key]
		if !ok {
			e = newNode()
			n.edges[key] = e
		}
		n = e
	}
	return n
}

// matchScheme returns the edge node for the given scheme, host and path.
func (n *node) matchScheme(scheme, host, path string) *node {
	if e, ok := n.edges[scheme]; ok {
		if hostNode, ok := e.leaf.(*node); ok {
			if e = hostNode.matchHost(host, path); e != nil {
				return e
			}
		}
	}
	if e, ok := n.edges[variableKey]; ok {
		if hostNode, ok := e.leaf.(*node); ok {
			if e = hostNode.matchHost(host, path); e != nil {
				return e
			}
		}
	}
	if e, ok := n.edges[wildcardKey]; ok {
		if hostNode, ok := e.leaf.(*node); ok {
			if e = hostNode.matchHost(host, path); e != nil {
				return e
			}
		}
	}
	return nil
}

// matchHost returns the edge node for the given host and path.
func (n *node) matchHost(host, path string) *node {
	next := ""
	if idx := strings.IndexByte(host, '.'); idx >= 0 {
		host, next = host[:idx], host[idx+1:]
	}
	if e, ok := n.edges[host]; ok {
		if len(next) == 0 {
			if pathNode, ok := e.leaf.(*node); ok {
				if e = pathNode.matchPath(path); e != nil {
					return e
				}
			}
		} else if e = e.matchHost(next, path); e != nil {
			return e
		}
	}
	if e, ok := n.edges[variableKey]; ok {
		if len(next) == 0 {
			if pathNode, ok := e.leaf.(*node); ok {
				if e = pathNode.matchPath(path); e != nil {
					return e
				}
			}
		} else if e = e.matchHost(next, path); e != nil {
			return e
		}
	}
	if e, ok := n.edges[wildcardKey]; ok {
		if pathNode, ok := e.leaf.(*node); ok {
			if e = pathNode.matchPath(path); e != nil {
				return e
			}
		}
	}
	return nil
}

// matchPath returns the edge node for the given path.
func (n *node) matchPath(path string) *node {
	next := ""
	if idx := strings.IndexByte(path, '/'); idx >= 0 {
		path, next = path[:idx], path[idx+1:]
	}
	if e, ok := n.edges[path]; ok {
		if len(next) == 0 {
			return e
		} else if e = e.matchPath(next); e != nil {
			return e
		}
	}
	if e, ok := n.edges[variableKey]; ok {
		if len(next) == 0 {
			return e
		} else if e = e.matchPath(next); e != nil {
			return e
		}
	}
	if e, ok := n.edges[wildcardKey]; ok {
		return e
	}
	return nil
}
