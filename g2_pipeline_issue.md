# Feature Request: Add pipeline data extraction capability

**Motivation and Ownership Rationale**
The `arrans_overlay_workflow_builder` project currently relies on a custom Python DSL script (`pipeline.py`) to scrape, parse, and extract version and download URL data from various external web sources (HTML, JSON, XML, RSS). This script is currently embedded as a giant base64-encoded executable blob in generated Web Binary and Web AppImage GitHub Actions workflows.

This is unacceptable as a long-term design. It makes generated workflows opaque, hard to review, and bloats the YAML. Because these generated workflows already utilize `g2` for operational package-maintenance functionality, `g2` is the architecturally correct ownership boundary for this extraction logic. It should have one normal, versioned implementation within `g2`, eliminating the need to transport and execute hidden Python payloads.

**CLI / API Capability Required**
A new command (e.g., `g2 extract` or a subcommand under `g2 pipeline`, adhering to existing `g2` CLI conventions) is needed to safely parse and execute a declarative extraction expression. The workflow-builder would emit a human-readable command like:
`tags="$(g2 extract '<readable pipeline expression>')"`

**Existing Workflow-Builder Pipeline Operators & Semantics to Preserve**
The new `g2` implementation must preserve the existing pipeline functionality, including:

*   **Network fetching**: `get(...)`
*   **Structured extraction**: `json(path)`, `xml`, `rss`, `atom`, `xpath(query)`
*   **Text processing**: `regex(pattern)`, `trim`
*   **HTML parsing**: `html_links`
*   **Data normalization**: `replace('old', 'new')`, `url.basename`, `link`
*   **List operations**: `first`, `last`, `single`, `exactly_one`

**Input / Output / Error Contracts**
*   **Input**: The command takes a string expression, and potentially a starting payload or variables.
*   **Output**: Extracted string values printed to stdout.
*   **Errors**: Must return a non-zero exit code and actionable error messages on stdout/stderr for:
    *   Malformed pipeline expressions.
    *   Network failures in `get(...)`.
    *   Parsing errors (e.g., invalid JSON/XML).

**Cardinality & Matching Semantics**
*   **Zero results**: A pipeline yielding an empty list without `single`/`exactly_one` should complete with no output (unless intentionally changed). Zero or ambiguous cardinality is only an error when `single` / `exactly_one` asserts exactly one result.
*   **One result**: Success, output the scalar value.
*   **Multiple results**:
    *   If expected (e.g., extracting multiple links), output them newline-separated.
    *   If unambiguous extraction is expected but multiple are found (e.g., when using `exactly_one` or `single`), it must result in an actionable ambiguity failure.
*   **Legitimate falsey scalar values**: Legitimate scalar values such as numeric JSON `0` or boolean `false` must not accidentally be confused with absence merely because of truthiness. The current empty-string policy (which explicitly requires an empty string to fail as zero cardinality under `exactly_one`) must preserve established semantics unless intentionally changed.

**Substitution Requirements**
The pipeline expression often needs to interpolate variables. The `g2` command must support replacing placeholders like `${VERSION}`, `${TAG}`, and `${RELEASE_FILENAME}` when they are provided (either via environment variables or explicit CLI flags).

**Safe Handling of Pipeline Expressions**
The pipeline expression should remain a visibly reviewable, safely quoted argument in shell scripts. There should be no runtime "decode and execute" behavior or opaque base64 encoding needed just to pass the expression to `g2`.

**Testing & Validation Requirements**
*   **Unit & Semantic Tests**: The implementation must include robust tests covering all operators (`get`, `regex`, `json`, `xpath`, `html_links`, etc.) and cardinality rules.
*   **Smoke Test (`which_browser` regression)**: The `which_browser` version string `0.2.6+44` extraction scenario (as established in workflow-builder #134) must remain a first-class end-to-end regression test to ensure generic artifacts are discovered and produce the correct normalized Gentoo version, `SRC_URI`, and manifest handoff.
*   **Compatibility**: The existing workflow-builder configuration syntax/semantics must continue to work seamlessly when translated to the new `g2` command.

**Note**
This issue is intended to unblock `arrans_overlay_workflow_builder` issue #160. Workflow builder generation of Web Binary and Web AppImage workflows is currently architecturally blocked from migrating off its base64 Python implementation until this `g2` capability exists.
