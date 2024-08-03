package arrans_overlay_workflow_builder

import (
	"sort"
	"strings"
	"unicode"
)

type GroupedFilenamePartMeaning struct {
	*FilenamePartMeaning
	Key string
}

func (m *GroupedFilenamePartMeaning) Match(s string) bool {
	if m.FilenamePartMeaning.CaseInsensitive {
		return strings.EqualFold(s, m.Key)
	} else {
		return s == m.Key
	}
}

func GroupAndSort(wordMap map[string]*FilenamePartMeaning) map[string][]*GroupedFilenamePartMeaning {
	keyGroups := make(map[string][]*GroupedFilenamePartMeaning)
	for key := range wordMap {
		meaning := wordMap[key]
		firstChar := string(key[0])
		keyGroups[firstChar] = append(keyGroups[firstChar], &GroupedFilenamePartMeaning{
			FilenamePartMeaning: wordMap[key],
			Key:                 key,
		})
		if meaning.CaseInsensitive {
			if unicode.IsUpper(rune(firstChar[0])) {
				firstChar = string(unicode.ToLower(rune(firstChar[0])))
			} else {
				firstChar = string(unicode.ToUpper(rune(firstChar[0])))
			}
			keyGroups[firstChar] = append(keyGroups[firstChar], &GroupedFilenamePartMeaning{
				FilenamePartMeaning: wordMap[key],
				Key:                 key,
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

func DecodeFilename(groupedWordMap map[string][]*GroupedFilenamePartMeaning, filename string) []*FilenamePartMeaning {
	var result []*FilenamePartMeaning
	length := len(filename)
	suffixOnly := false
	unmatched := -1
	var sep *FilenamePartMeaning
	for i := 0; i < length; {
		matched := false
		firstChar := string(filename[i])
		if meanings, found := groupedWordMap[firstChar]; found {
			for _, meaning := range meanings {
				keyLen := len(meaning.Key)
				if i+keyLen <= length && meaning.Match(filename[i:i+keyLen]) {
					if unmatched != -1 {
						if unmatched < i-2 {
							result = append(result, &FilenamePartMeaning{
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
					if suffixOnly && !meaning.FilenamePartMeaning.SuffixOnly {
						continue
					}
					var fi = *meaning.FilenamePartMeaning
					fi.Captured = filename[i : i+keyLen]
					result = append(result, &fi)
					if meaning.FilenamePartMeaning.SuffixOnly {
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
			sep = &FilenamePartMeaning{
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
			result = append(result, &FilenamePartMeaning{
				Unmatched:  true,
				Captured:   filename[unmatched:],
				SuffixOnly: suffixOnly,
			})
		}
		unmatched = -1
	}

	return result
}

type FilenamePartMeaning struct {
	// Core properties
	// Gentoo keyword
	Keyword string
	OS      string
	// Generally msvc, gnu, musl, etc
	Toolchain string
	// Like tar, or zip, also a bit of bz2, and gz but not proper "containers", later replaced by the container of the
	// contained file
	Container string
	// The separator -_-
	Separator bool
	Captured  string

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
	Unmatched bool
}

func GenerateWordMeanings(gitRepo string, versions []string, tags []string) map[string]*FilenamePartMeaning {
	wordMap := map[string]*FilenamePartMeaning{
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
		"linux":                         {OS: "linux", CaseInsensitive: true},
		"lin":                           {OS: "linux", CaseInsensitive: true},
		"windows":                       {OS: "windows", CaseInsensitive: true},
		"win":                           {OS: "windows", CaseInsensitive: true},
		"win32":                         {OS: "windows", Keyword: "~x86", CaseInsensitive: true},
		"win64":                         {OS: "windows", Keyword: "~amd64", CaseInsensitive: true},
		"macosx":                        {OS: "macosx"},
		"macos":                         {OS: "macosx"},
		"darwin":                        {OS: "macosx"},
		"gnu":                           {Toolchain: "gnu", CaseInsensitive: true},
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
		wordMap[gitRepo] = &FilenamePartMeaning{ProjectName: true, CaseInsensitive: true}
	}
	for _, version := range versions {
		if v, ok := wordMap[version]; ok {
			v.Version = true
		} else {
			wordMap[version] = &FilenamePartMeaning{Version: true}
		}
	}
	for _, tag := range tags {
		if v, ok := wordMap[tag]; ok {
			v.Tag = true
		} else {
			wordMap[tag] = &FilenamePartMeaning{Tag: true}
		}
	}

	return wordMap
}
