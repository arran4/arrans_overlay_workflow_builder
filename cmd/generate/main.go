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
