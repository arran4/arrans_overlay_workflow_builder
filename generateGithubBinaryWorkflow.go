package arrans_overlay_workflow_builder

import (
	"encoding/xml"
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/arran4/g2"
	"github.com/stoewer/go-strcase"
)

type GenerateGithubBinaryTemplateData struct {
	*GenerateGithubWorkflowBase
	_programsAsAlternatives        map[string][]string
	_reverseProgramsAsAlternatives map[string][]string
	MustntHaveUseFlags             map[string]map[string][]string
	MustHaveUseFlags               map[string]map[string][]string
}

func (ggbtd *GenerateGithubBinaryTemplateData) TemplateFileName() string {
	return "github-binary.tmpl"
}

func (ggbtd *GenerateGithubBinaryTemplateData) WorkflowName() string {
	return fmt.Sprintf("%s/%s update", ggbtd.Category, ggbtd.PackageName())
}

func (ggbtd *GenerateGithubBinaryTemplateData) KeywordList() []string {
	keywords := make([]string, 0)
	for programName := range ggbtd.Programs {
		for key := range ggbtd.Programs[programName].Binary {
			kw, _, _ := ggbtd.ParseKeywordAndUseFlags(key)
			keywords = append(keywords, kw)
		}
	}
	sort.Strings(keywords)
	return slices.Compact(keywords)
}

func (ggbtd *GenerateGithubBinaryTemplateData) ShellCompletionShells() []string {
	shells := make([]string, 0)
	for programName := range ggbtd.Programs {
		for _, shellMapping := range ggbtd.Programs[programName].ShellCompletionScripts {
			for shell := range shellMapping {
				shells = append(shells, shell)
			}
		}
	}
	sort.Strings(shells)
	return slices.CompactFunc(shells, strings.EqualFold)
}

func (ggbtd *GenerateGithubBinaryTemplateData) MainDependencies() []string {
	alternativeApps := ggbtd.ReverseProgramsAsAlternatives()
	deps := make([]string, 0)
	for programName := range ggbtd.Programs {
		if _, ok := alternativeApps[programName]; ok {
			continue
		}
		deps = append(deps, ggbtd.Programs[programName].Dependencies...)
	}
	sort.Strings(deps)
	deps = slices.CompactFunc(deps, strings.EqualFold)
	deps = slices.DeleteFunc(deps, func(s string) bool {
		return s == "sys-libs/glibc"
	})
	return deps
}

func (ggbtd *GenerateGithubBinaryTemplateData) AlternativeDependencies() map[string][]string {
	alternativeApps := ggbtd.ReverseProgramsAsAlternatives()
	altDeps := make(map[string][]string)
	for programName, prog := range ggbtd.Programs {
		if _, ok := alternativeApps[programName]; !ok {
			continue
		}
		altDeps[programName] = append(altDeps[programName], prog.Dependencies...)
		sort.Strings(altDeps[programName])
		altDeps[programName] = slices.CompactFunc(altDeps[programName], strings.EqualFold)
		altDeps[programName] = slices.DeleteFunc(altDeps[programName], func(s string) bool {
			return s == "sys-libs/glibc"
		})
	}
	return altDeps
}

func (ggbtd *GenerateGithubBinaryTemplateData) Keywords() string {
	return strings.Join(ggbtd.KeywordList(), " ")
}

func (ggbtd *GenerateGithubBinaryTemplateData) MaskedKeywords() string {
	list := ggbtd.KeywordList()
	for i := range list {
		list[i] = "~" + strings.TrimPrefix(list[i], "~")
	}
	return strings.Join(list, " ")
}

func (ggbtd *GenerateGithubBinaryTemplateData) WorkflowFileName() string {
	return fmt.Sprintf("%s-%s-update.yaml", ggbtd.Category, ggbtd.PackageName())
}

func (ggbtd *GenerateGithubBinaryTemplateData) PackageName() string {
	return strings.TrimSuffix(ggbtd.EbuildName, ".ebuild")
}

func (ggbtd *GenerateGithubBinaryTemplateData) HasDesktopFile() bool {
	for _, p := range ggbtd.Programs {
		if p.HasDesktopFile() {
			return true
		}
	}
	return false
}

