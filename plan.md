1. **Fix `src_uri.tmpl` structurally.**
   - Ensure the `\${PV}` append is removed when `.DownloadPipeline` is used.
   - Refactor `src_uri.tmpl` to unconditionally use `${resolved_download_urls["[[ $i ]]"]}` without any suffixes when `DownloadPipeline` is present, putting `ReleaseFilename` append inside the non-pipeline `else` branch.
2. **Apply the same fix to every Manifest branch.**
   - Audit and modify all `manifest_upsert.tmpl` branches (including semantic version prerelease paths) to guarantee that when `DownloadPipeline` is active, the exact cached `resolved_download_urls["[[ $i ]]"]` is used without appending `ReleaseFilename` or `PV`.
3. **Fix helper materialization using the EFFECTIVE pipeline.**
   - Update `web-binary.tmpl` to use `.InputConfig.GetDownloadPipeline` (and version equivalent) when determining if `g2-pipeline.py` needs to be materialized.
   - Add a test case with `TagsCommand` and `DownloadRegex` to trigger the fallback regression path and ensure Python materialization.
4. **Fix placeholder substitution before inserting RELEASE_FILENAME.**
   - In `.tmpl` scripts (like `web-binary.tmpl` and `web-appimage.tmpl`), alter the bash variable substitution order so that `G2_PIPELINE="${G2_PIPELINE//\${RELEASE_FILENAME}/$resolved_filename}"` evaluates safely without leaving raw `${VERSION}` bash variables stranded in the pipeline script string.
5. **Add a real multi-resource test.**
   - Create a new `txtar` test fixture with at least two different `ExternalResources` to verify that `resolved_download_urls["0"]` and `["1"]` are correctly resolved, injected into `SRC_URI`, and utilized independently in Manifest commands.
6. **Restore `DownloadBaseUrl` validation.**
   - In `inputconfig.go`, remove `emptyOrOnlyOrFail(...)` discarding the error and replace it with proper error handling and propagation for `DownloadBaseUrl`.
7. **Finish the `replace()` parser cleanup.**
   - In `pipeline.py`, replace the manual string parser for `replace()` with `shlex.split(args_str)` (restoring the `shlex` import).
   - Ensure escape sequences and escaped quotes are handled properly. Add tests for these scenarios in `pipeline_test.go`.
8. **Add exact legacy compatibility tests.**
   - In `inputconfig_test.go`, add unit tests asserting the exact literal output strings for `GetDownloadPipeline()` when generated from `DownloadRegex` and `DownloadXPath` properties.
9. **Fix the README.**
   - Update documentation reflecting `html_links` behavior, `xml` capabilities, `xpath(...)` support, and `DownloadPipeline` substitutions (`TAG`, `VERSION`, `RELEASE_FILENAME`).
10. **Fix CI and execute checks.**
    - Handle `w.Write` errors in the mock HTTP server in `pipeline_test.go` to satisfy `govulncheck` / `golangci-lint`.
    - Run the full test suite (`go test ./...`), update textars (`-update-txtar`), and run `actionlint`.
11. **Submit Changes.**
