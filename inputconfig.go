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

const (
	defaultDesktopFileEnabled = false
)

/* TODO

Type Github AppImage
GithubProjectUrl https://github.com/janhq/jan/
DesktopFile jan.desktop
InstalledFilename jan.AppImage
EbuildName jan-appimage
Description Jan is an open source alternative to ChatGPT that runs 100% offline on your computer. Multiple engine support (llama.cpp, TensorRT-LLM)
Homepage https://jan.ai/
License GNU Affero General Public License v3.0
ReleasesFilename amd64=>jan-linux-x86_64-0.5.1.AppImage

To support 2 modes, repo default named, and multiple executable nemed:

Type Github AppImage
GithubProjectUrl https://github.com/janhq/jan/
EbuildName jan-appimage
Description Jan is an open source alternative to ChatGPT that runs 100% offline on your computer. Multiple engine support (llama.cpp, TensorRT-LLM)
Homepage https://jan.ai/
License GNU Affero General Public License v3.0
DesktopFile jan.desktop
InstalledFilename jan.AppImage
ReleasesFilename amd64=>jan-linux-x86_64-0.5.1.AppImage

^ defaults to reponame so "jan"

Type Github AppImage
GithubProjectUrl https://github.com/janhq/jan/
EbuildName jan-appimage
Description Jan is an open source alternative to ChatGPT that runs 100% offline on your computer. Multiple engine support (llama.cpp, TensorRT-LLM)
Homepage https://jan.ai/
License GNU Affero General Public License v3.0
DesktopFile jan.desktop
InstalledFilename jan.AppImage
ReleasesFilename amd64=>jan-linux-x86_64-0.5.1.AppImage
ProgramName bill
DesktopFile bill.desktop
InstalledFilename bill.AppImage
ReleasesFilename amd64=>bill-linux-x86_64-0.5.1.AppImage
ProgramName carter
DesktopFile carter.desktop
InstalledFilename carter.AppImage
ReleasesFilename amd64=>carter-linux-x86_64-0.5.1.AppImage


*/

type Program struct {
	ProgramName       string
	InstalledFilename string
	DesktopFile       string
	ReleasesFilename  map[string]string
}

func (p *Program) String() string {
	var sb strings.Builder
	if p.ProgramName != "" {
		sb.WriteString(fmt.Sprintf("ProgramName %s\n", p.ProgramName))
	}
	if p.DesktopFile != "" {
		sb.WriteString(fmt.Sprintf("DesktopFile %s\n", p.DesktopFile))
	}
	if p.InstalledFilename != "" {
		sb.WriteString(fmt.Sprintf("InstalledFilename %s\n", p.InstalledFilename))
	}
	keywords := make([]string, 0, len(p.ReleasesFilename))
	for key := range p.ReleasesFilename {
		keywords = append(keywords, key)
	}
	sort.Strings(keywords)
	for _, kw := range keywords {
		sb.WriteString(fmt.Sprintf("ReleasesFilename %s=>%s\n", kw, p.ReleasesFilename[kw]))
	}
	return sb.String()
}

func (p *Program) IsEmpty() bool {
	return len(p.InstalledFilename) == 0 &&
		len(p.DesktopFile) == 0 &&
		len(p.ReleasesFilename) == 0
}

// InputConfig represents a single configuration entry.
type InputConfig struct {
	EntryNumber      int
	Type             string
	GithubProjectUrl string
	Category         string
	EbuildName       string
	Description      string
	Homepage         string
	GithubRepo       string
	GithubOwner      string
	License          string
	Programs         map[string]*Program
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
		var programs []string
		for key := range c.Programs {
			programs = append(programs, key)
		}
		sort.Strings(programs)
		for _, programName := range programs {
			sb.WriteString(c.Programs[programName].String())
		}
	default:
		sb.WriteString(fmt.Sprintf("# Unknown type\n"))
	}

	return sb.String()
}

