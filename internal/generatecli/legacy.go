// Package generatecli owns the legacy CLI entrypoint and its command handlers.
// Keeping this handwritten adapter outside cmd/generate allows the command
// launcher to be regenerated or replaced without discarding CLI behavior.
package generatecli

import (
	"flag"
	"fmt"
	"github.com/arran4/arrans_overlay_workflow_builder"
	"log"
	"os"
	"strings"
	"time"
)

type MainArgConfig struct {
	Version                      string
	Commit                       string
	Date                         string
	GenerateMd5Cache             bool
	GenerateOverlay              bool
	OverlayRepoName              *string
	UpsertOverlayRepoName        bool
	GeneratedEbuildGentooAuthors *bool
	GeneratedEbuildSourceLicense *string
}

func valueOrDefault(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func parseBoolValue(s string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "t", "1", "yes", "on":
		return true, true
	case "false", "f", "0", "no", "off":
		return false, true
	default:
		return false, false
	}
}

func isKnownCommand(s string) bool {
	switch s {
	case "generate", "oneshot", "config", "version":
		return true
	default:
		return false
	}
}

// Run executes the existing command-line interface using caller-supplied build metadata.
func Run(version, commit, date string) {
	fs := flag.NewFlagSet("", flag.ExitOnError)
	config := &MainArgConfig{
		Version: version,
		Commit:  commit,
		Date:    date,
	}
	var globalGenerateMd5Cache bool
	// A workaround for global flags being parsed before the command is parsed
	var globalGenerateOverlay bool
	var globalOverlayRepoName string
	var globalOverlayRepoNameSet bool
	var globalUpsertOverlayRepoName bool
	var globalGeneratedEbuildGentooAuthors *bool
	var globalGeneratedEbuildSourceLicense *string

	var globalGentooAuthorsVal *bool
	var globalGenEbuildGentooAuthorsVal *bool
	var globalSourceLicenseVal *string
	var globalGenEbuildSourceLicenseVal *string

	var newArgs []string
	if len(os.Args) > 0 {
		newArgs = append(newArgs, os.Args[0])
	}
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if arg == "--" {
			newArgs = append(newArgs, os.Args[i+1:]...)
			break
		}
		if !strings.HasPrefix(arg, "-") {
			newArgs = append(newArgs, os.Args[i:]...)
			break
		}
		switch {
		case arg == "--generate-md5-cache" || arg == "-generate-md5-cache":
			globalGenerateMd5Cache = true
		case arg == "--generate-overlay" || arg == "-generate-overlay":
			globalGenerateOverlay = true
		case arg == "--upsert-overlay-repo-name" || arg == "-upsert-overlay-repo-name":
			globalUpsertOverlayRepoName = true
		case arg == "--overlay-repo-name" || arg == "-overlay-repo-name":
			if i+1 < len(os.Args) && !strings.HasPrefix(os.Args[i+1], "-") && !isKnownCommand(os.Args[i+1]) {
				globalOverlayRepoName = os.Args[i+1]
				globalOverlayRepoNameSet = true
				if globalOverlayRepoName == "" {
					log.Fatalf("explicit empty canonical identity is invalid input")
				}
				i++
			} else {
				log.Fatalf("missing required value for flag %s", arg)
			}
		case strings.HasPrefix(arg, "--gentoo-authors=") || strings.HasPrefix(arg, "-gentoo-authors="):
			parts := strings.SplitN(arg, "=", 2)
			b, ok := parseBoolValue(parts[1])
			if !ok {
				log.Fatalf("invalid boolean value for flag %s: %s", parts[0], parts[1])
			}
			if globalGentooAuthorsVal != nil && *globalGentooAuthorsVal != b {
				log.Fatalf("conflicting flags: multiple %s with conflicting values", parts[0])
			}
			globalGentooAuthorsVal = &b
		case strings.HasPrefix(arg, "--generated-ebuild-gentoo-authors=") || strings.HasPrefix(arg, "-generated-ebuild-gentoo-authors="):
			parts := strings.SplitN(arg, "=", 2)
			b, ok := parseBoolValue(parts[1])
			if !ok {
				log.Fatalf("invalid boolean value for flag %s: %s", parts[0], parts[1])
			}
			if globalGenEbuildGentooAuthorsVal != nil && *globalGenEbuildGentooAuthorsVal != b {
				log.Fatalf("conflicting flags: multiple %s with conflicting values", parts[0])
			}
			globalGenEbuildGentooAuthorsVal = &b
		case arg == "--gentoo-authors" || arg == "-gentoo-authors":
			var b bool
			if i+1 < len(os.Args) {
				if parsed, ok := parseBoolValue(os.Args[i+1]); ok {
					b = parsed
					i++
				} else {
					b = true
				}
			} else {
				b = true
			}
			if globalGentooAuthorsVal != nil && *globalGentooAuthorsVal != b {
				log.Fatalf("conflicting flags: multiple %s with conflicting values", arg)
			}
			globalGentooAuthorsVal = &b
		case arg == "--generated-ebuild-gentoo-authors" || arg == "-generated-ebuild-gentoo-authors":
			var b bool
			if i+1 < len(os.Args) {
				if parsed, ok := parseBoolValue(os.Args[i+1]); ok {
					b = parsed
					i++
				} else {
					b = true
				}
			} else {
				b = true
			}
			if globalGenEbuildGentooAuthorsVal != nil && *globalGenEbuildGentooAuthorsVal != b {
				log.Fatalf("conflicting flags: multiple %s with conflicting values", arg)
			}
			globalGenEbuildGentooAuthorsVal = &b
		case strings.HasPrefix(arg, "--source-license=") || strings.HasPrefix(arg, "-source-license="):
			parts := strings.SplitN(arg, "=", 2)
			if parts[1] == "" {
				log.Fatalf("missing required value for flag %s", parts[0])
			}
			val := parts[1]
			if globalSourceLicenseVal != nil && *globalSourceLicenseVal != val {
				log.Fatalf("conflicting flags: multiple %s with conflicting values", parts[0])
			}
			globalSourceLicenseVal = &val
		case strings.HasPrefix(arg, "--generated-ebuild-source-license=") || strings.HasPrefix(arg, "-generated-ebuild-source-license="):
			parts := strings.SplitN(arg, "=", 2)
			if parts[1] == "" {
				log.Fatalf("missing required value for flag %s", parts[0])
			}
			val := parts[1]
			if globalGenEbuildSourceLicenseVal != nil && *globalGenEbuildSourceLicenseVal != val {
				log.Fatalf("conflicting flags: multiple %s with conflicting values", parts[0])
			}
			globalGenEbuildSourceLicenseVal = &val
		case arg == "--source-license" || arg == "-source-license":
			if i+1 < len(os.Args) && !strings.HasPrefix(os.Args[i+1], "-") && !isKnownCommand(os.Args[i+1]) {
				val := os.Args[i+1]
				if globalSourceLicenseVal != nil && *globalSourceLicenseVal != val {
					log.Fatalf("conflicting flags: multiple %s with conflicting values", arg)
				}
				globalSourceLicenseVal = &val
				i++
			} else {
				log.Fatalf("missing required value for flag %s", arg)
			}
		case arg == "--generated-ebuild-source-license" || arg == "-generated-ebuild-source-license":
			if i+1 < len(os.Args) && !strings.HasPrefix(os.Args[i+1], "-") && !isKnownCommand(os.Args[i+1]) {
				val := os.Args[i+1]
				if globalGenEbuildSourceLicenseVal != nil && *globalGenEbuildSourceLicenseVal != val {
					log.Fatalf("conflicting flags: multiple %s with conflicting values", arg)
				}
				globalGenEbuildSourceLicenseVal = &val
				i++
			} else {
				log.Fatalf("missing required value for flag %s", arg)
			}
		default:
			newArgs = append(newArgs, arg)
		}
	}
	os.Args = newArgs

	if globalGentooAuthorsVal != nil && globalGenEbuildGentooAuthorsVal != nil {
		if *globalGentooAuthorsVal != *globalGenEbuildGentooAuthorsVal {
			log.Fatalf("conflicting flags: --gentoo-authors=%t and --generated-ebuild-gentoo-authors=%t", *globalGentooAuthorsVal, *globalGenEbuildGentooAuthorsVal)
		}
		globalGeneratedEbuildGentooAuthors = globalGentooAuthorsVal
	} else if globalGentooAuthorsVal != nil {
		globalGeneratedEbuildGentooAuthors = globalGentooAuthorsVal
	} else if globalGenEbuildGentooAuthorsVal != nil {
		globalGeneratedEbuildGentooAuthors = globalGenEbuildGentooAuthorsVal
	}

	if globalSourceLicenseVal != nil && globalGenEbuildSourceLicenseVal != nil {
		if *globalSourceLicenseVal != *globalGenEbuildSourceLicenseVal {
			log.Fatalf("conflicting flags: --source-license=%q and --generated-ebuild-source-license=%q", *globalSourceLicenseVal, *globalGenEbuildSourceLicenseVal)
		}
		globalGeneratedEbuildSourceLicense = globalSourceLicenseVal
	} else if globalSourceLicenseVal != nil {
		globalGeneratedEbuildSourceLicense = globalSourceLicenseVal
	} else if globalGenEbuildSourceLicenseVal != nil {
		globalGeneratedEbuildSourceLicense = globalGenEbuildSourceLicenseVal
	}
	config.GenerateMd5Cache = globalGenerateMd5Cache
	config.GenerateOverlay = globalGenerateOverlay
	config.UpsertOverlayRepoName = globalUpsertOverlayRepoName
	if globalOverlayRepoNameSet {
		config.OverlayRepoName = &globalOverlayRepoName
	}
	config.GeneratedEbuildGentooAuthors = globalGeneratedEbuildGentooAuthors
	config.GeneratedEbuildSourceLicense = globalGeneratedEbuildSourceLicense

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
	ForceDate *string
}

