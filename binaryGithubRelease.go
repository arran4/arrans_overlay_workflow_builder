package arrans_overlay_workflow_builder

import (
	"archive/tar"
	"archive/zip"
	"compress/bzip2"
	"compress/gzip"
	"errors"
	"fmt"
	"github.com/arran4/arrans_overlay_workflow_builder/util"
	"github.com/google/go-github/v62/github"
	"io"
	"log"
	"os"
	"path"
	"strings"
)

type BinaryReleaseFileInfo struct {
	// Core properties
	// Gentoo keyword
	Keyword string
	OS      string
	// Generally msvc, gnu, musl, etc
	Toolchain string
	// Like tar, or zip, also a bit of bz2, and gz but not proper "containers", later replaced by the container of the
	// contained file
	ProgramName string

	// Compiled only
	Containers []string
	// Binary filename, not container
	Filename string

	// Relevant restraint + identification
	Binary bool

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
	ArchivePathname  string
	ExecutableBit    bool
	Container        *BinaryReleaseFileInfo
}

func ConfigAddBinaryGithubReleases(toConfig, gitRepo, tagOverride, tagPrefix string) error {
	ic, err := GenerateBinaryGithubReleaseConfigEntry(gitRepo, tagOverride, tagPrefix)
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

func ConfigViewBinaryGithubReleases(gitRepo, tagOverride, tagPrefix string) error {
	ic, err := GenerateBinaryGithubReleaseConfigEntry(gitRepo, tagOverride, tagPrefix)
	if err != nil {
		return err
	}

	log.Printf("Showing potential addition to config as entry id: %d", ic.EntryNumber)
	_ = os.Stderr.Sync()
	fmt.Printf("%s\n", ic.String())
	return nil
}

func GenerateBinaryGithubReleaseConfigEntry(gitRepo, tagOverride, prefix string) (*InputConfig, error) {
	repoName, ic, versions, tags, releaseInfo, config, err := NewInputConfigurationFromRepo(gitRepo, tagOverride, prefix)
	if err != nil {
		return config, err
	}

	var wordMap = GroupAndSort(GenerateWordMeanings(repoName, versions, tags))

	var files []*BinaryReleaseFileInfo
	for _, asset := range releaseInfo.Assets {
		files = append(files, &BinaryReleaseFileInfo{
			Filename:     asset.GetName(),
			ReleaseAsset: asset,
		})
	}
	rootFiles := BinaryReleaseFiles(files).FindFiles(wordMap, nil)
	if len(rootFiles.Binaries) == 0 && len(rootFiles.CompressedArchives) > 0 {
		log.Printf("No binaries found, but some archives / compressed files")
		for _, container := range rootFiles.CompressedArchives {
			log.Printf("Searching: %s", container.Filename)
			archivedFiles, err := container.SearchArchiveForFiles()
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
			containerFiles := BinaryReleaseFiles(archivedFiles).FindFiles(wordMap, rootFiles)
			for _, nce := range containerFiles.CompressedArchives {
				if len(nce.tempFile) == 0 {
					continue
				}
				if err := os.Remove(nce.tempFile); err != nil {
					log.Printf("Error removing temp file: %s", err)
				}
				nce.tempFile = ""
			}
			rootFiles.CompressedArchiveContent[container.Filename] = containerFiles
		}
	}
	if rootFiles.CountBinaries() == 0 && rootFiles.CountCompressedArchives() == 0 {
		return nil, fmt.Errorf("no binaries or archives/compressed files found")
	}
	if ic.Programs == nil {
		ic.Programs = map[string]*Program{}
	}
	binaries := rootFiles.AllBinaries()
	for _, binary := range binaries {
		p, ok := ic.Programs[binary.ProgramName]
		if !ok {
			p = &Program{
				ProgramName:       binary.ProgramName,
				InstalledFilename: fmt.Sprintf("%s", binary.ProgramName),
				DesktopFile:       "",
				Icons:             []string{},
				Dependencies:      []string{},
				ReleasesFilename:  map[string]string{},
				ArchiveFilename:   map[string]string{},
			}
			ic.Programs[binary.ProgramName] = p
		}
		if binary.Container != nil {
			p.ReleasesFilename[strings.TrimPrefix(binary.Keyword, "~")] = binary.Container.Filename
			p.ArchiveFilename[strings.TrimPrefix(binary.Keyword, "~")] = binary.ArchivePathname
		} else {
			p.ReleasesFilename[strings.TrimPrefix(binary.Keyword, "~")] = binary.Filename
		}
	}
	return ic, nil
}

func (container *BinaryReleaseFileInfo) SearchArchiveForFiles() ([]*BinaryReleaseFileInfo, error) {
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

	var archivedFiles []*BinaryReleaseFileInfo
	// TODO support weirdly nested containers.
	switch strings.ToLower(strings.Join(container.Containers, ".")) {
	case "tar.gz", "tar.bz2", "tar":
		var cr io.Reader
		f, err := os.Open(container.tempFile)
		if err != nil {
			return archivedFiles, fmt.Errorf("opening file: %s: %w", url, err)
		}
		defer func() {
			if err := f.Close(); err != nil {
				log.Printf("Error closing file: %s: %s", container.tempFile, err)
			}
		}()
		if len(container.Containers) >= 2 {
			t := container.Containers[1]
			switch strings.ToLower(t) {
			case "gz":
				cr, err = gzip.NewReader(f)
				if err != nil {
					return archivedFiles, fmt.Errorf("opening gzip file: %s: %w", url, err)
				}
			case "bz2":
				cr = bzip2.NewReader(f)
			default:
				return archivedFiles, fmt.Errorf("unknown format for file: %s", url)
			}
		} else {
			cr = f
		}
		tr := tar.NewReader(cr)

		for {
			zfh, err := tr.Next()
			if zfh == nil || errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return archivedFiles, fmt.Errorf("reading next tar file: %s: %w", url, err)
			}
			if zfh.FileInfo().IsDir() {
				continue
			}
			tmpFile, err := util.SaveReaderToTempFile(tr)
			if err != nil {
				return archivedFiles, fmt.Errorf("extracting file %s from %s: %w", zfh.Name, url, err)
			}
			_, fn := path.Split(zfh.Name)
			archivedFiles = append(archivedFiles, &BinaryReleaseFileInfo{
				Container:       container,
				ArchivePathname: zfh.Name,
				Filename:        fn,
				tempFile:        tmpFile,
				ReleaseAsset:    container.ReleaseAsset,
				ExecutableBit:   (zfh.Mode & 0o0500) == 0o0500,
			})
		}
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
			if f.Mode().IsDir() {
				continue
			}
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
			_, fn := path.Split(f.Name)
			archivedFiles = append(archivedFiles, &BinaryReleaseFileInfo{
				Container:       container,
				ArchivePathname: f.Name,
				Filename:        fn,
				tempFile:        tmpFile,
				ReleaseAsset:    container.ReleaseAsset,
				ExecutableBit:   (f.Mode().Perm() & 0o500) == 0o500,
			})
		}
	}
	return archivedFiles, nil
}

