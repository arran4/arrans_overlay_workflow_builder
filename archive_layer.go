package arrans_overlay_workflow_builder

import "strings"

type ArchiveLayer[T any] struct {
	Archive T
	Files   []T
	Layers  []*ArchiveLayer[T]
}

func isArchiveFilename(name string) bool {
	l := strings.ToLower(name)
	switch {
	case strings.HasSuffix(l, ".tar"),
		strings.HasSuffix(l, ".tar.gz"),
		strings.HasSuffix(l, ".tgz"),
		strings.HasSuffix(l, ".tar.bz2"),
		strings.HasSuffix(l, ".tbz2"),
		strings.HasSuffix(l, ".zip"):
		return true
	default:
		return false
	}
}

func containersFromName(name string) []string {
	l := strings.ToLower(name)
	switch {
	case strings.HasSuffix(l, ".tar.gz"), strings.HasSuffix(l, ".tgz"):
		return []string{"tar", "gz"}
	case strings.HasSuffix(l, ".tar.bz2"), strings.HasSuffix(l, ".tbz2"):
		return []string{"tar", "bz2"}
	case strings.HasSuffix(l, ".tar"):
		return []string{"tar"}
	case strings.HasSuffix(l, ".zip"):
		return []string{"zip"}
	default:
		return nil
	}
}
