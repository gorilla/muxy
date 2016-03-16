package mpath

import (
	"testing"
)

type parseTest struct {
	pattern string
	parts   []string
}

var parseTests = []parseTest{
	// static
	{"/", []string{""}},
	{"/foo", []string{"foo"}},
	{"/foo/", []string{"foo", ""}},
	{"/foo/bar", []string{"foo", "bar"}},
	{"/foo/bar/", []string{"foo", "bar", ""}},
	// variable
	{"/:foo", []string{":foo"}},
	{"/:foo/", []string{":foo", ""}},
	{"/:foo/:bar", []string{":foo", ":bar"}},
	{"/:foo/:bar/", []string{":foo", ":bar", ""}},
	// wildcard
	{"/*", []string{"*"}},
	{"/foo/*", []string{"foo", "*"}},
	{"/foo/:bar/*", []string{"foo", ":bar", "*"}},
	// percent encodings
	//{"/foo%2Fbar", {"foo/bar"}},
	//{"/%E4%B8%96%E7%95%8C", {"世界"}},
	//{"/%25", {"%"}},
	//{"/%25/", {"%", ""}},
	// parsing errors
	{"/*/", nil},     // invalid wildcard
	{"/*name", nil},  // invalid wildcard
	{"/:1name", nil}, // invalid variable name
	//{"/%2x", nil},    // invalid percent encoding
	//{"/%2", nil},     // invalid percent encoding
}

func TestParse(t *testing.T) {
	for _, v := range parseTests {
		parts, err := parse(v.pattern)
		if (err == nil) != (parts != nil) {
			if err == nil {
				t.Errorf("Expected error parsing %q", v.pattern)
			} else {
				t.Errorf("%q: expected %v; failed to parse with error %q", v.pattern, v.parts, err)
			}
		} else if err == nil {
			if !equalParts(v.parts, parts) {
				t.Errorf("%q: expected %v; got %v", v.pattern, v.parts, parts)
			}
		}
	}
}

func TestRoute(t *testing.T) {
}

func TestMatch(t *testing.T) {
}

func TestBuild(t *testing.T) {
}

func equalParts(p1, p2 []string) bool {
	if len(p1) != len(p2) {
		return false
	}
	for k, v := range p1 {
		if v != p2[k] {
			return false
		}
	}
	return true
}
