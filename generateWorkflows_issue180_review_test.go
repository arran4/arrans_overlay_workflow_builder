package arrans_overlay_workflow_builder

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/arran4/arrans_overlay_workflow_builder/util"
	"github.com/stretchr/testify/require"
)

// Use the real filesystem for truncation, permissions and removal semantics;
// inject deterministic failures only at the operation under examination.
type issue180FaultFS struct {
	util.OSFS
	read           func(string) ([]byte, error)
	write          func(string, []byte, fs.FileMode) error
	remove         func(string) error
	lstat          func(string) (os.FileInfo, error)
	chmod          func(string, fs.FileMode) error
	mkdir          func(string, fs.FileMode) error
	writes, mkdirs int
}

func (f *issue180FaultFS) ReadFile(p string) ([]byte, error) {
	if f.read != nil {
		return f.read(p)
	}
	return os.ReadFile(p)
}
func (f *issue180FaultFS) WriteFile(p string, b []byte, m fs.FileMode) error {
	f.writes++
	if f.write != nil {
		return f.write(p, b, m)
	}
	return os.WriteFile(p, b, m)
}
func (f *issue180FaultFS) MkdirAll(p string, m fs.FileMode) error {
	f.mkdirs++
	if f.mkdir != nil {
		return f.mkdir(p, m)
	}
	return os.MkdirAll(p, m)
}
func (f *issue180FaultFS) Remove(p string) error {
	if f.remove != nil {
		return f.remove(p)
	}
	return os.Remove(p)
}
func issue180Path(dir, name string) string {
	return filepath.Join(dir, "app-misc-"+name+"-update.yaml")
}
func issue180Run(dir string, fsys util.FileSystem, names ...string) error {
	var configs []*InputConfig
	for _, name := range names {
		configs = append(configs, makeValidConfig("app-misc", name))
	}
	return GenerateGithubWorkflowsFromInputConfigs("test.config", configs, dir, "1.0.0", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), fsys)
}
func issue180AssertFile(t *testing.T, path, content string, mode fs.FileMode) {
	t.Helper()
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, content, string(b))
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, mode, info.Mode().Perm())
}

