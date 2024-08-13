package arrans_overlay_workflow_builder

import (
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
)

type GenerateGithubBinaryTemplateData struct {
	*InputConfig
	Now                            time.Time
	ConfigFile                     string
	_programsAsAlternatives        map[string][]string
	_reverseProgramsAsAlternatives map[string][]string
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

func (ggbtd *GenerateGithubBinaryTemplateData) ManualPages() (result []*KeywordedManualPageReference) {
	for _, p := range ggbtd.Programs {
		for kw, mps := range p.ManualPage {
			for _, mp := range mps {
				result = append(result, (*KeywordedManualPageReference)(&KeywordedFilenameReference{
					Filepath: mp,
					Keyword:  kw,
				}))
			}
		}
	}
	return
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
	if len(requiredKeywords) == 0 {
		return result
	}
	return []KeywordGrouped[T]{{
		Grouped: result[0].Grouped,
	}}
}

func (ggbtd *GenerateGithubBinaryTemplateData) HasShellCompletion(shell string) bool {
	for _, p := range ggbtd.Programs {
		if p.HasShellCompletion(shell) {
			return true
		}
	}
	return false
}

func (ggbtd *GenerateGithubBinaryTemplateData) ShellCompletion(shell string) []*KeywordedFilenameReference {
	result := make([]*KeywordedFilenameReference, 0)
	for _, p := range ggbtd.Programs {
		v := p.ShellCompletion(shell)
		if len(v) > 0 {
			result = append(result, v...)
		}
	}
	return result
}

func (ggbtd *GenerateGithubBinaryTemplateData) IsArchived(keyword string) bool {
	for _, p := range ggbtd.Programs {
		if p.IsArchived(keyword) {
			return true
		}
	}
	return false
}

type ExternalResourceKeywordExtended struct {
	ExternalResource   *ExternalResource
	HaveKeywords       []string
	MustntHaveKeywords []string
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

func (ggbtd *GenerateGithubBinaryTemplateData) ExternalResources() map[string]*ExternalResourceKeywordExtended {
	archAlts := ggbtd.ProgramsAsAlternatives()
	progAlts := ggbtd.ReverseProgramsAsAlternatives()
	result := make(map[string]*ExternalResourceKeywordExtended)
	for programName := range ggbtd.Programs {
		for kw, rfn := range ggbtd.Programs[programName].Binary {
			e := &ExternalResourceKeywordExtended{
				ExternalResource: &ExternalResource{
					Keyword:         kw,
					ReleaseFilename: rfn[0],
					Archived:        len(rfn) > 2,
				},
				HaveKeywords:       []string{kw},
				MustntHaveKeywords: []string{},
			}
			result[rfn[0]] = e
			if programName == "" || programName == ggbtd.GithubRepo {
				alts, ok := archAlts[kw]
				if !ok || len(alts) == 0 {
					continue
				}
				for _, alt := range alts {
					e.MustntHaveKeywords = append(e.MustntHaveKeywords, alt)
				}
			}
			if v, ok := progAlts[programName]; ok && len(v) > 0 {
				alts, ok := archAlts[kw]
				if !ok || len(alts) == 0 {
					continue
				}
				e.HaveKeywords = append(e.HaveKeywords, programName)
				for _, alt := range alts {
					if alt == programName {
						continue
					}
					e.MustntHaveKeywords = append(e.MustntHaveKeywords, alt)
				}
			}
		}
	}
	return result
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
