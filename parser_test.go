package muxy

import (
	"testing"
)

type parserTest struct {
	pattern string
	parts   parts
}

var parserTests = []parserTest{
	// static
	{"/", parts{
		{staticPart, ""},
	}},
	{"/foo", parts{
		{staticPart, "foo"},
	}},
	{"/foo/", parts{
		{staticPart, "foo"},
		{staticPart, ""},
	}},
	{"/foo/bar", parts{
		{staticPart, "foo"},
		{staticPart, "bar"},
	}},
	{"/foo/bar/", parts{
		{staticPart, "foo"},
		{staticPart, "bar"},
		{staticPart, ""},
	}},
	// variable
	{"/{foo}", parts{
		{variablePart, "foo"},
	}},
	{"/{foo}/", parts{
		{variablePart, "foo"},
		{staticPart, ""},
	}},
	{"/{foo}/{bar}", parts{
		{variablePart, "foo"},
		{variablePart, "bar"},
	}},
	{"/{foo}/{bar}/", parts{
		{variablePart, "foo"},
		{variablePart, "bar"},
		{staticPart, ""},
	}},
	// wildcard
	{"/{*}", parts{
		{wildcardPart, ""},
	}},
	{"/foo/{*}", parts{
		{staticPart, "foo"},
		{wildcardPart, ""},
	}},
	{"/foo/{bar}/{*}", parts{
		{staticPart, "foo"},
		{variablePart, "bar"},
		{wildcardPart, ""},
	}},
	// parsing errors
	{"//foo", nil},     // double separator
	{"/fo{o", nil},     // variable delimiter in static part
	{"/fo}o", nil},     // variable delimiter in static part
	{"/foo{bar}", nil}, // variable in bad place
	{"/{bar}foo", nil}, // static in bad place
	{"/{foo", nil},     // unclosed variable
	{"/{foo}}", nil},   // too closed variable
	{"/{*}/", nil},     // wildcard in bad place
	{"/{*name}", nil},  // invalid variable name
	{"/{1name}", nil},  // invalid variable name
}

func TestParsePaths(t *testing.T) {
	for _, v := range parserTests {
		parts, err := parse(v.pattern, '/')
		if err == nil {
			if !equalParts(v.parts, parts) {
				t.Errorf("%q: expected %v; got %v", v.pattern, v.parts, parts)
			}
			raw := v.parts.raw('/')
			if v.pattern != raw {
				t.Errorf("%q: bad conversion to string; got %v", v.pattern, raw)
			}
		} else if v.parts != nil {
			t.Errorf("%q: expected %v; failed to parse with error %q", v.pattern, v.parts, err)
		}
	}
}

func equalParts(p1, p2 parts) bool {
	if len(p1) != len(p2) {
		return false
	}
	for k, v := range p1 {
		if v.typ != p2[k].typ || v.val != p2[k].val {
			return false
		}
	}
	return true
}
