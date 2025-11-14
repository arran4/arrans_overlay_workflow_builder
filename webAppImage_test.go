package arrans_overlay_workflow_builder

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGenerateWebAppImageConfigEntry(t *testing.T) {
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

	ic, links, err := GenerateWebAppImageConfigEntry(ts.URL, "", "")
	if err != nil {
		t.Fatalf("GenerateWebAppImageConfigEntry returned error: %v", err)
	}
	if ic.Type != "Web AppImage" {
		t.Fatalf("unexpected type %q", ic.Type)
	}
	if ic.DownloadPageUrl != ts.URL {
		t.Fatalf("unexpected download page url %q", ic.DownloadPageUrl)
	}
	if ic.DownloadMatch == "" {
		t.Fatalf("expected download match to be set")
	}
	if ic.EbuildName != "beeper-appimage.ebuild" {
		t.Fatalf("unexpected ebuild name %q", ic.EbuildName)
	}
	prog, ok := ic.Programs["beeper"]
	if !ok {
		t.Fatalf("program 'beeper' missing")
	}
	binaries := prog.Binary["~amd64"]
	if len(binaries) != 2 {
		t.Fatalf("expected two binary entries, got %d", len(binaries))
	}
	if binaries[0] != "Beeper-${VERSION}.AppImage" {
		t.Fatalf("unexpected binary pattern %q", binaries[0])
	}
	if binaries[1] != "beeper.AppImage" {
		t.Fatalf("unexpected installed name %q", binaries[1])
	}
	if len(links) != 2 {
		t.Fatalf("expected 2 links, got %d", len(links))
	}
}

func TestGenerateWebAppImageConfigEntryWithExtension(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			_, err := fmt.Fprint(w, `<html><body>
                                <a href="/download/com.automattic.beeper.desktop">Beeper</a>
                        </body></html>`)
			if err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		case "/download/com.automattic.beeper.desktop":
			http.Redirect(w, r, "/builds/Beeper-4.1.276-arm64.AppImage", http.StatusFound)
		case "/builds/Beeper-4.1.276-arm64.AppImage":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	ic, links, err := GenerateWebAppImageConfigEntry(ts.URL, "", "desktop")
	if err != nil {
		t.Fatalf("GenerateWebAppImageConfigEntry returned error: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("expected one link, got %d", len(links))
	}
	if ic.DownloadMatch == "" {
		t.Fatalf("expected download match to be set for extension override")
	}
	lowerMatch := strings.ToLower(ic.DownloadMatch)
	if !strings.Contains(lowerMatch, "desktop$") {
		t.Fatalf("expected download match to target desktop files, got %q", ic.DownloadMatch)
	}
	if strings.Contains(lowerMatch, "appimage") {
		t.Fatalf("expected download match to avoid appimage pattern, got %q", ic.DownloadMatch)
	}
	prog := ic.Programs["beeper"]
	binaries := prog.Binary["~amd64"]
	if binaries[0] != "Beeper-${VERSION}.AppImage" {
		t.Fatalf("unexpected binary pattern %q", binaries[0])
	}
}
