# g2 CLI Usability & Feature Report

## Overview
This document summarizes observed usability friction, flag handling edge cases, and feature enhancements for the `g2` tool based on workflow generator integration and CI run analysis.

---

## 1. `g2 lint` Repository Scope vs. Targeted Package Scope
* **Observed Behavior**: `g2 lint .` defaults to scanning and linting the entire overlay repository (`.`).
* **CI Impact**: When GitHub Action workflows are generated per-package, executing `g2 lint .` causes package-specific workflows to fail if *any other unrelated package* in the repository contains lint warnings or errors.
* **Workaround Implemented**: Updated generator templates to explicitly pass the target package category and name: `action: 'lint . ${{ env.ecn }}/${{ env.epn }}'`.
* **Suggested Enhancement**: 
  - Consider adding a `--changed-only` or `--staged` flag to `g2 lint` so it automatically limits linting to packages modified in the current git workspace/commit.

---

## 2. Strict Flag Parsing Order in `g2 ebuild next-revision`
* **Observed Behavior**: Passing flags after positional arguments causes `g2` to fail with a usage error.
  * **Failing Command**:
    ```bash
    g2 ebuild next-revision ./app-misc/pkg-0.1.0 -inspect ./app-misc/pkg-0.1.0.ebuild.tmp
    ```
    *Output*: `ebuild error: ebuild next-revision: usage: g2 ebuild next-revision [--inspect <new_ebuild_file>] <ebuildDir> <version>`
  * **Working Command**:
    ```bash
    g2 ebuild next-revision --inspect ./app-misc/pkg-0.1.0.ebuild.tmp ./app-misc 0.1.0
    ```
* **Root Cause**: Go's standard `flag` package halts option parsing at the first positional argument (`./app-misc/pkg-0.1.0`). Any flags specified after positional arguments remain unparsed in `flag.Args()`.
* **Suggested Enhancement**: Use a relaxed flag parser (such as `pflag` or `cobra`) to support intermixed positional arguments and flags regardless of command-line ordering.

---

## 3. Parameter Format Handling for `<version>` in `next-revision`
* **Observed Behavior**:
  * Passing pure version `0.1.0` works as expected.
  * Passing prefixed `<epn>-<version>` (e.g. `pkg-0.1.0`) or suffixed `<version>.ebuild` can lead to unexpected revision target matching or exit behavior depending on input string formatting.
* **Suggested Enhancement**: Automatically trim leading `<epn>-` prefixes and trailing `.ebuild` / `.tmp` suffixes from `<version>` parameters inside `g2 ebuild next-revision` for robust string handling.

---

## 4. Exit Code Convention for `--inspect`
* **Observed Behavior**: When `--inspect` detects that the generated ebuild content is identical to the existing highest revision (i.e. no update needed), `g2` exits with status `1`.
* **Impact**: Returning a non-zero exit status for non-error control flow ("no content change") requires shell scripts using `set -e` to append explicit checks or `|| true`.
* **Suggested Enhancement**: Provide a dedicated flag or return status code documentation separating operational errors from "no diff detected" outcomes.

## 5. `g2 ebuild next-revision` version matching ignores exact string literal matching on filename and requires exact GentooVersion match on stripped parameters

* **Observed Behavior**: The workflow generator currently passes unsterilized variable inputs (which may include `v` prefixes, `.ebuild` or `.tmp` suffixes, or even `<epn>-` package name prefixes if incorrectly parsed by custom tag extraction commands) as the `<version>` parameter to `g2 ebuild next-revision`. When this occurs:
  1. `g2 ebuild next-revision`'s `getNextRevision` function strictly matches the *parsed base Gentoo Version string* against the provided `<version>` string. For instance, if the existing file is `which_browser-0.2.6.44-r1.ebuild`, its parsed base `PV` is `0.2.6.44`. If the `<version>` argument provided is `0.2.6.44.ebuild`, the condition `base == version` evaluates to `"0.2.6.44" == "0.2.6.44.ebuild"`, which is false.
  2. Finding no matches, `next-revision` silently assumes this is an entirely new base version. It immediately returns the unsterilized, erroneous string (e.g. `0.2.6.44.ebuild`) and exits with a `0` exit code.
  3. Consequently, workflow scripts construct the next filename dynamically using the returned string (e.g. `which_browser-0.2.6.44.ebuild.ebuild`) or incorrectly recreate base non-revisioned `.ebuild` files, causing an endless cycle of file recreation because it never acknowledges that the `0.2.6.44-r1` revision already exists for that version tag.

* **Impact**: Unsanitized parameters lead to false negatives in revision scanning, directly triggering the constant recreation of unrevisioned (or incorrectly named) ebuild files in automated environments.

* **Suggested Enhancement**:
  * **Option A (Parameter Sanitization)**: Implement automatic preprocessing within `g2 ebuild next-revision` to aggressively strip common filename artifacts (e.g. `.ebuild`, `.tmp`) and repository-specific prefixes (`v`, `<epn>-`) from the `<version>` argument before evaluating `base == version`.
  * **Option B (Robust Version Parsing)**: Instead of a strict literal string comparison (`base == version`), parse the incoming `<version>` argument using `g2.ParseGentooVersion(version)`. If valid, compare the parsed output against the `base` of existing ebuilds in the directory, effectively ignoring surrounding noise and enforcing strict semantic/Gentoo version equivalency.
