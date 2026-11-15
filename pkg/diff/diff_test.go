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

func TestIsIgnoredPath(t *testing.T) {
	cases := []struct {
		ignore []string
		name   string
		want   bool
	}{
		{nil, "var/log/dpkg.log", false},
		{[]string{"var/log/dpkg.log"}, "var/log/dpkg.log", true},
		{[]string{"/var/log/dpkg.log"}, "var/log/dpkg.log", true},
		{[]string{"./var/log/dpkg.log"}, "var/log/dpkg.log", true},
		{[]string{"var/log/dpkg.log"}, "/var/log/dpkg.log", true},
		{[]string{"var/log/dpkg.log"}, "./var/log/dpkg.log", true},
		{[]string{"/var/log"}, "var/log", true},
		{[]string{"/var/log"}, "var/log/", true},
		{[]string{"/var/log/"}, "var/log", true},
		{[]string{"/var/log"}, "var/log/apt/history.log", true},
		{[]string{"/var/log"}, "var/logfoo", false},
		{[]string{"/var/log"}, "var", false},
		{[]string{"/var/log/dpkg.log"}, "var/log/dpkg.log.1", false},
		{[]string{"/tmp", "/var/log"}, "var/log/dpkg.log", true},
		{[]string{"/tmp", "/var/log"}, "etc/passwd", false},
	}
	for _, c := range cases {
		if got := isIgnoredPath(c.ignore, c.name); got != c.want {
			t.Errorf("isIgnoredPath(%q, %q) = %v, want %v", c.ignore, c.name, got, c.want)
		}
	}
}
