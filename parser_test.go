package muxy

import (
	"testing"
)

type parserTest struct {
	pattern string
	parts   []part
}

var parserTests = []parserTest{
	// static
	{"/", []part{
		{staticPart, ""},
	}},
	{"/foo", []part{
		{staticPart, "foo"},
	}},
	{"/foo/", []part{
		{staticPart, "foo"},
		{staticPart, ""},
	}},
	{"/foo/bar", []part{
		{staticPart, "foo"},
		{staticPart, "bar"},
	}},
	{"/foo/bar/", []part{
		{staticPart, "foo"},
		{staticPart, "bar"},
		{staticPart, ""},
	}},
	// variable
	{"/{foo}", []part{
		{variablePart, "foo"},
	}},
	{"/{foo}/", []part{
		{variablePart, "foo"},
		{staticPart, ""},
	}},
	{"/{foo}/{bar}", []part{
		{variablePart, "foo"},
		{variablePart, "bar"},
	}},
	{"/{foo}/{bar}/", []part{
		{variablePart, "foo"},
		{variablePart, "bar"},
		{staticPart, ""},
	}},
	// wildcard
	{"/{*}", []part{
		{wildcardPart, ""},
	}},
	{"/foo/{*}", []part{
		{staticPart, "foo"},
		{wildcardPart, ""},
	}},
	{"/foo/{bar}/{*}", []part{
		{staticPart, "foo"},
		{variablePart, "bar"},
		{wildcardPart, ""},
	}},
	{"/foo%2fbar", []part{
		{staticPart, "foo/bar"},
	}},
	{"/%E4%B8%96%E7%95%8C", []part{
		{staticPart, "世界"},
	}},
	{"/%25", []part{
		{staticPart, "%"},
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
	{"/foo/{*}/", nil}, // wildcard in bad place
	{"/{*name}", nil},  // invalid variable name
	{"/{1name}", nil},  // invalid variable name
	{"junk", nil},      // path does not start with /
	{"/%2x", nil},      // invalid percent encoding
	{"/%2", nil},       // invalid percent encoding
}

func TestParsePaths(t *testing.T) {
	for _, v := range parserTests {
		parts, err := parse(v.pattern, '/')
		if err == nil {
			if !equalParts(v.parts, parts) {
				t.Errorf("%q: expected %v; got %v", v.pattern, v.parts, parts)
			}
		} else if v.parts != nil {
			t.Errorf("%q: expected %v; failed to parse with error %q", v.pattern, v.parts, err)
		}
	}
}

func equalParts(p1, p2 []part) bool {
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