func (ggbtd *GenerateGithubBinaryTemplateData) HasManualPages() bool {
	for _, p := range ggbtd.Programs {
		if p.HasManualPage() {
			return true
		}
	}
	return false
}

func (ggbtd *GenerateGithubBinaryTemplateData) HasCompressedManualPages() bool {
	for _, p := range ggbtd.Programs {
		if p.HasCompressedManualPages() {
			return true
		}
	}
	return false
}

func (ggbtd *GenerateGithubBinaryTemplateData) HasDocuments() bool {
	for _, p := range ggbtd.Programs {
		if p.HasDocuments() {
			return true
		}
	}
	return false
}

type KeywordedManualPageReference KeywordedFilenameReference

func (kmpr KeywordedManualPageReference) Page() int {
	if len(kmpr.Filepath) == 0 {
		return 0
	}
	v, _ := strconv.Atoi(filepath.Ext(kmpr.Filepath[len(kmpr.Filepath)-1]))
	return v
}

func (kmpr KeywordedManualPageReference) SourceFilepath() string {
	return ((*KeywordedFilenameReference)(&kmpr)).SourceFilepath()
}

func (kmpr KeywordedManualPageReference) DestinationFilename() string {
	return ((*KeywordedFilenameReference)(&kmpr)).DestinationFilename()
}

func (kmpr KeywordedManualPageReference) Compressed() bool {
	sf := kmpr.SourceFilepath()
	switch strings.ToLower(filepath.Ext(sf)) {
	case ".gz", ".bz2":
		return true
	}
	return false
}

func (kmpr KeywordedManualPageReference) UncompressedSourceFilepath() string {
	sf := kmpr.SourceFilepath()
	ext := filepath.Ext(sf)
	switch strings.ToLower(ext) {
	case ".gz", ".bz2":
		return strings.TrimSuffix(sf, ext)
	}
	return sf
}

func (kmpr KeywordedManualPageReference) Decompressor() string {
	sf := kmpr.SourceFilepath()
	switch strings.ToLower(filepath.Ext(sf)) {
	case ".gz":
		return "gzip -d"
	case ".bz2":
		return "bzip2 -d"
	}
	return "touch"
}

func (ggbtd *GenerateGithubBinaryTemplateData) ManualPages() (result []KeywordGrouped[*KeywordedManualPageReference]) {
	m := map[string]int{}
	for _, p := range ggbtd.Programs {
		for kw, mps := range p.ManualPage {
			for _, mp := range mps {
				offset, ok := m[kw]
				if !ok {
					m[kw] = len(result)
					offset = m[kw]
					result = append(result, KeywordGrouped[*KeywordedManualPageReference]{
						Keyword: kw,
					})
				}
				result[offset].Grouped = append(result[offset].Grouped, (*KeywordedManualPageReference)(&KeywordedFilenameReference{
					Filepath: mp,
					Keyword:  kw,
				}))
			}
		}
	}
	return ggbtd.CompressGroupedKeywordedanualPageReference(result)
}

func (ggbtd *GenerateGithubBinaryTemplateData) CompressedManualPages() (result []KeywordGrouped[*KeywordedManualPageReference]) {
	m := map[string]int{}
	for _, p := range ggbtd.Programs {
		for kw, mps := range p.ManualPage {
			for _, mp := range mps {
				manPage := (*KeywordedManualPageReference)(&KeywordedFilenameReference{
					Filepath: mp,
					Keyword:  kw,
				})
				if manPage.Compressed() {
					offset, ok := m[kw]
					if !ok {
						m[kw] = len(result)
						offset = m[kw]
						result = append(result, KeywordGrouped[*KeywordedManualPageReference]{
							Keyword: kw,
						})
					}
					result[offset].Grouped = append(result[offset].Grouped, manPage)
				}
			}
		}
	}
	return ggbtd.CompressGroupedKeywordedanualPageReference(result)
}

type KeywordGrouped[T any] struct {
	Keyword string
	Grouped []T
}

