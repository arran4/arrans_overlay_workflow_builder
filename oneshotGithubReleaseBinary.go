package arrans_overlay_workflow_builder

import (
	"fmt"
	"log"
	"os"
	"time"
)

func CmdOneshotGithubReleaseBinary(gitRepo, tagOverride, tagPrefix, outputDir string) error {
	ic, err := GenerateBinaryGithubReleaseConfigEntry(gitRepo, tagOverride, tagPrefix)
	if err != nil {
		return err
	}

	log.Printf("Showing potential addition to config as entry id: %d", ic.EntryNumber)
	_ = os.Stderr.Sync()
	fmt.Printf("%s\n", ic.String())

	missing := false
	if ic.Category == "" {
		log.Printf("%s needs a category", ic.EbuildName)
		missing = true
	}
	if missing {
		return fmt.Errorf("missing required fields")
	}

	templates, err := ParseWorkflowTemplates()
	if err != nil {
		return err
	}
	now := time.Now()
	_ = os.MkdirAll(outputDir, 0755)
	if err := ic.GenerateGithubWorkflow("-", now, templates, outputDir); err != nil {
		return err
	}
	return nil
}
