package arrans_overlay_workflow_builder

import (
	"testing"
)

func TestLookupSymbolIncludesWaylandCursor(t *testing.T) {
	tests := map[string]string{
		"libwayland-cursor.so":   "dev-libs/wayland",
		"libwayland-cursor.so.0": "dev-libs/wayland",
	}

	for library, expected := range tests {
		t.Run(library, func(t *testing.T) {
			if dep, ok := lookupSymbol(library); !ok || dep != expected {
				t.Fatalf("expected %s to map to %s, got %s (present: %t)", library, expected, dep, ok)
			}
		})
	}
}

func TestLookupSymbolWayland(t *testing.T) {
	pkg, ok := lookupSymbol("libwayland-cursor.so.0")
	if !ok {
		t.Fatalf("expected lookup to succeed for libwayland-cursor.so.0")
	}
	if pkg != "dev-libs/wayland" {
		t.Fatalf("expected package to be dev-libs/wayland, got %s", pkg)
	}
}
