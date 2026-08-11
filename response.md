# Response to Issue #88: 5 ebuilds require 2 workarounds

The issues raised in this ticket have been addressed by the following implemented features:

1. **Mechanism for Preserving Custom Ebuild Bash Logic:**
   The `EbuildInclude` directive has been fully implemented and is supported across the templates. You can now inject custom bash logic directly into specific ebuild phases. For example:
   ```
   EbuildInclude pkg_postinst => echo "Please add your user to the video group."
   ```

2. **Bash Context Evaluation Handling for Binary Paths:**
   Variable evaluation within generated bash scripts has been standardized. The templates now use `actionvardoublequoted` and `ebuildvardoublequoted` template functions to correctly resolve literal placeholders like `${TAG}` and `${VERSION}` to the appropriate loop variables (e.g., `${tag}`) or ebuild variables (e.g., `\${PV}`).

### Specific Examples Addressed:
* **`app-misc/gocdm-bin`**: The requirement for a custom `pkg_postinst` block warning the user about system group permissions can now be achieved using the `EbuildInclude pkg_postinst => ...` directive.
* **`dev-util/codex-bin`**: The unusual tag format (`rust-v0.1.0`) is now fully supported by the expanded `Workaround Tag Prefix => rust-v` directive, which strips the prefix correctly when generating Gentoo PV logic.
* **`media-sound/go-playerctl-bin`**: Automatically emitting `dosym` commands during `src_install` is now supported via the `Symlink target => destination` directive (e.g., `Symlink go-playerctl => /usr/bin/go-playerctl-bin`).
* **`app-misc/flutter-jules-bin`**: The `License` field is now supported in the configuration parser, preventing the fallback to `unknown`. Custom `sed` statements for `.desktop` files can be injected using `EbuildInclude src_prepare => ...` or `EbuildInclude src_install => ...`.
* **`www-misc/which_browser`**: While highly complex, specialized unpacking and dependency replacements can similarly be mitigated by leveraging the `EbuildInclude` feature for `src_prepare` and `src_install`, minimizing the need for manual file overwrites, thus making the "Ignore" flag less necessary.

To verify these capabilities, integration tests (txtar) have been added to demonstrate `Symlink` and `EbuildInclude` behaviors matching expected Gentoo ebuild outputs.
