package arrans_overlay_workflow_builder

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/arran4/arrans_overlay_workflow_builder/util"
	"github.com/google/go-github/v62/github"
	"github.com/stoewer/go-strcase"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

var (
	DefaultDesktopFileEnabled = false
)

type Program struct {
	ProgramName            string
	Binary                 map[string][]string
	DesktopFile            string
	Icons                  []string
	Documents              map[string][][]string
	ManualPage             map[string][][]string
	ShellCompletionScripts map[string]map[string][]string
	Dependencies           []string
}

func (p *Program) HasDesktopFile() bool {
	return p.DesktopFile != ""
}

func (p *Program) FirstIcons() string {
	if len(p.Icons) == 0 {
		return ""
	}
	return p.Icons[0]
}

func (p *Program) InstalledFilename() string {
	for _, b := range p.Binary {
		if len(b) > 0 {
			return b[len(b)-1]
		}
	}
	return ""
}

func (p *Program) IsArchived(arch string) bool {
	return len(p.Binary[arch]) > 2
}

func (p *Program) String() string {
	var sb strings.Builder
	if p.ProgramName != "" {
		sb.WriteString(fmt.Sprintf("ProgramName %s\n", p.ProgramName))
	}
	if p.DesktopFile != "" {
		sb.WriteString(fmt.Sprintf("DesktopFile %s\n", p.DesktopFile))
	}
	if len(p.Icons) > 0 {
		sb.WriteString(fmt.Sprintf("Icons %s\n", strings.Join(p.Icons, " ")))
	}
	if len(p.Dependencies) > 0 {
		sb.WriteString(fmt.Sprintf("Dependencies %s\n", strings.Join(p.Dependencies, " ")))
	}
	MapDoubleStringer(&sb, "Document", p.Documents)
	MapDoubleStringer(&sb, "ManualPage", p.ManualPage)
	DoubleMapStringer(&sb, "ShellCompletionScript", p.ShellCompletionScripts)
	MapStringer(&sb, "Binary", p.Binary)
	return sb.String()
}

func MapStringer(sb *strings.Builder, key string, valueMap map[string][]string) {
	keywords := make([]string, 0, len(valueMap))
	for key := range valueMap {
		keywords = append(keywords, key)
	}
	sort.Strings(keywords)
	for _, kw := range keywords {
		sb.WriteString(fmt.Sprintf("%s %s=>%s\n", key, kw, strings.Join(valueMap[kw], " > ")))
	}
}

func MapDoubleStringer(sb *strings.Builder, key string, valueMap map[string][][]string) {
	keywords := make([]string, 0, len(valueMap))
	for key := range valueMap {
		keywords = append(keywords, key)
	}
	sort.Strings(keywords)
	for _, kw := range keywords {
		for _, values := range valueMap[kw] {
			sb.WriteString(fmt.Sprintf("%s %s=>%s\n", key, kw, strings.Join(values, " > ")))
		}
	}
}

func DoubleMapStringer(sb *strings.Builder, key string, valueMap map[string]map[string][]string) {
	keywords := make([]string, 0, len(valueMap))
	for key := range valueMap {
		keywords = append(keywords, key)
	}
	sort.Strings(keywords)
	for _, kw := range keywords {
		subKeywords := make([]string, 0, len(valueMap[kw]))
		for subKey := range valueMap[kw] {
			subKeywords = append(subKeywords, subKey)
		}
		sort.Strings(subKeywords)
		for _, skw := range subKeywords {
			sb.WriteString(fmt.Sprintf("%s %s:%s=>%s\n", key, kw, skw, strings.Join(valueMap[kw][skw], " > ")))
		}
	}
}

func (p *Program) IsEmpty() bool {
	return len(p.Binary) == 0 &&
		len(p.DesktopFile) == 0 &&
		len(p.Icons) == 0 &&
		len(p.Documents) == 0 &&
		len(p.ManualPage) == 0 &&
		len(p.ShellCompletionScripts) == 0 &&
		len(p.Dependencies) == 0
}

