package util

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"golang.org/x/net/html"
)

var (
	headClient    = newHeadClient()
	httpClient    = &http.Client{Timeout: 30 * time.Second}
	urlPattern    = regexp.MustCompile(`https?://[^\s"'<>\\]+`)
	semverPattern = regexp.MustCompile(`(?i)\b\d+\.\d+(?:\.\d+)?(?:[-+][0-9a-z.-]+)?\b`)
)

const (
	defaultUserAgent     = "arrans-overlay-workflow-builder/1.0"
	defaultLinkExtension = "AppImage"
)

func newHeadClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ForceAttemptHTTP2 = false
	transport.TLSNextProto = make(map[string]func(string, *tls.Conn) http.RoundTripper)
	return &http.Client{
		Timeout:   15 * time.Second,
		Transport: transport,
	}
}

// AppImageLink represents a discovered AppImage hyperlink.
type AppImageLink struct {
	// Href is the raw attribute value as seen in the HTML.
	Href string
	// URL is the fully resolved absolute URL that can be downloaded. When
	// redirects are encountered this will contain the final destination.
	URL string
	// Filename is the last path element of the resolved URL.
	Filename string
}

// FindAppImageLink returns the latest AppImage link discovered on the page.
func FindAppImageLink(pageURL string) (string, error) {
	links, err := FindAppImageLinks(pageURL, "")
	if err != nil {
		return "", err
	}
	if len(links) == 0 {
		return "", fmt.Errorf("no AppImage link found")
	}
	return links[len(links)-1].URL, nil
}

// FindAppImageLinks returns all AppImage links on the provided page sorted in
// version-aware order. When matchExpr is provided it is treated as a case
// insensitive regular expression applied to the raw href attribute.
func FindAppImageLinks(pageURL, matchExpr string) ([]AppImageLink, error) {
	return FindAppImageLinksWithExtension(pageURL, matchExpr, "AppImage")
}

// FindAppImageLinksWithExtension behaves like FindAppImageLinks but allows
// callers to override the file extension matched on the source page.
func FindAppImageLinksWithExtension(pageURL, matchExpr, extension string) ([]AppImageLink, error) {
	log.Printf("fetching download page %s", pageURL)

	req, err := http.NewRequest(http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("User-Agent", defaultUserAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", pageURL, err)
	}
	defer resp.Body.Close()

	log.Printf("%s -> %s", pageURL, resp.Status)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading body: %w", err)
	}

	log.Printf("read %d bytes from %s", len(body), pageURL)

	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parsing HTML: %w", err)
	}

	matchHref, err := compileLinkMatcher(matchExpr, extension)
	if err != nil {
		return nil, err
	}

	var links []AppImageLink
	seen := map[string]struct{}{}
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key != "href" {
					continue
				}
				if !matchHref(a.Val) {
					continue
				}
				resolved, err := resolveHref(pageURL, a.Val)
				if err != nil {
					log.Printf("failed to resolve href %q from %s: %v", a.Val, pageURL, err)
					continue
				}
				abs := resolved.String()
				if _, ok := seen[abs]; ok {
					continue
				}
				seen[abs] = struct{}{}
				log.Printf("following link candidate href=%q resolved=%s", a.Val, abs)
				finalURL, finalName := followRedirect(abs)
				if !strings.HasSuffix(strings.ToLower(finalName), ".appimage") {
					log.Printf("discarding %s because resolved filename %q is not an AppImage", abs, finalName)
					continue
				}
				log.Printf("accepted AppImage link %s -> %s", abs, finalURL)
				links = append(links, AppImageLink{
					Href:     a.Val,
					URL:      finalURL,
					Filename: finalName,
				})
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(doc)

	if len(links) == 0 {
		log.Printf("no AppImage links discovered in anchors, scanning text for fallbacks")
		links = append(links, extractLinksFromText(string(body), matchHref, seen)...)
	}

	links = preferSemanticVersionLinks(links)

	log.Printf("discovered %d candidate AppImage links before sorting", len(links))

	sort.Slice(links, func(i, j int) bool {
		return versionLess(links[i].Filename, links[j].Filename)
	})

	if len(links) > 0 {
		log.Printf("latest AppImage candidate %s", links[len(links)-1].Filename)
	}

	return links, nil
}

func compileLinkMatcher(matchExpr, extension string) (func(string) bool, error) {
	if matchExpr != "" {
		compiled, err := regexp.Compile("(?i)" + matchExpr)
		if err != nil {
			return nil, fmt.Errorf("compiling match expression: %w", err)
		}
		return compiled.MatchString, nil
	}

	ext := strings.TrimPrefix(extension, ".")
	if ext == "" {
		ext = defaultLinkExtension
	}
	lowerExt := strings.ToLower(ext)

	return func(href string) bool {
		trimmed := strings.TrimSpace(href)
		if trimmed == "" {
			return false
		}
		if idx := strings.IndexAny(trimmed, "?#"); idx >= 0 {
			trimmed = trimmed[:idx]
		}
		if parsed, err := url.Parse(trimmed); err == nil && parsed.Path != "" {
			linkExt := strings.ToLower(strings.TrimPrefix(path.Ext(parsed.Path), "."))
			if linkExt == lowerExt && linkExt != "" {
				return true
			}
			trimmed = parsed.Path
		}
		lower := strings.ToLower(trimmed)
		if strings.HasSuffix(lower, "."+lowerExt) {
			return true
		}
		return strings.Contains(lower, "."+lowerExt)
	}, nil
}