func (ggbtd *GenerateGithubBinaryTemplateData) Documents() (result []KeywordGrouped[*KeywordedFilenameReference]) {
	m := map[string]int{}
	for _, p := range ggbtd.Programs {
		for kw, mps := range p.Documents {
			for _, mp := range mps {
				offset, ok := m[kw]
				if !ok {
					m[kw] = len(result)
					offset = m[kw]
					result = append(result, KeywordGrouped[*KeywordedFilenameReference]{
						Keyword: kw,
					})
				}
				result[offset].Grouped = append(result[offset].Grouped, &KeywordedFilenameReference{
					Filepath: mp,
					Keyword:  kw,
				})
			}
		}
	}
	slices.SortFunc(result, func(a, b KeywordGrouped[*KeywordedFilenameReference]) int {
		return strings.Compare(a.Keyword, b.Keyword)
	})
	return ggbtd.CompressGroupedKeywordedFilenameReference(result)
}

func (ggbtd *GenerateGithubBinaryTemplateData) CompressGroupedKeywordedFilenameReference(result []KeywordGrouped[*KeywordedFilenameReference]) []KeywordGrouped[*KeywordedFilenameReference] {
	comparerFunc := func(reference *KeywordedFilenameReference, reference2 *KeywordedFilenameReference) bool {
		if len(reference.Filepath) == 0 && len(reference2.Filepath) == 0 {
			return true
		}
		if len(reference.Filepath) == 0 || len(reference2.Filepath) == 0 {
			return false
		}
		return slices.Equal(reference.Filepath[1:], reference2.Filepath[1:])
	}
	return KeywordGroupCompressor(ggbtd, result, comparerFunc)
}

func (ggbtd *GenerateGithubBinaryTemplateData) CompressGroupedKeywordedanualPageReference(result []KeywordGrouped[*KeywordedManualPageReference]) []KeywordGrouped[*KeywordedManualPageReference] {
	comparerFunc := func(reference *KeywordedManualPageReference, reference2 *KeywordedManualPageReference) bool {
		if len(reference.Filepath) == 0 && len(reference2.Filepath) == 0 {
			return true
		}
		if len(reference.Filepath) == 0 || len(reference2.Filepath) == 0 {
			return false
		}
		return slices.Equal(reference.Filepath[1:], reference2.Filepath[1:])
	}
	return KeywordGroupCompressor(ggbtd, result, comparerFunc)
}

func KeywordGroupCompressor[T any](ggbtd *GenerateGithubBinaryTemplateData, result []KeywordGrouped[T], comparerFunc func(reference T, reference2 T) bool) []KeywordGrouped[T] {
	if len(result) == 0 {
		return result
	}
	requiredKeywords := map[string]bool{}
	for _, kw := range ggbtd.KeywordList() {
		requiredKeywords[kw] = true
	}
	first := result[0]
	for _, kwg := range result {
		delete(requiredKeywords, kwg.Keyword)
		if !slices.EqualFunc(kwg.Grouped, first.Grouped, comparerFunc) {
			return result
		}
	}
	if len(requiredKeywords) != 0 {
		return result
	}
	return []KeywordGrouped[T]{{
		Grouped: result[0].Grouped,
	}}
}

func (ggbtd *GenerateGithubBinaryTemplateData) ShellCompletionInstallPath(shell string) (string, error) {
	switch shell {
	case "bash":
		return "/usr/share/bash-completion/completions", nil
	case "fish":
		return "/usr/share/fish/vendor_completions.d", nil
	case "zsh":
		return "/usr/share/zsh/site-functions", nil
	case "powershell":
		return "/usr/share/powershell/Modules", nil
	}
	return "", fmt.Errorf("unknown shell: %s", shell)
}

func (ggbtd *GenerateGithubBinaryTemplateData) HasShellCompletion(shell string) bool {
	for _, p := range ggbtd.Programs {
		if p.HasShellCompletion(shell) {
			return true
		}
	}
	return false
}

