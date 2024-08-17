package main

import (
	"flag"
	"fmt"
	"github.com/arran4/arrans_overlay_workflow_builder"
	"log"
	"os"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type MainArgConfig struct {
	Version string
	Commit  string
	Date    string
}

func main() {
	fs := flag.NewFlagSet("", flag.ExitOnError)
	config := &MainArgConfig{
		Version: version,
		Commit:  commit,
		Date:    date,
	}
	if err := fs.Parse(os.Args); err != nil {
		log.Printf("Flag parse error: %s", err)
		os.Exit(-1)
		return
	}
	if fs.NArg() <= 1 {
		log.Printf("Please specify an argument, try -help for help")
		os.Exit(-1)
		return
	}
	switch fs.Arg(1) {
	case "generate":
		if err := config.cmdGenerate(fs.Args()[2:]); err != nil {
			log.Printf("generate error: %s", err)
			os.Exit(-1)
			return
		}
	case "oneshot":
		if err := config.cmdOneshot(fs.Args()[2:]); err != nil {
			log.Printf("oneshot error: %s", err)
			os.Exit(-1)
			return
		}
	case "config":
		if err := config.cmdConfig(fs.Args()[2:]); err != nil {
			log.Printf("config error: %s", err)
			os.Exit(-1)
			return
		}
	case "version":
		if err := config.printVersion(); err != nil {
			log.Printf("config error: %s", err)
			os.Exit(-1)
			return
		}
	default:
		log.Printf("Unknown command %s", fs.Arg(1))
		log.Printf("Try %s for %s", "generate", "commands to generate github action workflows output")
		log.Printf("Try %s for %s", "oneshot", "does both the config and generate steps")
		log.Printf("Try %s for %s", "config", "commands to view results and content")
		log.Printf("Try %s for %s", "version", "commands to view version information")
		os.Exit(-1)
	}
}

type CmdGenerateArgConfig struct {
	*MainArgConfig
}

func (mac *MainArgConfig) cmdGenerate(args []string) error {
	fs := flag.NewFlagSet("", flag.ExitOnError)
	config := &CmdGenerateArgConfig{
		MainArgConfig: mac,
	}
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}
	switch fs.Arg(0) {
	case "workflows":
		if err := config.cmdGenerateGithubWorkflows(fs.Args()[1:]); err != nil {
			return fmt.Errorf("github appimage: %w", err)
		}
	default:
		log.Printf("Unknown command %s", fs.Arg(0))
		log.Printf("Try %s for %s", "workflows", "Generate workflows from a configfile.")
		os.Exit(-1)
	}
	return nil
}

type CmdGenerateGithubWorkflowsArgConfig struct {
	*CmdGenerateArgConfig
	InputFile *string
	OutputDir *string
}

func (mac *CmdGenerateArgConfig) cmdGenerateGithubWorkflows(args []string) error {
	config := &CmdGenerateGithubWorkflowsArgConfig{
		CmdGenerateArgConfig: mac,
	}
	fs := flag.NewFlagSet("", flag.ExitOnError)
	config.InputFile = fs.String("input-file", "input.config", "The input with config")
	config.OutputDir = fs.String("output-dir", "./output", "Directory to output workflows")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}
	switch fs.Arg(0) {
	case "":
		if config.InputFile == nil || *config.InputFile == "" {
			return fmt.Errorf("input file argument missing")
		}
		return arrans_overlay_workflow_builder.GenerateGithubWorkflows(*config.InputFile, *config.OutputDir, config.Version)
	default:
		log.Printf("Unknown command %s", fs.Arg(0))
		os.Exit(-1)
	}
	return nil
}

type CmdConfigArgConfig struct {
	*MainArgConfig
}

func (mac *MainArgConfig) cmdConfig(args []string) error {
	fs := flag.NewFlagSet("", flag.ExitOnError)
	config := &CmdConfigArgConfig{
		MainArgConfig: mac,
	}
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}
	switch fs.Arg(0) {
	case "add":
		if err := config.cmdConfigAdd(fs.Args()[1:]); err != nil {
			return fmt.Errorf("config add: %w", err)
		}
	case "view":
		if err := config.cmdConfigView(fs.Args()[1:]); err != nil {
			return fmt.Errorf("config view: %w", err)
		}
	default:
		log.Printf("Unknown command %s", fs.Arg(0))
		log.Printf("Try %s for %s", "add", "Adds an configuration to a configuration file.")
		log.Printf("Try %s for %s", "view", "Provides a bunch of options for viewing.")
		os.Exit(-1)
	}
	return nil
}

