package arrans_overlay_workflow_builder

import (
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
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
			keywords = append(keywords, key)
		}
	}
	sort.Strings(keywords)
	return keywords
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

func (ggbtd *GenerateGithubBinaryTemplateData) inferUseFlags() {
	if ggbtd.MustHaveUseFlags != nil && ggbtd.MustntHaveUseFlags != nil {
		return
	}
	archAlts := ggbtd.ProgramsAsAlternatives()
	progAlts := ggbtd.ReverseProgramsAsAlternatives()
	ggbtd.MustHaveUseFlags = map[string]map[string][]string{}
	ggbtd.MustntHaveUseFlags = map[string]map[string][]string{}
	for programName := range ggbtd.Programs {
		for kw := range ggbtd.Programs[programName].Binary {
			if v, ok := ggbtd.MustHaveUseFlags[programName]; !ok || v == nil {
				ggbtd.MustHaveUseFlags[programName] = map[string][]string{}
			}
			if v, ok := ggbtd.MustHaveUseFlags[programName][kw]; !ok || v == nil {
				ggbtd.MustHaveUseFlags[programName][kw] = []string{kw}
			}
			if v, ok := ggbtd.MustntHaveUseFlags[programName]; !ok || v == nil {
				ggbtd.MustntHaveUseFlags[programName] = map[string][]string{}
			}
			if v, ok := ggbtd.MustntHaveUseFlags[programName][kw]; !ok || v == nil {
				ggbtd.MustntHaveUseFlags[programName][kw] = []string{}
			}
			if programName == "" || programName == ggbtd.GithubRepo {
				alts, ok := archAlts[kw]
				if !ok || len(alts) <= 0 {
					continue
				}
				for _, alt := range alts {
					if alt == programName {
						continue
					}
					ggbtd.MustntHaveUseFlags[programName][kw] = append(ggbtd.MustntHaveUseFlags[programName][kw], alt)
				}
			}
			if v, ok := progAlts[programName]; ok && len(v) > 0 {
				alts, ok := archAlts[kw]
				if !ok || len(alts) <= 0 {
					continue
				}
				ggbtd.MustHaveUseFlags[programName][kw] = append(ggbtd.MustHaveUseFlags[programName][kw], programName)
				for _, alt := range alts {
					if alt == programName {
						continue
					}
					ggbtd.MustntHaveUseFlags[programName][kw] = append(ggbtd.MustntHaveUseFlags[programName][kw], alt)
				}
			}
		}
	}
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
		for kw, rfn := range ggbtd.Programs[programName].Binary {
			e := &ExternalResourceKeywordExtended{
				ExternalResource: &ExternalResource{
					Keyword:         kw,
					ReleaseFilename: rfn[0],
					Archived:        len(rfn) > 2,
				},
				MustHaveUseFlags:   ggbtd.GetMustHaveUseFlags(programName, kw),
				MustntHaveUseFlags: ggbtd.GetMustntHaveUseFlags(programName, kw),
			}
			m[rfn[0]] = e
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
	s, _ := ggbtd.Workarounds["Programs as Alternatives"]
	ss := strings.Split(s, " ")
	for _, each := range ss {
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
	return ggbtd._reverseProgramsAsAlternatives
}

func (ggbtd *GenerateGithubBinaryTemplateData) ProgramsAsAlternativesForArch(forArchitecture string) []string {
	v, ok := ggbtd.ProgramsAsAlternatives()[forArchitecture]
	if !ok || v == nil {
		return []string{}
	}
	return v
}
