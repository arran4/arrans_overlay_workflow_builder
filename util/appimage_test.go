package util

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFindAppImageLinks(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, err := fmt.Fprint(w, `<html><body>
                        <a href="/builds/Beeper-4.1.9.AppImage">Old</a>
                        <a href="/builds/Beeper-4.1.10.AppImage">New</a>
                        <a href="/other/Tool-1.0.AppImage">Other</a>
                </body></html>`)
		if err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer ts.Close()

	links, err := FindAppImageLinks(ts.URL, "")
	if err != nil {
		t.Fatalf("FindAppImageLinks returned error: %v", err)
	}
	if len(links) != 3 {
		t.Fatalf("expected 3 links, got %d", len(links))
	}
	filtered, err := FindAppImageLinks(ts.URL, "Beeper-")
	if err != nil {
		t.Fatalf("FindAppImageLinks with match returned error: %v", err)
	}
	if len(filtered) != 2 {
		t.Fatalf("expected 2 filtered links, got %d", len(filtered))
	}
	if filtered[1].URL != ts.URL+"/builds/Beeper-4.1.10.AppImage" {
		t.Fatalf("expected latest Beeper link to be 4.1.10, got %q", filtered[1].URL)
	}
}

func TestFindAppImageLinkLatest(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, err := fmt.Fprint(w, `<html><body>
                        <a href="/builds/Beeper-4.1.135.AppImage">Download</a>
                        <a href="/builds/Beeper-4.1.140.AppImage">Download</a>
                </body></html>`)
		if err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer ts.Close()

	link, err := FindAppImageLink(ts.URL)
	if err != nil {
		t.Fatalf("FindAppImageLink returned error: %v", err)
	}
	expected := ts.URL + "/builds/Beeper-4.1.140.AppImage"
	if link != expected {
		t.Fatalf("expected %q, got %q", expected, link)
	}
}

func TestFindAppImageLinksWithExtensionRedirect(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			_, err := fmt.Fprint(w, `<html><body>
                                <a href="/download/BeeperSetup.exe">Windows</a>
                                <a href="/download/com.automattic.beeper.desktop">Download</a>
                        </body></html>`)
			if err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		case "/download/BeeperSetup.exe":
			http.Redirect(w, r, "/builds/BeeperSetup.exe", http.StatusFound)
		case "/builds/BeeperSetup.exe":
			w.WriteHeader(http.StatusOK)
		case "/download/com.automattic.beeper.desktop":
			http.Redirect(w, r, "/builds/Beeper-4.1.276-arm64.AppImage", http.StatusFound)
		case "/builds/Beeper-4.1.276-arm64.AppImage":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	links, err := FindAppImageLinksWithExtension(ts.URL, "", "desktop")
	if err != nil {
		t.Fatalf("FindAppImageLinksWithExtension returned error: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(links))
	}
	expectedURL := ts.URL + "/builds/Beeper-4.1.276-arm64.AppImage"
	if links[0].URL != expectedURL {
		t.Fatalf("expected resolved URL %q, got %q", expectedURL, links[0].URL)
	}
	if links[0].Filename != "Beeper-4.1.276-arm64.AppImage" {
		t.Fatalf("expected filename to be AppImage, got %q", links[0].Filename)
	}
}

func TestFindAppImageLinksWithExtensionQueryAndDotPrefix(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			_, err := fmt.Fprint(w, `<html><body>
                                <a href="/download/com.automattic.beeper.desktop?channel=stable">Download</a>
                        </body></html>`)
			if err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		case "/download/com.automattic.beeper.desktop":
			if r.URL.RawQuery != "channel=stable" {
				t.Fatalf("unexpected query %q", r.URL.RawQuery)
			}
			http.Redirect(w, r, "/builds/Beeper-4.1.300.AppImage", http.StatusFound)
		case "/builds/Beeper-4.1.300.AppImage":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	links, err := FindAppImageLinksWithExtension(ts.URL, "", ".desktop")
	if err != nil {
		t.Fatalf("FindAppImageLinksWithExtension returned error: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(links))
	}
	if links[0].URL != ts.URL+"/builds/Beeper-4.1.300.AppImage" {
		t.Fatalf("unexpected resolved URL %q", links[0].URL)
	}
	if links[0].Filename != "Beeper-4.1.300.AppImage" {
		t.Fatalf("unexpected filename %q", links[0].Filename)
	}
}

func TestFindAppImageLinksWithExtensionScriptFallback(t *testing.T) {
	var ts *httptest.Server
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			page := fmt.Sprintf(`<html><body><script>var data={"platforms":[{"key":"windows","binaries":{"x64":{"productionURL":{"url":"%s/download/windows.desktop","type":"download"}}}},{"key":"linux","binaries":{"x64":{"productionURL":{"url":"%s/download/linux.desktop","type":"download"}}}}]};</script></body></html>`, ts.URL, ts.URL)
			_, err := fmt.Fprint(w, page)
			if err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		case "/download/windows.desktop":
			http.Redirect(w, r, "/builds/windows.exe", http.StatusFound)
		case "/builds/windows.exe":
			w.WriteHeader(http.StatusOK)
		case "/download/linux.desktop":
			http.Redirect(w, r, "/builds/Linux-2.0-x86_64.AppImage", http.StatusFound)
		case "/builds/Linux-2.0-x86_64.AppImage":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	links, err := FindAppImageLinksWithExtension(ts.URL, "", "desktop")
	if err != nil {
		t.Fatalf("FindAppImageLinksWithExtension returned error: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(links))
	}
	if links[0].URL != ts.URL+"/builds/Linux-2.0-x86_64.AppImage" {
		t.Fatalf("unexpected resolved URL %q", links[0].URL)
	}
	if links[0].Filename != "Linux-2.0-x86_64.AppImage" {
		t.Fatalf("unexpected filename %q", links[0].Filename)
	}
}

func TestFindAppImageLinksPrefersSemanticVersions(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, err := fmt.Fprint(w, `<html><body>
                        <a href="/builds/Beeper-nightly.AppImage">Nightly</a>
                        <a href="/builds/Beeper-4.1.400.AppImage">Stable</a>
                </body></html>`)
		if err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer ts.Close()

	links, err := FindAppImageLinks(ts.URL, "")
	if err != nil {
		t.Fatalf("FindAppImageLinks returned error: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("expected 1 filtered link, got %d", len(links))
	}
	if links[0].Filename != "Beeper-4.1.400.AppImage" {
		t.Fatalf("expected stable AppImage, got %q", links[0].Filename)
	}
}

func TestFindAppImageLinksKeepsNightlyWhenOnlyNightly(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body>
                        <a href="/builds/Beeper-nightly.AppImage">Nightly</a>
                </body></html>`)
	}))
	defer ts.Close()

	links, err := FindAppImageLinks(ts.URL, "")
	if err != nil {
		t.Fatalf("FindAppImageLinks returned error: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(links))
	}
	if links[0].Filename != "Beeper-nightly.AppImage" {
		t.Fatalf("expected nightly AppImage, got %q", links[0].Filename)
	}
}
