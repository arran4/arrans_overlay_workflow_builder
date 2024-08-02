package arrans_overlay_workflow_builder

import (
	"archive/zip"
	"fmt"
	"github.com/arran4/arrans_overlay_workflow_builder/util"
	"github.com/google/go-github/v62/github"
	"github.com/probonopd/go-appimage/src/goappimage"
	"log"
	"os"
	"slices"
	"sort"
	"strings"
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
	SuffixOnly      bool
	CaseInsensitive bool
	// Required for the URL only atm:
	ReleaseAsset *github.ReleaseAsset
	// Unmatched
	Unmatched []string

	// Transient information
	tempFile         string
	OriginalFilename string
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

func GenerateAppImageGithubReleaseConfigEntry(gitRepo, tagOverride, prefix string) (*InputConfig, error) {
	repoName, ic, versions, tags, releaseInfo, config, err := NewInputConfigurationFromRepo(gitRepo, tagOverride, prefix)
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
	for _, appImage := range appImages {
		if err := appImage.GetInformationFromAppImage(repoName, ic); err != nil {
			return nil, err
		}
		// Desktop icon: ai.Desktop.Section("Desktop Entry").Key("Icon").Value()
	}
	return ic, nil
}

func (appImage *AppImageFileInfo) GetInformationFromAppImage(repoName string, ic *InputConfig) error {
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
			DesktopFile:       "",
			Icons:             []string{},
			Dependencies:      []string{},
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
			archivedFiles = append(archivedFiles, &AppImageFileInfo{
				Container:    container.Filename,
				Filename:     f.Name,
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
				result.Unmatched = append(result.Unmatched, each.Captured)
			} else {
				result.ProgramName = each.Captured
			}
		}
	}
	return result, true
}
