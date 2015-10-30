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

// edge returns the edge node corresponding to s.
//
// A recursive lookup is performend slicing s into all substrings
// separated by sep.
func (n *node) edge(s string, sep byte) *node {
	next := ""
	if idx := strings.IndexByte(s, sep); idx >= 0 {
		s, next = s[:idx], s[idx+1:]
	}
	if e := n.edges[s]; e != nil {
		if len(next) == 0 {
			return e
		} else if e = e.edge(next, sep); e != nil {
			return e
		}
	}
	if e := n.edges[variableKey]; e != nil {
		if len(next) == 0 {
			return e
		} else if e = e.edge(next, sep); e != nil {
			return e
		}
	}
	if e := n.edges[wildcardKey]; e != nil {
		return e
	}
	return nil
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
