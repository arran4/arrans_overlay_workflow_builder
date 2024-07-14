package arrans_overlay_workflow_builder

import (
	"archive/zip"
	"context"
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/arran4/arrans_overlay_workflow_builder/util"
	"github.com/google/go-github/v62/github"
	"github.com/probonopd/go-appimage/src/goappimage"
	"github.com/stoewer/go-strcase"
	"log"
	"os"
	"sort"
	"strings"
	"unicode"
)

type FileInfo struct {
	// Core properties
	// Gentoo keyword
	Keyword string
	OS      string
	// Generally msvc, gnu, musl, etc
	Toolchain string
	// Like tar, or zip, also a bit of bz2, and gz but not proper "containers", later replaced by the container of the
	// contained file
	Container   string
	ProgramName string
	// The separator -_-
	Separator bool
	Captured  string

	// Compiled only
	Containers []string
	// App image filename, not container
	Filename string

	// Relevant restraint + identification
	AppImage bool

	// Identification
	Version     bool
	Tag         bool
	ProjectName bool

	// Match rules
	SuffixOnly      bool
	CaseInsensitive bool
	// Required for the URL only atm:
	ReleaseAsset *github.ReleaseAsset
	Unmatched    bool

	// Transient information
	tempFile         string
	OriginalFilename string
}

