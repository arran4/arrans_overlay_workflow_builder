# Final Audit Report for Issue #177

This audit report validates the codebase against the proposed engineering standards and specifically inspects the gaps related to issues #155, #166, #174, and newly discovered untracked defects.

## 1. Tracked Issue #155: Transactional generation (Failure-path nonmutation)
**Requirement**: A failure during candidate package state generation must not mutate the previously published state.
**Findings**:
*   `manifest_upsert.tmpl` executes sequentially. In a multi-architecture configuration (e.g. `amd64`, `arm64`), if an early asset hashes successfully but a subsequent asset fails to download or hash, `g2 manifest upsert-from-url` has already mutated the `Manifest` file.
*   **Risk**: The repository can be left in a partially updated state on error.
*   **Recommendation**: This is already tracked via #155. The resolution will involve building a complete temporary `Manifest` and `ebuild` in a staging directory, validating them, and moving them atomically.

## 2. Tracked Issue #166: Validate generated binary ebuilds against real g2 cache and lint
**Requirement**: Ensure that generator output actually passes `g2 lint` and `g2 cache` policies rather than just passing the mocked CI steps.
**Findings**:
*   `smoke_test.go` stubs the `g2` binary via bash commands (e.g., `echo "g2 cache generate called"`). It does not invoke actual `g2` logic.
*   `generateWorkflows_extended_test.go` lacks a test demonstrating that generated files pass `g2 lint`.
*   **Risk**: Generated workflow outputs might contain subtle formatting or syntax errors that bypass the generator's internal validation but fail when processed by the real Gentoo/g2 toolchain.
*   **Recommendation**: This is accurately tracked in #166.

## 3. Tracked Issue #174: Standardize reusable HTTP and binary fixtures in tests
**Requirement**: Minimize temporary file usage and extract repeated test setup logic (e.g., `httptest.Server`).
**Findings**:
*   Inline `httptest.NewServer` mock logic is duplicated extensively across `pipeline_test.go`, `smoke_test.go`, `util/appimage_test.go`, and `webAppImage_test.go`.
*   `os.WriteFile` and `t.TempDir()` are used heavily in `smoke_test.go` and `generateWorkflows_test.go`.
*   **Risk**: Test setup is verbose and hard to read, violating the standard "Tests should explain contracts instead of setup machinery."
*   **Recommendation**: This is tracked in #174.

## 4. Untracked Gap: Sequential Workflow Output Generation
**Requirement**: Generation should be safe and dry-runs should not mutate user/repository state.
**Findings**:
*   In `generateWorkflows.go:110`, the generator explicitly discards `MkdirAll` errors: `_ = fsys.MkdirAll(outputDir, 0755)`.
*   `GenerateGithubWorkflowsFromInputConfigs` iterates through `inputConfigs`. If a config fails (`inputConfig.GenerateGithubWorkflow(...) != nil`), it immediately returns the error. Because writes happen inline within the loop (via `fsys.WriteFile`), previously generated valid configs have already been written to the filesystem.
*   **Risk**: A partial generation failure can leave incomplete or mixed batches of generated workflow `.yaml` files in the output directory. Errors related to directory creation are silently suppressed.
*   **Proposed Separate Implementation Issue**: Create a new issue titled "Make batch workflow generation transactional and handle MkdirAll errors". The acceptance criteria should mandate a regression test proving that if the second config in a batch fails, the outputs from the first config are not saved (or existing files are not overwritten), and that `MkdirAll` errors are correctly propagated instead of ignored (`_ =`).

## 5. Verification Commands Run
*   `go test ./...` (All existing tests pass)
*   `grep` and `cat` across `generateWorkflows.go`, `inputconfig.go`, and `cmd/generate/main.go` to inspect the filesystem and loop logic.
*   Analysis of `smoke_test.go` for `g2` integration mechanisms.
