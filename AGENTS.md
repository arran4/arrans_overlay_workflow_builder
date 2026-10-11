Ensure `templates/github-binary.tmpl` properly uses `echo` with tabs.
The generator supports a `Web Binary` configuration type (using `generateWebBinaryWorkflow.go` and `web-binary.tmpl`) for handling non-GitHub releases. It relies on the `DownloadBaseUrl` field to construct the `SRC_URI` and strictly requires `TagsCommand` to fetch version tags (e.g., using a custom `curl` command to parse HTML release notes).

Every txtar test must contain the complete generated output in `expected.yaml`. Do not use txtar files for partial-output or single-assertion tests; use regular unit tests with fixtures under `testdata/` for those cases instead. Keep txtar coverage for most major generator features and code paths because these archives are also used for manual evaluation of generated workflows. For each workaround and each template, there must be at least one (often more for testing interactions) `.txtar` test verifying the output.

Txtar inputs must be fully qualified enough to generate a functional ebuild for every major generator type. Inject nondeterministic values such as cron schedules and timestamps in txtar tests so unrelated input changes do not rewrite snapshots. Before changing a `uses:` reference, verify the current appropriate major version against the action's official GitHub repository.

Every workaround added (like `Workaround Version Replacement`, `Workaround Gentoo Version Regular Expression`, etc.) must have a txtar test proving that it works for the corresponding outputs (e.g., github-cmake, web-binary, github-binary, github-appimage, etc.).

## Engineering Standards & Guidelines

### Non-Mutating Generation and Planning vs. Apply
- Validate all inputs, parse templates, and render candidate workflow outputs completely in memory before writing to disk.
- Generation must fail closed: if any configuration entry or template execution fails during batch generation, do not leave partially written or corrupted workflow files on disk.
- Prefer in-memory filesystem abstractions (`util.NewMockFS()`) or isolated temporary directories during candidate evaluation and testing to prevent mutating repo state or user environments.

### Reproducible Workflow Output
- Keep workflow generation deterministic and idempotent. Normalize trailing whitespace and blank lines within template outputs (`normalizeGeneratedWorkflow`).
- When testing or rendering workflows, inject fixed nondeterministic values (such as cron schedules and timestamps) so unrelated input changes do not churn snapshots.
- Ensure embedded bash scripts and `echo` statements use tab indentation where required by Gentoo ebuild / workflow conventions.

### Test Fixture Selection and Boundary Testing
- **Txtar Archives (`testdata/txtar/`)**: Reserved for full-output snapshot verification. Every `.txtar` file MUST contain the complete expected generated workflow in `expected.yaml`. Do not use `.txtar` for single-assertion or partial-output tests.
- **Unit Test Fixtures (`testdata/config/`, `util.NewMockFS()`)**: Use unit tests with `util.NewMockFS()` or focused `testdata/` fixtures for testing specific options, partial output matches, error paths, or edge cases without creating temporary files or altering process environments.
- **Boundary Integration Tests (`smoke_test.go`)**: Maintain hermetic subprocess/shell execution tests to verify script validity (`bash -n`) and runtime execution with stubbed binaries (`gh`, `g2`, `git`, `wget`) without touching live external services or rebuilding process environment directories (such as XDG/home paths).

For further rationale on engineering standards and Go test isolation, see:
- [Suggested Coding Standards](https://arran4.github.io/blog/post/2026/055-suggested-coding-standards-for-repositories-and-agents/)
- [Stop Rebuilding XDG Directories in Go Tests](https://arran4.github.io/blog/post/2026/054-stop-rebuilding-xdg-directories-in-go-tests/)
