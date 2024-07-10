package arrans_overlay_workflow_builder

import (
	"bufio"
	"fmt"
	"github.com/arran4/arrans_overlay_workflow_builder/util"
	"io"
	"sort"
	"strings"
	"unicode"
)

// InputConfig represents a single configuration entry.
type InputConfig struct {
	EntryNumber       int
	Type              string
	GithubProjectUrl  string
	Category          string
	EbuildName        string
	Description       string
	Homepage          string
	GithubRepo        string
	GithubOwner       string
	License           string
	InstalledFilename string
	DesktopFile       string
	ReleasesFilename  map[string]string
}

const (
	DefaultCategory = "app-misc"
	DefaultLicense  = "unknown"
)

// String serializes the InputConfig struct back into the configuration file format.
func (c *InputConfig) String() string {
	var sb strings.Builder

	if c.Type != "" {
		sb.WriteString(fmt.Sprintf("Type %s\n", c.Type))
	}
	switch c.Type {
	case "Github AppImage":
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
		if c.License != "" {
			sb.WriteString(fmt.Sprintf("License %s\n", c.License))
		}
		keywords := make([]string, 0, len(c.ReleasesFilename))
		for key := range c.ReleasesFilename {
			keywords = append(keywords, key)
		}
		sort.Strings(keywords)
		for _, kw := range keywords {
			sb.WriteString(fmt.Sprintf("ReleasesFilename %s=>%s\n", kw, c.ReleasesFilename[kw]))
		}
	default:
		sb.WriteString(fmt.Sprintf("# Unknown type\n"))
	}

	return sb.String()
}

