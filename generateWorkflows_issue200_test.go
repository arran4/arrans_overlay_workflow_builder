package arrans_overlay_workflow_builder

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type issue200Workflow struct {
	Env  map[string]string `yaml:"env"`
	Jobs map[string]struct {
		Env   map[string]string `yaml:"env"`
		Steps []struct {
			Name string            `yaml:"name"`
			If   string            `yaml:"if"`
			Run  string            `yaml:"run"`
			With map[string]string `yaml:"with"`
		} `yaml:"steps"`
	} `yaml:"jobs"`
}

func generateIssue200Workflow(t *testing.T, configStr string, opts ...any) issue200Workflow {
	t.Helper()
	tmpls, err := ParseWorkflowTemplates()
	require.NoError(t, err)

	configs, err := ParseInputConfigReader(strings.NewReader(configStr))
	require.NoError(t, err)
	require.Len(t, configs, 1)

	outDir := t.TempDir()
	err = configs[0].GenerateGithubWorkflow(
		"test.config",
		time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC),
		tmpls,
		outDir,
		"1.0.0",
		opts...,
	)
	require.NoError(t, err)

	entries, err := os.ReadDir(outDir)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	data, err := os.ReadFile(filepath.Join(outDir, entries[0].Name()))
	require.NoError(t, err)

	var wf issue200Workflow
	require.NoError(t, yaml.Unmarshal(data, &wf))
	return wf
}

func resolveIssue200StepScript(t *testing.T, wf issue200Workflow, stepName string, extraVars map[string]string) string {
	t.Helper()
	job, ok := wf.Jobs["check-and-create-ebuild"]
	require.True(t, ok, "expected check-and-create-ebuild job")

	var rawRun string
	for _, s := range job.Steps {
		if s.Name == stepName {
			rawRun = s.Run
			break
		}
	}
	require.NotEmpty(t, rawRun, "step %q must have non-empty run script", stepName)

	script := rawRun
	for k, v := range wf.Env {
		script = strings.ReplaceAll(script, fmt.Sprintf("${{ env.%s }}", k), v)
	}
	for k, v := range job.Env {
		script = strings.ReplaceAll(script, fmt.Sprintf("${{ env.%s }}", k), v)
	}
	for k, v := range extraVars {
		script = strings.ReplaceAll(script, fmt.Sprintf("${{ env.%s }}", k), v)
	}

	var prefix strings.Builder
	prefix.WriteString("set -euo pipefail\n")
	if tag, ok := extraVars["generated_tag"]; ok {
		prefix.WriteString(fmt.Sprintf("generated_tag=%q\n", tag))
	}
	return prefix.String() + script
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v in %s failed:\n%s", args, dir, string(out))
	return string(out)
}

func initBareAndClone(t *testing.T, seedFiles map[string]string) (bareDir string, cloneDir string) {
	t.Helper()
	root := t.TempDir()
	bareDir = filepath.Join(root, "remote.git")
	runGit(t, root, "init", "--bare", "--initial-branch=main", bareDir)

	seedDir := filepath.Join(root, "seed")
	runGit(t, root, "clone", bareDir, seedDir)
	runGit(t, seedDir, "config", "user.name", "Test Bot")
	runGit(t, seedDir, "config", "user.email", "test@example.com")
	runGit(t, seedDir, "config", "commit.gpgsign", "false")

	for rel, content := range seedFiles {
		abs := filepath.Join(seedDir, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o755))
		require.NoError(t, os.WriteFile(abs, []byte(content), 0o644))
	}
	runGit(t, seedDir, "add", "-A")
	runGit(t, seedDir, "commit", "-m", "Initial overlay commit")
	runGit(t, seedDir, "push", "-u", "origin", "main")

	cloneDir = filepath.Join(root, "work")
	runGit(t, root, "clone", bareDir, cloneDir)
	runGit(t, cloneDir, "config", "user.name", "Updater Bot")
	runGit(t, cloneDir, "config", "user.email", "updater@example.com")
	runGit(t, cloneDir, "config", "commit.gpgsign", "false")
	return bareDir, cloneDir
}