type CmdConfigAddArgConfig struct {
	*CmdConfigArgConfig
}

func (mac *CmdConfigArgConfig) cmdConfigAdd(args []string) error {
	config := &CmdConfigAddArgConfig{
		CmdConfigArgConfig: mac,
	}
	fs := flag.NewFlagSet("", flag.ExitOnError)
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}
	switch fs.Arg(0) {
	case "github-release-appimage":
		if err := config.cmdConfigAddAppImageGithubReleases(fs.Args()[1:]); err != nil {
			return fmt.Errorf("config add: %w", err)
		}
	case "github-release-binary":
		if err := config.cmdConfigAddBinaryGithubReleases(fs.Args()[1:]); err != nil {
			return fmt.Errorf("config add: %w", err)
		}
	default:
		log.Printf("Unknown command %s", fs.Arg(0))
		log.Printf("Try %s for %s", "github-release-appimage", "To generate a config file from a github release with semantic version for AppImages.")
		log.Printf("Try %s for %s", "github-release-binary", "To generate a config file from a github release with semantic version for Binary Releases.")
		os.Exit(-1)
	}
	return nil
}

type CmdConfigAddAppImageGithubReleasesArgConfig struct {
	*CmdConfigAddArgConfig
	GithubUrl          *string
	ConfigFile         *string
	SelectedVersionTag *string
	TagPrefix          *string
}

func (mac *CmdConfigAddArgConfig) cmdConfigAddAppImageGithubReleases(args []string) error {
	config := &CmdConfigAddAppImageGithubReleasesArgConfig{
		CmdConfigAddArgConfig: mac,
	}
	fs := flag.NewFlagSet("", flag.ExitOnError)
	config.ConfigFile = fs.String("to", "input.config", "The input with config")
	config.GithubUrl = fs.String("github-url", "https://github.com/owner/repo/", "The github URL to add")
	config.SelectedVersionTag = fs.String("version-tag", "", "Version / tag override")
	config.TagPrefix = fs.String("tag-prefix", "", "Tag prefix for app to select on and remove")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}
	switch fs.Arg(0) {
	case "":
		if config.ConfigFile == nil || *config.ConfigFile == "" {
			return fmt.Errorf("config file to modify argument missing")
		}
		if config.GithubUrl == nil || *config.GithubUrl == "" {
			return fmt.Errorf("github URL to add is missing")
		}
		return arrans_overlay_workflow_builder.ConfigAddAppImageGithubReleases(*config.ConfigFile, *config.GithubUrl, *config.SelectedVersionTag, *config.TagPrefix)
	default:
		log.Printf("Unknown command %s", fs.Arg(0))
		log.Printf("Try %s for %s", "github-appimage", "Adds an configuration to a configuration file.")
		os.Exit(-1)
	}
	return nil
}

type CmdConfigAddBinaryGithubReleasesArgConfig struct {
	*CmdConfigAddArgConfig
	GithubUrl          *string
	ConfigFile         *string
	SelectedVersionTag *string
	TagPrefix          *string
}

func (mac *CmdConfigAddArgConfig) cmdConfigAddBinaryGithubReleases(args []string) error {
	config := &CmdConfigAddBinaryGithubReleasesArgConfig{
		CmdConfigAddArgConfig: mac,
	}
	fs := flag.NewFlagSet("", flag.ExitOnError)
	config.ConfigFile = fs.String("to", "input.config", "The input with config")
	config.GithubUrl = fs.String("github-url", "https://github.com/owner/repo/", "The github URL to add")
	config.SelectedVersionTag = fs.String("version-tag", "", "Version / tag override")
	config.TagPrefix = fs.String("tag-prefix", "", "Tag prefix for app to select on and remove")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}
	switch fs.Arg(0) {
	case "":
		if config.ConfigFile == nil || *config.ConfigFile == "" {
			return fmt.Errorf("config file to modify argument missing")
		}
		if config.GithubUrl == nil || *config.GithubUrl == "" {
			return fmt.Errorf("github URL to add is missing")
		}
		return arrans_overlay_workflow_builder.ConfigAddBinaryGithubReleases(*config.ConfigFile, *config.GithubUrl, *config.SelectedVersionTag, *config.TagPrefix)
	default:
		log.Printf("Unknown command %s", fs.Arg(0))
		log.Printf("Try %s for %s", "github-binary", "Adds an configuration to a configuration file.")
		os.Exit(-1)
	}
	return nil
}

