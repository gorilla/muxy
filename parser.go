package muxy

import (
	"fmt"
	"strings"
	"unicode"
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
//     - static segment values are percent decoded;
//
// Example:
//
//     // returns three parts: static "foo", variable "bar" and wildcard ""
//     parts, err := parse("/foo/{bar}/{*}", '/')
func parse(pattern string, sep byte) ([]part, error) {

	index := 0 // index in pattern to beginning of next segment
	if sep == '/' {
		if len(pattern) == 0 || pattern[0] != '/' {
			return nil, fmt.Errorf("%q: path pattern must start with '/'", pattern)
		}
		index = 1
	}

	var parts []part
	for index <= len(pattern) {

		i := strings.IndexByte(pattern[index:], sep)
		if i < 0 {
			i = len(pattern[index:])
		}
		segment := pattern[index : index+i]
		index += i + 1

		switch {
		case segment == "":
			switch {
			case sep == '.':
				return nil, fmt.Errorf("%q: empty segment not allowed", pattern)
			case index <= len(pattern):
				return nil, fmt.Errorf("%q: empty segment not allowed in middle of pattern", pattern)
			}
			parts = append(parts, part{typ: staticPart, val: segment})
		case segment[0] == '{':
			if segment[len(segment)-1] != '}' {
				return nil, fmt.Errorf("%q: malformed variable specification %q", pattern, segment)
			}
			name := segment[1 : len(segment)-1]
			switch name {
			case "":
				return nil, fmt.Errorf("%q: empty variable name not allowed", pattern)
			case "*":
				if index <= len(pattern) {
					return nil, fmt.Errorf("%q: {*} not allowed in middle of pattern", pattern)
				}
				parts = append(parts, part{typ: wildcardPart})
			default:
				for i, r := range name {
					if r == '_' || unicode.IsLetter(r) {
						continue
					}
					if i > 0 && unicode.IsDigit(r) {
						continue
					}
					return nil, fmt.Errorf("%q: invalid variable name %q", pattern, name)
				}
				parts = append(parts, part{typ: variablePart, val: name})
			}
		default:
			if strings.IndexByte(segment, '{') >= 0 || strings.IndexByte(segment, '}') >= 0 {
				return nil, fmt.Errorf("%q: '{' and '}' not allowed in pattern segment", pattern)
			}
			if sep == '/' {
				var err error
				segment, err = percentDecode(segment)
				if err != nil {
					return nil, err
				}
			}
			parts = append(parts, part{typ: staticPart, val: segment})
		}
	}
	return parts, nil
}

// -----------------------------------------------------------------------------

func isHex(b byte) bool {
	return '0' <= b && b <= '9' ||
		'a' <= b && b <= 'f' ||
		'A' <= b && b <= 'F'
}

func hexValue(b byte) byte {
	if b <= '9' {
		return b & 0xf
	} else {
		return 9 + (b & 0xf)
	}
}

func percentDecode(s string) (string, error) {
	n := 0
	for i := 0; i < len(s); {
		if s[i] != '%' {
			i++
		} else if i+2 < len(s) && isHex(s[i+1]) && isHex(s[i+2]) {
			i += 3
			n += 1
		} else {
			s = s[i:]
			if len(s) > 3 {
				s = s[:3]
			}
			return "", fmt.Errorf("bad percent escape %q", s)
		}
	}
	if n == 0 {
		return s, nil
	}

	p := make([]byte, len(s)-2*n)
	j := 0
	for i := 0; i < len(s); {
		b := s[i]
		if b != '%' {
			p[j] = b
			i++
			j++
		} else {
			p[j] = hexValue(s[i+1])<<4 | hexValue(s[i+2])
			i += 3
			j++
		}
	}
	return string(p), nil
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

// -----------------------------------------------------------------------------
