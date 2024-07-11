package main

import (
	"flag"
	"fmt"
	"github.com/arran4/arrans_overlay_workflow_builder"
	"log"
	"os"
)

type MainArgConfig struct {
}

func main() {
	fs := flag.NewFlagSet("", flag.ExitOnError)
	config := &MainArgConfig{}
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
	case "config":
		if err := config.cmdConfig(fs.Args()[2:]); err != nil {
			log.Printf("config error: %s", err)
			os.Exit(-1)
			return
		}
	default:
		log.Printf("Unknown command %s", fs.Arg(1))
		log.Printf("Try %s for %s", "generate", "commands to generate github action workflows output")
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
	case "github-appimage":
		if err := config.cmdGenerateGithubAppImage(fs.Args()[1:]); err != nil {
			return fmt.Errorf("github appimage: %w", err)
		}
	default:
		log.Printf("Unknown command %s", fs.Arg(0))
		log.Printf("Try %s for %s", "github-appimage", "a command specific to generating appimage ebuilds from github repos that use github releases to release appimages.")
		os.Exit(-1)
	}
	return nil
}

type CmdGenerateGithubAppImageArgConfig struct {
	*CmdGenerateArgConfig
	InputFile *string
}

func (mac *CmdGenerateArgConfig) cmdGenerateGithubAppImage(args []string) error {
	config := &CmdGenerateGithubAppImageArgConfig{
		CmdGenerateArgConfig: mac,
	}
	fs := flag.NewFlagSet("", flag.ExitOnError)
	config.InputFile = fs.String("input-file", "input.config", "The input with config")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}
	switch fs.Arg(0) {
	case "":
		if config.InputFile == nil || *config.InputFile == "" {
			return fmt.Errorf("input file argument missing")
		}
		return arrans_overlay_workflow_builder.GenerateGithubAppImage(*config.InputFile)
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
	default:
		log.Printf("Unknown command %s", fs.Arg(0))
		os.Exit(-1)
	}
	return nil
}

type CmdConfigAddAppImageGithubReleasesArgConfig struct {
	*CmdConfigAddArgConfig
	GithubUrl  *string
	ConfigFile *string
}

func (mac *CmdConfigAddArgConfig) cmdConfigAddAppImageGithubReleases(args []string) error {
	config := &CmdConfigAddAppImageGithubReleasesArgConfig{
		CmdConfigAddArgConfig: mac,
	}
	fs := flag.NewFlagSet("", flag.ExitOnError)
	config.ConfigFile = fs.String("to", "input.config", "The input with config")
	config.GithubUrl = fs.String("github-url", "https://github.com/owner/repo/", "The github URL to add")
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
		return arrans_overlay_workflow_builder.ConfigAddAppImageGithubReleases(*config.ConfigFile, *config.GithubUrl)
	default:
		log.Printf("Unknown command %s", fs.Arg(0))
		log.Printf("Try %s for %s", "github-appimage", "Adds an configuration to a configuration file.")
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
	default:
		log.Printf("Unknown command %s", fs.Arg(0))
		os.Exit(-1)
	}
	return nil
}

type CmdConfigViewAppImageGithubReleasesArgConfig struct {
	*CmdConfigViewArgConfig
	GithubUrl *string
}

func (mac *CmdConfigViewArgConfig) cmdConfigViewAppImageGithubReleases(args []string) error {
	config := &CmdConfigViewAppImageGithubReleasesArgConfig{
		CmdConfigViewArgConfig: mac,
	}
	fs := flag.NewFlagSet("", flag.ExitOnError)
	config.GithubUrl = fs.String("github-url", "https://github.com/owner/repo/", "The github URL to view")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}
	switch fs.Arg(0) {
	case "":
		if config.GithubUrl == nil || *config.GithubUrl == "" {
			return fmt.Errorf("github URL to view is missing")
		}
		return arrans_overlay_workflow_builder.ConfigViewAppImageGithubReleases(*config.GithubUrl)
	default:
		log.Printf("Unknown command %s", fs.Arg(0))
		log.Printf("Try %s for %s", "github-appimage", "Views an addition to a configuration file for a particular query.")
		os.Exit(-1)
	}
	return nil
}
