package arrans_overlay_workflow_builder

import (
	"context"
	"errors"
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/arran4/arrans_overlay_workflow_builder/util"
	"github.com/google/go-github/v62/github"
	"github.com/probonopd/go-appimage/src/goappimage"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
)

type Meaning struct {
	// Core properties
	Keyword   string
	OS        string
	Toolchain string
	Container string

	// Compiled only
	Containers []string
	Filename   string

	// Relevant restraint + identification
	AppImage bool

	// Identification
	Version     bool
	ProjectName bool

	// Match rules
	SuffixOnly    bool
	CaseSensitive bool
	ReleaseAsset  *github.ReleaseAsset
}

func ConfigAddAppImageGithubReleases(toConfig string, gitRepo string) error {
	client := github.NewClient(nil)
	if token, ok := os.LookupEnv("GITHUB_TOKEN"); ok {
		client = client.WithAuthToken(token)
	}
	ownerName, repoName, err := util.ExtractGithubOwnerRepo(gitRepo)
	if err != nil {
		return fmt.Errorf("github url parse: %w", err)
	}
	log.Printf("Getting details for %s's %s", ownerName, repoName)
	ctx := context.Background()
	repo, _, err := client.Repositories.Get(ctx, ownerName, repoName)
	if err != nil {
		return fmt.Errorf("github repo fetch: %w", err)
	}
	var licenseName *string
	if repo.License != nil {
		licenseName = repo.License.Name
	}
	ic := &InputConfig{
		Type:             "Github AppImage",
		GithubProjectUrl: gitRepo,
		//Category:          "",
		EbuildName:  fmt.Sprintf("%s-appimage", util.TrimSuffixes(repoName, "-AppImage", "-appimage")),
		Description: StringOrDefault(repo.Description, "TODO"),
		Homepage:    StringOrDefault(repo.Homepage, ""),
		GithubRepo:  repoName,
		GithubOwner: ownerName,
		License:     StringOrDefault(licenseName, "unknown"),
	}
	latestRelease, _, err := client.Repositories.GetLatestRelease(ctx, ownerName, repoName)
	if err != nil {
		return fmt.Errorf("github latest release fetch: %w", err)
	}

	v, err := semver.NewVersion(latestRelease.GetTagName())
	if err != nil {
		return fmt.Errorf("github latest release tag parse %s: %w", latestRelease.GetTagName(), err)
	}
	version := v.String()

	log.Printf("Latest release %s", version)

	var wordMap = GroupAndSort(GenerateWordMeanings(repoName, version))

	var appImages []*Meaning
	for _, asset := range latestRelease.Assets {
		n := asset.GetName()
		log.Printf("Is %s an AppImage?", n)
		result := DecodeFilename(wordMap, n)
		if len(result) == 0 {
			log.Printf("Can't decode %s", n)
			continue
		}
		compiled, ok := CompileMeanings(result, asset, n)
		if !ok {
			log.Printf("Can't simplify %s", n)
			continue
		}
		if !compiled.AppImage {
			log.Printf("Doesn't have AppImage in it %s", n)
			continue
		}
		if compiled.OS != "" && compiled.OS != "linux" {
			log.Printf("Not for linux %s", n)
			continue
		}
		if compiled.Keyword == "" {
			// Default to amd64 because that's just a thing you do.
			compiled.Keyword = "~amd64"
		}
		appImages = append(appImages, compiled)
	}

	for _, appImage := range appImages {
		url := appImage.ReleaseAsset.GetBrowserDownloadURL()
		log.Printf("Downloading %s", url)
		tempFile, err := downloadUrlToTempFile(url)
		if err != nil {
			return fmt.Errorf("downloading release: %w", err)
		}
		log.Printf("Got %s", tempFile)
		var programName string // TODO from app image (create a mode "unidentified string" - can be used to figure out programName)
		ic.Programs[programName] = &Program{
			ProgramName:       programName,
			InstalledFilename: fmt.Sprintf("%s.AppImage", ic.GithubRepo),
			ReleasesFilename:  map[string]string{},
		}
		ic.Programs[programName].ReleasesFilename[strings.TrimPrefix(appImage.Keyword, "~")] = appImage.Filename
		ai, err := goappimage.NewAppImage(tempFile)
		if err != nil {
			return fmt.Errorf("reading AppImage %s: %w", url, err)
		}
		files := ai.ListFiles(".")
		for _, f := range files {
			if strings.HasSuffix(f, ".desktop") {
				ic.Programs[programName].DesktopFile = f
				log.Printf("Found a desktop file %s", ic.Programs[programName].DesktopFile)
				break
			}
		}
		// Desktop icon: ai.Desktop.Section("Desktop Entry").Key("Icon").Value()
		if err := os.Remove(tempFile); err != nil {
			log.Printf("Error removing temp file: %s", err)
		}
	}

	log.Printf("Reading config")
	config, err := ReadConfigurationFile(toConfig)
	if err != nil {
		return fmt.Errorf("reading configuration file: %s: %w", toConfig, err)
	}

	for _, entry := range config {
		if entry.EntryNumber >= ic.EntryNumber {
			ic.EntryNumber = entry.EntryNumber + 1
		}
	}

	log.Printf("Appending to config as entry id: %d", ic.EntryNumber)
	if err := AppendToConfigurationFile(toConfig, ic); err != nil {
		return fmt.Errorf("appending to configuration file: %s: %w", toConfig, err)
	}
	return nil
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

func downloadUrlToTempFile(url string) (string, error) {
	// Create a temporary file
	file, err := os.CreateTemp("", "download-*.tmp")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %v", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Printf("Temp file close issue: %s", err)
		}
	}(file)
	response, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download file: %v", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("File download close issue: %s", err)
		}
	}(response.Body)

	_, err = io.Copy(file, response.Body)
	if err != nil {
		return "", fmt.Errorf("writing %s to file %s: %v", url, file.Name(), err)
	}

	return file.Name(), nil
}

