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