type BinaryReleaseFiles []*BinaryReleaseFileInfo

type FileTypes struct {
	CompressedArchives       []*BinaryReleaseFileInfo
	CompressedArchiveContent map[string]*FileTypes
	Binaries                 []*BinaryReleaseFileInfo
	ManualPages              []*BinaryReleaseFileInfo
	ShellCompletion          []*BinaryReleaseFileInfo
	Root                     *FileTypes
}

func (t *FileTypes) CountBinaries() int {
	result := len(t.Binaries)
	for _, each := range t.CompressedArchiveContent {
		result += len(each.Binaries)
	}
	return result
}

func (t *FileTypes) CountCompressedArchives() int {
	result := len(t.CompressedArchives)
	for _, each := range t.CompressedArchiveContent {
		result += len(each.CompressedArchives)
	}
	return result
}

func (t *FileTypes) AllBinaries() (result []*BinaryReleaseFileInfo) {
	result = append([]*BinaryReleaseFileInfo{}, t.Binaries...)
	for _, archive := range t.CompressedArchiveContent {
		result = append(result, archive.AllBinaries()...)
	}
	return result
}

func (base BinaryReleaseFiles) FindFiles(wordMap map[string][]*GroupedFilenamePartMeaning, root *FileTypes) *FileTypes {
	result := &FileTypes{
		CompressedArchives:       []*BinaryReleaseFileInfo{},
		Binaries:                 []*BinaryReleaseFileInfo{},
		ManualPages:              []*BinaryReleaseFileInfo{},
		ShellCompletion:          []*BinaryReleaseFileInfo{},
		CompressedArchiveContent: map[string]*FileTypes{},
		Root:                     root,
	}
	for _, base := range base {
		log.Printf("Is %s a binary?", base.Filename)
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
			result.CompressedArchives = append(result.CompressedArchives, compiled)
			log.Printf("Is %s an Binary? - Maybe archived", base.Filename)
		case compiled.Binary && len(compiled.Containers) == 0:
			result.Binaries = append(result.Binaries, compiled)
			log.Printf("Is %s an Binary? - Yes", base.Filename)
		default:
			log.Printf("Doesn't have Binary, or a archived Binary in it %s", base.Filename)
			continue
		}
	}
	return result
}

func (base *BinaryReleaseFileInfo) CompileMeanings(input []*FilenamePartMeaning) (*BinaryReleaseFileInfo, bool) {
	result := &BinaryReleaseFileInfo{
		SuffixOnly: true,
	}
	if base != nil {
		result.ReleaseAsset = base.ReleaseAsset
		result.OriginalFilename = base.Filename
		result.ArchivePathname = base.ArchivePathname
		result.OS = base.OS
		result.Keyword = base.Keyword
		result.Toolchain = base.Toolchain
		result.tempFile = base.tempFile
		result.ExecutableBit = base.ExecutableBit
		result.Binary = base.ExecutableBit
		if base.Container != nil {
			result.Container = base.Container
			if result.OS == "" {
				result.OS = base.Container.OS
			}
			if result.Keyword == "" {
				result.Keyword = base.Container.Keyword
			}
			if result.Toolchain == "" {
				result.Toolchain = base.Container.Toolchain
			}
		}
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
			return nil, false
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