func extractLinksFromText(body string, matchHref func(string) bool, seen map[string]struct{}) []AppImageLink {
	var links []AppImageLink
	matches := urlPattern.FindAllString(body, -1)
	log.Printf("fallback matcher discovered %d raw URLs in page text", len(matches))
	for _, candidate := range matches {
		if !matchHref(candidate) {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		log.Printf("following fallback link %s", candidate)
		finalURL, finalName := followRedirect(candidate)
		if !strings.HasSuffix(strings.ToLower(finalName), ".appimage") {
			log.Printf("fallback link %s resolved to %q which is not an AppImage", candidate, finalName)
			continue
		}
		log.Printf("accepted fallback AppImage link %s -> %s", candidate, finalURL)
		links = append(links, AppImageLink{
			Href:     candidate,
			URL:      finalURL,
			Filename: finalName,
		})
	}
	return links
}

func preferSemanticVersionLinks(links []AppImageLink) []AppImageLink {
	if len(links) == 0 {
		return links
	}

	var semverLinks []AppImageLink
	for _, link := range links {
		if hasSemanticVersion(link.Filename) {
			semverLinks = append(semverLinks, link)
		}
	}

	if len(semverLinks) == 0 {
		return links
	}

	if len(semverLinks) != len(links) {
		log.Printf("filtered out %d non-semantic version downloads", len(links)-len(semverLinks))
	}

	return semverLinks
}

func hasSemanticVersion(name string) bool {
	if name == "" {
		return false
	}
	lower := strings.ToLower(name)
	if strings.Contains(lower, "nightly") || strings.Contains(lower, "snapshot") {
		return false
	}
	return semverPattern.FindString(name) != ""
}

func followRedirect(rawURL string) (string, string) {
	log.Printf("issuing HEAD request to resolve final URL for %s", rawURL)
	req, err := http.NewRequest(http.MethodHead, rawURL, nil)
	if err != nil {
		log.Printf("failed to construct HEAD request for %s: %v", rawURL, err)
		return fallbackURL(rawURL)
	}
	req.Header.Set("User-Agent", defaultUserAgent)

	resp, err := headClient.Do(req)
	if err != nil {
		log.Printf("HEAD request for %s failed: %v", rawURL, err)
		return fallbackURL(rawURL)
	}
	defer resp.Body.Close()
	final := resp.Request.URL
	if resp.StatusCode >= 400 || final == nil {
		log.Printf("HEAD request for %s returned status %d, falling back", rawURL, resp.StatusCode)
		return fallbackURL(rawURL)
	}
	name := path.Base(final.Path)
	if name == "" || name == "/" {
		_, fallbackName := fallbackURL(rawURL)
		name = fallbackName
	}
	return final.String(), name
}

func fallbackURL(rawURL string) (string, string) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		log.Printf("fallbackURL: unable to parse %s: %v", rawURL, err)
		return rawURL, path.Base(rawURL)
	}
	name := path.Base(parsed.Path)
	if name == "" || name == "/" {
		return rawURL, path.Base(rawURL)
	}
	return rawURL, name
}

func resolveHref(pageURL, href string) (*url.URL, error) {
	base, err := url.Parse(pageURL)
	if err != nil {
		return nil, fmt.Errorf("parsing base url: %w", err)
	}
	u, err := url.Parse(href)
	if err != nil {
		return nil, fmt.Errorf("parsing href: %w", err)
	}
	resolved := base.ResolveReference(u)
	return resolved, nil
}

type versionSegment struct {
	digits bool
	value  string
}

func splitVersionSegments(s string) []versionSegment {
	var segments []versionSegment
	current := strings.Builder{}
	digitMode := false
	started := false
	for _, r := range s {
		isDigit := unicode.IsDigit(r)
		if !started {
			digitMode = isDigit
			started = true
		}
		if isDigit != digitMode {
			segments = append(segments, versionSegment{digits: digitMode, value: current.String()})
			current.Reset()
			digitMode = isDigit
		}
		if digitMode {
			current.WriteRune(r)
		} else {
			current.WriteRune(unicode.ToLower(r))
		}
	}
	if current.Len() > 0 {
		segments = append(segments, versionSegment{digits: digitMode, value: current.String()})
	}
	return segments
}

func versionLess(a, b string) bool {
	sa := splitVersionSegments(a)
	sb := splitVersionSegments(b)
	for i := 0; i < len(sa) && i < len(sb); i++ {
		segA := sa[i]
		segB := sb[i]
		switch {
		case segA.digits && segB.digits:
			cmp := compareNumericSegment(segA.value, segB.value)
			if cmp != 0 {
				return cmp < 0
			}
		case segA.digits:
			return true
		case segB.digits:
			return false
		default:
			if segA.value != segB.value {
				return segA.value < segB.value
			}
		}
	}
	return len(sa) < len(sb)
}

func compareNumericSegment(a, b string) int {
	aTrim := strings.TrimLeft(a, "0")
	bTrim := strings.TrimLeft(b, "0")
	if aTrim == "" {
		aTrim = "0"
	}
	if bTrim == "" {
		bTrim = "0"
	}
	if len(aTrim) != len(bTrim) {
		if len(aTrim) < len(bTrim) {
			return -1
		}
		return 1
	}
	if aTrim != bTrim {
		if aTrim < bTrim {
			return -1
		}
		return 1
	}
	if len(a) != len(b) {
		if len(a) < len(b) {
			return -1
		}
		return 1
	}
	return 0
}