func TestIssue180Review_LateRenderLeavesNoDirectory(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "new", "workflows")
	cfg1, cfg2 := makeValidConfig("app-misc", "first"), makeValidConfig("app-misc", "bad")
	cfg2.Type = "invalid"
	require.Error(t, GenerateGithubWorkflowsFromInputConfigs("test.config", []*InputConfig{cfg1, cfg2}, dir, "1.0.0"))
	entries, err := os.ReadDir(parent)
	require.NoError(t, err)
	require.Empty(t, entries)
}
func TestIssue180Review_BackupReadFailsClosed(t *testing.T) {
	for _, cause := range []error{os.ErrPermission, errors.New("injected I/O failure")} {
		t.Run(cause.Error(), func(t *testing.T) {
			dir := t.TempDir()
			first, second := issue180Path(dir, "first"), issue180Path(dir, "second")
			require.NoError(t, os.WriteFile(first, []byte("first-original"), 0600))
			require.NoError(t, os.WriteFile(second, []byte("second-original"), 0640))
			f := &issue180FaultFS{read: func(p string) ([]byte, error) {
				if p == second {
					return nil, &os.PathError{Op: "read", Path: p, Err: cause}
				}
				return os.ReadFile(p)
			}}
			err := issue180Run(dir, f, "first", "second")
			require.ErrorIs(t, err, cause)
			require.Contains(t, err.Error(), second)
			require.Zero(t, f.writes)
			require.Zero(t, f.mkdirs)
			issue180AssertFile(t, first, "first-original", 0600)
			issue180AssertFile(t, second, "second-original", 0640)
		})
	}
}
func TestIssue180Review_PartialWriteRestoresFailingDestination(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(map[bool]string{false: "absent", true: "existing"}[existing], func(t *testing.T) {
			dir := t.TempDir()
			first, second, third := issue180Path(dir, "first"), issue180Path(dir, "second"), issue180Path(dir, "third")
			require.NoError(t, os.WriteFile(first, []byte("first-original"), 0640))
			if existing {
				require.NoError(t, os.WriteFile(third, []byte("third-original"), 0600))
			}
			cause := errors.New("partial third write")
			f := &issue180FaultFS{}
			f.write = func(p string, b []byte, m fs.FileMode) error {
				if f.writes == 3 {
					require.NoError(t, os.WriteFile(p, []byte("TRUNCATED"), m))
					require.NoError(t, os.Chmod(p, 0777))
					return cause
				}
				return os.WriteFile(p, b, m)
			}
			err := issue180Run(dir, f, "first", "second", "third")
			require.ErrorIs(t, err, cause)
			issue180AssertFile(t, first, "first-original", 0640)
			_, err = os.Stat(second)
			require.ErrorIs(t, err, os.ErrNotExist)
			if existing {
				issue180AssertFile(t, third, "third-original", 0600)
			} else {
				_, err = os.Stat(third)
				require.ErrorIs(t, err, os.ErrNotExist)
			}
		})
	}
}
func TestIssue180Review_RollbackFailuresReportedAndRecoveryContinues(t *testing.T) {
	dir := t.TempDir()
	first, second, third := issue180Path(dir, "first"), issue180Path(dir, "second"), issue180Path(dir, "third")
	require.NoError(t, os.WriteFile(first, []byte("first-original"), 0640))
	require.NoError(t, os.WriteFile(third, []byte("third-original"), 0600))
	publishErr, restoreErr, removeErr := errors.New("partial write"), errors.New("restore unavailable"), errors.New("remove unavailable")
	f := &issue180FaultFS{}
	f.write = func(p string, b []byte, m fs.FileMode) error {
		if f.writes == 3 {
			require.NoError(t, os.WriteFile(p, []byte("PARTIAL"), m))
			return publishErr
		}
		if p == third {
			return restoreErr
		}
		return os.WriteFile(p, b, m)
	}
	f.remove = func(p string) error {
		if p == second {
			return removeErr
		}
		return os.Remove(p)
	}
	err := issue180Run(dir, f, "first", "second", "third")
	require.ErrorIs(t, err, publishErr)
	require.ErrorIs(t, err, restoreErr)
	require.ErrorIs(t, err, removeErr)
	require.Contains(t, err.Error(), third)
	require.Contains(t, err.Error(), second)
	issue180AssertFile(t, first, "first-original", 0640)
	issue180AssertFile(t, third, "PARTIAL", 0600)
	require.FileExists(t, second)
}
func TestIssue180Review_DuplicateTargetsRejectedBeforeMutation(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "new")
	f := &issue180FaultFS{}
	err := issue180Run(dir, f, "same", "same")
	require.ErrorContains(t, err, "duplicate")
	require.Zero(t, f.writes)
	require.Zero(t, f.mkdirs)
	_, err = os.Stat(dir)
	require.ErrorIs(t, err, os.ErrNotExist)
}
func TestIssue180Review_SuccessPreservesPermissions(t *testing.T) {
	dir := t.TempDir()
	path := issue180Path(dir, "first")
	require.NoError(t, os.WriteFile(path, []byte("old"), 0600))
	require.NoError(t, issue180Run(dir, &issue180FaultFS{}, "first", "second"))
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(b), "app-misc/first update")
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, fs.FileMode(0600), info.Mode().Perm())
	info, err = os.Stat(issue180Path(dir, "second"))
	require.NoError(t, err)
	require.Equal(t, fs.FileMode(0644), info.Mode().Perm())
}

