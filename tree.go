package muxy

import (
	"fmt"
	"strings"
)

const (
	variableKey = "{"
	wildcardKey = "}"
)

type leaf interface {
	setParts([]part)
	rawParts() string
}

func newNode() *node {
	return &node{edges: map[string]*node{}}
}

type node struct {
	leaf  leaf             // leaf node, if any
	edges map[string]*node // edge nodes, if any
}

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

func (n *node) setLeaf(s string, sep byte, l leaf) error {
	p, err := parse(s, sep)
	if err != nil {
		return err
	}
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
	if n.leaf != nil {
		return fmt.Errorf("%q already has a registered equivalent: %s", s, n.leaf.rawParts())
	}
	l.setParts(p)
	n.leaf = l
	return nil
}