func (ggbtd *GenerateGithubBinaryTemplateData) ShellCompletion(shell string) []KeywordGrouped[*KeywordedFilenameReference] {
	m := map[string]int{}
	result := make([]KeywordGrouped[*KeywordedFilenameReference], 0)
	for _, p := range ggbtd.Programs {
		scs := p.ShellCompletion(shell)
		for _, sc := range scs {
			offset, ok := m[sc.Keyword]
			if !ok {
				m[sc.Keyword] = len(result)
				offset = m[sc.Keyword]
				result = append(result, KeywordGrouped[*KeywordedFilenameReference]{
					Keyword: sc.Keyword,
				})
			}
			result[offset].Grouped = append(result[offset].Grouped, sc)
		}
	}
	return ggbtd.CompressGroupedKeywordedFilenameReference(result)
}

func (ggbtd *GenerateGithubBinaryTemplateData) IsArchived(keyword string) bool {
	for _, p := range ggbtd.Programs {
		if p.IsArchived(keyword) {
			return true
		}
	}
	return false
}

func (ggbtd *GenerateGithubBinaryTemplateData) ParseKeywordAndUseFlags(kw string) (string, []string, []string) {
	// e.g. amd64[+onnx,-cuda]
	start := strings.Index(kw, "[")
	if start == -1 {
		return kw, nil, nil
	}
	end := strings.LastIndex(kw, "]")
	if end == -1 || end < start {
		return kw, nil, nil
	}
	arch := kw[:start]
	flagsPart := kw[start+1 : end]
	if len(strings.TrimSpace(flagsPart)) == 0 {
		return arch, nil, nil
	}
	flags := strings.Split(flagsPart, ",")
	var mustHave []string
	var mustntHave []string
	for _, f := range flags {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		if strings.HasPrefix(f, "+") {
			if flag := strings.TrimSpace(f[1:]); flag != "" {
				mustHave = append(mustHave, flag)
			}
		} else if strings.HasPrefix(f, "-") {
			if flag := strings.TrimSpace(f[1:]); flag != "" {
				mustntHave = append(mustntHave, flag)
			}
		} else {
			mustHave = append(mustHave, f)
		}
	}
	return arch, mustHave, mustntHave
}

func (ggbtd *GenerateGithubBinaryTemplateData) inferUseFlags() {
	if ggbtd.MustHaveUseFlags != nil && ggbtd.MustntHaveUseFlags != nil {
		return
	}
	archAlts := ggbtd.ProgramsAsAlternatives()
	progAlts := ggbtd.ReverseProgramsAsAlternatives()
	ggbtd.MustHaveUseFlags = map[string]map[string][]string{}
	ggbtd.MustntHaveUseFlags = map[string]map[string][]string{}
	for programName := range ggbtd.Programs {
		for kwWithFlags := range ggbtd.Programs[programName].Binary {
			kw, mustHave, mustntHave := ggbtd.ParseKeywordAndUseFlags(kwWithFlags)
			if v, ok := ggbtd.MustHaveUseFlags[programName]; !ok || v == nil {
				ggbtd.MustHaveUseFlags[programName] = map[string][]string{}
			}
			if v, ok := ggbtd.MustHaveUseFlags[programName][kwWithFlags]; !ok || v == nil {
				ggbtd.MustHaveUseFlags[programName][kwWithFlags] = []string{kw}
			}
			ggbtd.MustHaveUseFlags[programName][kwWithFlags] = append(ggbtd.MustHaveUseFlags[programName][kwWithFlags], mustHave...)

			if v, ok := ggbtd.MustntHaveUseFlags[programName]; !ok || v == nil {
				ggbtd.MustntHaveUseFlags[programName] = map[string][]string{}
			}
			if v, ok := ggbtd.MustntHaveUseFlags[programName][kwWithFlags]; !ok || v == nil {
				ggbtd.MustntHaveUseFlags[programName][kwWithFlags] = []string{}
			}
			ggbtd.MustntHaveUseFlags[programName][kwWithFlags] = append(ggbtd.MustntHaveUseFlags[programName][kwWithFlags], mustntHave...)

			if programName == "" || programName == ggbtd.GithubRepo {
				alts, ok := archAlts[kw]
				if !ok || len(alts) <= 0 {
					continue
				}
				for _, alt := range alts {
					if alt == programName {
						continue
					}
					ggbtd.MustntHaveUseFlags[programName][kwWithFlags] = append(ggbtd.MustntHaveUseFlags[programName][kwWithFlags], alt)
				}
			}
			if v, ok := progAlts[programName]; ok && len(v) > 0 {
				alts, ok := archAlts[kw]
				if !ok || len(alts) <= 0 {
					continue
				}
				ggbtd.MustHaveUseFlags[programName][kwWithFlags] = append(ggbtd.MustHaveUseFlags[programName][kwWithFlags], programName)
				for _, alt := range alts {
					if alt == programName {
						continue
					}
					ggbtd.MustntHaveUseFlags[programName][kwWithFlags] = append(ggbtd.MustntHaveUseFlags[programName][kwWithFlags], alt)
				}
			}
		}
	}
}

