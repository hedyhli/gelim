package main

import (
	"errors"
	"testing"
)

type ParseMetaResult struct {
	mediatype string
	params    map[string]string
	err       error
}

func TestParseMeta(t *testing.T) {
	var tests = []struct {
		meta string
		res  ParseMetaResult
	}{
		{"", ParseMetaResult{"text/gemini", map[string]string{"charset": "utf-8"}, nil}},
		{"text/gemini;", ParseMetaResult{"text/gemini", make(map[string]string), nil}},
		{"text/gemini; lang=en", ParseMetaResult{"text/gemini", map[string]string{"lang": "en"}, nil}},
		{"text/gemini; charset=utf-8;", ParseMetaResult{"text/gemini", map[string]string{"charset": "utf-8"}, nil}},
		{"text/plain; charset=utf-8;", ParseMetaResult{"text/plain", map[string]string{"charset": "utf-8"}, nil}},
		{"text/gemini; lang=es;", ParseMetaResult{"text/gemini", map[string]string{"lang": "es"}, nil}},
		{"text/plain; charset=utf-8;lang=ru", ParseMetaResult{"text/plain", map[string]string{"charset": "utf-8", "lang": "ru"}, nil}},
		{"foobar", ParseMetaResult{"foobar", make(map[string]string), nil}},
		{"foo=;", ParseMetaResult{"", make(map[string]string), errors.New("mime: expected slash after first token")}},
		{";", ParseMetaResult{"", make(map[string]string), errors.New("mime: no media type")}},
		{"application/octet-stream", ParseMetaResult{"application/octet-stream", make(map[string]string), nil}},
	}

	for _, test := range tests {
		mediatype, params, err := ParseMeta(test.meta)
		if mediatype != test.res.mediatype {
			t.Errorf("parseMeta(%q) mediatype = %q, want %q", test.meta, mediatype, test.res.mediatype)
		}
		for k, v := range test.res.params {
			if params[k] != v {
				t.Errorf("parseMeta(%q) params[%q] = %q, want %q", test.meta, k, params[k], v)
			}
		}
		if err == nil && test.res.err == nil {
			continue
		}
		if (err != nil && test.res.err == nil) || (err.Error() != test.res.err.Error()) {
			t.Errorf("parseMeta(%q) err = %q, want %q", test.meta, err, test.res.err)
		}
	}
}
