1. Add explicit string replacement variables `{{.Tag}}` and `{{.Version}}` parsing to `actionvardoublequoted` and `ebuildvardoublequoted` functions in `generateWorkflows.go`.
2. Ensure bash script inside `github-binary.tmpl`, `github-appimage.tmpl`, `github-cmake.tmpl`, `web-binary.tmpl`, and `web-appimage.tmpl` export the missing `${version}`, `${tag}` (as `TAG`) and `${originalVersion}` variables properly before constructing the `SRC_URI` block, so that it can be used inside `CustomDownloadUrl` or templates correctly (for `replace` calls).
3. Validate fixes by testing template output with `go test -update-txtar ./...`.
4. Perform pre commit checks (using `pre_commit_instructions`).
5. Submit the PR.
