package arrans_overlay_workflow_builder

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestReleaseGraphInvariants(t *testing.T) {
	content, err := os.ReadFile(".github/workflows/ci.yml")
	if err != nil {
		t.Fatalf("failed to read ci.yml: %v", err)
	}
	yamlStr := string(content)

	// 1. External eligible tag push routes to release
	if !strings.Contains(yamlStr, `if [[ "${{ github.ref }}" == refs/tags/v* ]]; then
                run_release=true`) {
		t.Errorf("Missing explicit routing for v* tag push")
	}

	// 2. release-context allows skipped prepare-release-tag and reaches GoReleaser
	if !strings.Contains(yamlStr, `if: ${{ always() && !failure() && !cancelled() && needs.route.outputs.run_release == 'true' && (needs.discover.outputs.has_go != 'true' || needs.go-test.result == 'success') && (needs.prepare-release-tag.result == 'success' || needs.prepare-release-tag.result == 'skipped') }}`) {
		t.Errorf("release-context condition missing skipped prepare-release-tag handling or misses go-test success validation")
	}

	// 3. publish-tag on branch is rejected via publish-tag mode invoked on a non-tag ref
	if !strings.Contains(yamlStr, `if [[ "${{ github.ref_type }}" != "tag" ]]; then`) || !strings.Contains(yamlStr, `echo "Error: publish-tag mode invoked on a non-tag ref." >&2`) {
		t.Errorf("publish-tag non-tag ref rejection missing")
	}

	// 4. Release tag validation executes before early exits
	earlyExitRegex := regexp.MustCompile(`(?s)# Validate tag format against eligible release-tag policy.*if \[\[ "\$\{\{ github.event_name \}\}" == "push" \]\]; then`)
	if !earlyExitRegex.MatchString(yamlStr) {
		t.Errorf("Tag validation doesn't appear before early exits")
	}

	// 5. prerelease modes retain their GoReleaser snapshot semantics
	snapshotRegex := regexp.MustCompile(`args: 'release --clean \$\{\{ \(github\.event_name == ''workflow_dispatch'' && inputs\.mode == ''publish-tag'' && \(contains\(github\.ref_name, ''-test''\) \|\| contains\(github\.ref_name, ''-rc''\) \|\| contains\(github\.ref_name, ''-alpha''\)\)\) && ''--snapshot'' \|\| '''' \}\}'`)
	if !snapshotRegex.MatchString(yamlStr) {
		t.Errorf("Prerelease modes do not preserve GoReleaser snapshot semantics")
	}

	// 6. release: published remains downstream-only
	if !strings.Contains(yamlStr, `release)
              # Downstream notification only. Never create the same release again here.
              ;;`) {
		t.Errorf("release: published must be downstream-only")
	}

	// 7. GoReleaser must be sole creator: Ensure softprops/action-gh-release is NOT present
	if strings.Contains(yamlStr, `softprops/action-gh-release`) {
		t.Errorf("softprops/action-gh-release should not be present; GoReleaser must be sole creator")
	}
}