type CmdConfigViewArgConfig struct {
	*CmdConfigArgConfig
}

func (mac *CmdConfigArgConfig) cmdConfigView(args []string) error {
	config := &CmdConfigViewArgConfig{
		CmdConfigArgConfig: mac,
	}
	fs := flag.NewFlagSet("", flag.ExitOnError)
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}
	switch fs.Arg(0) {
	case "github-release-appimage":
		if err := config.cmdConfigViewAppImageGithubReleases(fs.Args()[1:]); err != nil {
			return fmt.Errorf("config view: %w", err)
		}
	case "github-release-binary":
		if err := config.cmdConfigViewBinaryGithubReleases(fs.Args()[1:]); err != nil {
			return fmt.Errorf("config view: %w", err)
		}
	default:
		log.Printf("Unknown command %s", fs.Arg(0))
		os.Exit(-1)
	}
	return nil
}

type CmdConfigViewAppImageGithubReleasesArgConfig struct {
	*CmdConfigViewArgConfig
	GithubUrl          *string
	SelectedVersionTag *string
	TagPrefix          *string
}

func (mac *CmdConfigViewArgConfig) cmdConfigViewAppImageGithubReleases(args []string) error {
	config := &CmdConfigViewAppImageGithubReleasesArgConfig{
		CmdConfigViewArgConfig: mac,
	}
	fs := flag.NewFlagSet("", flag.ExitOnError)
	config.GithubUrl = fs.String("github-url", "https://github.com/owner/repo/", "The github URL to view")
	config.SelectedVersionTag = fs.String("version-tag", "", "Version / tag override")
	config.TagPrefix = fs.String("tag-prefix", "", "Tag prefix for app to select on and remove")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}
	switch fs.Arg(0) {
	case "":
		if config.GithubUrl == nil || *config.GithubUrl == "" {
			return fmt.Errorf("github URL to view is missing")
		}
		return arrans_overlay_workflow_builder.ConfigViewAppImageGithubReleases(*config.GithubUrl, *config.SelectedVersionTag, *config.TagPrefix)
	default:
		log.Printf("Unknown command %s", fs.Arg(0))
		log.Printf("Try %s for %s", "github-appimage", "Views an addition to a configuration file for a particular query.")
		os.Exit(-1)
	}
	return nil
}

type CmdConfigViewBinaryGithubReleasesArgConfig struct {
	*CmdConfigViewArgConfig
	GithubUrl          *string
	SelectedVersionTag *string
	TagPrefix          *string
}

func (mac *CmdConfigViewArgConfig) cmdConfigViewBinaryGithubReleases(args []string) error {
	config := &CmdConfigViewBinaryGithubReleasesArgConfig{
		CmdConfigViewArgConfig: mac,
	}
	fs := flag.NewFlagSet("", flag.ExitOnError)
	config.GithubUrl = fs.String("github-url", "https://github.com/owner/repo/", "The github URL to view")
	config.SelectedVersionTag = fs.String("version-tag", "", "Version / tag override")
	config.TagPrefix = fs.String("tag-prefix", "", "Tag prefix for app to select on and remove")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}
	switch fs.Arg(0) {
	case "":
		if config.GithubUrl == nil || *config.GithubUrl == "" {
			return fmt.Errorf("github URL to view is missing")
		}
		return arrans_overlay_workflow_builder.ConfigViewBinaryGithubReleases(*config.GithubUrl, *config.SelectedVersionTag, *config.TagPrefix)
	default:
		log.Printf("Unknown command %s", fs.Arg(0))
		log.Printf("Try %s for %s", "github-binary", "Views an addition to a configuration file for a particular query.")
		os.Exit(-1)
	}
	return nil
}