func CompileMeanings(input []*Meaning, releaseAsset *github.ReleaseAsset, filename string) (*Meaning, bool) {
	result := &Meaning{
		Filename:     filename,
		ReleaseAsset: releaseAsset,
		SuffixOnly:   true,
	}
	for _, each := range input {

		if each.Keyword != "" {
			if result.Keyword != "" && result.Keyword != each.Keyword {
				return nil, false
			}
			if result.Keyword == "" {
				result.Keyword = each.Keyword
			}
		}
		if each.OS != "" {
			if result.OS != "" && result.OS != each.OS {
				return nil, false
			}
			if result.OS == "" {
				result.OS = each.OS
			}
		}
		if each.Toolchain != "" {
			if result.Toolchain != "" && result.Toolchain != each.Toolchain {
				return nil, false
			}
			if result.Toolchain == "" {
				result.Toolchain = each.Toolchain
			}
		}
		if each.Container != "" {
			result.Containers = append(result.Containers, each.Container)
		}

		if each.Version {
			result.Version = each.Version
		}

		if each.ProjectName {
			result.ProjectName = each.ProjectName
		}

		if each.AppImage {
			result.AppImage = each.AppImage
		}
	}
	return result, true
}

func DecodeFilename(groupedWordMap map[string][]*KeyedMeaning, filename string) []*Meaning {
	var result []*Meaning
	length := len(filename)
	suffixOnly := false

	for i := 0; i < length; {
		firstChar := string(filename[i])
		if meanings, found := groupedWordMap[firstChar]; found {
			matched := false
			for _, meaning := range meanings {
				keyLen := len(meaning.Key)
				if i+keyLen <= length && filename[i:i+keyLen] == meaning.Key {
					if suffixOnly && !meaning.SuffixOnly {
						continue
					}
					result = append(result, meaning.Meaning)
					if meaning.SuffixOnly {
						suffixOnly = true
					}
					i += keyLen
					matched = true
					break
				}
			}
			if !matched {
				return nil
			}
		} else {
			return nil
		}

		// Skip separators
		for i < length-1 && (filename[i] == '-' || filename[i] == '_' || filename[i] == '.') {
			i++
		}
	}

	return result
}

type KeyedMeaning struct {
	*Meaning
	Key string
}

func GroupAndSort(wordMap map[string]*Meaning) map[string][]*KeyedMeaning {
	keyGroups := make(map[string][]*KeyedMeaning)
	for key := range wordMap {
		firstChar := string(key[0])
		keyGroups[firstChar] = append(keyGroups[firstChar], &KeyedMeaning{
			Meaning: wordMap[key],
			Key:     key,
		})
	}
	for letter := range keyGroups {
		sort.Slice(keyGroups[letter], func(i, j int) bool {
			return len(keyGroups[letter][i].Key) > len(keyGroups[letter][j].Key)
		})
	}
	return keyGroups
}

