package arrans_overlay_workflow_builder

import (
	"sort"
	"unicode"
)

type KeyedMeaning[Embedded any] struct {
	Embedded Embedded
	Key      string
}

type Embeddable interface {
	IsCaseInsensitive() bool
}

func GroupAndSort[Embedded Embeddable](wordMap map[string]Embedded) map[string][]*KeyedMeaning[Embedded] {
	keyGroups := make(map[string][]*KeyedMeaning[Embedded])
	for key := range wordMap {
		meaning := wordMap[key]
		firstChar := string(key[0])
		keyGroups[firstChar] = append(keyGroups[firstChar], &KeyedMeaning[Embedded]{
			Embedded: wordMap[key],
			Key:      key,
		})
		if meaning.IsCaseInsensitive() {
			if unicode.IsUpper(rune(firstChar[0])) {
				firstChar = string(unicode.ToLower(rune(firstChar[0])))
			} else {
				firstChar = string(unicode.ToUpper(rune(firstChar[0])))
			}
			keyGroups[firstChar] = append(keyGroups[firstChar], &KeyedMeaning[Embedded]{
				Embedded: wordMap[key],
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
