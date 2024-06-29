package arrans_overlay_workflow_builder

import (
	"bufio"
	"fmt"
	"io"
	"net/url"
	"strings"
	"unicode"
)

// InputConfig represents a single configuration entry.
type InputConfig struct {
	EntryNumber       int
	Type              string
	GithubProjectUrl  string
	DesktopFile       string
	InstalledFilename string
	Category          string
	EbuildName        string
	Description       string
	Homepage          string
}

const (
	DefaultCategory = "app-misc"
)

// String serializes the InputConfig struct back into the configuration file format.
func (c *InputConfig) String() string {
	var sb strings.Builder

	if c.Type != "" {
		sb.WriteString(fmt.Sprintf("Type %s\n", c.Type))
	}
	if c.GithubProjectUrl != "" {
		sb.WriteString(fmt.Sprintf("GithubProjectUrl %s\n", c.GithubProjectUrl))
	}
	if c.DesktopFile != "" {
		sb.WriteString(fmt.Sprintf("DesktopFile %s\n", c.DesktopFile))
	}
	if c.InstalledFilename != "" {
		sb.WriteString(fmt.Sprintf("InstalledFilename %s\n", c.InstalledFilename))
	}
	if c.Category != "" {
		sb.WriteString(fmt.Sprintf("Category %s\n", c.Category))
	}
	if c.EbuildName != "" {
		sb.WriteString(fmt.Sprintf("EbuildName %s\n", c.EbuildName))
	}
	if c.Description != "" {
		sb.WriteString(fmt.Sprintf("Description %s\n", c.Description))
	}
	if c.Homepage != "" {
		sb.WriteString(fmt.Sprintf("Homepage %s\n", c.Homepage))
	}

	return sb.String()
}

// ParseInputConfigFile parses the given configuration file and returns a slice of InputConfig structures.
func ParseInputConfigFile(file io.Reader) ([]*InputConfig, error) {
	var configs []*InputConfig
	var currentConfig *InputConfig
	var parseFields map[string]*string
	scanner := bufio.NewScanner(file)
	entryNumber := 0
	breakCount := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") {
			continue
		}

		if line == "" {
			if breakCount < 0 || currentConfig == nil {
				continue
			}
			breakCount++
			var err error
			configs, err = SanitizeAndAppendInputConfig(currentConfig, configs)
			if err != nil {
				return nil, fmt.Errorf("sanitiization issue with %d: %w", currentConfig.EntryNumber, err)
			}
			currentConfig = nil
			continue
		}

		if currentConfig == nil {
			currentConfig = &InputConfig{
				Category:    DefaultCategory,
				EntryNumber: entryNumber,
			}
			entryNumber++
			parseFields = map[string]*string{
				"Type":              &currentConfig.Type,
				"GithubProjectUrl":  &currentConfig.GithubProjectUrl,
				"InstalledFilename": &currentConfig.InstalledFilename,
				"DesktopFile":       &currentConfig.DesktopFile,
				"Category":          &currentConfig.Category,
				"EbuildName":        &currentConfig.EbuildName,
				"Description":       &currentConfig.Description,
				"Homepage":          &currentConfig.Homepage,
			}
		}

		matched := false
		for prefix, field := range parseFields {
			if strings.HasPrefix(line, prefix) {
				withoutPrefix := strings.TrimPrefix(line, prefix)
				if withoutPrefix != "" && !unicode.IsSpace(rune(withoutPrefix[0])) {
					continue
				}
				for len(withoutPrefix) > 0 && unicode.IsSpace(rune(withoutPrefix[0])) {
					withoutPrefix = withoutPrefix[1:]
				}
				value := strings.TrimSpace(withoutPrefix)
				*field = value
				matched = true
				break
			}
		}

		if !matched {
			return nil, fmt.Errorf("invalid line: %s", line)
		}
		breakCount = 0
	}

	if currentConfig != nil {
		var err error
		configs, err = SanitizeAndAppendInputConfig(currentConfig, configs)
		if err != nil {
			return nil, fmt.Errorf("sanitiization issue with %d: %w", currentConfig.EntryNumber, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return configs, nil
}

func SanitizeAndAppendInputConfig(currentConfig *InputConfig, configs []*InputConfig) ([]*InputConfig, error) {
	_, repo, err := ExtractOrgRepo(currentConfig.GithubProjectUrl)
	if err != nil {
		return nil, fmt.Errorf("github url parser: %w", err)
	}
	if currentConfig.Category == "" {
		currentConfig.Category = DefaultCategory
	}
	if currentConfig.EbuildName == "" {
		currentConfig.EbuildName = repo
	}
	currentConfig.EbuildName = TrimSuffixes(strings.TrimSuffix(currentConfig.EbuildName, ".ebuild"), "-appimage", "-AppImage") + "-appimage.ebuild"
	if currentConfig.DesktopFile == "" {
		currentConfig.DesktopFile = repo
	}
	currentConfig.DesktopFile = TrimSuffixes(currentConfig.DesktopFile, ".desktop") + ".desktop"
	configs = append(configs, currentConfig)
	return configs, nil
}

// TrimSuffixes removes the first matching suffix from the input string.
func TrimSuffixes(s string, suffixes ...string) string {
	for _, suffix := range suffixes {
		if strings.HasSuffix(s, suffix) {
			return strings.TrimSuffix(s, suffix)
		}
	}
	return s
}

// ExtractOrgRepo extracts the organization and repository from a GitHub URL.
func ExtractOrgRepo(githubURL string) (string, string, error) {
	parsedURL, err := url.Parse(githubURL)
	if err != nil {
		return "", "", err
	}

	// Ensure the URL is a GitHub URL
	if !strings.Contains(parsedURL.Host, "github.com") {
		return "", "", fmt.Errorf("not a valid GitHub URL: %s", githubURL)
	}

	// Split the path and get the org and repo
	pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(pathParts) < 2 {
		return "", "", fmt.Errorf("URL does not contain enough parts to extract org and repo: %s", githubURL)
	}

	org := pathParts[0]
	repo := pathParts[1]
	return org, repo, nil
}
