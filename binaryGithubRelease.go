package arrans_overlay_workflow_builder

import (
	"archive/tar"
	"archive/zip"
	"compress/bzip2"
	"compress/gzip"
	"debug/elf"
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
	tempFileUsage    int
	Installer        bool
	AppImage         bool
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
	if rootFiles.CountBinaries() == 0 && rootFiles.CountMaybeBinaries() >= 0 {
		log.Printf("No binaries found however some suspected binaries downloading to check them")
		if err := rootFiles.CheckMaybes(); err != nil {
			return nil, fmt.Errorf("checking maybes: %w", err)
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

func (brfi *BinaryReleaseFileInfo) SearchArchiveForFiles() ([]*BinaryReleaseFileInfo, error) {
	url, closeFn, err := brfi.FetchContent()
	if closeFn != nil {
		defer closeFn()
	}
	if err != nil {
		return nil, err
	}

	var archivedFiles []*BinaryReleaseFileInfo
	// TODO support weirdly nested containers.
	switch strings.ToLower(strings.Join(brfi.Containers, ".")) {
	case "tar.gz", "tar.bz2", "tar":
		var cr io.Reader
		f, err := os.Open(brfi.tempFile)
		if err != nil {
			return archivedFiles, fmt.Errorf("opening file: %s: %w", url, err)
		}
		defer func() {
			if err := f.Close(); err != nil {
				log.Printf("Error closing file: %s: %s", brfi.tempFile, err)
			}
		}()
		if len(brfi.Containers) >= 2 {
			t := brfi.Containers[1]
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
				Container:       brfi,
				ArchivePathname: zfh.Name,
				Filename:        fn,
				tempFile:        tmpFile,
				ReleaseAsset:    brfi.ReleaseAsset,
				ExecutableBit:   (zfh.Mode & 0o0500) == 0o0500,
			})
		}
	case "zip":
		zf, err := zip.OpenReader(brfi.tempFile)
		if err != nil {
			return archivedFiles, fmt.Errorf("opening zip file: %s: %w", url, err)
		}
		defer func() {
			if err := zf.Close(); err != nil {
				log.Printf("Error closing file: %s: %s", brfi.tempFile, err)
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
				Container:       brfi,
				ArchivePathname: f.Name,
				Filename:        fn,
				tempFile:        tmpFile,
				ReleaseAsset:    brfi.ReleaseAsset,
				ExecutableBit:   (f.Mode().Perm() & 0o500) == 0o500,
			})
		}
	}
	return archivedFiles, nil
}

func (brfi *BinaryReleaseFileInfo) close() {
	// TODO use lock - no threading atm so no need
	if brfi.tempFileUsage < 0 {
		brfi.tempFileUsage = 0
		return
	}
	brfi.tempFileUsage--
	if brfi.tempFileUsage != 0 || brfi.tempFile == "" {
		return
	}
	if err := os.Remove(brfi.tempFile); err != nil {
		log.Printf("Error removing temp file: %s", err)
	}
	brfi.tempFile = ""
}

func (brfi *BinaryReleaseFileInfo) FetchContent() (string, func(), error) {
	if brfi.tempFile != "" {
		brfi.tempFileUsage++
		return brfi.tempFile, brfi.close, nil
	}
	brfi.tempFileUsage++
	url := brfi.ReleaseAsset.GetBrowserDownloadURL()
	log.Printf("Downloading %s", url)
	var err error
	brfi.tempFile, err = util.DownloadUrlToTempFile(url)
	if err != nil {
		return "", nil, fmt.Errorf("downloading release: %w", err)
	}
	log.Printf("Got %s => %s", url, brfi.tempFile)
	// TODO change the way this works so it doesn't clean up this way, this is horrible.
	return url, brfi.close, nil
}

type BinaryReleaseFiles []*BinaryReleaseFileInfo

