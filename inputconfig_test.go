package arrans_overlay_workflow_builder

import (
	"bytes"
	"fmt"
	"testing"
)

const testConfigData = `
# Example config
Type Github AppImage
GithubProjectUrl https://github.com/janhq/jan/
DesktopFile jan
InstalledFilename jan
Category app-misc
EbuildName jan-appimage
Description Jan is an open source alternative to ChatGPT that runs 100% offline on your computer. Multiple engine support (llama.cpp, TensorRT-LLM)
Homepage https://jan.ai/
ReleasesFilename amd64=>jan-linux-x86_64-${VERSION}.AppImage
ReleasesFilename arm64=>jan-linux-arm64-${VERSION}.AppImage

Type Github AppImage
GithubProjectUrl https://github.com/anotherorg/anotherrepo/
InstalledFilename anotherapp
ReleasesFilename amd64=>anotherrepo-${VERSION}.AppImage
`

func TestParseConfigFile(t *testing.T) {
	configs, err := ParseInputConfigFile(bytes.NewReader([]byte(testConfigData)))
	if err != nil {
		t.Fatalf("error parsing config file: %v", err)
	}

	if len(configs) != 2 {
		t.Fatalf("expected 2 config entries, got %d", len(configs))
	}

	expectedConfigs := []*InputConfig{
		{
			EntryNumber:       0,
			Type:              "Github AppImage",
			GithubProjectUrl:  "https://github.com/janhq/jan/",
			DesktopFile:       "jan.desktop",
			InstalledFilename: "jan",
			Category:          "app-misc",
			EbuildName:        "jan-appimage.ebuild",
			Description:       "Jan is an open source alternative to ChatGPT that runs 100% offline on your computer. Multiple engine support (llama.cpp, TensorRT-LLM)",
			Homepage:          "https://jan.ai/",
			ReleasesFilename: map[string]string{
				"amd64": "jan-linux-x86_64-${VERSION}.AppImage",
				"arm64": "jan-linux-arm64-${VERSION}.AppImage",
			},
		},
		{
			Type:              "Github AppImage",
			GithubProjectUrl:  "https://github.com/anotherorg/anotherrepo/",
			DesktopFile:       "anotherrepo.desktop",
			Category:          "app-misc",
			EbuildName:        "anotherrepo-appimage.ebuild",
			InstalledFilename: "anotherapp",
			ReleasesFilename: map[string]string{
				"amd64": "anotherrepo-${VERSION}.AppImage",
			},
		},
	}

	// Assertion loop remains the same as before
	for i, expected := range expectedConfigs {
		if configs[i].Type != expected.Type ||
			configs[i].GithubProjectUrl != expected.GithubProjectUrl ||
			configs[i].DesktopFile != expected.DesktopFile ||
			configs[i].InstalledFilename != expected.InstalledFilename ||
			configs[i].Category != expected.Category ||
			configs[i].EbuildName != expected.EbuildName ||
			configs[i].Description != expected.Description ||
			configs[i].Homepage != expected.Homepage ||
			!releasesFilenamesEqual(configs[i].ReleasesFilename, expected.ReleasesFilename) {
			t.Errorf("unexpected config[%d]:\nexpected: %+v\ngot:      %+v", i, expected, configs[i])

			// Print the table only when there is an error
			fmt.Println("------------------------------------------------------------------------------------------------------------------------------------------------------------------------")
			fmt.Printf("| %-10s | %-20s | %-80s | %-80s |\n", "Status", "Field", "Expected Value", "Result Value")
			fmt.Println("------------------------------------------------------------------------------------------------------------------------------------------------------------------------")
			fmt.Printf("| %-10s | %-20s | %-80s | %-80s |\n", getStatus(expected.Type, configs[i].Type), "Type", expected.Type, configs[i].Type)
			fmt.Printf("| %-10s | %-20s | %-80s | %-80s |\n", getStatus(expected.GithubProjectUrl, configs[i].GithubProjectUrl), "GithubProjectUrl", expected.GithubProjectUrl, configs[i].GithubProjectUrl)
			fmt.Printf("| %-10s | %-20s | %-80s | %-80s |\n", getStatus(expected.DesktopFile, configs[i].DesktopFile), "DesktopFile", expected.DesktopFile, configs[i].DesktopFile)
			fmt.Printf("| %-10s | %-20s | %-80s | %-80s |\n", getStatus(expected.InstalledFilename, configs[i].InstalledFilename), "InstalledFilename", expected.InstalledFilename, configs[i].InstalledFilename)
			fmt.Printf("| %-10s | %-20s | %-80s | %-80s |\n", getStatus(expected.Category, configs[i].Category), "Category", expected.Category, configs[i].Category)
			fmt.Printf("| %-10s | %-20s | %-80s | %-80s |\n", getStatus(expected.EbuildName, configs[i].EbuildName), "EbuildName", expected.EbuildName, configs[i].EbuildName)
			fmt.Printf("| %-10s | %-20s | %-80s | %-80s |\n", getStatus(expected.Description, configs[i].Description), "Description", expected.Description, configs[i].Description)
			fmt.Printf("| %-10s | %-20s | %-80s | %-80s |\n", getStatus(expected.Homepage, configs[i].Homepage), "Homepage", expected.Homepage, configs[i].Homepage)
			fmt.Printf("| %-10s | %-20s | %-80s | %-80s |\n", getStatus(formatReleasesFilename(expected.ReleasesFilename), formatReleasesFilename(configs[i].ReleasesFilename)), "ReleasesFilename", formatReleasesFilename(expected.ReleasesFilename), formatReleasesFilename(configs[i].ReleasesFilename))
			fmt.Println("------------------------------------------------------------------------------------------------------------------------------------------------------------------------")
		}
	}
}

