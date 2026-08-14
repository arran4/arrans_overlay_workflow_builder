package arrans_overlay_workflow_builder

import (
	"io"
	"log"
	"os"
)

func ConfigAddCmakeSourceGithubReleases(configFile string, githubUrl string, selectedVersionTag string, tagPrefix string) error {
	repoName, ic, versions, tags, releaseInfo, _, err := NewInputConfigurationFromRepo(githubUrl, selectedVersionTag, tagPrefix, "", "Github Cmake Release")
	if err != nil {
		return err
	}
	_ = repoName
	_ = versions
	_ = tags
	_ = releaseInfo

	return AppendToConfigurationFile(configFile, ic)
}

func ConfigViewCmakeSourceGithubReleases(githubUrl string, selectedVersionTag string, tagPrefix string) error {
	repoName, ic, versions, tags, releaseInfo, _, err := NewInputConfigurationFromRepo(githubUrl, selectedVersionTag, tagPrefix, "", "Github Cmake Release")
	if err != nil {
		return err
	}
	_ = repoName
	_ = versions
	_ = tags
	_ = releaseInfo

	_, err = io.WriteString(os.Stdout, ic.String())
	return err
}

func CmdOneshotGithubReleaseCmakeSource(githubUrl string, selectedVersionTag string, tagPrefix string, outputDir string, version string, opts ...any) error {
	repoName, ic, versions, tags, releaseInfo, _, err := NewInputConfigurationFromRepo(githubUrl, selectedVersionTag, tagPrefix, "", "Github Cmake Release")
	if err != nil {
		return err
	}
	_ = repoName
	_ = versions
	_ = tags
	_ = releaseInfo

	log.Printf("Generating from repo")
	return GenerateGithubWorkflowsFromInputConfigs("oneshot", []*InputConfig{ic}, outputDir, version, opts...)
}
