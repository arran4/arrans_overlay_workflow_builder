Ensure `templates/github-binary.tmpl` properly uses `echo` with tabs.
The generator supports a `Web Binary` configuration type (using `generateWebBinaryWorkflow.go` and `web-binary.tmpl`) for handling non-GitHub releases. It relies on the `DownloadBaseUrl` field to construct the `SRC_URI` and strictly requires `TagsCommand` to fetch version tags (e.g., using a custom `curl` command to parse HTML release notes).

Every txtar test must contain the complete generated output in `expected.yaml`. Do not use txtar files for partial-output or single-assertion tests; use regular unit tests with fixtures under `testdata/` for those cases instead. Keep txtar coverage for most major generator features and code paths because these archives are also used for manual evaluation of generated workflows. For each workaround and each template, there must be at least one (often more for testing interactions) `.txtar` test verifying the output.

Txtar inputs must be fully qualified enough to generate a functional ebuild for every major generator type. Inject nondeterministic values such as cron schedules and timestamps in txtar tests so unrelated input changes do not rewrite snapshots. Before changing a `uses:` reference, verify the current appropriate major version against the action's official GitHub repository.

Every workaround added (like `Workaround Version Replacement`, `Workaround Gentoo Version Regular Expression`, etc.) must have a txtar test proving that it works for the corresponding outputs (e.g., github-cmake, web-binary, github-binary, github-appimage, etc.).

## Deterministic generation, safe preview and review

- Treat configuration and templates as source; generated workflows and ebuilds are outputs. Where a preview/dry-run mode exists, it must not rewrite repository files, destination overlays or pre-existing user data, even temporarily. Calculate a plan and compare candidate output in isolated storage; apply only in explicit write mode. Follow existing transactional-generation issue #155 rather than opening a duplicate.
- Reject incomplete source metadata or incompatible releases rather than silently emitting a plausible but incorrect package. Preserve g2-compatible generated metadata and validate it with the real downstream checker where that is the contract (issue #166).
- Keep the existing complete-output txtar `expected.yaml` requirements. Use narrowly scoped Go tests for individual helpers, reusable HTTP responses and binary data, and full txtar fixtures for end-to-end generator outputs (issue #174).
- Avoid generic global environment overrides or network calls in routine test scenarios. Use injected sources or fixture transport where possible; keep real upstream and Gentoo checks as labelled integration coverage.

## Engineering guidance and agent audit

Use https://arran4.github.io/blog/post/2026/055-suggested-coding-standards-for-repositories-and-agents/ selectively, without superseding the more specific rules above. A coding agent must inspect implementation/tests, distinguish verified gaps from hypotheses, and record results and existing issue overlap at https://github.com/arran4/arrans_overlay_workflow_builder/issues/177. Do not equate adding instructions with completing the implementation audit; do not merge automatically.