func (ggbtd *GenerateGithubBinaryTemplateData) IUseFlags() []g2.Flag {
	seen := make(map[string]bool)
	var finalUses []g2.Flag

	addFlag := func(name, text string) {
		cleaned := strcase.SnakeCase(strings.TrimSpace(name))
		if cleaned != "" && !seen[cleaned] {
			seen[cleaned] = true
			finalUses = append(finalUses, g2.Flag{
				Name: cleaned,
				Text: text,
			})
		}
	}

	revAlts := ggbtd.ReverseProgramsAsAlternatives()
	altProgs := make([]string, 0, len(revAlts))
	for use := range revAlts {
		altProgs = append(altProgs, use)
	}
	sort.Strings(altProgs)
	for _, use := range altProgs {
		addFlag(use, fmt.Sprintf("Install %s binary", use))
	}
	if ggbtd.HasManualPages() {
		addFlag("man", "Install manual pages")
	}
	if ggbtd.HasDocuments() {
		addFlag("doc", "Install documentation")
	}
	for _, shell := range ggbtd.ShellCompletionShells() {
		addFlag(shell, fmt.Sprintf("Install %s completion", shell))
	}
	for _, use := range ggbtd.IUse {
		addFlag(use, fmt.Sprintf("Enable %s", use))
	}
	for _, use := range ggbtd.ExtractedUseFlags() {
		addFlag(use, fmt.Sprintf("Enable %s", use))
	}

	slices.SortFunc(finalUses, func(a, b g2.Flag) int {
		return strings.Compare(a.Name, b.Name)
	})

	return finalUses
}

func (ggbtd *GenerateGithubBinaryTemplateData) ExtractedUseFlags() []string {
	ggbtd.inferUseFlags()
	flagsSet := make(map[string]struct{})
	keywordList := ggbtd.KeywordList()
	for _, progMap := range ggbtd.MustHaveUseFlags {
		for _, flags := range progMap {
			for _, f := range flags {
				if !slices.Contains(keywordList, f) {
					flagsSet[f] = struct{}{}
				}
			}
		}
	}
	for _, progMap := range ggbtd.MustntHaveUseFlags {
		for _, flags := range progMap {
			for _, f := range flags {
				if !slices.Contains(keywordList, f) {
					flagsSet[f] = struct{}{}
				}
			}
		}
	}
	var res []string
	for f := range flagsSet {
		res = append(res, f)
	}
	sort.Strings(res)
	return res
}

type ExternalResourceKeywordExtended struct {
	ExternalResource   *ExternalResource
	MustHaveUseFlags   []string
	MustntHaveUseFlags []string
}

func (erke *ExternalResourceKeywordExtended) Keyword() string {
	return erke.ExternalResource.Keyword
}

func (erke *ExternalResourceKeywordExtended) ReleaseFilename() string {
	return erke.ExternalResource.ReleaseFilename
}

func (erke *ExternalResourceKeywordExtended) Archived() bool {
	return erke.ExternalResource.Archived
}