// ParseInputConfigReader parses the given configuration file and returns a slice of InputConfig structures.
func ParseInputConfigReader(file io.Reader) ([]*InputConfig, error) {
	var configs []*InputConfig
	var parseFields map[string][]string
	var parseProgramFields map[string]map[string][]string
	scanner := bufio.NewScanner(file)
	breakCount := 0
	var lastProgramName string
	var lineNumber = 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lineNumber++
		if strings.HasPrefix(line, "#") {
			continue
		}

		if line == "" {
			if breakCount < 0 || parseFields == nil {
				continue
			}
			breakCount++
			var err error
			configs, err = CreateSanitizeAndAppendInputConfig(parseFields, parseProgramFields, configs)
			if err != nil {
				return nil, fmt.Errorf("line %d: sanitiization issue with %d: %w", lineNumber, len(configs), err)
			}
			parseFields = nil
			parseProgramFields = nil
			lastProgramName = ""
			continue
		}

		if parseFields == nil {
			parseFields = map[string][]string{
				"Type":              nil,
				"GithubProjectUrl":  nil,
				"Category":          {DefaultCategory},
				"EbuildName":        nil,
				"Description":       nil,
				"Homepage":          nil,
				"License":           {DefaultLicense},
				"ProgramName":       nil,
				"InstalledFilename": nil,
				"DesktopFile":       nil,
				"ReleasesFilename":  nil,
			}
			parseProgramFields = map[string]map[string][]string{}
			lastProgramName = ""
		}

		matched := false
		switch {
		case lastProgramName != "":
			for prefix := range parseProgramFields[lastProgramName] {
				if strings.HasPrefix(line, prefix) {
					withoutPrefix := strings.TrimPrefix(line, prefix)
					if withoutPrefix != "" && !unicode.IsSpace(rune(withoutPrefix[0])) {
						continue
					}
					for len(withoutPrefix) > 0 && unicode.IsSpace(rune(withoutPrefix[0])) {
						withoutPrefix = withoutPrefix[1:]
					}
					value := strings.TrimSpace(withoutPrefix)
					if prefix == "ProgramName" {
						lastProgramName = value
						parseProgramFields[lastProgramName] = map[string][]string{
							"ProgramName":       {value},
							"InstalledFilename": nil,
							"DesktopFile":       nil,
							"ReleasesFilename":  nil,
						}
					}
					parseProgramFields[lastProgramName][prefix] = append(parseProgramFields[lastProgramName][prefix], value)
					matched = true
					break
				}
			}
			if matched {
				break
			}
			fallthrough
		default:
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
					if prefix == "ProgramName" {
						lastProgramName = value
						parseProgramFields[lastProgramName] = map[string][]string{
							"ProgramName":       {value},
							"InstalledFilename": nil,
							"DesktopFile":       nil,
							"ReleasesFilename":  nil,
						}
					}
					parseFields[prefix] = append(parseFields[prefix], value)
					matched = true
					break
				}
			}
		}

		if !matched {
			return nil, fmt.Errorf("invalid line: %s", line)
		}
		breakCount = 0
	}

	if parseFields != nil {
		var err error
		configs, err = CreateSanitizeAndAppendInputConfig(parseFields, parseProgramFields, configs)
		if err != nil {
			return nil, fmt.Errorf("sanitiization issue with last(%d): %w", len(configs), err)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return configs, nil
}

func CreateSanitizeAndAppendInputConfig(parsedFields map[string][]string, parsedProgramFields map[string]map[string][]string, configs []*InputConfig) ([]*InputConfig, error) {
	var err error
	currentConfig := &InputConfig{}
	currentConfig.Type, err = onlyOrFail(parsedFields["Type"])
	if err != nil {
		return nil, fmt.Errorf("on Type: %v: %w", parsedFields["Type"], err)
	}
	switch currentConfig.Type {
	case "Github AppImage":
		currentConfig.GithubProjectUrl, err = onlyOrFail(parsedFields["GithubProjectUrl"])
		if err != nil {
			return nil, fmt.Errorf("on GithubProjectUrl: %v: %w", parsedFields["GithubProjectUrl"], err)
		}
		currentConfig.Category, err = emptyOrLast(parsedFields["Category"])
		if err != nil {
			return nil, fmt.Errorf("on Category: %v: %w", parsedFields["Category"], err)
		}
		currentConfig.EbuildName, err = emptyOrOnlyOrFail(parsedFields["EbuildName"])
		if err != nil {
			return nil, fmt.Errorf("on EbuildName: %v: %w", parsedFields["EbuildName"], err)
		}
		currentConfig.Description, err = emptyOrOnlyOrFail(parsedFields["Description"])
		if err != nil {
			return nil, fmt.Errorf("on Description: %v: %w", parsedFields["Description"], err)
		}
		currentConfig.Homepage, err = emptyOrOnlyOrFail(parsedFields["Homepage"])
		if err != nil {
			return nil, fmt.Errorf("on Homepage: %v: %w", parsedFields["Homepage"], err)
		}
		currentConfig.License, err = emptyOrLast(parsedFields["License"])
		if err != nil {
			return nil, fmt.Errorf("on License: %v: %w", parsedFields["License"], err)
		}
		currentConfig.GithubOwner, currentConfig.GithubRepo, err = util.ExtractGithubOwnerRepo(currentConfig.GithubProjectUrl)
		if err != nil {
			return nil, fmt.Errorf("github url parser: %w", err)
		}
		if currentConfig.EbuildName == "" {
			currentConfig.EbuildName = currentConfig.GithubRepo
		}
		currentConfig.EbuildName = util.TrimSuffixes(strings.TrimSuffix(currentConfig.EbuildName, ".ebuild"), "-appimage", "-AppImage") + "-appimage.ebuild"
		if currentConfig.Programs == nil {
			currentConfig.Programs = map[string]*Program{}
		}
		program, err := currentConfig.CreateAndSanitizeInputConfigProgram("", parsedFields)
		if err != nil {
			return nil, err
		}
		if program != nil && !program.IsEmpty() {
			currentConfig.Programs[""] = program
		}

		for programName, programFields := range parsedProgramFields {
			program, err := currentConfig.CreateAndSanitizeInputConfigProgram(programName, programFields)
			if err != nil {
				return nil, err
			}
			if program != nil && !program.IsEmpty() {
				currentConfig.Programs[program.ProgramName] = program
			}
		}
	default:
		return nil, fmt.Errorf("uknown type: %s", currentConfig.Type)
	}
	configs = append(configs, currentConfig)
	return configs, nil
}

func (c *InputConfig) CreateAndSanitizeInputConfigProgram(programName string, programFields map[string][]string) (*Program, error) {
	if programFields == nil {
		return nil, fmt.Errorf("lacking program fields")
	}
	var program = &Program{
		ProgramName: programName,
	}
	var err error
	program.InstalledFilename, err = emptyOrOnlyOrFail(programFields["InstalledFilename"])
	if err != nil {
		return nil, fmt.Errorf("on InstalledFilename: %v: %w", programFields["InstalledFilename"], err)
	}
	program.DesktopFile, err = emptyOrOnlyOrFail(programFields["DesktopFile"])
	if err != nil {
		return nil, fmt.Errorf("on DesktopFile: %v: %w", programFields["DesktopFile"], err)
	}
	program.ReleasesFilename, err = parseMapType1(programFields["ReleasesFilename"])
	if err != nil {
		return nil, fmt.Errorf("on ReleasesFilename: %v: %w", programFields["ReleasesFilename"], err)
	}
	if defaultDesktopFileEnabled && program.DesktopFile == "" {
		program.DesktopFile = c.GithubRepo
	}
	if program.DesktopFile != "" {
		program.DesktopFile = util.TrimSuffixes(program.DesktopFile, ".desktop") + ".desktop"
	}
	return program, nil
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
