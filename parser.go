package muxy

import (
	"bytes"
	"fmt"
	"unicode"
	"unicode/utf8"
)

// parse splits s into all segments separated by sep and returns a slice of
// the part type contained between those separators. For each segment:
//
//     - if it is not enclosed by curly braces, it is a static part;
//     - if it is enclosed by curly braces, it is a variable part;
//     - {*} is a special wildcard variable only allowed at the end of s;
//     - curly braces are only allowed enclosing a whole segment;
//     - a variable name must be a vald Go identifier or *;
//     - an empty segment is only allowed as the last part;
//
// Example:
//
//     // returns three parts: static "foo", variable "bar" and wildcard ""
//     parts, err := parse("/foo/{bar}/{*}", '/')
func parse(s string, sep byte) (parts, error) {
	if s != "" && s[0] == sep {
		s = s[1:]
	}
	n, r := 1, rune(sep)
	for _, v := range s {
		if v == r {
			n++
		}
	}
	p := parser{src: s, sep: r, dst: make([]part, n)}
	if err := p.parseParts(); err != nil {
		return nil, err
	}
	return p.dst, nil
}

// -----------------------------------------------------------------------------

type partType uint8

func (t partType) String() string {
	switch t {
	case staticPart:
		return "static"
	case variablePart:
		return "variable"
	}
	return "wildcard"
}

const (
	staticPart partType = iota
	variablePart
	wildcardPart
)

// part represents a segment to be matched.
type part struct {
	typ partType
	val string
}

type parts []part

func (p parts) raw(sep byte) string {
	b := new(bytes.Buffer)
	if sep == '/' {
		b.WriteByte(sep)
	}
	for k, v := range p {
		switch v.typ {
		case staticPart:
			b.WriteString(v.val)
		case variablePart:
			b.WriteString("{" + v.val + "}")
		case wildcardPart:
			b.WriteString("{*}")
		}
		if k < len(p)-1 {
			b.WriteByte(sep)
		}
	}
	return b.String()
}

// -----------------------------------------------------------------------------

const eof = -1

// parser parses the declaration syntax.
type parser struct {
	src string
	sep rune
	pos int
	idx int
	dst parts
}

// next returns the next rune in the input.
func (p *parser) next() rune {
	if p.pos >= len(p.src) {
		return eof
	}
	r, w := utf8.DecodeRuneInString(p.src[p.pos:])
	p.pos += w
	return r
}

// setPart adds a part to the destination slice.
func (p *parser) setPart(typ partType, val string) {
	p.dst[p.idx] = part{typ: typ, val: val}
	p.idx++
}

// parseParts consumes all parts recursively.
//
// The separator was already consumed when this method is called.
func (p *parser) parseParts() error {
	pin := p.pos
	switch p.next() {
	case eof:
		p.setPart(staticPart, "")
		return nil
	case '{':
		if err := p.parseVariable(); err != nil {
			return err
		}
		val := p.src[pin+1 : p.pos-1]
		if val == "*" {
			p.setPart(wildcardPart, "")
		} else {
			p.setPart(variablePart, val)
		}
		switch p.next() {
		case eof:
			return nil
		case p.sep:
			return p.parseParts()
		}
		return p.errorf("unexpected characters after variable declaration")
	case p.sep:
		return p.errorf("unexpected double %q", p.sep)
	}
	for {
		switch p.next() {
		case p.sep:
			p.setPart(staticPart, p.src[pin:p.pos-1])
			return p.parseParts()
		case eof:
			p.setPart(staticPart, p.src[pin:p.pos])
			return nil
		case '{', '}':
			return p.errorf("variables must be at the start of a segment")
		}
	}
}

// parseVariable consumes the variable name including the closing curly brace.
//
// The opening curly brace was already consumed when this method is called.
func (p *parser) parseVariable() error {
	switch r := p.next(); {
	case r == eof:
		return p.errorf("unexpected eof after '{'")
	case r == '*':
		if r = p.next(); r != '}' {
			return p.errorf("expected '}' after '*'")
		}
		if r = p.next(); r != eof {
			return p.errorf("expected eof after '{*}'")
		}
		return nil
	case r != '_' && !unicode.IsLetter(r):
		return p.errorf("expected underscore or letter starting a variable name; got %q", r)
	}
	for {
		switch r := p.next(); {
		case r == '}':
			return nil
		case r == eof:
			return p.errorf("unexpected eof in variable name")
		case r == p.sep:
			return p.errorf("missing '}' in variable declaration")
		case r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r):
			return p.errorf("unexpected %q in variable name", r)
		}
	}
}

// errorf returns an error prefixed by the string being parsed.
func (p *parser) errorf(format string, args ...interface{}) error {
	return fmt.Errorf(fmt.Sprintf("mux: %q: %s", p.src, format), args...)
}
