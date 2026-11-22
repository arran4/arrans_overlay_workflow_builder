package arrans_overlay_workflow_builder

import (
    "os"
    "strings"
    "testing"
)

// TestIssue202LargeBinaryUpdaterBudget keeps the checked-in multi-architecture
// chezmoi example below the per-run command limit. This is a focused regression
// check; issue #203 will enforce budgets across all rendered workflows.
func TestIssue202LargeBinaryUpdaterBudget(t *testing.T) {
    contents, err := os.ReadFile("examples/.github/workflows/app-admin-chezmoi-bin-update.yaml")
    if err != nil { t.Fatal(err) }
    workflow := string(contents)
    start := strings.Index(workflow, "      - name: Process each release\n")
    if start < 0 { t.Fatal("missing release processing step") }
    run := strings.Index(workflow[start:], "        run: |\n")
    if run < 0 { t.Fatal("missing release processing command") }
    start += run + len("        run: |\n")
    end := strings.Index(workflow[start:], "\n      - name: Deduplicate ebuilds")
    if end < 0 { t.Fatal("missing next step boundary") }

    // YAML literal-block indentation is not part of the executed run command.
    raw := strings.Split(workflow[start:start+end], "\n")
    for i, line := range raw {
        if line != "" && !strings.HasPrefix(line, "          ") {
            t.Fatalf("unexpected command indentation at line %d", i+1)
        }
        raw[i] = strings.TrimPrefix(line, "          ")
    }
    script := strings.Join(raw, "\n")
    if len(script) >= 20000 {
        t.Fatalf("generated release command exceeds 20,000-byte safety budget: %d", len(script))
    }
    if count := strings.Count(script, "g2 manifest upsert-from-url"); count != 11 {
        t.Fatalf("expected 11 manifest asset operations, got %d", count)
    }
    if count := strings.Count(script, "${release_base}/"); count != 22 {
        t.Fatalf("expected SRC_URI and Manifest URLs for 11 assets, got %d", count)
    }
    if !strings.Contains(script, "release_base=\"https://github.com/") {
        t.Fatal("release URL base was not initialized")
    }
}