func getStatus(expected, result string) string {
	if expected == result {
		return "equal"
	}
	return "not equal"
}

func releasesFilenamesEqual(expected, result map[string]string) bool {
	if len(expected) != len(result) {
		return false
	}
	for key, expectedValue := range expected {
		resultValue, ok := result[key]
		if !ok || resultValue != expectedValue {
			return false
		}
	}
	return true
}

func formatReleasesFilename(releases map[string]string) string {
	var formatted string
	for arch, filename := range releases {
		formatted += fmt.Sprintf("%s=>%s, ", arch, filename)
	}
	return formatted[:len(formatted)-2] // Remove trailing comma and space
}

func TestConfigString(t *testing.T) {
	config := &InputConfig{
		EntryNumber:       0,
		Type:              "Github AppImage",
		GithubProjectUrl:  "https://github.com/janhq/jan/",
		DesktopFile:       "jan.desktop",
		InstalledFilename: "jan",
		Category:          "app-misc",
		EbuildName:        "jan-appimage.ebuild",
		Description:       "Jan is an open source alternative to ChatGPT that runs 100% offline on your computer. Multiple engine support (llama.cpp, TensorRT-LLM)",
		Homepage:          "https://jan.ai/",
		ReleasesFilename: map[string]string{
			"amd64": "anotherrepo-${VERSION}.AppImage",
		},
	}

	expected := `Type Github AppImage
GithubProjectUrl https://github.com/janhq/jan/
DesktopFile jan.desktop
InstalledFilename jan
Category app-misc
EbuildName jan-appimage.ebuild
Description Jan is an open source alternative to ChatGPT that runs 100% offline on your computer. Multiple engine support (llama.cpp, TensorRT-LLM)
Homepage https://jan.ai/
ReleasesFilename amd64=>anotherrepo-${VERSION}.AppImage
`

	result := config.String()
	if result != expected {
		t.Errorf("InputConfig.String() = \n'%s'\n\nwant\n'%s'", result, expected)
	}
}
