package main

import (
	"net/url"
	"testing"
)

func mustParse(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

func TestBuildPrompt(t *testing.T) {
	var tests = []struct {
		u    *url.URL
		conf string
		res  string
	}{
		{
			mustParse("gemini://example.org/a/b?query"),
			"%U %%",
			"gemini://example.org/a/b %",
		},
		{
			mustParse("gemini://example.org/a/b?query"),
			"%u %z",
			"example.org/a/b %z",
		},
		{
			mustParse("gemini://example.org/a/b?query"),
			"%P",
			"/a/b",
		},
		{
			mustParse("gemini://example.org/a/b?query"),
			"%p",
			"b",
		},
		{
			mustParse("gemini://example.org/a/b?query"),
			"%a ><$^&*()+-_{}[]|\\!`~",
			"%a ><$^&*()+-_{}[]|\\!`~",
		},
		{
			mustParse("gemini://example.org/"),
			"%p",
			"/",
		},
		{
			mustParse("gemini://x.example.org:9000/"),
			"%H %h",
			"x.example.org:9000 x.example.org",
		},
	}

	for _, test := range tests {
		p := BuildPrompt(test.u, test.conf)
		if p != test.res {
			t.Errorf("buildPrompt(%q, %q)\ngot  = %q\nwant = %q", test.u.String(), test.conf, p, test.res)
		}
	}
}
