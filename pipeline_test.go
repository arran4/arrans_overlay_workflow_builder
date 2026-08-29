package arrans_overlay_workflow_builder

import (
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
)

func TestPythonPipeline(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rss":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<rss><channel><item><link>http://example.com/v1.0.tar.gz</link></item></channel></rss>`))
		case "/atom":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<feed xmlns="http://www.w3.org/2005/Atom"><entry><link href="http://example.com/v2.0.tar.gz"/></entry></feed>`))
		case "/json":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"releases": [{"tag": "v2.0"}]}`))
		case "/html":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<html><body><a href="relative.tar.gz">link</a><a href="other.tar.gz">link2</a></body></html>`))
		case "/xml":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<root><link>http://example.com/v3.0.tar.gz</link></root>`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	tests := []struct {
		name     string
		pipeline string
		expected string
		fail     bool
	}{
		{"RSS", "get(" + ts.URL + "/rss) | rss | first | link | url.basename | regex(v(.*)\\.tar\\.gz)", "1.0", false},
		{"Atom", "get(" + ts.URL + "/atom) | atom | first | link | url.basename | regex(v(.*)\\.tar\\.gz)", "2.0", false},
		{"JSON", "get(" + ts.URL + "/json) | json(releases.0.tag) | replace('v', '')", "2.0", false},
		{"HTML Links", "get(" + ts.URL + "/html) | html_links | first", ts.URL + "/relative.tar.gz", false},
		{"XML XPath", "get(" + ts.URL + "/xml) | xml | xpath(.//link) | first | regex(v(.*)\\.tar\\.gz)", "3.0", false},
		{"Replace comma", "get(" + ts.URL + "/rss) | replace('tar.gz', 'zip')", "<rss><channel><item><link>http://example.com/v1.0.zip</link></item></channel></rss>", false},
		{"Replace spaces after comma", "get(" + ts.URL + "/rss) | replace('tar.gz',  'zip')", "<rss><channel><item><link>http://example.com/v1.0.zip</link></item></channel></rss>", false},
		{"Replace commas inside args", "get(" + ts.URL + "/json) | replace('v2.0', 'v3.0,v') | json(releases.0.tag)", "v3.0,v", false},
		{"Replace single and double quotes", "get(" + ts.URL + "/json) | replace('\"tag\": \"v2.0\"', '\"tag\": \"v3.0\"') | json(releases.0.tag)", "v3.0", false},
		{"Replace escaped quotes", "get(" + ts.URL + "/json) | json(releases.0.tag) | replace('v2.0', 'v\\'3.0')", "v'3.0", false},
		{"Replace backslashes", "get(" + ts.URL + "/json) | json(releases.0.tag) | replace('v2', 'v\\\\2')", "v\\2.0", false},
		{"Replace empty string", "get(" + ts.URL + "/json) | replace('v', '') | json(releases.0.tag)", "2.0", false},
		{"Fail Replace less args", "get(" + ts.URL + "/rss) | replace('a')", "", true},
		{"Fail Replace more args", "get(" + ts.URL + "/rss) | replace('a', 'b', 'c')", "", true},
		{"Fail Replace non-string", "get(" + ts.URL + "/rss) | replace('a', 1)", "", true},
		{"Fail Replace unterminated quotes", "get(" + ts.URL + "/rss) | replace('a', 'b)", "", true},

		{"Regex alternation", "get(" + ts.URL + "/html) | html_links | regex(((?:relative|other)\\.tar\\.gz)) | last", "other.tar.gz", false},
		{"Regex escaped paren", "get(" + ts.URL + "/json) | replace('v2.0', '(v2.0)') | regex(\\((.*?)\\))", "v2.0", false},
		{"Replace spaces and quotes", "get(" + ts.URL + "/json) | replace('\"tag\": \"v2.0\"', '\"tag\": \"v3.0\"') | json(releases.0.tag)", "v3.0", false},
		{"Fail 404", "get(" + ts.URL + "/404) | trim", "", true},
		{"Fail Unknown", "unknown_cmd", "", true},
		{"Fail Empty", "get(" + ts.URL + "/rss) | ", "", true},
		{"Fail Unbalanced", "get(" + ts.URL + "/rss | trim", "", true},
		{"Fail Dangling Escape", "get(" + ts.URL + "/rss) | regex(v\\\\", "", true},
		{"Fail Replace Args", "get(" + ts.URL + "/rss) | replace('a')", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("python3", "templates/_partials/pipeline.py", tt.pipeline)
			output, err := cmd.CombinedOutput()
			outStr := strings.TrimSpace(string(output))

			if tt.fail {
				if err == nil {
					t.Errorf("Expected failure for %q but it succeeded. Output: %q", tt.pipeline, outStr)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected failure for %q: %v. Output: %q", tt.pipeline, err, outStr)
				return
			}

			if outStr != tt.expected {
				t.Errorf("For %q expected %q but got %q", tt.pipeline, tt.expected, outStr)
			}
		})
	}
}