func cloneRemoteInspect(t *testing.T, bareDir string) string {
	t.Helper()
	inspectDir := filepath.Join(t.TempDir(), "inspect")
	runGit(t, filepath.Dir(inspectDir), "clone", "--branch", "main", bareDir, inspectDir)
	return inspectDir
}

func runBashScript(dir string, env []string, script string) (string, error) {
	cmd := exec.Command("bash", "-c", script)
	cmd.Dir = dir
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestIssue200_GoReleaser43794460683_UnavailableMastersDeletedCache(t *testing.T) {
	verifyG2Available(t)

	for _, featureMd5 := range []bool{false, true} {
		name := fmt.Sprintf("FeatureGenerateMd5Cache=%v", featureMd5)
		t.Run(name, func(t *testing.T) {
			cfg := `Type Github Binary Release
GithubProjectUrl https://github.com/goreleaser/goreleaser
EbuildName goreleaser-bin
Category dev-go
Description Deliver Go binaries as fast and easily as possible
Homepage https://goreleaser.com
License MIT
Binary amd64=>goreleaser_Linux_x86_64.tar.gz > goreleaser > goreleaser
`
			if featureMd5 {
				cfg += "Feature Generate Md5 Cache\n"
			}
			wf := generateIssue200Workflow(t, cfg)

			bareDir, workDir := initBareAndClone(t, map[string]string{
				"profiles/repo_name":                            "arrans-overlay",
				"profiles/eapi":                                 "8\n",
				"metadata/layout.conf":                          "masters = gentoo\ncache-formats = md5-dict\n",
				"dev-go/goreleaser-bin/goreleaser-bin-2.17.1.ebuild": "EAPI=8\ninherit unpacker\nDESCRIPTION=\"GoReleaser\"\nSLOT=\"0\"\n",
				"dev-go/goreleaser-bin/Manifest":                "DIST goreleaser-2.20.2.tar.gz 100 BLAKE2B 00 SHA512 00\n",
				"metadata/md5-cache/dev-go/goreleaser-bin-2.17.1": "EAPI=8\nDESCRIPTION=GoReleaser\n_md5_=1111\n",
				"metadata/md5-cache/app-misc/other-pkg-1.0.0":   "EAPI=8\nDESCRIPTION=Other\n_md5_=2222\n",
			})

			// Simulate upgrade from 2.17.1 -> 2.20.2 where deduplication removed 2.17.1.ebuild
			require.NoError(t, os.Remove(filepath.Join(workDir, "dev-go/goreleaser-bin/goreleaser-bin-2.17.1.ebuild")))
			require.NoError(t, os.WriteFile(
				filepath.Join(workDir, "dev-go/goreleaser-bin/goreleaser-bin-2.20.2.ebuild"),
				[]byte("EAPI=8\ninherit unpacker\nDESCRIPTION=\"GoReleaser\"\nSLOT=\"0\"\n"),
				0o644,
			))
			require.NoError(t, os.WriteFile(
				filepath.Join(workDir, "dev-go/goreleaser-bin/Manifest"),
				[]byte("DIST goreleaser-2.20.2.tar.gz 123 BLAKE2B 00 SHA512 00\n"),
				0o644,
			))

			cacheScript := resolveIssue200StepScript(t, wf, "Generate md5-cache", nil)
			cacheOut, err := runBashScript(workDir, []string{"G2_MASTER_REPOS="}, cacheScript)
			require.NoError(t, err, "Generate md5-cache failed:\n%s", cacheOut)
			require.Contains(t, cacheOut, "Authoritative md5-cache generation requires unavailable master repository context")

			// Obsolete 2.17.1 cache must be deleted; 2.20.2 cache must not be fabricated; unrelated cache must remain.
			_, err = os.Stat(filepath.Join(workDir, "metadata/md5-cache/dev-go/goreleaser-bin-2.17.1"))
			require.True(t, os.IsNotExist(err), "obsolete 2.17.1 md5-cache must be removed")
			_, err = os.Stat(filepath.Join(workDir, "metadata/md5-cache/dev-go/goreleaser-bin-2.20.2"))
			require.True(t, os.IsNotExist(err), "unauthoritative 2.20.2 md5-cache must not be fabricated")
			_, err = os.Stat(filepath.Join(workDir, "metadata/md5-cache/app-misc/other-pkg-1.0.0"))
			require.NoError(t, err, "unrelated package md5-cache must remain untouched")

			commitScript := resolveIssue200StepScript(t, wf, "Commit and push changes", map[string]string{
				"generated_tag": "v2.20.2",
			})
			commitOut, err := runBashScript(workDir, nil, commitScript)
			require.NoError(t, err, "Commit and push changes failed:\n%s", commitOut)

			// Verify remote publication via fresh clone
			inspectDir := cloneRemoteInspect(t, bareDir)
			_, err = os.Stat(filepath.Join(inspectDir, "dev-go/goreleaser-bin/goreleaser-bin-2.20.2.ebuild"))
			require.NoError(t, err, "2.20.2 ebuild must be published to remote")
			_, err = os.Stat(filepath.Join(inspectDir, "dev-go/goreleaser-bin/goreleaser-bin-2.17.1.ebuild"))
			require.True(t, os.IsNotExist(err), "2.17.1 ebuild deletion must be published to remote")
			_, err = os.Stat(filepath.Join(inspectDir, "metadata/md5-cache/dev-go/goreleaser-bin-2.17.1"))
			require.True(t, os.IsNotExist(err), "2.17.1 md5-cache deletion must be published to remote")
			otherBytes, err := os.ReadFile(filepath.Join(inspectDir, "metadata/md5-cache/app-misc/other-pkg-1.0.0"))
			require.NoError(t, err)
			require.Equal(t, "EAPI=8\nDESCRIPTION=Other\n_md5_=2222\n", string(otherBytes))
		})
	}
}

func TestIssue200_RetainedEbuildPreservesTrackedCacheOnCISkip(t *testing.T) {
	verifyG2Available(t)

	cfg := `Type Github AppImage Release
GithubProjectUrl https://github.com/example/app
EbuildName myapp-appimage
Category app-misc
Description Test AppImage
Homepage https://example.com
License MIT
Binary amd64=>myapp-${TAG}.AppImage > myapp.AppImage
`
	wf := generateIssue200Workflow(t, cfg)

	bareDir, workDir := initBareAndClone(t, map[string]string{
		"profiles/repo_name":                             "arrans-overlay",
		"profiles/eapi":                                  "8\n",
		"metadata/layout.conf":                           "masters = gentoo\ncache-formats = md5-dict\n",
		"app-misc/myapp-appimage/myapp-appimage-1.0.0.ebuild": "EAPI=8\ninherit xdg-utils\nDESCRIPTION=\"MyApp\"\nSLOT=\"0\"\n",
		"app-misc/myapp-appimage/Manifest":               "DIST myapp-1.0.0.AppImage 100 BLAKE2B 00 SHA512 00\n",
		"metadata/md5-cache/app-misc/myapp-appimage-1.0.0": "EAPI=8\nDESCRIPTION=MyApp\n_md5_=aaaa\n",
	})

	// Add 2.0.0 while retaining 1.0.0
	require.NoError(t, os.WriteFile(
		filepath.Join(workDir, "app-misc/myapp-appimage/myapp-appimage-2.0.0.ebuild"),
		[]byte("EAPI=8\ninherit xdg-utils\nDESCRIPTION=\"MyApp\"\nSLOT=\"0\"\n"),
		0o644,
	))

	cacheScript := resolveIssue200StepScript(t, wf, "Generate md5-cache", nil)
	cacheOut, err := runBashScript(workDir, []string{"G2_MASTER_REPOS="}, cacheScript)
	require.NoError(t, err, "Generate md5-cache failed:\n%s", cacheOut)

	// Retained 1.0.0 ebuild's tracked cache must be preserved
	preserved, err := os.ReadFile(filepath.Join(workDir, "metadata/md5-cache/app-misc/myapp-appimage-1.0.0"))
	require.NoError(t, err)
	require.Equal(t, "EAPI=8\nDESCRIPTION=MyApp\n_md5_=aaaa\n", string(preserved))

	commitScript := resolveIssue200StepScript(t, wf, "Commit and push changes", map[string]string{
		"generated_tag": "v2.0.0",
	})
	commitOut, err := runBashScript(workDir, nil, commitScript)
	require.NoError(t, err, "Commit and push failed:\n%s", commitOut)

	inspectDir := cloneRemoteInspect(t, bareDir)
	_, err = os.Stat(filepath.Join(inspectDir, "app-misc/myapp-appimage/myapp-appimage-2.0.0.ebuild"))
	require.NoError(t, err)
	preservedRemote, err := os.ReadFile(filepath.Join(inspectDir, "metadata/md5-cache/app-misc/myapp-appimage-1.0.0"))
	require.NoError(t, err)
	require.Equal(t, "EAPI=8\nDESCRIPTION=MyApp\n_md5_=aaaa\n", string(preservedRemote))
}

func TestIssue200_ConcurrentWritersBothPublishCleanly(t *testing.T) {
	cfgA := `Type Github Binary Release
GithubProjectUrl https://github.com/example/alpha
EbuildName alpha-bin
Category app-admin
Description Alpha
Homepage https://example.com
License MIT
Binary amd64=>alpha.tar.gz > alpha > alpha
`
	cfgB := `Type Github AppImage Release
GithubProjectUrl https://github.com/example/beta
EbuildName beta-appimage
Category app-misc
Description Beta
Homepage https://example.com
License MIT
Binary amd64=>beta.AppImage > beta.AppImage
`
	wfA := generateIssue200Workflow(t, cfgA)
	wfB := generateIssue200Workflow(t, cfgB)

	bareDir, workA := initBareAndClone(t, map[string]string{
		"profiles/repo_name":                          "arrans-overlay",
		"profiles/eapi":                               "8\n",
		"metadata/layout.conf":                        "cache-formats = md5-dict\n",
		"app-admin/alpha-bin/alpha-bin-1.0.0.ebuild":  "EAPI=8\nSLOT=\"0\"\n",
		"app-admin/alpha-bin/Manifest":                "DIST alpha-1.0.0.tar.gz 10 BLAKE2B 00 SHA512 00\n",
		"app-misc/beta-appimage/beta-appimage-1.0.0.ebuild": "EAPI=8\nSLOT=\"0\"\n",
		"app-misc/beta-appimage/Manifest":             "DIST beta-1.0.0.AppImage 10 BLAKE2B 00 SHA512 00\n",
		"metadata/md5-cache/app-misc/other-1.0.0":     "EAPI=8\n_md5_=9999\n",
	})

	workB := filepath.Join(t.TempDir(), "workB")
	runGit(t, filepath.Dir(workB), "clone", bareDir, workB)
	runGit(t, workB, "config", "user.name", "Beta Bot")
	runGit(t, workB, "config", "user.email", "beta@example.com")
	runGit(t, workB, "config", "commit.gpgsign", "false")

	// Both writers prepare disjoint updates from the exact same base commit
	require.NoError(t, os.WriteFile(filepath.Join(workA, "app-admin/alpha-bin/alpha-bin-1.1.0.ebuild"), []byte("EAPI=8\nSLOT=\"0\"\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(workA, "metadata/md5-cache/app-admin"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(workA, "metadata/md5-cache/app-admin/alpha-bin-1.1.0"), []byte("EAPI=8\n_md5_=aaaa\n"), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(workB, "app-misc/beta-appimage/beta-appimage-2.0.0.ebuild"), []byte("EAPI=8\nSLOT=\"0\"\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(workB, "metadata/md5-cache/app-misc"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(workB, "metadata/md5-cache/app-misc/beta-appimage-2.0.0"), []byte("EAPI=8\n_md5_=bbbb\n"), 0o644))

	scriptA := resolveIssue200StepScript(t, wfA, "Commit and push changes", map[string]string{"generated_tag": "v1.1.0"})
	scriptB := resolveIssue200StepScript(t, wfB, "Commit and push changes", map[string]string{"generated_tag": "v2.0.0"})

	var wg sync.WaitGroup
	var outA, outB string
	var errA, errB error
	env := []string{"G2_PUSH_RETRY_MAX_ATTEMPTS=5", "G2_PUSH_RETRY_BASE_SLEEP_SECS=0"}

	wg.Add(2)
	go func() {
		defer wg.Done()
		outA, errA = runBashScript(workA, env, scriptA)
	}()
	go func() {
		defer wg.Done()
		outB, errB = runBashScript(workB, env, scriptB)
	}()
	wg.Wait()

	require.NoError(t, errA, "Writer A failed:\n%s", outA)
	require.NoError(t, errB, "Writer B failed:\n%s", outB)

	inspectDir := cloneRemoteInspect(t, bareDir)
	_, err := os.Stat(filepath.Join(inspectDir, "app-admin/alpha-bin/alpha-bin-1.1.0.ebuild"))
	require.NoError(t, err, "Writer A ebuild must exist on remote")
	_, err = os.Stat(filepath.Join(inspectDir, "metadata/md5-cache/app-admin/alpha-bin-1.1.0"))
	require.NoError(t, err, "Writer A md5-cache must exist on remote")
	_, err = os.Stat(filepath.Join(inspectDir, "app-misc/beta-appimage/beta-appimage-2.0.0.ebuild"))
	require.NoError(t, err, "Writer B ebuild must exist on remote")
	_, err = os.Stat(filepath.Join(inspectDir, "metadata/md5-cache/app-misc/beta-appimage-2.0.0"))
	require.NoError(t, err, "Writer B md5-cache must exist on remote")
	_, err = os.Stat(filepath.Join(inspectDir, "metadata/md5-cache/app-misc/other-1.0.0"))
	require.NoError(t, err, "Unrelated package md5-cache must remain on remote")
}

func TestIssue200_NoChangeUpdateIsCleanNoOp(t *testing.T) {
	cfg := `Type Github Binary Release
GithubProjectUrl https://github.com/example/noop
EbuildName noop-bin
Category app-admin
Description Noop
Homepage https://example.com
License MIT
Binary amd64=>noop.tar.gz > noop > noop
`
	wf := generateIssue200Workflow(t, cfg)
	bareDir, workDir := initBareAndClone(t, map[string]string{
		"app-admin/noop-bin/noop-bin-1.0.0.ebuild": "EAPI=8\n",
	})
	headBefore := strings.TrimSpace(runGit(t, workDir, "rev-parse", "HEAD"))

	script := resolveIssue200StepScript(t, wf, "Commit and push changes", map[string]string{"generated_tag": "v1.0.0"})
	out, err := runBashScript(workDir, nil, script)
	require.NoError(t, err, "No-change update should succeed cleanly:\n%s", out)
	require.Contains(t, out, "No package changes to commit or push.")

	inspectDir := cloneRemoteInspect(t, bareDir)
	remoteHead := strings.TrimSpace(runGit(t, inspectDir, "rev-parse", "HEAD"))
	require.Equal(t, headBefore, remoteHead, "No-change update must not create a commit")
}

func TestIssue200_DirtyWorktreeFailsClosed(t *testing.T) {
	cfg := `Type Github Binary Release
GithubProjectUrl https://github.com/example/dirty
EbuildName dirty-bin
Category app-admin
Description Dirty
Homepage https://example.com
License MIT
Binary amd64=>dirty.tar.gz > dirty > dirty
`
	wf := generateIssue200Workflow(t, cfg)
	bareDir, workDir := initBareAndClone(t, map[string]string{
		"app-admin/dirty-bin/dirty-bin-1.0.0.ebuild": "EAPI=8\n",
		"README.md": "original\n",
	})
	headBefore := strings.TrimSpace(runGit(t, workDir, "rev-parse", "HEAD"))

	// Legitimate package update PLUS unexpected dirty tracked modification outside package scope
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "app-admin/dirty-bin/dirty-bin-1.1.0.ebuild"), []byte("EAPI=8\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "README.md"), []byte("unexpected dirty change\n"), 0o644))

	script := resolveIssue200StepScript(t, wf, "Commit and push changes", map[string]string{"generated_tag": "v1.1.0"})
	out, err := runBashScript(workDir, nil, script)
	require.Error(t, err, "Dirty worktree must fail closed")
	require.Contains(t, out, "unexpected unstaged or untracked changes")

	inspectDir := cloneRemoteInspect(t, bareDir)
	remoteHead := strings.TrimSpace(runGit(t, inspectDir, "rev-parse", "HEAD"))
	require.Equal(t, headBefore, remoteHead, "Dirty worktree must not publish to remote")
}

func TestIssue200_GenuinePushRejectionFailsImmediately(t *testing.T) {
	cfg := `Type Github Binary Release
GithubProjectUrl https://github.com/example/reject
EbuildName reject-bin
Category app-admin
Description Reject
Homepage https://example.com
License MIT
Binary amd64=>reject.tar.gz > reject > reject
`
	wf := generateIssue200Workflow(t, cfg)
	bareDir, workDir := initBareAndClone(t, map[string]string{
		"app-admin/reject-bin/reject-bin-1.0.0.ebuild": "EAPI=8\n",
	})

	// Install a bare remote pre-receive hook simulating a ruleset/permission rejection
	hookPath := filepath.Join(bareDir, "hooks", "pre-receive")
	require.NoError(t, os.MkdirAll(filepath.Dir(hookPath), 0o755))
	require.NoError(t, os.WriteFile(hookPath, []byte("#!/bin/sh\necho 'error: GH006: Protected branch update failed' >&2\nexit 1\n"), 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(workDir, "app-admin/reject-bin/reject-bin-1.1.0.ebuild"), []byte("EAPI=8\n"), 0o644))

	script := resolveIssue200StepScript(t, wf, "Commit and push changes", map[string]string{"generated_tag": "v1.1.0"})
	out, err := runBashScript(workDir, []string{"G2_PUSH_RETRY_MAX_ATTEMPTS=5", "G2_PUSH_RETRY_BASE_SLEEP_SECS=0"}, script)
	require.Error(t, err, "Genuine push rejection must fail nonzero")
	require.Contains(t, out, "non-retryable error")
	require.NotContains(t, out, "retrying in", "Non-retryable push rejection must not retry")
}

func TestIssue200_RebaseConflictFailsAndAbortsCleanly(t *testing.T) {
	cfg := `Type Github Binary Release
GithubProjectUrl https://github.com/example/conflict
EbuildName conflict-bin
Category app-admin
Description Conflict
Homepage https://example.com
License MIT
Binary amd64=>conflict.tar.gz > conflict > conflict
`
	wf := generateIssue200Workflow(t, cfg)
	bareDir, workA := initBareAndClone(t, map[string]string{
		"app-admin/conflict-bin/Manifest": "DIST conflict-1.0.0.tar.gz 10 BLAKE2B 00 SHA512 00\n",
	})

	// Another writer pushes a concurrent change to the same package Manifest
	workB := filepath.Join(t.TempDir(), "workB")
	runGit(t, filepath.Dir(workB), "clone", bareDir, workB)
	runGit(t, workB, "config", "user.name", "Other Bot")
	runGit(t, workB, "config", "user.email", "other@example.com")
	runGit(t, workB, "config", "commit.gpgsign", "false")
	require.NoError(t, os.WriteFile(filepath.Join(workB, "app-admin/conflict-bin/Manifest"), []byte("DIST conflict-2.0.0.tar.gz 20 BLAKE2B 00 SHA512 00\n"), 0o644))
	runGit(t, workB, "add", "-A")
	runGit(t, workB, "commit", "-m", "Concurrent 2.0.0")
	runGit(t, workB, "push", "origin", "main")

	// WorkA tries to publish 1.5.0 modifying the same package
	require.NoError(t, os.WriteFile(filepath.Join(workA, "app-admin/conflict-bin/Manifest"), []byte("DIST conflict-1.5.0.tar.gz 15 BLAKE2B 00 SHA512 00\n"), 0o644))
	script := resolveIssue200StepScript(t, wf, "Commit and push changes", map[string]string{"generated_tag": "v1.5.0"})
	out, err := runBashScript(workA, []string{"G2_PUSH_RETRY_MAX_ATTEMPTS=3", "G2_PUSH_RETRY_BASE_SLEEP_SECS=0"}, script)
	require.Error(t, err, "Concurrent same-package update must fail closed")
	require.Contains(t, out, "refusing to replay potentially stale package state")
}

func TestIssue200_ExhaustedRetryLimitFailsNonzero(t *testing.T) {
	cfg := `Type Github Binary Release
GithubProjectUrl https://github.com/example/exhaust
EbuildName exhaust-bin
Category app-admin
Description Exhaust
Homepage https://example.com
License MIT
Binary amd64=>exhaust.tar.gz > exhaust > exhaust
`
	wf := generateIssue200Workflow(t, cfg)
	bareDir, workA := initBareAndClone(t, map[string]string{
		"app-admin/exhaust-bin/exhaust-bin-1.0.0.ebuild": "EAPI=8\n",
	})

	// Second writer clone that advances main on an unrelated package before each push attempt
	workB := filepath.Join(t.TempDir(), "workB")
	runGit(t, filepath.Dir(workB), "clone", bareDir, workB)
	runGit(t, workB, "config", "user.name", "Racer Bot")
	runGit(t, workB, "config", "user.email", "racer@example.com")
	runGit(t, workB, "config", "commit.gpgsign", "false")

	realGit, err := exec.LookPath("git")
	require.NoError(t, err)
	binDir := t.TempDir()
	counterFile := filepath.Join(t.TempDir(), "count")
	gitWrapper := fmt.Sprintf(`#!/bin/bash
if [[ "$1" == "push" ]]; then
  n=0
  if [[ -f %q ]]; then n=$(cat %q); fi
  n=$((n + 1))
  echo "$n" > %q
  (
    cd %q || exit 1
    %q pull --rebase origin main >/dev/null 2>&1 || true
    echo "tick $n" > "unrelated-$n.txt"
    %q add "unrelated-$n.txt"
    %q commit -m "concurrent $n" >/dev/null 2>&1
    %q push origin main >/dev/null 2>&1
  )
fi
exec %q "$@"
`, counterFile, counterFile, counterFile, workB, realGit, realGit, realGit, realGit, realGit)
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "git"), []byte(gitWrapper), 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(workA, "app-admin/exhaust-bin/exhaust-bin-1.1.0.ebuild"), []byte("EAPI=8\n"), 0o644))
	script := resolveIssue200StepScript(t, wf, "Commit and push changes", map[string]string{"generated_tag": "v1.1.0"})
	out, err := runBashScript(workA, []string{
		"PATH=" + binDir + ":" + os.Getenv("PATH"),
		"G2_PUSH_RETRY_MAX_ATTEMPTS=2",
		"G2_PUSH_RETRY_BASE_SLEEP_SECS=0",
	}, script)
	require.Error(t, err, "Exhausted retry limit must fail nonzero")
	require.Contains(t, out, "failed to push package update after 2 attempts")
}
