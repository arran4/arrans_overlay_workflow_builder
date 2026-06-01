package arrans_overlay_workflow_builder

import (
	"testing"
)

func TestLookupSymbolWayland(t *testing.T) {
	pkg, ok := lookupSymbol("libwayland-cursor.so.0")
	if !ok {
		t.Fatalf("expected lookup to succeed for libwayland-cursor.so.0")
	}
	if pkg != "dev-libs/wayland" {
		t.Fatalf("expected package to be dev-libs/wayland, got %s", pkg)
	}
}