type CmdOneshotArgConfig struct {
	*MainArgConfig
}

func (mac *MainArgConfig) cmdOneshot(args []string) error {
	fs := flag.NewFlagSet("", flag.ExitOnError)
	config := &CmdOneshotArgConfig{
		MainArgConfig: mac,
	}
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}
	switch fs.Arg(0) {
	case "github-release-appimage":
		if err := config.cmdOneshotGithubReleaseAppImage(fs.Args()[1:]); err != nil {
			return fmt.Errorf("github appimage: %w", err)
		}
	case "github-release-binary":
		if err := config.cmdOneshotGithubReleaseBinary(fs.Args()[1:]); err != nil {
			return fmt.Errorf("github binary: %w", err)
		}
	default:
		log.Printf("Unknown command %s", fs.Arg(0))
		log.Printf("Try %s for %s", "workflows", "Oneshot workflows from a configfile.")
		os.Exit(-1)
	}
	return nil
}

func (mac *MainArgConfig) printVersion() error {
	fmt.Printf("Arrans Overlay Workflow Builder %s, commit %s, built at %s", version, commit, date)
	return nil
}

type CmdOneshotGithubReleaseAppImageArgConfig struct {
	*CmdOneshotArgConfig
	GithubUrl          *string
	SelectedVersionTag *string
	TagPrefix          *string
	OutputDir          *string
}

func (mac *CmdOneshotArgConfig) cmdOneshotGithubReleaseAppImage(args []string) error {
	config := &CmdOneshotGithubReleaseAppImageArgConfig{
		CmdOneshotArgConfig: mac,
	}
	fs := flag.NewFlagSet("", flag.ExitOnError)
	config.GithubUrl = fs.String("github-url", "https://github.com/owner/repo/", "The github URL to view")
	config.SelectedVersionTag = fs.String("version-tag", "", "Version / tag override")
	config.TagPrefix = fs.String("tag-prefix", "", "Tag prefix for app to select on and remove")
	config.OutputDir = fs.String("output-dir", "./output", "Directory to output workflows")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}
	switch fs.Arg(0) {
	case "":
		if config.GithubUrl == nil || *config.GithubUrl == "" {
			return fmt.Errorf("github URL to view is missing")
		}
		return arrans_overlay_workflow_builder.CmdOneshotGithubReleaseAppImage(*config.GithubUrl, *config.SelectedVersionTag, *config.TagPrefix, *config.OutputDir, config.Version)
	default:
		log.Printf("Unknown command %s", fs.Arg(0))
		os.Exit(-1)
	}
	return nil
}

type CmdOneshotGithubReleaseBinaryArgConfig struct {
	*CmdOneshotArgConfig
	GithubUrl          *string
	SelectedVersionTag *string
	TagPrefix          *string
	OutputDir          *string
}

func (mac *CmdOneshotArgConfig) cmdOneshotGithubReleaseBinary(args []string) error {
	config := &CmdOneshotGithubReleaseBinaryArgConfig{
		CmdOneshotArgConfig: mac,
	}
	fs := flag.NewFlagSet("", flag.ExitOnError)
	config.GithubUrl = fs.String("github-url", "https://github.com/owner/repo/", "The github URL to view")
	config.SelectedVersionTag = fs.String("version-tag", "", "Version / tag override")
	config.TagPrefix = fs.String("tag-prefix", "", "Tag prefix for app to select on and remove")
	config.OutputDir = fs.String("output-dir", "./output", "Directory to output workflows")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}
	switch fs.Arg(0) {
	case "":
		if config.GithubUrl == nil || *config.GithubUrl == "" {
			return fmt.Errorf("github URL to view is missing")
		}
		return arrans_overlay_workflow_builder.CmdOneshotGithubReleaseBinary(*config.GithubUrl, *config.SelectedVersionTag, *config.TagPrefix, *config.OutputDir, config.Version)
	default:
		log.Printf("Unknown command %s", fs.Arg(0))
		os.Exit(-1)
	}
	return nil
}
