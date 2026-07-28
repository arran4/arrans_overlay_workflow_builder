Ensure `templates/github-binary.tmpl` properly uses `echo` with tabs.
The generator supports a `Web Binary` configuration type (using `generateWebBinaryWorkflow.go` and `web-binary.tmpl`) for handling non-GitHub releases. It relies on the `DownloadBaseUrl` field to construct the `SRC_URI` and strictly requires `TagsCommand` to fetch version tags (e.g., using a custom `curl` command to parse HTML release notes).

Every txtar test must contain the complete generated output in `expected.yaml`. Do not use txtar files for partial-output or single-assertion tests; use regular unit tests with fixtures under `testdata/` for those cases instead. Keep txtar coverage for most major generator features and code paths because these archives are also used for manual evaluation of generated workflows.

Txtar inputs must be fully qualified enough to generate a functional ebuild for every major generator type. Inject nondeterministic values such as cron schedules and timestamps in txtar tests so unrelated input changes do not rewrite snapshots. Before changing a `uses:` reference, verify the current appropriate major version against the action's official GitHub repository.
