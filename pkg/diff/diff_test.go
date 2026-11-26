package diff

import "testing"

func TestCanonicalPath(t *testing.T) {
	cases := map[string]string{
		"":             "",
		"/":            ".",
		"./":           ".",
		".":            ".",
		"foo":          "foo",
		"/foo":         "foo",
		"./foo":        "foo",
		"foo/":         "foo/",
		"/foo/bar":     "foo/bar",
		"./foo/bar/":   "foo/bar/",
		"../foo":       "../foo",
		".hidden":      ".hidden",
		"/.hidden/foo": ".hidden/foo",
	}
	for in, want := range cases {
		if got := canonicalPath(in); got != want {
			t.Errorf("canonicalPath(%q) = %q, want %q", in, got, want)
		}
	}
}