func (p *Program) HasManualPage() bool {
	for _, e := range p.ManualPage {
		for _, ee := range e {
			if len(ee) > 0 {
				return true
			}
		}
	}
	return false
}

func (p *Program) HasCompressedManualPages() bool {
	for _, e := range p.ManualPage {
		for _, ee := range e {
			if len(ee) > 2 {
				switch strings.ToLower(filepath.Ext(ee[len(ee)-2])) {
				case ".gz", ".bz2":
					return true
				}
			}
		}
	}
	return false
}

func (p *Program) HasDocuments() bool {
	for _, e := range p.Documents {
		for _, ee := range e {
			if len(ee) > 0 {
				return true
			}
		}
	}
	return false
}

func (p *Program) HasShellCompletion(shell string) bool {
	for _, e := range p.ShellCompletionScripts {
		for ee := range e {
			if strings.EqualFold(ee, shell) {
				return true
			}
		}
	}
	return false
}

type KeywordedFilenameReference struct {
	Filepath []string
	Keyword  string
}

func (kr *KeywordedFilenameReference) SourceFilepath() string {
	if len(kr.Filepath) <= 1 {
		return strings.Join(kr.Filepath, "/")
	}
	return strings.Join(kr.Filepath[1:len(kr.Filepath)-1], "/")
}

func (kr *KeywordedFilenameReference) DestinationFilename() string {
	if len(kr.Filepath) == 0 {
		return ""
	}
	return kr.Filepath[len(kr.Filepath)-1]
}

func (p *Program) ShellCompletion(shell string) (result []*KeywordedFilenameReference) {
	for kw, e := range p.ShellCompletionScripts {
		for shellName, fp := range e {
			if strings.EqualFold(shellName, shell) {
				result = append(result, &KeywordedFilenameReference{
					Keyword:  kw,
					Filepath: fp,
				})
			}
		}
	}
	return
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
	Workarounds      map[string]string
	Programs         map[string]*Program
}

const (
	DefaultCategory = "app-misc"
	DefaultLicense  = "unknown"
)

// String serializes the InputConfig struct back into the configuration file format.
func (ic *InputConfig) String() string {
	var sb strings.Builder

	if ic.Type != "" {
		sb.WriteString(fmt.Sprintf("Type %s\n", ic.Type))
	}
	switch ic.Type {
	case "Github AppImage Release":
		if ic.GithubProjectUrl != "" {
			sb.WriteString(fmt.Sprintf("GithubProjectUrl %s\n", ic.GithubProjectUrl))
		}
		if ic.Category != "" {
			sb.WriteString(fmt.Sprintf("Category %s\n", ic.Category))
		}
		if ic.EbuildName != "" {
			sb.WriteString(fmt.Sprintf("EbuildName %s\n", ic.EbuildName))
		}
		if ic.Description != "" {
			sb.WriteString(fmt.Sprintf("Description %s\n", ic.Description))
		}
		if ic.Homepage != "" {
			sb.WriteString(fmt.Sprintf("Homepage %s\n", ic.Homepage))
		}
		if ic.License != "" {
			sb.WriteString(fmt.Sprintf("License %s\n", ic.License))
		}
		workarounds := ic.WorkaroundString()
		for _, workaround := range workarounds {
			if len(ic.Workarounds[workaround]) == 0 {
				sb.WriteString(fmt.Sprintf("Workaround %s\n", workaround))
			} else {
				sb.WriteString(fmt.Sprintf("Workaround %s => %s\n", workaround, ic.Workarounds[workaround]))
			}
		}
		programs := ic.ProgramsString()
		for _, programName := range programs {
			sb.WriteString(ic.Programs[programName].String())
		}
	case "Github Binary Release":
		if ic.GithubProjectUrl != "" {
			sb.WriteString(fmt.Sprintf("GithubProjectUrl %s\n", ic.GithubProjectUrl))
		}
		if ic.Category != "" {
			sb.WriteString(fmt.Sprintf("Category %s\n", ic.Category))
		}
		if ic.EbuildName != "" {
			sb.WriteString(fmt.Sprintf("EbuildName %s\n", ic.EbuildName))
		}
		if ic.Description != "" {
			sb.WriteString(fmt.Sprintf("Description %s\n", ic.Description))
		}
		if ic.Homepage != "" {
			sb.WriteString(fmt.Sprintf("Homepage %s\n", ic.Homepage))
		}
		if ic.License != "" {
			sb.WriteString(fmt.Sprintf("License %s\n", ic.License))
		}
		workarounds := ic.WorkaroundString()
		for _, workaround := range workarounds {
			if len(ic.Workarounds[workaround]) == 0 {
				sb.WriteString(fmt.Sprintf("Workaround %s\n", workaround))
			} else {
				sb.WriteString(fmt.Sprintf("Workaround %s => %s\n", workaround, ic.Workarounds[workaround]))
			}
		}
		programs := ic.ProgramsString()
		for _, programName := range programs {
			sb.WriteString(ic.Programs[programName].String())
		}
	default:
		sb.WriteString(fmt.Sprintf("# Unknown type\n"))
	}

	return sb.String()
}

