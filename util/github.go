package util

import (
	"fmt"
	"net/url"
	"strings"
)

// ExtractGithubOwnerRepo extracts the organization and repository from a GitHub URL.
func ExtractGithubOwnerRepo(githubURL string) (string, string, error) {
	parsedURL, err := url.Parse(githubURL)
	if err != nil {
		return "", "", err
	}

	// Ensure the URL is a GitHub URL
	if !strings.Contains(strings.ToLower(parsedURL.Host), "github.com") {
		return "", "", fmt.Errorf("not a valid GitHub URL: %s", githubURL)
	}

	// Split the path and get the owner and repo
	pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(pathParts) < 2 {
		return "", "", fmt.Errorf("URL does not contain enough parts to extract owner and repo: %s", githubURL)
	}

	owner := pathParts[0]
	repo := pathParts[1]
	return owner, repo, nil
}