func (mac *CmdGenerateArgConfig) cmdGenerateGithubWorkflows(args []string) error {
	config := &CmdGenerateGithubWorkflowsArgConfig{
		CmdGenerateArgConfig: mac,
	}

	var normalizedArgs []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if (arg == "-gentoo-authors" || arg == "--gentoo-authors" ||
			arg == "-generated-ebuild-gentoo-authors" || arg == "--generated-ebuild-gentoo-authors") &&
			i+1 < len(args) {
			if b, ok := parseBoolValue(args[i+1]); ok {
				normalizedArgs = append(normalizedArgs, fmt.Sprintf("%s=%t", arg, b))
				i++
				continue
			}
		}
		normalizedArgs = append(normalizedArgs, arg)
	}
	args = normalizedArgs

	fs := flag.NewFlagSet("", flag.ExitOnError)
	config.InputFile = fs.String("input-file", "input.config", "The input with config")
	config.OutputDir = fs.String("output-dir", "./output", "Directory to output workflows")
	config.ForceDate = fs.String("force-date", "", "Force a specific date (e.g. 2026-08-13 00:00:00 +0000 UTC) for testing")
	gentooAuthors := fs.Bool("gentoo-authors", false, "Emit Gentoo Authors copyright header instead of generic MIT.")
	sourceLicense := fs.String("source-license", "", "Source-level license to emit in headers (e.g., MIT).")
	generatedGentooAuthors := fs.Bool("generated-ebuild-gentoo-authors", false, "Emit Gentoo Authors copyright header instead of generic MIT.")
	generatedSourceLicense := fs.String("generated-ebuild-source-license", "", "Source-level license to emit in headers (e.g., MIT).")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}
	switch fs.Arg(0) {
	case "":
		if config.InputFile == nil || *config.InputFile == "" {
			return fmt.Errorf("input file argument missing")
		}

		var opts []any
		if config.ForceDate != nil && *config.ForceDate != "" {
			t, err := time.Parse("2006-01-02 15:04:05 -0700 MST", *config.ForceDate)
			if err == nil {
				opts = append(opts, t)
			} else {
				log.Printf("Warning: failed to parse force-date: %v. Expected format: '2006-01-02 15:04:05 -0700 MST'", err)
			}
		}
		if config.GenerateMd5Cache {
			opts = append(opts, arrans_overlay_workflow_builder.OptGenerateMd5Cache(true))
		}
		if config.GenerateOverlay {
			opts = append(opts, arrans_overlay_workflow_builder.OptGenerateOverlay(true))
		}
		if config.OverlayRepoName != nil {
			opts = append(opts, arrans_overlay_workflow_builder.OptOverlayRepoName(*config.OverlayRepoName))
		}
		if config.UpsertOverlayRepoName {
			opts = append(opts, arrans_overlay_workflow_builder.OptUpsertOverlayRepoName(true))
		}

		var subGentooAuthorsVal *bool
		var subGenEbuildGentooAuthorsVal *bool
		var subSourceLicenseVal *string
		var subGenEbuildSourceLicenseVal *string

		fs.Visit(func(f *flag.Flag) {
			switch f.Name {
			case "gentoo-authors":
				b := *gentooAuthors
				subGentooAuthorsVal = &b
			case "generated-ebuild-gentoo-authors":
				b := *generatedGentooAuthors
				subGenEbuildGentooAuthorsVal = &b
			case "source-license":
				s := *sourceLicense
				subSourceLicenseVal = &s
			case "generated-ebuild-source-license":
				s := *generatedSourceLicense
				subGenEbuildSourceLicenseVal = &s
			}
		})

		var subcommandGentooAuthors *bool
		if subGentooAuthorsVal != nil && subGenEbuildGentooAuthorsVal != nil {
			if *subGentooAuthorsVal != *subGenEbuildGentooAuthorsVal {
				return fmt.Errorf("conflicting flags: -gentoo-authors=%t and -generated-ebuild-gentoo-authors=%t", *subGentooAuthorsVal, *subGenEbuildGentooAuthorsVal)
			}
			subcommandGentooAuthors = subGentooAuthorsVal
		} else if subGentooAuthorsVal != nil {
			subcommandGentooAuthors = subGentooAuthorsVal
		} else if subGenEbuildGentooAuthorsVal != nil {
			subcommandGentooAuthors = subGenEbuildGentooAuthorsVal
		}

		var subcommandSourceLicense *string
		if subSourceLicenseVal != nil && subGenEbuildSourceLicenseVal != nil {
			if *subSourceLicenseVal != *subGenEbuildSourceLicenseVal {
				return fmt.Errorf("conflicting flags: -source-license=%q and -generated-ebuild-source-license=%q", *subSourceLicenseVal, *subGenEbuildSourceLicenseVal)
			}
			subcommandSourceLicense = subSourceLicenseVal
		} else if subSourceLicenseVal != nil {
			subcommandSourceLicense = subSourceLicenseVal
		} else if subGenEbuildSourceLicenseVal != nil {
			subcommandSourceLicense = subGenEbuildSourceLicenseVal
		}

		if config.GeneratedEbuildGentooAuthors != nil {
			opts = append(opts, arrans_overlay_workflow_builder.OptGeneratedEbuildGentooAuthors(*config.GeneratedEbuildGentooAuthors))
		}
		if config.GeneratedEbuildSourceLicense != nil {
			opts = append(opts, arrans_overlay_workflow_builder.OptGeneratedEbuildSourceLicense(*config.GeneratedEbuildSourceLicense))
		}
		if subcommandGentooAuthors != nil {
			opts = append(opts, arrans_overlay_workflow_builder.OptGentooAuthors(*subcommandGentooAuthors))
		}
		if subcommandSourceLicense != nil {
			opts = append(opts, arrans_overlay_workflow_builder.OptSourceLicense(*subcommandSourceLicense))
		}
		return arrans_overlay_workflow_builder.GenerateGithubWorkflows(*config.InputFile, *config.OutputDir, config.Version, opts...)
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
	case "github-cmake-source":
		if err := config.cmdConfigAddCmakeSourceGithubReleases(fs.Args()[1:]); err != nil {
			return fmt.Errorf("config add: %w", err)
		}
	case "web-appimage":
		if err := config.cmdConfigAddWebAppImage(fs.Args()[1:]); err != nil {
			return fmt.Errorf("config add: %w", err)
		}
	default:
		log.Printf("Unknown command %s", fs.Arg(0))
		log.Printf("Try %s for %s", "github-release-appimage", "To generate a config file from a github release with semantic version for AppImages.")
		log.Printf("Try %s for %s", "github-release-binary", "To generate a config file from a github release with semantic version for Binary Releases.")
		log.Printf("Try %s for %s", "web-appimage", "To generate a config file from a webpage that links to AppImages.")
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

type CmdConfigAddWebAppImageArgConfig struct {
	*CmdConfigAddArgConfig
	PageUrl    *string
	MatchExpr  *string
	ConfigFile *string
	Extension  *string
}

func (mac *CmdConfigAddArgConfig) cmdConfigAddWebAppImage(args []string) error {
	config := &CmdConfigAddWebAppImageArgConfig{CmdConfigAddArgConfig: mac}
	fs := flag.NewFlagSet("", flag.ExitOnError)
	config.ConfigFile = fs.String("to", "input.config", "The input with config")
	config.PageUrl = fs.String("url", "", "The webpage to inspect for AppImages")
	config.MatchExpr = fs.String("match", "", "Optional regular expression used to filter AppImage links")
	config.Extension = fs.String("extension", "", "File extension to match (defaults to AppImage)")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}
	switch fs.Arg(0) {
	case "":
		if config.ConfigFile == nil || *config.ConfigFile == "" {
			return fmt.Errorf("config file to modify argument missing")
		}
		if config.PageUrl == nil || *config.PageUrl == "" {
			return fmt.Errorf("url missing")
		}
		return arrans_overlay_workflow_builder.ConfigAddWebAppImage(*config.ConfigFile, *config.PageUrl, *config.MatchExpr, valueOrDefault(config.Extension))
	default:
		log.Printf("Unknown command %s", fs.Arg(0))
		log.Printf("Try %s for %s", "web-appimage", "Adds a configuration generated from a webpage hosting AppImages.")
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
	case "github-cmake-source":
		if err := config.cmdConfigViewCmakeSourceGithubReleases(fs.Args()[1:]); err != nil {
			return fmt.Errorf("config view: %w", err)
		}
	case "web-appimage":
		if err := config.cmdConfigViewWebAppImage(fs.Args()[1:]); err != nil {
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

type CmdConfigViewWebAppImageArgConfig struct {
	*CmdConfigViewArgConfig
	PageUrl   *string
	MatchExpr *string
	Extension *string
}

func (mac *CmdConfigViewArgConfig) cmdConfigViewWebAppImage(args []string) error {
	config := &CmdConfigViewWebAppImageArgConfig{CmdConfigViewArgConfig: mac}
	fs := flag.NewFlagSet("", flag.ExitOnError)
	config.PageUrl = fs.String("url", "", "Webpage containing AppImage link")
	config.MatchExpr = fs.String("match", "", "Optional regular expression used to filter AppImage links")
	config.Extension = fs.String("extension", "", "File extension to match (defaults to AppImage)")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}
	switch fs.Arg(0) {
	case "":
		if config.PageUrl == nil || *config.PageUrl == "" {
			return fmt.Errorf("url missing")
		}
		return arrans_overlay_workflow_builder.ConfigViewWebAppImage(*config.PageUrl, *config.MatchExpr, valueOrDefault(config.Extension))
	default:
		log.Printf("Unknown command %s", fs.Arg(0))
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
	case "github-cmake-source":
		if err := config.cmdOneshotGithubReleaseCmakeSource(fs.Args()[1:]); err != nil {
			return fmt.Errorf("github cmake source: %w", err)
		}
	case "web-appimage":
		if err := config.cmdOneshotWebAppImage(fs.Args()[1:]); err != nil {
			return fmt.Errorf("web appimage: %w", err)
		}
	default:
		log.Printf("Unknown command %s", fs.Arg(0))
		log.Printf("Try %s for %s", "workflows", "Oneshot workflows from a configfile.")
		os.Exit(-1)
	}
	return nil
}

func (mac *MainArgConfig) printVersion() error {
	fmt.Printf("Arrans Overlay Workflow Builder %s, commit %s, built at %s", mac.Version, mac.Commit, mac.Date)
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
		var opts []any
		if config.GenerateMd5Cache {
			opts = append(opts, arrans_overlay_workflow_builder.OptGenerateMd5Cache(true))
		}
		if config.GenerateOverlay {
			opts = append(opts, arrans_overlay_workflow_builder.OptGenerateOverlay(true))
		}
		if config.OverlayRepoName != nil {
			opts = append(opts, arrans_overlay_workflow_builder.OptOverlayRepoName(*config.OverlayRepoName))
		}
		if config.UpsertOverlayRepoName {
			opts = append(opts, arrans_overlay_workflow_builder.OptUpsertOverlayRepoName(true))
		}
		return arrans_overlay_workflow_builder.CmdOneshotGithubReleaseAppImage(*config.GithubUrl, *config.SelectedVersionTag, *config.TagPrefix, *config.OutputDir, config.Version, opts...)
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
		var opts []any
		if config.GenerateMd5Cache {
			opts = append(opts, arrans_overlay_workflow_builder.OptGenerateMd5Cache(true))
		}
		if config.GenerateOverlay {
			opts = append(opts, arrans_overlay_workflow_builder.OptGenerateOverlay(true))
		}
		if config.OverlayRepoName != nil {
			opts = append(opts, arrans_overlay_workflow_builder.OptOverlayRepoName(*config.OverlayRepoName))
		}
		if config.UpsertOverlayRepoName {
			opts = append(opts, arrans_overlay_workflow_builder.OptUpsertOverlayRepoName(true))
		}
		return arrans_overlay_workflow_builder.CmdOneshotGithubReleaseBinary(*config.GithubUrl, *config.SelectedVersionTag, *config.TagPrefix, *config.OutputDir, config.Version, opts...)
	default:
		log.Printf("Unknown command %s", fs.Arg(0))
		os.Exit(-1)
	}
	return nil
}

type CmdOneshotWebAppImageArgConfig struct {
	*CmdOneshotArgConfig
	PageUrl   *string
	MatchExpr *string
	Extension *string
	OutputDir *string
}

func (mac *CmdOneshotArgConfig) cmdOneshotWebAppImage(args []string) error {
	config := &CmdOneshotWebAppImageArgConfig{CmdOneshotArgConfig: mac}
	fs := flag.NewFlagSet("", flag.ExitOnError)
	config.PageUrl = fs.String("url", "", "Webpage containing AppImage link")
	config.MatchExpr = fs.String("match", "", "Optional regular expression used to filter AppImage links")
	config.Extension = fs.String("extension", "", "File extension to match (defaults to AppImage)")
	config.OutputDir = fs.String("output-dir", "./output", "Directory to output workflows")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}
	switch fs.Arg(0) {
	case "":
		if config.PageUrl == nil || *config.PageUrl == "" {
			return fmt.Errorf("url missing")
		}
		var opts []any
		if config.GenerateMd5Cache {
			opts = append(opts, arrans_overlay_workflow_builder.OptGenerateMd5Cache(true))
		}
		if config.GenerateOverlay {
			opts = append(opts, arrans_overlay_workflow_builder.OptGenerateOverlay(true))
		}
		if config.OverlayRepoName != nil {
			opts = append(opts, arrans_overlay_workflow_builder.OptOverlayRepoName(*config.OverlayRepoName))
		}
		if config.UpsertOverlayRepoName {
			opts = append(opts, arrans_overlay_workflow_builder.OptUpsertOverlayRepoName(true))
		}
		return arrans_overlay_workflow_builder.CmdOneshotWebAppImage(*config.PageUrl, *config.MatchExpr, valueOrDefault(config.Extension), *config.OutputDir, config.Version, opts...)
	default:
		log.Printf("Unknown command %s", fs.Arg(0))
		os.Exit(-1)
	}
	return nil
}

