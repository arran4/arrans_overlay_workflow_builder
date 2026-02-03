package arrans_overlay_workflow_builder

import (
	"archive/tar"
	"archive/zip"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"github.com/arran4/arrans_overlay_workflow_builder/util"
	"github.com/google/go-github/v62/github"
	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz"
	"io"
	"log"
	"os"
	"slices"
	"sort"
	"strings"
	"unicode"
)

type AppImageFileInfo struct {
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
	SuffixOnly       bool
	CaseInsensitive  bool
	KeywordDefaulted bool
	// Required for the URL only atm:
	ReleaseAsset *github.ReleaseAsset
	// Unmatched
	Unmatched []string

	// Transient information
	tempFile         string
	OriginalFilename string
	Installer        bool
}

func ConfigAddAppImageGithubReleases(toConfig, gitRepo, tagOverride, tagPrefix string) error {
	ic, err := GenerateAppImageGithubReleaseConfigEntry(gitRepo, tagOverride, tagPrefix)
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

func ConfigViewAppImageGithubReleases(gitRepo, tagOverride, tagPrefix string) error {
	ic, err := GenerateAppImageGithubReleaseConfigEntry(gitRepo, tagOverride, tagPrefix)
	if err != nil {
		return err
	}

	log.Printf("Showing potential addition to config as entry id: %d", ic.EntryNumber)
	_ = os.Stderr.Sync()
	fmt.Printf("%s\n", ic.String())
	return nil
}

func GenerateAppImageGithubReleaseConfigEntry(gitRepo, tagOverride, tagPrefix string) (*InputConfig, error) {
	repoName, ic, versions, tags, releaseInfo, config, err := NewInputConfigurationFromRepo(gitRepo, tagOverride, tagPrefix, "-appimage", "Github AppImage Release")
	if err != nil {
		return config, err
	}

	var wordMap = GroupAndSort(GenerateWordMeanings(repoName, versions, tags))

	var files []*AppImageFileInfo
	for _, asset := range releaseInfo.Assets {
		files = append(files, &AppImageFileInfo{
			Filename:     asset.GetName(),
			ReleaseAsset: asset,
		})
	}
	appImages, containers := AppImageFiles(files).ExtractAppImagesAndContainers(wordMap)
	if len(appImages) == 0 && len(containers) > 0 {
		log.Printf("No app images found, but some archives / compressed files")
		for _, container := range containers {
			log.Printf("Searching: %s", container.Filename)
			archivedFiles, err := container.SearchArchiveForAppImageFiles()
			if err != nil {
				for _, af := range archivedFiles {
					if err := os.Remove(af.tempFile); err != nil {
						log.Printf("Error removing temp file: %s", err)
					}
					af.tempFile = ""
				}
				return nil, err
			}
			nai, nc := AppImageFiles(archivedFiles).ExtractAppImagesAndContainers(wordMap)
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
	appImages = selectPrimaryAppImage(appImages, repoName)
	for _, appImage := range appImages {
		if err := appImage.GetInformationFromAppImage(repoName, ic); err != nil {
			return nil, err
		}
		// Desktop icon: ai.Desktop.Section("Desktop Entry").Key("Icon").Value()
	}
	return ic, nil
}

func selectPrimaryAppImage(appImages []*AppImageFileInfo, repoName string) []*AppImageFileInfo {
	if len(appImages) <= 1 {
		return appImages
	}
	repoNorm := normalizeName(repoName)
	sort.Slice(appImages, func(i, j int) bool {
		ai, aj := appImages[i], appImages[j]
		key := func(a *AppImageFileInfo) (int, int) {
			norm := normalizeName(a.ProgramName)
			switch {
			case norm == repoNorm:
				return 0, 0
			case a.ProgramName == "":
				return 1, 0
			default:
				return 2, stringDistance(norm, repoNorm)
			}
		}
		ki1, ki2 := key(ai)
		kj1, kj2 := key(aj)
		if ki1 != kj1 {
			return ki1 < kj1
		}
		if ki2 != kj2 {
			return ki2 < kj2
		}
		return ai.OriginalFilename < aj.OriginalFilename
	})
	log.Printf("Multiple AppImages found, using %s", appImages[0].OriginalFilename)
	return []*AppImageFileInfo{appImages[0]}
}

func normalizeName(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

func stringDistance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	la, lb := len(ra), len(rb)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	dp := make([][]int, la+1)
	for i := range dp {
		dp[i] = make([]int, lb+1)
	}
	for i := 0; i <= la; i++ {
		dp[i][0] = i
	}
	for j := 0; j <= lb; j++ {
		dp[0][j] = j
	}
	for i := 1; i <= la; i++ {
		for j := 1; j <= lb; j++ {
			cost := 0
			if ra[i-1] != rb[j-1] {
				cost = 1
			}
			dp[i][j] = dp[i-1][j-1] + cost
			if dp[i][j-1]+1 < dp[i][j] {
				dp[i][j] = dp[i][j-1] + 1
			}
			if dp[i-1][j]+1 < dp[i][j] {
				dp[i][j] = dp[i-1][j] + 1
			}
		}
	}
	return dp[la][lb]
}

func (appImage *AppImageFileInfo) GetInformationFromAppImage(repoName string, ic *InputConfig) error {
	url := appImage.ReleaseAsset.GetBrowserDownloadURL()
	if appImage.tempFile == "" {
		var err error
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
	programName := appImage.ProgramName
	if programName == "" {
		programName = repoName
	}
	program, ok := ic.Programs[programName]
	if !ok {
		program = &Program{
			ProgramName:  programName,
			DesktopFile:  "",
			Icons:        []string{},
			Dependencies: []string{},
			Binary:       map[string][]string{},
		}
		ic.Programs[programName] = program
	}
	keyword := strings.TrimPrefix(appImage.Keyword, "~")
	program.Binary[keyword] = []string{}
	if appImage.Container != "" {
		program.Binary[keyword] = append(program.Binary[keyword], appImage.Container)
	}
	program.Binary[keyword] = append(program.Binary[keyword], appImage.Filename)
	program.Binary[keyword] = append(program.Binary[keyword], fmt.Sprintf("%s.AppImage", programName))
	ai, err := NewAppImage(appImage.tempFile)
	if err != nil {
		return fmt.Errorf("reading AppImage %s %s: %w", appImage.Filename, url, err)
	}
	defer func() {
		if err := ai.Close(); err != nil {
			log.Printf("Error closing app image: %s", err)
		}
	}()
	for _, f := range ai.ListFiles("usr/share/icons/hicolor/128x128/apps") {
		if strings.HasSuffix(f, ".png") {
			program.Icons = append(program.Icons, "hicolor-apps")
			break
		}
	}
	for _, f := range ai.ListFiles("usr/share/pixmaps") {
		if strings.HasSuffix(f, ".png") {
			program.Icons = append(program.Icons, "pixmaps")
			break
		}
	}
	found := false
	for _, f := range ai.ListFiles(".") {
		if strings.HasSuffix(f, ".png") {
			found = true
			program.Icons = append(program.Icons, "root")
		}
		if strings.HasSuffix(f, ".desktop") {
			program.DesktopFile = f
			log.Printf("Found a desktop file %s", program.DesktopFile)
		}
		if found && program.DesktopFile != "" {
			break
		}
	}

	sort.Strings(program.Icons)
	program.Icons = slices.Compact(program.Icons)

	unknownSymbols, err := ReadDependencies(appImage.tempFile, program)
	if err != nil {
		return err
	}

	if len(unknownSymbols) > 0 {
		return fmt.Errorf("unknown dependencies: %s", strings.Join(unknownSymbols, ", "))
	}

	return nil
}

func (container *AppImageFileInfo) SearchArchiveForAppImageFiles() ([]*AppImageFileInfo, error) {
	switch strings.ToLower(strings.Join(container.Containers, ".")) {
	case "deb", "rpm":
		// Skip repo archives for the moment.
		return nil, nil
	}
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

	var archivedFiles []*AppImageFileInfo
	// TODO support weirdly nested containers.
	containerType := strings.ToLower(strings.Join(container.Containers, "."))
	switch containerType {
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
			if err != nil {
				return archivedFiles, fmt.Errorf("extracting file %s from %s: %w", f.Name, url, err)
			}
			tmpFile, err := util.SaveReaderToTempFile(zfr)
			if err != nil {
				return archivedFiles, fmt.Errorf("saving file %s to temp file: %w", f.Name, err)
			}
			defer func() {
				if err := zfr.Close(); err != nil {
					log.Printf("error closing zip file %s from %s: %s", f.Name, url, err)
				}
			}()
			archivedFiles = append(archivedFiles, &AppImageFileInfo{
				Container:    container.Filename,
				Filename:     f.Name,
				tempFile:     tmpFile,
				ReleaseAsset: container.ReleaseAsset,
			})
		}
	case "tar", "tar.gz", "tgz", "tar.bz2", "tbz", "tbz2", "tar.xz", "tar.zst":
		f, err := os.Open(container.tempFile)
		if err != nil {
			return archivedFiles, fmt.Errorf("opening tar file: %s: %w", url, err)
		}
		defer func() {
			if err := f.Close(); err != nil {
				log.Printf("Error closing file: %s: %s", container.tempFile, err)
			}
		}()
		var r io.Reader = f
		switch containerType {
		case "tar.gz", "tgz":
			gr, err := gzip.NewReader(f)
			if err != nil {
				return archivedFiles, fmt.Errorf("opening gzip file: %s: %w", url, err)
			}
			defer func() {
				if err := gr.Close(); err != nil {
					log.Printf("Error closing gzip file %s: %s", container.tempFile, err)
				}
			}()
			r = gr
		case "tar.bz2", "tbz", "tbz2":
			r = bzip2.NewReader(f)
		case "tar.xz":
			xzr, err := xz.NewReader(f)
			if err != nil {
				return archivedFiles, fmt.Errorf("opening xz file: %s: %w", url, err)
			}
			r = xzr
		case "tar.zst":
			zr, err := zstd.NewReader(f)
			if err != nil {
				return archivedFiles, fmt.Errorf("opening zst file: %s: %w", url, err)
			}
			defer zr.Close()
			r = zr
		}
		tr := tar.NewReader(r)
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return archivedFiles, fmt.Errorf("extracting file %s from %s: %w", container.tempFile, url, err)
			}
			if hdr.FileInfo().IsDir() {
				continue
			}
			tmpFile, err := util.SaveReaderToTempFile(tr)
			if err != nil {
				return archivedFiles, fmt.Errorf("saving file %s to temp file: %w", hdr.Name, err)
			}
			archivedFiles = append(archivedFiles, &AppImageFileInfo{
				Container:    container.Filename,
				Filename:     hdr.Name,
				tempFile:     tmpFile,
				ReleaseAsset: container.ReleaseAsset,
			})
		}
	}
	return archivedFiles, nil
}

type AppImageFiles []*AppImageFileInfo

func (base AppImageFiles) ExtractAppImagesAndContainers(wordMap map[string][]*GroupedFilenamePartMeaning) ([]*AppImageFileInfo, []*AppImageFileInfo) {
	var appImages []*AppImageFileInfo
	var containers []*AppImageFileInfo
	for _, base := range base {
		log.Printf("Is %s an AppImage?", base.Filename)
		results := DecodeFilename(wordMap, base.Filename)
		if len(results) == 0 {
			log.Printf("Can't decode %s", base.Filename)
			continue
		}
		compiled, ok := base.CompileMeanings(results)
		if !ok {
			log.Printf("Can't simplify %s", base.Filename)
			continue
		}
		if len(compiled.Unmatched) > 0 {
			log.Printf("Unmatched tokens in name: %s: %#v", base.Filename, compiled.Unmatched)
			continue
		}
		if compiled.Installer {
			log.Printf("Binary is an installer: %s: skpping", base.Filename)
			continue
		}
		if compiled.OS != "" && compiled.OS != "linux" {
			log.Printf("Not for linux %s", base.Filename)
			continue
		}
		if compiled.Keyword == "" {
			// Default to amd64 because that's just a thing you do.
			compiled.Keyword = "~amd64"
			compiled.KeywordDefaulted = true
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

func (base *AppImageFileInfo) CompileMeanings(input []*FilenamePartMeaning) (*AppImageFileInfo, bool) {
	result := &AppImageFileInfo{
		SuffixOnly: true,
	}
	if base != nil {
		result.ReleaseAsset = base.ReleaseAsset
		result.Container = base.Container
		result.OriginalFilename = base.Filename
		result.OS = base.OS
		result.Keyword = base.Keyword
		result.KeywordDefaulted = base.KeywordDefaulted
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
			if result.Keyword != "" && !result.KeywordDefaulted && result.Keyword != each.Keyword {
				return nil, false
			}
			if result.Keyword == "" || result.KeywordDefaulted {
				result.Keyword = each.Keyword
				result.KeywordDefaulted = false
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

		if each.Installer {
			result.Installer = each.Installer
		}

		if each.AppImage {
			result.AppImage = each.AppImage
		}

		if each.Unmatched {
			if result.ProgramName != "" || each.SuffixOnly {
				result.Unmatched = append(result.Unmatched, each.Captured)
			} else {
				result.ProgramName = each.Captured
			}
		}
	}
	return result, true
}