func (ggbtd *GenerateGithubBinaryTemplateData) ExternalResources() []*ExternalResourceKeywordExtended {
	ggbtd.inferUseFlags()
	m := make(map[string]*ExternalResourceKeywordExtended)
	for programName := range ggbtd.Programs {
		for kwWithFlags, rfn := range ggbtd.Programs[programName].Binary {
			kw, _, _ := ggbtd.ParseKeywordAndUseFlags(kwWithFlags)
			e := &ExternalResourceKeywordExtended{
				ExternalResource: &ExternalResource{
					Keyword:         kw,
					ReleaseFilename: rfn[0],
					Archived:        len(rfn) > 2,
				},
				MustHaveUseFlags:   ggbtd.GetMustHaveUseFlags(programName, kwWithFlags),
				MustntHaveUseFlags: ggbtd.GetMustntHaveUseFlags(programName, kwWithFlags),
			}
			if existing, ok := m[rfn[0]]; ok {
				existing.MustHaveUseFlags = append(existing.MustHaveUseFlags, ggbtd.GetMustHaveUseFlags(programName, kwWithFlags)...)
				slices.Sort(existing.MustHaveUseFlags)
				existing.MustHaveUseFlags = slices.Compact(existing.MustHaveUseFlags)

				existing.MustntHaveUseFlags = append(existing.MustntHaveUseFlags, ggbtd.GetMustntHaveUseFlags(programName, kwWithFlags)...)
				slices.Sort(existing.MustntHaveUseFlags)
				existing.MustntHaveUseFlags = slices.Compact(existing.MustntHaveUseFlags)
			} else {
				m[rfn[0]] = e
			}
		}
	}
	result := make([]*ExternalResourceKeywordExtended, 0, len(ggbtd.Programs))
	for _, each := range m {
		result = append(result, each)
	}
	slices.SortFunc(result, func(a, b *ExternalResourceKeywordExtended) int {
		v1 := strings.Compare(a.ExternalResource.Keyword, b.ExternalResource.Keyword)
		if v1 != 0 {
			return v1
		}
		return strings.Compare(a.ExternalResource.ReleaseFilename, b.ExternalResource.ReleaseFilename)
	})
	return result
}

func (ggbtd *GenerateGithubBinaryTemplateData) GetMustHaveUseFlags(programName string, kw string) []string {
	if ggbtd.MustHaveUseFlags == nil {
		ggbtd.inferUseFlags()
	}
	if v, ok := ggbtd.MustHaveUseFlags[programName]; !ok || v == nil {
		return []string{}
	}
	if v, ok := ggbtd.MustHaveUseFlags[programName][kw]; !ok || v == nil {
		return []string{}
	} else {
		return v
	}
}

func (ggbtd *GenerateGithubBinaryTemplateData) GetMustntHaveUseFlags(programName string, kw string) []string {
	if ggbtd.MustntHaveUseFlags == nil {
		ggbtd.inferUseFlags()
	}
	if v, ok := ggbtd.MustntHaveUseFlags[programName]; !ok || v == nil {
		return []string{}
	}
	if v, ok := ggbtd.MustntHaveUseFlags[programName][kw]; !ok || v == nil {
		return []string{}
	} else {
		return v
	}
}

func (ggbtd *GenerateGithubBinaryTemplateData) ProgramsAsAlternatives() map[string][]string {
	if ggbtd._programsAsAlternatives != nil {
		return ggbtd._programsAsAlternatives
	}
	if ggbtd.Workarounds == nil {
		return map[string][]string{}
	}
	ggbtd._programsAsAlternatives = map[string][]string{}
	s := ggbtd.Workarounds["Programs as Alternatives"]
	ss := strings.SplitSeq(s, " ")
	for each := range ss {
		e := strings.Split(each, ":")
		if len(e) != 2 {
			continue
		}
		ggbtd._programsAsAlternatives[strings.TrimSpace(e[0])] = append(ggbtd._programsAsAlternatives[strings.TrimSpace(e[0])], strings.TrimSpace(e[1]))
	}
	return ggbtd._programsAsAlternatives
}

func (ggbtd *GenerateGithubBinaryTemplateData) ReverseProgramsAsAlternatives() map[string][]string {
	if ggbtd._reverseProgramsAsAlternatives != nil {
		return ggbtd._reverseProgramsAsAlternatives
	}
	if ggbtd.Workarounds == nil {
		return map[string][]string{}
	}
	ggbtd._reverseProgramsAsAlternatives = map[string][]string{}
	for arch, progs := range ggbtd.ProgramsAsAlternatives() {
		for _, prog := range progs {
			ggbtd._reverseProgramsAsAlternatives[prog] = append(ggbtd._reverseProgramsAsAlternatives[prog], arch)
		}
	}
	for _, archs := range ggbtd._reverseProgramsAsAlternatives {
		sort.Strings(archs)
	}
	return ggbtd._reverseProgramsAsAlternatives
}

