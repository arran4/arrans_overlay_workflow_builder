# Batch workflow publication safety (#180)

The batch workflow generator renders and validates every YAML output in memory
and rejects duplicate destinations before touching the filesystem. It inspects
and backs up all existing regular output files before creating directories or
writing any output; read/stat errors abort publication. Existing symlink or
non-regular destinations are rejected. Success preserves existing file modes
and uses 0644 (subject to umask) for new workflows.

If publication fails, recovery includes the currently failing file (which may
have been truncated), restores previous bytes and permissions, and removes new
files and directory ancestors created by this batch. Recovery continues after
individual failures and returns the original error plus all restoration/removal
errors, with affected paths. Callers must handle reported recovery failures.

This is **best-effort rollback**, not multi-file atomic publication. Readers may
observe intermediate writes; concurrent writers are unsupported. SIGKILL,
crashes, power loss or recovery I/O failures may leave partial state; there is no
fsync/crash journal. The API has no preview flag. Per-entry rendering can be used
without publication. This contract covers generated workflow YAML only;
generated ebuild/Manifest transactions remain issue #155.
