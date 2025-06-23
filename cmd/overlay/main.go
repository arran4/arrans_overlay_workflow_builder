package main

import (
	"context"
	"flag"
	aowb "github.com/arran4/arrans_binary_overlay_builder"
	"github.com/google/go-github/v62/github"
	"log"
	"os"
	"strings"
)

func main() {
	configFile := flag.String("config-file", "input.config", "Configuration file")
	outputDir := flag.String("output-dir", "./overlay", "Directory to place generated overlay")
	flag.Parse()

	tmpl, err := aowb.ParseWorkflowTemplates()
	if err != nil {
		log.Fatalf("parsing ebuild template: %v", err)
	}

	b, err := os.ReadFile(*configFile)
	if err != nil {
		log.Fatalf("reading %s: %v", *configFile, err)
	}
	configs, err := aowb.ParseInputConfigReader(strings.NewReader(string(b)))
	if err != nil {
		log.Fatalf("parsing %s: %v", *configFile, err)
	}

	for _, ic := range configs {
		if ic.Category == "" {
			ic.Category = aowb.DefaultCategory
		}
		version, err := latestVersion(ic.GithubOwner, ic.GithubRepo, ic.WorkaroundTagPrefix())
		if err != nil {
			log.Printf("fetching version for %s: %v", ic.EbuildName, err)
			continue
		}
		if err := ic.GenerateEbuild(tmpl, *outputDir, version); err != nil {
			log.Printf("writing ebuild for %s: %v", ic.EbuildName, err)
		}
	}

}

func latestVersion(owner, repo, prefix string) (string, error) {
	client := github.NewClient(nil)
	if token, ok := os.LookupEnv("GITHUB_TOKEN"); ok {
		client = client.WithAuthToken(token)
	}
	rel, _, err := client.Repositories.GetLatestRelease(context.Background(), owner, repo)
	if err != nil {
		return "", err
	}
	tag := rel.GetTagName()
	if prefix != "" && strings.HasPrefix(tag, prefix) {
		tag = strings.TrimPrefix(tag, prefix)
	}
	tag = strings.TrimPrefix(tag, "v")
	return tag, nil
}