func (f *issue180FaultFS) Lstat(p string) (os.FileInfo, error) {
	if f.lstat != nil {
		return f.lstat(p)
	}
	return os.Lstat(p)
}
func (f *issue180FaultFS) Chmod(p string, m fs.FileMode) error {
	if f.chmod != nil {
		return f.chmod(p, m)
	}
	return os.Chmod(p, m)
}
func TestIssue180Review_BackupStatFailsClosed(t *testing.T) {
	dir := t.TempDir()
	first, second := issue180Path(dir, "first"), issue180Path(dir, "second")
	require.NoError(t, os.WriteFile(first, []byte("original"), 0600))
	cause := errors.New("stat unavailable")
	f := &issue180FaultFS{lstat: func(p string) (os.FileInfo, error) {
		if p == second {
			return nil, cause
		}
		return os.Lstat(p)
	}}
	err := issue180Run(dir, f, "first", "second")
	require.ErrorIs(t, err, cause)
	require.Zero(t, f.writes)
	require.Zero(t, f.mkdirs)
	issue180AssertFile(t, first, "original", 0600)
}
func TestIssue180Review_PermissionRecoveryFailureReported(t *testing.T) {
	dir := t.TempDir()
	first, second := issue180Path(dir, "first"), issue180Path(dir, "second")
	require.NoError(t, os.WriteFile(first, []byte("original"), 0600))
	cause, chmodErr := errors.New("publication unavailable"), errors.New("chmod unavailable")
	f := &issue180FaultFS{}
	f.write = func(p string, b []byte, m fs.FileMode) error {
		if f.writes == 2 {
			require.NoError(t, os.Chmod(first, 0777))
			return cause
		}
		return os.WriteFile(p, b, m)
	}
	f.chmod = func(p string, m fs.FileMode) error { return chmodErr }
	err := issue180Run(dir, f, "first", "second")
	require.ErrorIs(t, err, cause)
	require.ErrorIs(t, err, chmodErr)
	require.Contains(t, err.Error(), "restoring permissions for "+first)
	issue180AssertFile(t, first, "original", 0777)
	_, err = os.Stat(second)
	require.ErrorIs(t, err, os.ErrNotExist)
}
func TestIssue180Review_PublicationFailureCleansNewDirectories(t *testing.T) {
	for _, duringMkdir := range []bool{false, true} {
		t.Run(map[bool]string{false: "partial-file", true: "partial-mkdir"}[duringMkdir], func(t *testing.T) {
			parent := t.TempDir()
			dir := filepath.Join(parent, "new", "workflows")
			cause := errors.New("injected publication failure")
			f := &issue180FaultFS{}
			if duringMkdir {
				f.mkdir = func(p string, m fs.FileMode) error { require.NoError(t, os.MkdirAll(filepath.Dir(p), m)); return cause }
			} else {
				f.write = func(p string, b []byte, m fs.FileMode) error {
					require.NoError(t, os.WriteFile(p, []byte("partial"), m))
					return cause
				}
			}
			require.ErrorIs(t, issue180Run(dir, f, "first"), cause)
			entries, err := os.ReadDir(parent)
			require.NoError(t, err)
			require.Empty(t, entries)
		})
	}
}
func TestIssue180Review_RejectsNonRegularDestination(t *testing.T) {
	for _, symlink := range []bool{false, true} {
		t.Run(map[bool]string{false: "directory", true: "symlink"}[symlink], func(t *testing.T) {
			dir := t.TempDir()
			first, second := issue180Path(dir, "first"), issue180Path(dir, "second")
			require.NoError(t, os.WriteFile(first, []byte("first-original"), 0600))
			target := filepath.Join(t.TempDir(), "target")
			require.NoError(t, os.WriteFile(target, []byte("target-original"), 0600))
			if symlink {
				require.NoError(t, os.Symlink(target, second))
			} else {
				require.NoError(t, os.Mkdir(second, 0755))
			}
			f := &issue180FaultFS{}
			require.ErrorContains(t, issue180Run(dir, f, "first", "second"), "not a regular file")
			require.Zero(t, f.writes)
			require.Zero(t, f.mkdirs)
			issue180AssertFile(t, first, "first-original", 0600)
			issue180AssertFile(t, target, "target-original", 0600)
		})
	}
}
