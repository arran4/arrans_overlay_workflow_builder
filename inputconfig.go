package arrans_overlay_workflow_builder

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/arran4/arrans_overlay_workflow_builder/util"
	"github.com/google/go-github/v62/github"
	"io"
	"log"
	"os"
	"sort"
	"strings"
	"unicode"
)

var (
	DefaultDesktopFileEnabled = false
)

type Program struct {
	ProgramName       string
	InstalledFilename string
	DesktopFile       string
	Icons             []string
	Docs              []string
	Dependencies      []string
	ReleasesFilename  map[string]string
	ArchiveFilename   map[string]string
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

func (p *Program) IsArchived() bool {
	return len(p.ArchiveFilename) > 0
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
	if len(p.Docs) > 0 {
		sb.WriteString(fmt.Sprintf("Docs %s\n", strings.Join(p.Docs, " ")))
	}
	if len(p.Dependencies) > 0 {
		sb.WriteString(fmt.Sprintf("Dependencies %s\n", strings.Join(p.Dependencies, " ")))
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
	keywords = make([]string, 0, len(p.ArchiveFilename))
	for key := range p.ArchiveFilename {
		keywords = append(keywords, key)
	}
	sort.Strings(keywords)
	for _, kw := range keywords {
		sb.WriteString(fmt.Sprintf("ArchiveFilename %s=>%s\n", kw, p.ArchiveFilename[kw]))
	}
	return sb.String()
}

func (p *Program) IsEmpty() bool {
	return len(p.InstalledFilename) == 0 &&
		len(p.DesktopFile) == 0 &&
		len(p.Icons) == 0 &&
		len(p.Docs) == 0 &&
		len(p.Dependencies) == 0 &&
		len(p.ReleasesFilename) == 0 &&
		len(p.ArchiveFilename) == 0
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
	case "Github AppImage":
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
				"Icons":             nil,
				"Docs":              nil,
				"Dependencies":      nil,
				"ReleasesFilename":  nil,
				"Workaround":        nil,
				"ArchiveFilename":   nil,
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
							"Dependencies":      nil,
							"Icons":             nil,
							"Docs":              nil,
							"ReleasesFilename":  nil,
							"ArchiveFilename":   nil,
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
							"Icons":             nil,
							"Docs":              nil,
							"Dependencies":      nil,
							"ReleasesFilename":  nil,
							"ArchiveFilename":   nil,
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
		currentConfig.Workarounds, err = parseOptionalMapType1(parsedFields["Workaround"])
		if err != nil {
			return nil, fmt.Errorf("on Workarounds: %v: %w", parsedFields["Workaround"], err)
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

func (ic *InputConfig) CreateAndSanitizeInputConfigProgram(programName string, programFields map[string][]string) (*Program, error) {
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
	program.Icons, err = emptyOrAppendStringArray(program.Icons, programFields["Icons"])
	if err != nil {
		return nil, fmt.Errorf("on Icons: %v: %w", programFields["Icons"], err)
	}
	program.Docs, err = emptyOrAppendStringArray(program.Docs, programFields["Docs"])
	if err != nil {
		return nil, fmt.Errorf("on Docs: %v: %w", programFields["Docs"], err)
	}
	program.Dependencies, err = emptyOrAppendStringArray(program.Dependencies, programFields["Dependencies"])
	if err != nil {
		return nil, fmt.Errorf("on Dependencies: %v: %w", programFields["Dependencies"], err)
	}
	program.ReleasesFilename, err = parseMapType1(programFields["ReleasesFilename"])
	if err != nil {
		return nil, fmt.Errorf("on ReleasesFilename: %v: %w", programFields["ReleasesFilename"], err)
	}
	program.ArchiveFilename, err = parseMapType1(programFields["ArchiveFilename"])
	if err != nil {
		return nil, fmt.Errorf("on ArchiveFilename: %v: %w", programFields["ArchiveFilename"], err)
	}
	if DefaultDesktopFileEnabled && program.DesktopFile == "" {
		program.DesktopFile = ic.GithubRepo
	}
	if program.DesktopFile != "" {
		program.DesktopFile = util.TrimSuffixes(program.DesktopFile, ".desktop") + ".desktop"
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

func NewInputConfigurationFromRepo(gitRepo string, tagOverride string, prefix string) (string, *InputConfig, []string, []string, *github.RepositoryRelease, *InputConfig, error) {
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
		Type:             "Github Binary Release",
		GithubProjectUrl: gitRepo,
		//Category:          "",
		EbuildName:  fmt.Sprintf("%s-bin", ebuildNamePart),
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
			tag := release.GetTagName()
			if prefix != "" {
				if !strings.HasPrefix(tag, prefix) {
					continue
				}
				tag = strings.TrimPrefix(tag, prefix)
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

		tag := releaseInfo.GetTagName()
		if prefix != "" {
			if !strings.HasPrefix(tag, prefix) {
				return "", nil, nil, nil, nil, nil, fmt.Errorf("github latest release tag %s doesn't have prefix %s", tag, prefix)
			}
			tag = strings.TrimPrefix(tag, prefix)
			ic.Workarounds["Tag Prefix"] = prefix
		}
		v, err := semver.NewVersion(tag)
		if err != nil {
			return "", nil, nil, nil, nil, nil, fmt.Errorf("github latest release tag parse %s: %w", tag, err)
		}
		if strings.HasPrefix(tag, "v") {
			tags = []string{tag}
			versions = []string{v.String()}
		} else {
			tags = []string{tag}
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
		if prefix != "" {
			if !strings.HasSuffix(tag, prefix) {
				return "", nil, nil, nil, nil, nil, fmt.Errorf("github latest release tag %s doesn't have prefix %s", tag, prefix)
			}
			tag = strings.TrimPrefix(tag, prefix)
			ic.Workarounds["Tag Prefix"] = prefix
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