func (ggbtd *GenerateGithubBinaryTemplateData) ProgramsAsAlternativesForArch(forArchitecture string) []string {
	v, ok := ggbtd.ProgramsAsAlternatives()[forArchitecture]
	if !ok || v == nil {
		return []string{}
	}
	return v
}

func (ggbtd *GenerateGithubBinaryTemplateData) NeedsSrcUnpack() bool {
	for _, r := range ggbtd.ExternalResources() {
		if r.Archived() {
			return true
		}
	}
	return ggbtd.HasCompressedManualPages()
}

func (ggbtd *GenerateGithubBinaryTemplateData) Metadata() (string, error) {
	pkgMd := &g2.PkgMetadata{
		XMLName: xml.Name{
			Local: "pkgmetadata",
		},
		Upstream: &g2.Upstream{
			RemoteID: []g2.RemoteID{},
		},
	}
	if ggbtd.MaintainerEmail != "" {
		pkgMd.Maintainers = append(pkgMd.Maintainers, g2.Maintainer{
			Email: ggbtd.MaintainerEmail,
			Name:  ggbtd.MaintainerName,
			Type:  "person",
		})
	}
	if ggbtd.GithubOwner != "" && ggbtd.GithubRepo != "" {
		pkgMd.Upstream.RemoteID = append(pkgMd.Upstream.RemoteID, g2.RemoteID{
			Type: "github",
			Text: fmt.Sprintf("%s/%s", ggbtd.GithubOwner, ggbtd.GithubRepo),
		})
	}
	if len(pkgMd.Upstream.RemoteID) == 0 {
		pkgMd.Upstream = nil
	}

	if len(pkgMd.Use) == 0 {
		pkgMd.Use = []g2.Use{{}}
	}

	pkgMd.Use[0].Flags = append(pkgMd.Use[0].Flags, ggbtd.IUseFlags()...)

	for i := range pkgMd.Use {
		var newFlags []g2.Flag
		for _, f := range pkgMd.Use[i].Flags {
			if len(f.Name) > 0 {
				newFlags = append(newFlags, f)
			}
		}
		pkgMd.Use[i].Flags = newFlags
		if len(pkgMd.Use[i].Flags) == 0 {
			pkgMd.Use[i].Flags = nil
		}
	}

	var newUses []g2.Use
	for _, u := range pkgMd.Use {
		if len(u.Flags) > 0 {
			newUses = append(newUses, u)
		}
	}
	pkgMd.Use = newUses
	if len(pkgMd.Use) == 0 {
		pkgMd.Use = nil
	}

	o, err := xml.MarshalIndent(pkgMd, "", "	")
	if err != nil {
		return "", fmt.Errorf("marshalling metadata: %w", err)
	}
	return fmt.Sprintf("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<!DOCTYPE pkgmetadata SYSTEM \"http://www.gentoo.org/dtd/metadata.dtd\">\n%s", string(o)), nil
}

func (ggbtd *GenerateGithubBinaryTemplateData) G2MetadataArgs() string {
	if ggbtd == nil || ggbtd.GenerateGithubWorkflowBase == nil {
		return ""
	}
	args := ggbtd.GenerateGithubWorkflowBase.G2MetadataArgs() + " "

	for _, f := range ggbtd.IUseFlags() {
		args += fmt.Sprintf("--use-add \"%s:%s\" ", f.Name, f.Text)
	}
	return strings.TrimSpace(args)
}

type ReverseProgramAlternative struct {
	UseFlag       string
	Architectures []string
}

func (ggbtd *GenerateGithubBinaryTemplateData) SortedReverseProgramsAsAlternatives() []ReverseProgramAlternative {
	alts := ggbtd.ReverseProgramsAsAlternatives()
	keys := make([]string, 0, len(alts))
	for k := range alts {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var res []ReverseProgramAlternative
	for _, k := range keys {
		archs := alts[k]
		sort.Strings(archs)
		res = append(res, ReverseProgramAlternative{UseFlag: k, Architectures: archs})
	}
	return res
}