func (ic *InputConfig) ProgramsString() []string {
	var programs []string
	for key := range ic.Programs {
		programs = append(programs, key)
	}
	sort.Strings(programs)
	return programs
}

func (ic *InputConfig) WorkaroundString() []string {
	var workarounds []string
	for key := range ic.Workarounds {
		workarounds = append(workarounds, key)
	}
	sort.Strings(workarounds)
	return workarounds
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
				"Type":                  nil,
				"GithubProjectUrl":      nil,
				"Category":              {DefaultCategory},
				"EbuildName":            nil,
				"Description":           nil,
				"Homepage":              nil,
				"License":               {DefaultLicense},
				"ProgramName":           nil,
				"DesktopFile":           nil,
				"Icons":                 nil,
				"ManualPage":            nil,
				"Document":              nil,
				"ShellCompletionScript": nil,
				"Dependencies":          nil,
				"Workaround":            nil,
				"Binary":                nil,
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
							"ProgramName":           {value},
							"DesktopFile":           nil,
							"Dependencies":          nil,
							"Icons":                 nil,
							"ManualPage":            nil,
							"Document":              nil,
							"ShellCompletionScript": nil,
							"Binary":                nil,
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
							"ProgramName":           {value},
							"DesktopFile":           nil,
							"Icons":                 nil,
							"ManualPage":            nil,
							"Document":              nil,
							"ShellCompletionScript": nil,
							"Dependencies":          nil,
							"Binary":                nil,
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
	currentConfig.Workarounds, err = parseOptionalMapType1(parsedFields["Workaround"])
	if err != nil {
		return nil, fmt.Errorf("on Workarounds: %v: %w", parsedFields["Workaround"], err)
	}
	switch currentConfig.Type {
	case "Github AppImage Release":
		if currentConfig.EbuildName == "" {
			currentConfig.EbuildName = currentConfig.GithubRepo
		}
		currentConfig.EbuildName = util.TrimSuffixes(strings.TrimSuffix(currentConfig.EbuildName, ".ebuild"), "-appimage", "-AppImage") + "-appimage.ebuild"
		if currentConfig.Programs == nil {
			currentConfig.Programs = map[string]*Program{}
		}
	case "Github Binary Release":
		if currentConfig.EbuildName == "" {
			currentConfig.EbuildName = currentConfig.GithubRepo
		}
		currentConfig.EbuildName = util.TrimSuffixes(strings.TrimSuffix(currentConfig.EbuildName, ".ebuild"), "-bin") + "-bin.ebuild"
		if currentConfig.Programs == nil {
			currentConfig.Programs = map[string]*Program{}
		}
	default:
		return nil, fmt.Errorf("uknown type: %s", currentConfig.Type)
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
	configs = append(configs, currentConfig)
	return configs, nil
}

func (ic *InputConfig) CreateAndSanitizeInputConfigProgram(programName string, programFields map[string][]string) (*Program, error) {
	if programFields == nil {
		return nil, fmt.Errorf("lacking program fields")
	}
	var program = &Program{
		ProgramName: programName,
	}
	var err error
	program.Dependencies, err = emptyOrAppendStringArray(program.Dependencies, programFields["Dependencies"])
	if err != nil {
		return nil, fmt.Errorf("on Dependencies: %v: %w", programFields["Dependencies"], err)
	}
	program.Binary, err = parseMapStringListType1(programFields["Binary"])
	if err != nil {
		return nil, fmt.Errorf("on Binary: %v: %w", programFields["Binary"], err)
	}
	switch ic.Type {
	case "Github AppImage Release":
		program.DesktopFile, err = emptyOrOnlyOrFail(programFields["DesktopFile"])
		if err != nil {
			return nil, fmt.Errorf("on DesktopFile: %v: %w", programFields["DesktopFile"], err)
		}
		program.Icons, err = emptyOrAppendStringArray(program.Icons, programFields["Icons"])
		if err != nil {
			return nil, fmt.Errorf("on Icons: %v: %w", programFields["Icons"], err)
		}
		if DefaultDesktopFileEnabled && program.DesktopFile == "" {
			program.DesktopFile = ic.GithubRepo
		}
		if program.DesktopFile != "" {
			program.DesktopFile = util.TrimSuffixes(program.DesktopFile, ".desktop") + ".desktop"
		}
	case "Github Binary Release":
		program.Documents, err = parseMapDoubleStringListType1(programFields["Document"])
		if err != nil {
			return nil, fmt.Errorf("on Document: %v: %w", programFields["Document"], err)
		}
		program.ManualPage, err = parseMapDoubleStringListType1(programFields["ManualPage"])
		if err != nil {
			return nil, fmt.Errorf("on ManualPage: %v: %w", programFields["ManualPage"], err)
		}
		program.ShellCompletionScripts, err = parseDoubleMapStringListType1(programFields["ShellCompletionScript"])
		if err != nil {
			return nil, fmt.Errorf("on ShellCompletionScript: %v: %w", programFields["ShellCompletionScript"], err)
		}
	default:
		return nil, fmt.Errorf("uknown type: %s", ic.Type)
	}
	return program, nil
}

func (ic *InputConfig) WorkaroundSemanticVersionWithoutV() bool {
	if ic.Workarounds == nil {
		return false
	}
	_, ok := ic.Workarounds["Semantic Version Without V"]
	return ok
}

func (ic *InputConfig) WorkaroundSemanticVersionPrereleaseHack1() bool {
	if ic.Workarounds == nil {
		return false
	}
	_, ok := ic.Workarounds["Semantic Version Prerelease Hack 1"]
	return ok
}

func (ic *InputConfig) WorkaroundTagPrefix() string {
	if ic.Workarounds == nil {
		return ""
	}
	s, _ := ic.Workarounds["Tag Prefix"]
	return s
}

func (ic *InputConfig) Validate() error {
	// TODO more validation
	for workaround := range ic.Workarounds {
		switch workaround {
		case "Semantic Version Without V":
		case "Semantic Version Prerelease Hack 1":
		case "Tag Prefix":
		case "Programs as Alternatives":
		default:
			return fmt.Errorf("unknown workaround: %s", workaround)
		}
	}
	return nil
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

func parseMapStringListType1(a []string) (map[string][]string, error) {
	result := make(map[string][]string, len(a))
	for i, v := range a {
		s := strings.SplitN(v, "=>", 2)
		if len(s) != 2 {
			return nil, fmt.Errorf("entry %d, can't split %#v", i, v)
		}
		for _, e := range strings.Split(strings.TrimSpace(s[1]), ">") {
			result[strings.TrimSpace(s[0])] = append(result[strings.TrimSpace(s[0])], strings.TrimSpace(e))
		}
	}
	return result, nil
}

func parseMapDoubleStringListType1(a []string) (map[string][][]string, error) {
	result := make(map[string][][]string, len(a))
	for i, v := range a {
		s := strings.SplitN(v, "=>", 2)
		if len(s) != 2 {
			return nil, fmt.Errorf("entry %d, can't split %#v", i, v)
		}
		var v []string
		for _, e := range strings.Split(strings.TrimSpace(s[1]), ">") {
			v = append(v, strings.TrimSpace(e))
		}
		result[strings.TrimSpace(s[0])] = append(result[strings.TrimSpace(s[0])], v)
	}
	return result, nil
}

func parseDoubleMapStringListType1(a []string) (map[string]map[string][]string, error) {
	result := make(map[string]map[string][]string, len(a))
	for i, line := range a {
		kvSplit := strings.SplitN(line, "=>", 2)
		if len(kvSplit) != 2 {
			return nil, fmt.Errorf("entry %d, can't split %#v", i, line)
		}
		keySplit := strings.Split(kvSplit[0], ":")
		if len(kvSplit) != 2 {
			return nil, fmt.Errorf("entry %d, can't split key %#v", i, kvSplit[0])
		}
		for _, eps := range strings.Split(strings.TrimSpace(kvSplit[1]), ">") {
			keySplit1 := strings.TrimSpace(keySplit[0])
			keySplit2 := strings.TrimSpace(keySplit[1])
			if _, ok := result[keySplit1]; !ok {
				result[keySplit1] = make(map[string][]string)
			}
			result[keySplit1][keySplit2] = append(result[keySplit1][keySplit2], strings.TrimSpace(eps))
		}
	}
	return result, nil
}

func parseOptionalMapType1(a []string) (map[string]string, error) {
	result := make(map[string]string, len(a))
	for _, v := range a {
		s := strings.SplitN(v, "=>", 2)
		if len(s) != 2 {
			result[strings.TrimSpace(s[0])] = ""
		} else {
			result[strings.TrimSpace(s[0])] = strings.TrimSpace(s[1])
		}
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

func emptyOrAppendStringArray(o []string, i []string) ([]string, error) {
	if o == nil {
		o = []string{}
	}
	if len(i) > 0 {
		for _, e := range i {
			for _, s := range strings.Split(e, " ") {
				o = append(o, strings.TrimSpace(s))
			}
		}
	}
	return o, nil
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

func AppendToConfigurationFile(config string, ic *InputConfig) error {
	f, err := os.OpenFile(config, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening configuration file to append: %w", err)
	}

	if _, err := f.WriteString("\n" + ic.String() + "\n"); err != nil {
		return fmt.Errorf("writing: %w", err)
	}

	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("Error closing file: %s", err)
		}
	}()
	return nil
}

func ReadConfigurationFile(configFn string) ([]*InputConfig, error) {
	var config []*InputConfig
	f, err := os.Open(configFn)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("opening configuration file: %w", err)
	} else if err == nil {
		config, err = ParseInputConfigReader(f)
		if err != nil {
			return nil, fmt.Errorf("parsing configuration file: %w", err)
		}
		defer func() {
			if err := f.Close(); err != nil {
				log.Printf("Error closing file: %s", err)
			}
		}()
	} else {
		config = make([]*InputConfig, 0)
	}
	return config, nil
}

func NewInputConfigurationFromRepo(gitRepo, tagOverride, tagPrefix, ebuildSuffix, sourceType string) (string, *InputConfig, []string, []string, *github.RepositoryRelease, *InputConfig, error) {
	client := github.NewClient(nil)
	if token, ok := os.LookupEnv("GITHUB_TOKEN"); ok {
		client = client.WithAuthToken(token)
	}
	ownerName, repoName, err := util.ExtractGithubOwnerRepo(gitRepo)
	if err != nil {
		return "", nil, nil, nil, nil, nil, fmt.Errorf("github url parse: %w", err)
	}
	log.Printf("Getting details for %s's %s", ownerName, repoName)
	ctx := context.Background()
	repo, _, err := client.Repositories.Get(ctx, ownerName, repoName)
	if err != nil {
		return "", nil, nil, nil, nil, nil, fmt.Errorf("github repo fetch: %w", err)
	}
	var licenseName *string
	if repo.License != nil {
		licenseName = repo.License.Name
	}
	ebuildNamePart := strings.ReplaceAll(repoName, ".", "-")
	ic := &InputConfig{
		Type:             sourceType,
		GithubProjectUrl: gitRepo,
		//Category:          "",
		EbuildName:  strcase.KebabCase(fmt.Sprintf("%s%s", ebuildNamePart, ebuildSuffix)),
		Description: util.StringOrDefault(repo.Description, "TODO"),
		Homepage:    util.StringOrDefault(repo.Homepage, ""),
		GithubRepo:  repoName,
		GithubOwner: ownerName,
		Workarounds: map[string]string{},
		Programs:    map[string]*Program{},
		License:     util.StringOrDefault(licenseName, "unknown"),
	}
	var versions = []string{}
	var tags = []string{}
	if tagOverride != "" {
		tags = append(tags, tagOverride)
	}
	var releaseInfo *github.RepositoryRelease
	if tagOverride == "" {
		var releasesList []*github.RepositoryRelease
		releasesList, _, err = client.Repositories.ListReleases(ctx, ownerName, repoName, &github.ListOptions{})
		if err != nil {
			return "", nil, nil, nil, nil, nil, fmt.Errorf("github list releases fetch: %w", err)
		}
		for _, release := range releasesList {
			originalTag := release.GetTagName()
			tag := originalTag
			if tagPrefix != "" {
				if !strings.HasPrefix(tag, tagPrefix) {
					continue
				}
				tag = strings.TrimPrefix(tag, tagPrefix)
			}
			v, err := semver.NewVersion(tag)
			if err != nil {
				continue
			}
			if v.Prerelease() != "" {
				ic.Workarounds["Semantic Version Prerelease Hack 1"] = ""
			}
			if releaseInfo == nil {
				releaseInfo = release
			}
		}
		if releaseInfo == nil {
			releaseInfo, _, err = client.Repositories.GetLatestRelease(ctx, ownerName, repoName)
			if err != nil {
				return "", nil, nil, nil, nil, nil, fmt.Errorf("github latest release fetch: %w", err)
			}
		}

		originalTag := releaseInfo.GetTagName()
		tag := originalTag
		if tagPrefix != "" {
			if !strings.HasPrefix(tag, tagPrefix) {
				return "", nil, nil, nil, nil, nil, fmt.Errorf("github latest release tag %s doesn't have prefix %s", tag, tagPrefix)
			}
			tag = strings.TrimPrefix(tag, tagPrefix)
			ic.Workarounds["Tag Prefix"] = tagPrefix
		}
		v, err := semver.NewVersion(tag)
		if err != nil {
			return "", nil, nil, nil, nil, nil, fmt.Errorf("github latest release tag parse %s: %w", tag, err)
		}
		if strings.HasPrefix(tag, "v") {
			tags = []string{originalTag}
			versions = []string{v.String()}
		} else {
			tags = []string{originalTag}
			ic.Workarounds["Semantic Version Without V"] = ""
		}
	} else {
		releaseInfo, _, err = client.Repositories.GetReleaseByTag(ctx, ownerName, repoName, tagOverride)
		if err != nil {
			return "", nil, nil, nil, nil, nil, fmt.Errorf("github latest release fetch: %w", err)
		}
		if !strings.HasPrefix(tagOverride, "v") {
			ic.Workarounds["Semantic Version Without V"] = ""
		}
		tag := releaseInfo.GetTagName()
		if tagPrefix != "" {
			if !strings.HasSuffix(tag, tagPrefix) {
				return "", nil, nil, nil, nil, nil, fmt.Errorf("github latest release tag %s doesn't have prefix %s", tag, tagPrefix)
			}
			tag = strings.TrimPrefix(tag, tagPrefix)
			ic.Workarounds["Tag Prefix"] = tagPrefix
		}
		v, err := semver.NewVersion(tag)
		if err != nil {
			return "", nil, nil, nil, nil, nil, fmt.Errorf("github latest release tag parse %s: %w", tag, err)
		}
		if v.Prerelease() != "" {
			ic.Workarounds["Semantic Version Prerelease Hack 1"] = ""
		}
	}

	log.Printf("Latest release %v", versions)
	return repoName, ic, versions, tags, releaseInfo, nil, nil
}