// ParseInputConfigFile parses the given configuration file and returns a slice of InputConfig structures.
func ParseInputConfigFile(file io.Reader) ([]*InputConfig, error) {
	var configs []*InputConfig
	var parseFields map[string][]string
	scanner := bufio.NewScanner(file)
	breakCount := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") {
			continue
		}

		if line == "" {
			if breakCount < 0 || parseFields == nil {
				continue
			}
			breakCount++
			var err error
			configs, err = SanitizeAndAppendInputConfig(parseFields, configs)
			if err != nil {
				return nil, fmt.Errorf("sanitiization issue with %d: %w", len(configs), err)
			}
			parseFields = nil
			continue
		}

		if parseFields == nil {
			parseFields = map[string][]string{
				"Type":              nil,
				"GithubProjectUrl":  nil,
				"InstalledFilename": nil,
				"DesktopFile":       nil,
				"Category":          {DefaultCategory},
				"EbuildName":        nil,
				"Description":       nil,
				"Homepage":          nil,
				"License":           {DefaultLicense},
				"ReleasesFilename":  nil,
			}
		}

		matched := false
		for prefix := range parseFields {
			if strings.HasPrefix(line, prefix) {
				withoutPrefix := strings.TrimPrefix(line, prefix)
				if withoutPrefix != "" && !unicode.IsSpace(rune(withoutPrefix[0])) {
					continue
				}
				for len(withoutPrefix) > 0 && unicode.IsSpace(rune(withoutPrefix[0])) {
					withoutPrefix = withoutPrefix[1:]
				}
				value := strings.TrimSpace(withoutPrefix)
				parseFields[prefix] = append(parseFields[prefix], value)
				matched = true
				break
			}
		}

		if !matched {
			return nil, fmt.Errorf("invalid line: %s", line)
		}
		breakCount = 0
	}

	if parseFields != nil {
		var err error
		configs, err = SanitizeAndAppendInputConfig(parseFields, configs)
		if err != nil {
			return nil, fmt.Errorf("sanitiization issue with %d: %w", len(configs), err)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return configs, nil
}

func SanitizeAndAppendInputConfig(parseFields map[string][]string, configs []*InputConfig) ([]*InputConfig, error) {
	var err error
	currentConfig := &InputConfig{}
	currentConfig.Type, err = onlyOrFail(parseFields["Type"])
	if err != nil {
		return nil, fmt.Errorf("on Type: %v: %w", parseFields["Type"], err)
	}
	switch currentConfig.Type {
	case "Github AppImage":
		currentConfig.GithubProjectUrl, err = onlyOrFail(parseFields["GithubProjectUrl"])
		if err != nil {
			return nil, fmt.Errorf("on GithubProjectUrl: %v: %w", parseFields["GithubProjectUrl"], err)
		}
		currentConfig.InstalledFilename, err = emptyOrOnlyOrFail(parseFields["InstalledFilename"])
		if err != nil {
			return nil, fmt.Errorf("on InstalledFilename: %v: %w", parseFields["InstalledFilename"], err)
		}
		currentConfig.DesktopFile, err = emptyOrOnlyOrFail(parseFields["DesktopFile"])
		if err != nil {
			return nil, fmt.Errorf("on DesktopFile: %v: %w", parseFields["DesktopFile"], err)
		}
		currentConfig.Category, err = emptyOrLast(parseFields["Category"])
		if err != nil {
			return nil, fmt.Errorf("on Category: %v: %w", parseFields["Category"], err)
		}
		currentConfig.EbuildName, err = emptyOrOnlyOrFail(parseFields["EbuildName"])
		if err != nil {
			return nil, fmt.Errorf("on EbuildName: %v: %w", parseFields["EbuildName"], err)
		}
		currentConfig.Description, err = emptyOrOnlyOrFail(parseFields["Description"])
		if err != nil {
			return nil, fmt.Errorf("on Description: %v: %w", parseFields["Description"], err)
		}
		currentConfig.Homepage, err = emptyOrOnlyOrFail(parseFields["Homepage"])
		if err != nil {
			return nil, fmt.Errorf("on Homepage: %v: %w", parseFields["Homepage"], err)
		}
		currentConfig.License, err = emptyOrLast(parseFields["License"])
		if err != nil {
			return nil, fmt.Errorf("on License: %v: %w", parseFields["License"], err)
		}
		currentConfig.ReleasesFilename, err = parseMapType1(parseFields["ReleasesFilename"])
		if err != nil {
			return nil, fmt.Errorf("on ReleasesFilename: %v: %w", parseFields["ReleasesFilename"], err)
		}
		currentConfig.GithubOwner, currentConfig.GithubRepo, err = util.ExtractGithubOwnerRepo(currentConfig.GithubProjectUrl)
		if err != nil {
			return nil, fmt.Errorf("github url parser: %w", err)
		}
		if currentConfig.EbuildName == "" {
			currentConfig.EbuildName = currentConfig.GithubRepo
		}
		currentConfig.EbuildName = util.TrimSuffixes(strings.TrimSuffix(currentConfig.EbuildName, ".ebuild"), "-appimage", "-AppImage") + "-appimage.ebuild"
		if currentConfig.DesktopFile == "" {
			currentConfig.DesktopFile = currentConfig.GithubRepo
		}
		currentConfig.DesktopFile = util.TrimSuffixes(currentConfig.DesktopFile, ".desktop") + ".desktop"
	default:
		return nil, fmt.Errorf("uknown type: %s", currentConfig.Type)
	}
	configs = append(configs, currentConfig)
	return configs, nil
}

func parseMapType1(a []string) (map[string]string, error) {
	result := make(map[string]string, len(a))
	for i, v := range a {
		s := strings.SplitN(v, "=>", 2)
		if len(s) != 2 {
			return nil, fmt.Errorf("entry %d, can't split %#v", i, v)
		}
		result[strings.TrimSpace(s[0])] = strings.TrimSpace(s[1])
	}
	return result, nil
}

func emptyOrOnlyOrFail(i []string) (string, error) {
	switch len(i) {
	case 0:
		return "", nil
	case 1:
		return i[0], nil
	default:
		return "", fmt.Errorf("too many values")
	}
}

func emptyOrLast(i []string) (string, error) {
	switch len(i) {
	case 0:
		return "", nil
	default:
		return i[len(i)-1], nil
	}
}

func onlyOrFail(i []string) (string, error) {
	switch len(i) {
	case 0:
		return "", fmt.Errorf("no values")
	case 1:
		return i[0], nil
	default:
		return "", fmt.Errorf("too many values")
	}
}