type CmdConfigAddCmakeSourceGithubReleasesArgConfig struct {
	*CmdConfigAddArgConfig
	GithubUrl          *string
	ConfigFile         *string
	SelectedVersionTag *string
	TagPrefix          *string
}

func (mac *CmdConfigAddArgConfig) cmdConfigAddCmakeSourceGithubReleases(args []string) error {
	config := &CmdConfigAddCmakeSourceGithubReleasesArgConfig{
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
		return arrans_overlay_workflow_builder.ConfigAddCmakeSourceGithubReleases(*config.ConfigFile, *config.GithubUrl, *config.SelectedVersionTag, *config.TagPrefix)
	default:
		log.Printf("Unknown command %s", fs.Arg(0))
		log.Printf("Try %s for %s", "github-cmake-source", "Adds an configuration to a configuration file.")
		os.Exit(-1)
	}
	return nil
}

type CmdConfigViewCmakeSourceGithubReleasesArgConfig struct {
	*CmdConfigViewArgConfig
	GithubUrl          *string
	SelectedVersionTag *string
	TagPrefix          *string
}

func (mac *CmdConfigViewArgConfig) cmdConfigViewCmakeSourceGithubReleases(args []string) error {
	config := &CmdConfigViewCmakeSourceGithubReleasesArgConfig{
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
		return arrans_overlay_workflow_builder.ConfigViewCmakeSourceGithubReleases(*config.GithubUrl, *config.SelectedVersionTag, *config.TagPrefix)
	default:
		log.Printf("Unknown command %s", fs.Arg(0))
		log.Printf("Try %s for %s", "github-cmake-source", "Views an addition to a configuration file for a particular query.")
		os.Exit(-1)
	}
	return nil
}

