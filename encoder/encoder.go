package encoder

import (
	"fmt"
	"strings"
)

// This file contains functions to encode and decode URL path segments as
// described in RFC 3986. The net/url package contains similar functionality,
// but it's not exported. Because the net/url package operates on the entire
// path, the package does not encode '/' as we want here.

var shouldEncode [256]byte

func init() {
	const (
		// RFC 2234 §6.1
		//  ALPHA = %x41-5A / %x61-7A   ; A-Z / a-z
		ALPHA = "abcdefghijklmnopqrstuvwxyz" + "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		//  DIGIT = %x30-39 ; 0-9
		DIGIT = "0123456789"

		// RFC 3986 §2.2
		//  unreserved = ALPHA / DIGIT / "-" / "." / "_" / "~"
		unreserved = ALPHA + DIGIT + "-._~"
		//  sub-delims = "!" / "$" / "&" / "'" / "(" / ")" / "*" / "+" / "," / ";" / "="
		sub_delims = "!$&'()*+,;="

		// RFC 3986 §2.1
		//  pct-encoded = "%" HEXDIG HEXDIG

		// RFC 3986 §3.3
		//  segment = *pchar
		//  pchar = unreserved / pct-encoded / sub-delims / ":" / "@"
		pchar = unreserved + sub_delims + ":@"
	)

	for i := range shouldEncode {
		if strings.IndexByte(pchar, byte(i)) < 0 {
			shouldEncode[i] = 1
		}
	}
}

// EncodePathSegment percent encodes bytes not allowed in a path segment.
func EncodePathSegment(s string) string {
	// Count bytes to escape.
	n := 0
	for i := 0; i < len(s); i++ {
		if shouldEncode[s[i]] != 0 {
			n++
		}
	}
	if n == 0 {
		return s
	}

	// Escape the bytes.
	p := make([]byte, len(s)+2*n)
	j := 0
	for i := 0; i < len(s); i++ {
		b := s[i]
		if shouldEncode[s[i]] != 0 {
			p[j] = '%'
			p[j+1] = "0123456789ABCDEF"[b>>4]
			p[j+2] = "0123456789ABCDEF"[b&15]
			j += 3
		} else {
			p[j] = b
			j++
		}
	}
	return string(p)
}

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

// DecodePathSegment decodes percent encodings in a path segment.
func DecodePathSegment(s string) (string, error) {
	// Use optimized standard library function to quickly test for the common
	// case where no decoding is required.
	i := strings.IndexByte(s, '%')
	if i < 0 {
		return s, nil
	}

	// Count number of %'s and check syntax.
	n := 0
	for i < len(s) {
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

	// Decode the string.
	p := make([]byte, len(s)-2*n)
	for i, j := 0, 0; i < len(s); {
		b := s[i]
		if b != '%' {
			i++
		} else {
			b = hexValue(s[i+1])<<4 | hexValue(s[i+2])
			i += 3
		}
		p[j] = b
		j++
	}
	return string(p), nil
}