type FileTypes struct {
	CompressedArchives       []*BinaryReleaseFileInfo
	CompressedArchiveContent map[string]*FileTypes
	Binaries                 []*BinaryReleaseFileInfo
	ManualPages              []*BinaryReleaseFileInfo
	ShellCompletion          []*BinaryReleaseFileInfo
	Root                     *FileTypes
	MaybeBinaries            []*BinaryReleaseFileInfo
}

func (t *FileTypes) CountBinaries() int {
	result := len(t.Binaries)
	for _, each := range t.CompressedArchiveContent {
		result += len(each.Binaries)
	}
	return result
}

func (t *FileTypes) CountMaybeBinaries() int {
	result := len(t.MaybeBinaries)
	for _, each := range t.CompressedArchiveContent {
		result += len(each.MaybeBinaries)
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

func (t *FileTypes) CheckMaybes() error {
	for _, each := range t.MaybeBinaries {
		if ok, err := each.CheckMaybe(); err != nil {
			return fmt.Errorf("check maybes of %s: %w", each.Filename, err)
		} else if ok {
			t.Binaries = append(t.Binaries, each)
		}
	}
	for filename, archive := range t.CompressedArchiveContent {
		if err := archive.CheckMaybes(); err != nil {
			return fmt.Errorf("check maybes of container %s: %w", filename, err)
		}
	}
	return nil
}

func (base BinaryReleaseFiles) FindFiles(wordMap map[string][]*GroupedFilenamePartMeaning, root *FileTypes) *FileTypes {
	result := &FileTypes{
		CompressedArchives:       []*BinaryReleaseFileInfo{},
		Binaries:                 []*BinaryReleaseFileInfo{},
		MaybeBinaries:            []*BinaryReleaseFileInfo{},
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
		if compiled.Installer {
			log.Printf("Installer, not a binary sorry: %s", base.Filename)
			continue
		}
		if compiled.AppImage {
			log.Printf("AppImage, please use the app image version: %s", base.Filename)
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
			result.MaybeBinaries = append(result.MaybeBinaries, compiled)
			log.Printf("Is %s an Binary? - Unknown - Suspected", base.Filename)
			continue
		}
	}
	return result
}

func (brfi *BinaryReleaseFileInfo) CompileMeanings(input []*FilenamePartMeaning) (*BinaryReleaseFileInfo, bool) {
	result := &BinaryReleaseFileInfo{
		SuffixOnly: true,
	}
	if brfi != nil {
		result.ReleaseAsset = brfi.ReleaseAsset
		result.OriginalFilename = brfi.Filename
		result.ArchivePathname = brfi.ArchivePathname
		result.OS = brfi.OS
		result.Keyword = brfi.Keyword
		result.Toolchain = brfi.Toolchain
		result.tempFile = brfi.tempFile
		result.ExecutableBit = brfi.ExecutableBit
		result.Binary = brfi.ExecutableBit
		if brfi.Container != nil {
			result.Container = brfi.Container
			if result.OS == "" {
				result.OS = brfi.Container.OS
			}
			if result.Keyword == "" {
				result.Keyword = brfi.Container.Keyword
			}
			if result.Toolchain == "" {
				result.Toolchain = brfi.Container.Toolchain
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

func (brfi *BinaryReleaseFileInfo) CheckMaybe() (bool, error) {
	url, closeFn, err := brfi.FetchContent()
	if closeFn != nil {
		defer closeFn()
	}
	if err != nil {
		return false, fmt.Errorf("check maybe of %s: %w", url, err)
	}
	e, err := elf.Open(brfi.tempFile)
	if err != nil {
		log.Printf("%s is probably not a binary", brfi.Filename)
		return false, nil
	}
	defer func() {
		if err := e.Close(); err != nil {
			log.Printf("Error closing elf: %s", err)
		}
	}()
	log.Printf("%s has elf so probably is a binary", brfi.Filename)
	return true, nil
}