func ConfigAddAppImageGithubReleases(toConfig string, gitRepo string) error {
	ic, err := GenerateAppImageGithubReleaseConfigEntry(gitRepo, "")
	if err != nil {
		return err
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

func ConfigViewAppImageGithubReleases(gitRepo, tagOverride string) error {
	ic, err := GenerateAppImageGithubReleaseConfigEntry(gitRepo, tagOverride)
	if err != nil {
		return err
	}

	log.Printf("Showing potential addition to config as entry id: %d", ic.EntryNumber)
	_ = os.Stderr.Sync()
	fmt.Printf("%s", ic.String())
	return nil
}

func GenerateAppImageGithubReleaseConfigEntry(gitRepo, tagOverride string) (*InputConfig, error) {
	client := github.NewClient(nil)
	if token, ok := os.LookupEnv("GITHUB_TOKEN"); ok {
		client = client.WithAuthToken(token)
	}
	ownerName, repoName, err := util.ExtractGithubOwnerRepo(gitRepo)
	if err != nil {
		return nil, fmt.Errorf("github url parse: %w", err)
	}
	log.Printf("Getting details for %s's %s", ownerName, repoName)
	ctx := context.Background()
	repo, _, err := client.Repositories.Get(ctx, ownerName, repoName)
	if err != nil {
		return nil, fmt.Errorf("github repo fetch: %w", err)
	}
	var licenseName *string
	if repo.License != nil {
		licenseName = repo.License.Name
	}
	ic := &InputConfig{
		Type:             "Github AppImage",
		GithubProjectUrl: gitRepo,
		//Category:          "",
		EbuildName:  fmt.Sprintf("%s-appimage", util.TrimSuffixes(strcase.KebabCase(repoName), "-AppImage", "-appimage")),
		Description: util.StringOrDefault(repo.Description, "TODO"),
		Homepage:    util.StringOrDefault(repo.Homepage, ""),
		GithubRepo:  repoName,
		GithubOwner: ownerName,
		License:     util.StringOrDefault(licenseName, "unknown"),
	}
	var versions = []string{}
	var tags = []string{tagOverride}
	var releaseInfo *github.RepositoryRelease
	if tagOverride == "" {
		releaseInfo, _, err = client.Repositories.GetLatestRelease(ctx, ownerName, repoName)
		if err != nil {
			return nil, fmt.Errorf("github latest release fetch: %w", err)
		}

		v, err := semver.NewVersion(releaseInfo.GetTagName())
		if err != nil {
			return nil, fmt.Errorf("github latest release tag parse %s: %w", releaseInfo.GetTagName(), err)
		}
		versions = []string{v.String()}
		tags = []string{"v" + v.String()}
	} else {
		releaseInfo, _, err = client.Repositories.GetReleaseByTag(ctx, ownerName, repoName, tagOverride)
		if err != nil {
			return nil, fmt.Errorf("github latest release fetch: %w", err)
		}

	}

	log.Printf("Latest release %v", versions)

	var wordMap = GroupAndSort(GenerateWordMeanings(repoName, versions, tags))

	var files []*FileInfo
	for _, asset := range releaseInfo.Assets {
		files = append(files, &FileInfo{
			Filename:     asset.GetName(),
			ReleaseAsset: asset,
		})
	}
	appImages, containers := ExtractAppsAndContainers(files, wordMap)
	if len(appImages) == 0 && len(containers) > 0 {
		log.Printf("No app images found, but some archives / compressed files")
		for _, container := range containers {
			log.Printf("Searching: %s", container.Filename)
			archivedFiles, err := SearchArchiveForFiles(container)
			if err != nil {
				if archivedFiles != nil {
					for _, af := range archivedFiles {
						if err := os.Remove(af.tempFile); err != nil {
							log.Printf("Error removing temp file: %s", err)
						}
						af.tempFile = ""
					}
				}
				return nil, err
			}
			nai, nc := ExtractAppsAndContainers(archivedFiles, wordMap)
			for _, nce := range nc {
				if len(nce.tempFile) == 0 {
					continue
				}
				if err := os.Remove(nce.tempFile); err != nil {
					log.Printf("Error removing temp file: %s", err)
				}
				nce.tempFile = ""
			}
			if len(nai) > 0 {
				appImages = append(appImages, nai...)
			}
		}
	}
	if len(appImages) == 0 && len(containers) == 0 {
		return nil, fmt.Errorf("no app imagee or archives/compressed files found")
	}
	if ic.Programs == nil {
		ic.Programs = map[string]*Program{}
	}
	for _, appImage := range appImages {
		err := GetInformationFromAppImage(appImage, repoName, ic)
		if err != nil {
			return nil, err
		}
		// Desktop icon: ai.Desktop.Section("Desktop Entry").Key("Icon").Value()
	}
	return ic, nil
}

func GetInformationFromAppImage(appImage *FileInfo, repoName string, ic *InputConfig) error {
	url := appImage.ReleaseAsset.GetBrowserDownloadURL()
	var err error
	if appImage.tempFile == "" {
		log.Printf("Downloading %s", url)
		appImage.tempFile, err = util.DownloadUrlToTempFile(url)
		if err != nil {
			return fmt.Errorf("downloading release: %w", err)
		}
	}
	if len(appImage.tempFile) > 0 {
		defer func() {
			if err := os.Remove(appImage.tempFile); err != nil {
				log.Printf("Error removing temp file: %s", err)
			}
			appImage.tempFile = ""
		}()
	}
	log.Printf("Got %s", appImage.tempFile)
	var programName string = appImage.ProgramName
	if programName == "" {
		programName = repoName
	}
	program, ok := ic.Programs[programName]
	if !ok {
		program = &Program{
			ProgramName:       programName,
			InstalledFilename: fmt.Sprintf("%s.AppImage", programName),
			ReleasesFilename:  map[string]string{},
			ArchiveFilename:   map[string]string{},
		}
		ic.Programs[appImage.ProgramName] = program
	}
	if appImage.Container != "" {
		program.ReleasesFilename[strings.TrimPrefix(appImage.Keyword, "~")] = appImage.Container
		program.ArchiveFilename[strings.TrimPrefix(appImage.Keyword, "~")] = appImage.Filename
	} else {
		program.ReleasesFilename[strings.TrimPrefix(appImage.Keyword, "~")] = appImage.Filename
	}
	ai, err := goappimage.NewAppImage(appImage.tempFile)
	if err != nil {
		return fmt.Errorf("reading AppImage %s %s: %w", appImage.Filename, url, err)
	}
	files := ai.ListFiles(".")
	for _, f := range files {
		if strings.HasSuffix(f, ".desktop") {
			program.DesktopFile = f
			log.Printf("Found a desktop file %s", program.DesktopFile)
			break
		}
	}
	return nil
}

func SearchArchiveForFiles(container *FileInfo) ([]*FileInfo, error) {
	url := container.ReleaseAsset.GetBrowserDownloadURL()
	log.Printf("Downloading %s", url)
	var err error
	container.tempFile, err = util.DownloadUrlToTempFile(url)
	if err != nil {
		return nil, fmt.Errorf("downloading release: %w", err)
	}
	defer func() {
		if err := os.Remove(container.tempFile); err != nil {
			log.Printf("Error removing temp file: %s", err)
		}
		container.tempFile = ""
	}()

	log.Printf("Got %s => %s", url, container.tempFile)

	var archivedFiles []*FileInfo
	// TODO support weirdly nested containers.
	switch strings.Join(container.Containers, ".") {
	case "zip":
		zf, err := zip.OpenReader(container.tempFile)
		if err != nil {
			return archivedFiles, fmt.Errorf("opening zip file: %s: %w", url, err)
		}
		defer func() {
			if err := zf.Close(); err != nil {
				log.Printf("Error closing file: %s: %s", container.tempFile, err)
			}
		}()
		for _, f := range zf.File {
			zfr, err := f.Open()
			tmpFile, err := util.SaveReaderToTempFile(zfr)
			if err != nil {
				return archivedFiles, fmt.Errorf("extracting file %s from %s: %w", f.Name, url, err)
			}
			defer func() {
				if err := zfr.Close(); err != nil {
					log.Printf("error closing zip file %s from %s: %s", f.Name, url, err)
				}
			}()
			archivedFiles = append(archivedFiles, &FileInfo{
				Container:    container.Filename,
				Filename:     f.Name,
				tempFile:     tmpFile,
				ReleaseAsset: container.ReleaseAsset,
			})
		}
	}
	return archivedFiles, nil
}

func ExtractAppsAndContainers(base []*FileInfo, wordMap map[string][]*KeyedMeaning) ([]*FileInfo, []*FileInfo) {
	var appImages []*FileInfo
	var containers []*FileInfo
	for _, base := range base {
		log.Printf("Is %s an AppImage?", base.Filename)
		results := DecodeFilename(wordMap, base.Filename)
		if len(results) == 0 {
			log.Printf("Can't decode %s", base.Filename)
			continue
		}
		compiled, ok := CompileMeanings(results, base)
		if !ok {
			log.Printf("Can't simplify %s", base.Filename)
			continue
		}
		if compiled.OS != "" && compiled.OS != "linux" {
			log.Printf("Not for linux %s", base.Filename)
			continue
		}
		if compiled.Keyword == "" {
			// Default to amd64 because that's just a thing you do.
			compiled.Keyword = "~amd64"
		}
		switch {
		case len(compiled.Containers) > 0:
			containers = append(containers, compiled)
			log.Printf("Is %s an AppImage? - Maybe archived", base.Filename)
		case compiled.AppImage && len(compiled.Containers) == 0:
			appImages = append(appImages, compiled)
			log.Printf("Is %s an AppImage? - Yes", base.Filename)
		default:
			log.Printf("Doesn't have AppImage, or a archived AppImage in it %s", base.Filename)
			continue
		}
	}
	return appImages, containers
}

func CompileMeanings(input []*FileInfo, base *FileInfo) (*FileInfo, bool) {
	result := &FileInfo{
		SuffixOnly: true,
	}
	if base != nil {
		result.ReleaseAsset = base.ReleaseAsset
		result.Container = base.Container
		result.OriginalFilename = base.Filename
		result.OS = base.OS
		result.Keyword = base.Keyword
		result.Toolchain = base.Toolchain
		result.tempFile = base.tempFile
	}
	for _, each := range input {
		switch {
		case each.Version:
			result.Filename += "${VERSION}"
		case each.Tag:
			result.Filename += "${TAG}"
		default:
			result.Filename += each.Captured
		}
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

		if each.Tag {
			result.Tag = each.Tag
		}

		if each.ProjectName {
			result.ProjectName = each.ProjectName
		}

		if each.AppImage {
			result.AppImage = each.AppImage
		}

		if each.Unmatched {
			if result.ProgramName != "" || each.SuffixOnly {
				return nil, false
			}
			result.ProgramName = each.Captured
		}
	}
	return result, true
}

func DecodeFilename(groupedWordMap map[string][]*KeyedMeaning, filename string) []*FileInfo {
	var result []*FileInfo
	length := len(filename)
	suffixOnly := false
	unmatched := -1
	var sep *FileInfo
	for i := 0; i < length; {
		matched := false
		firstChar := string(filename[i])
		if meanings, found := groupedWordMap[firstChar]; found {
			for _, meaning := range meanings {
				keyLen := len(meaning.Key)
				if i+keyLen <= length && (!meaning.CaseInsensitive && filename[i:i+keyLen] == meaning.Key || meaning.CaseInsensitive && strings.EqualFold(filename[i:i+keyLen], meaning.Key)) {
					if unmatched != -1 {
						if unmatched < i-2 {
							result = append(result, &FileInfo{
								Unmatched:  true,
								Captured:   filename[unmatched : i-1],
								SuffixOnly: suffixOnly,
							})
							if sep != nil {
								result = append(result, sep)
								sep = nil
							}
						}
						unmatched = -1
					}
					if suffixOnly && !meaning.SuffixOnly {
						continue
					}
					var fi = *meaning.FileInfo
					fi.Captured = filename[i : i+keyLen]
					result = append(result, &fi)
					if meaning.SuffixOnly {
						suffixOnly = true
					}
					i += keyLen
					matched = true
					break
				}
			}
		}

		if !matched {
			if unmatched == -1 {
				unmatched = i
			}
			for i < length && !(filename[i] == '-' || filename[i] == '_' || filename[i] == '.') {
				i++
			}
		}

		// Skip separators
		if i < length && (filename[i] == '-' || filename[i] == '_' || filename[i] == '.') {
			sep = &FileInfo{
				Captured:  string(filename[i]),
				Separator: true,
			}
			if unmatched == -1 {
				result = append(result, sep)
				sep = nil
			}
			i++
		}
	}

	if unmatched != -1 {
		if unmatched < length {
			result = append(result, &FileInfo{
				Unmatched:  true,
				Captured:   filename[unmatched:],
				SuffixOnly: suffixOnly,
			})
		}
		unmatched = -1
	}

	return result
}

type KeyedMeaning struct {
	*FileInfo
	Key string
}

func GroupAndSort(wordMap map[string]*FileInfo) map[string][]*KeyedMeaning {
	keyGroups := make(map[string][]*KeyedMeaning)
	for key := range wordMap {
		meaning := wordMap[key]
		firstChar := string(key[0])
		keyGroups[firstChar] = append(keyGroups[firstChar], &KeyedMeaning{
			FileInfo: wordMap[key],
			Key:      key,
		})
		if meaning.CaseInsensitive {
			if unicode.IsUpper(rune(firstChar[0])) {
				firstChar = string(unicode.ToLower(rune(firstChar[0])))
			} else {
				firstChar = string(unicode.ToUpper(rune(firstChar[0])))
			}
			keyGroups[firstChar] = append(keyGroups[firstChar], &KeyedMeaning{
				FileInfo: wordMap[key],
				Key:      key,
			})
		}
	}
	for letter := range keyGroups {
		sort.Slice(keyGroups[letter], func(i, j int) bool {
			return len(keyGroups[letter][i].Key) > len(keyGroups[letter][j].Key)
		})
	}
	return keyGroups
}

func GenerateWordMeanings(gitRepo string, versions []string, tags []string) map[string]*FileInfo {
	wordMap := map[string]*FileInfo{
		"x86-64": {Keyword: "~amd64"},
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
		"lin":                           {OS: "linux"},
		"windows":                       {OS: "windows"},
		"win":                           {OS: "windows"},
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
		wordMap[gitRepo] = &FileInfo{ProjectName: true, CaseInsensitive: true}
	}
	for _, version := range versions {
		if v, ok := wordMap[version]; ok {
			v.Version = true
		} else {
			wordMap[version] = &FileInfo{Version: true}
		}
	}
	for _, tag := range tags {
		if v, ok := wordMap[tag]; ok {
			v.Tag = true
		} else {
			wordMap[tag] = &FileInfo{Tag: true}
		}
	}

	return wordMap
}