func GenerateWordMeanings(gitRepo string, version string) map[string]*Meaning {
	wordMap := map[string]*Meaning{
		// Gentoo
		"alpha":  {Keyword: "~alpha"},
		"~alpha": {Keyword: "~alpha"},
		"amd64":  {Keyword: "~amd64"},
		"~amd64": {Keyword: "~amd64"},
		"arm":    {Keyword: "~arm"},
		"~arm":   {Keyword: "~arm"},
		"arm64":  {Keyword: "~arm64"},
		"~arm64": {Keyword: "~arm64"},
		"hppa":   {Keyword: "~hppa"},
		"~hppa":  {Keyword: "~hppa"},
		"ia64":   {Keyword: "~ia64"},
		"~ia64":  {Keyword: "~ia64"},
		"mips":   {Keyword: "~mips"},
		"~mips":  {Keyword: "~mips"},
		"ppc":    {Keyword: "~ppc"},
		"~ppc":   {Keyword: "~ppc"},
		"ppc64":  {Keyword: "~ppc64"},
		"~ppc64": {Keyword: "~ppc64"},
		"riscv":  {Keyword: "~riscv"},
		"~riscv": {Keyword: "~riscv"},
		"s390":   {Keyword: "~s390"},
		"~s390":  {Keyword: "~s390"},
		"sparc":  {Keyword: "~sparc"},
		"~sparc": {Keyword: "~sparc"},
		"x86":    {Keyword: "~x86"},
		"~x86":   {Keyword: "~x86"},
		// Flutter / android
		"x64":   {Keyword: "~amd64"},
		"arm32": {Keyword: "~arm"},
		// Rust
		"aarch64-unknown-linux-gnu":     {Keyword: "~arm64", OS: "linux", Toolchain: "gnu"},
		"i686-pc-windows-gnu":           {Keyword: "~x86", OS: "windows", Toolchain: "gnu"},
		"i686-pc-windows-msvc":          {Keyword: "~x86", OS: "windows", Toolchain: "msvc"},
		"i686-unknown-linux-gnu":        {Keyword: "~x86", OS: "linux", Toolchain: "gnu"},
		"x86_64-apple-darwin":           {Keyword: "~amd64", OS: "macosx"},
		"x86_64-pc-windows-gnu":         {Keyword: "~amd64", OS: "windows", Toolchain: "gnu"},
		"x86_64-pc-windows-msvc":        {Keyword: "~amd64", OS: "windows", Toolchain: "msvc"},
		"x86_64-unknown-linux-gnu":      {Keyword: "~amd64", OS: "linux", Toolchain: "gnu"},
		"aarch64-unknown-linux-musl":    {Keyword: "~arm64", OS: "linux", Toolchain: "musl"},
		"arm-unknown-linux-gnueabi":     {Keyword: "~arm", OS: "linux", Toolchain: "gnueabi"},
		"arm-unknown-linux-gnueabihf":   {Keyword: "~arm", OS: "linux", Toolchain: "gnueabihf"},
		"armv7-unknown-linux-gnueabihf": {Keyword: "~arm", OS: "linux", Toolchain: "gnueabihf"},
		"powerpc-unknown-linux-gnu":     {Keyword: "~ppc", OS: "linux", Toolchain: "gnu"},
		"powerpc64-unknown-linux-gnu":   {Keyword: "~ppc64", OS: "linux", Toolchain: "gnu"},
		"powerpc64le-unknown-linux-gnu": {Keyword: "~ppc64", OS: "linux", Toolchain: "gnu"},
		"riscv64gc-unknown-linux-gnu":   {Keyword: "~riscv", OS: "linux", Toolchain: "gnu"},
		"s390x-unknown-linux-gnu":       {Keyword: "~s390", OS: "linux", Toolchain: "gnu"},
		"x86_64-unknown-linux-musl":     {Keyword: "~amd64", OS: "linux", Toolchain: "musl"},
		"unknown":                       {},
		"linux":                         {OS: "linux"},
		"windows":                       {OS: "windows"},
		"win32":                         {OS: "windows", Keyword: "~x86"},
		"win64":                         {OS: "windows", Keyword: "~amd64"},
		"macosx":                        {OS: "macosx"},
		"macos":                         {OS: "macosx"},
		"darwin":                        {OS: "macosx"},
		"gnu":                           {Toolchain: "gnu"},
		"musl":                          {Toolchain: "musl"},
		"gnueabi":                       {Toolchain: "gnueabi"},
		"gnueabihf":                     {Toolchain: "gnueabihf"},
		"msvc":                          {Toolchain: "msvc"},
		"armv7":                         {Keyword: "~arm"},
		"powerpc":                       {Keyword: "~ppc"},
		"powerpc64":                     {Keyword: "~ppc64"},
		"powerpc64le":                   {Keyword: "~ppc64"},
		"riscv64gc":                     {Keyword: "~riscv"},
		"s390x":                         {Keyword: "~s390"},
		"x86_64":                        {Keyword: "~amd64"},
		"i686":                          {Keyword: "~x86"},
		"armhf":                         {Keyword: "~arm"},
		"aarch64":                       {Keyword: "~arm64"},
		// AppImage
		"AppImage": {AppImage: true, OS: "linux", SuffixOnly: true},
		"deb":      {Container: "deb", OS: "linux", SuffixOnly: true},
		"rpm":      {Container: "deb", OS: "linux", SuffixOnly: true},
		"exe":      {OS: "windows", SuffixOnly: true},
		"dmg":      {OS: "macosx", SuffixOnly: true},
		"pkg":      {OS: "macosx", SuffixOnly: true},
		"gz":       {Container: "gz", SuffixOnly: true},
		"bz2":      {Container: "bz2", SuffixOnly: true},
		"tar":      {Container: "tar", SuffixOnly: true},
		"zip":      {Container: "zip", SuffixOnly: true},
	}
	if v, ok := wordMap[gitRepo]; ok {
		v.ProjectName = true
	} else {
		wordMap[gitRepo] = &Meaning{ProjectName: true}
	}
	if v, ok := wordMap[version]; ok {
		v.Version = true
	} else {
		wordMap[version] = &Meaning{Version: true}
	}
	if v, ok := wordMap["v"+version]; ok {
		v.Version = true
	} else {
		wordMap["v"+version] = &Meaning{Version: true}
	}

	return wordMap
}

func StringOrDefault(description *string, defaultStr string) string {
	if description == nil {
		return defaultStr
	}
	return *description
}
