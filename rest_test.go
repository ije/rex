package rex

import (
	"testing"
)

func TestMatch(t *testing.T) {
	testMatch(t, "/", "/", true)
	testMatch(t, "/", "/foo", false)
	testMatch(t, "/foo/*", "/bar", false)
	testMatch(t, "/foo/*", "/foo", false)

	p := testMatch(t, "/*", "/foo", true)
	if p.Params["*"] != "foo" {
		t.Fatalf("param '*' not match, want: %s, got: %s", "foo", p.Params["*"])
	}

	p = testMatch(t, "/*", "/foo/bar", true)
	if p.Params["*"] != "foo/bar" {
		t.Fatalf("param '*' not match, want: %s, got: %s", "foo/bar", p.Params["*"])
	}

	p = testMatch(t, "/foo/*", "/foo/bar", true)
	if p.Params["*"] != "bar" {
		t.Fatalf("param '*' not match, want: %s, got: %s", "bar", p.Params["*"])
	}

	p = testMatch(t, "/foo/:key", "/foo/bar", true)
	if p.Params["key"] != "bar" {
		t.Fatalf("param 'key' not match, want: %s, got: %s", "bar", p.Params["key"])
	}
}

func testMatch(t *testing.T, pattern string, path string, ok bool) *Path {
	ctx := &Context{
		Path: &Path{
			raw:      path,
			segments: splitPath(path),
			Params:   Params{},
		},
	}
	if match(splitPath(pattern), ctx) != ok {
		t.Fatalf("match failed, pattern: %s, path: %s", pattern, path)
	}
	return ctx.Path
}
