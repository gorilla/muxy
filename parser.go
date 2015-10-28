package muxy

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// parse parses a string s into parts separated by sep. for each part:
//
//     - curly braces are only allowed enclosing a whole part;
//     - if the part is not enclosed by curly braces, it is a static part;
//     - if the part is enclosed by curly braces, it is a {variable};
//     - a variable name must be a vald Go identifier or *;
//     - * is a special wildcard variable that can only be at the end of s;
//     - an empty part is only allowed as the last element of parts;
func parse(s string, sep rune) ([]part, error) {
	if i := strings.IndexRune(s, sep); i == 0 {
		s = s[utf8.RuneLen(sep):]
	}
	n := 1
	for _, r := range s {
		if r == sep {
			n++
		}
	}
	p := &parser{src: s, sep: sep, dst: make([]part, n)}
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

type part struct {
	typ partType
	val string
}

// -----------------------------------------------------------------------------

const eof = -1

type parser struct {
	src string
	sep rune
	pos int
	idx int
	dst []part
}

func (p *parser) setPart(typ partType, val string) {
	p.dst[p.idx] = part{typ: typ, val: val}
	p.idx++
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

// parseParts consumes all parts recursively.
// The leading separator was already consumed.
func (p *parser) parseParts() error {
	pin := p.pos
	r := p.next()
	switch r {
	case eof:
		p.setPart(staticPart, "")
		return nil
	case p.sep:
		return p.errorf("unexpected double %q", p.sep)
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
	}
	for {
		r = p.next()
		switch r {
		case p.sep:
			p.setPart(staticPart, p.src[pin:p.pos-utf8.RuneLen(p.sep)])
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
// The opening curly brace was already consumed.
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
	return nil
}

func (p *parser) errorf(format string, args ...interface{}) error {
	return fmt.Errorf(fmt.Sprintf("%q: %s", p.src, format), args...)
}