type CmdOneshotGithubReleaseCmakeSourceArgConfig struct {
	*CmdOneshotArgConfig
	GithubUrl          *string
	SelectedVersionTag *string
	TagPrefix          *string
	OutputDir          *string
}

func (mac *CmdOneshotArgConfig) cmdOneshotGithubReleaseCmakeSource(args []string) error {
	config := &CmdOneshotGithubReleaseCmakeSourceArgConfig{
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
		var opts []any
		if config.GenerateMd5Cache {
			opts = append(opts, arrans_overlay_workflow_builder.OptGenerateMd5Cache(true))
		}
		if config.GenerateOverlay {
			opts = append(opts, arrans_overlay_workflow_builder.OptGenerateOverlay(true))
		}
		if config.OverlayRepoName != nil {
			opts = append(opts, arrans_overlay_workflow_builder.OptOverlayRepoName(*config.OverlayRepoName))
		}
		if config.UpsertOverlayRepoName {
			opts = append(opts, arrans_overlay_workflow_builder.OptUpsertOverlayRepoName(true))
		}
		return arrans_overlay_workflow_builder.CmdOneshotGithubReleaseCmakeSource(*config.GithubUrl, *config.SelectedVersionTag, *config.TagPrefix, *config.OutputDir, config.Version, opts...)
	default:
		log.Printf("Unknown command %s", fs.Arg(0))
		os.Exit(-1)
	}
	return nil
}
