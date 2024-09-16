package arrans_overlay_workflow_builder

import (
	"bytes"
	"log"
	"reflect"
	"testing"
	"time"
)

func TestGenerateGithubBinaryTemplateData_GetMustHaveUseFlags_and_GetMustHaveUseFlags(t *testing.T) {
	tests := []struct {
		name                      string
		ggbtd                     *GenerateGithubBinaryTemplateData
		programName               string
		kw                        string
		wantGetMustHaveUseFlags   []string
		wantGetMustntHaveUseFlags []string
	}{
		{
			name: "Test 1",
			ggbtd: NewGenerateGithubBinaryTemplateDataFromString(`Type Github Binary Release
GithubProjectUrl https://github.com/arran4/anytype-to-linkwarden
EbuildName anytype-to-linkwarden-bin
Description TODO
License unknown
Workaround Programs as Alternatives => amd64:anytype-to-linkwarden
ProgramName anytype-to-linkwarden
Document amd64=>anytype-to-linkwarden_${VERSION}_linux_386.tar.gz > readme.md > readme.md
Document amd64=>anytype-to-linkwarden_${VERSION}_linux_amd64.tar.gz > readme.md > readme.md
Document arm64=>anytype-to-linkwarden_${VERSION}_linux_arm64.tar.gz > readme.md > readme.md
Binary amd64=>anytype-to-linkwarden_${VERSION}_linux_amd64.tar.gz > anytype-to-linkwarden > anytype-to-linkwarden
Binary arm64=>anytype-to-linkwarden_${VERSION}_linux_arm64.tar.gz > anytype-to-linkwarden > anytype-to-linkwarden
`),
			programName:               "anytype-to-linkwarden",
			kw:                        "amd64",
			wantGetMustHaveUseFlags:   []string{"amd64"},
			wantGetMustntHaveUseFlags: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ggbtd.GetMustHaveUseFlags(tt.programName, tt.kw); !reflect.DeepEqual(got, tt.wantGetMustHaveUseFlags) {
				t.Errorf("GetMustHaveUseFlags() = %v, want %v", got, tt.wantGetMustHaveUseFlags)
			}
			if got := tt.ggbtd.GetMustntHaveUseFlags(tt.programName, tt.kw); !reflect.DeepEqual(got, tt.wantGetMustntHaveUseFlags) {
				t.Errorf("GetMustntHaveUseFlags() = %v, want %v", got, tt.wantGetMustntHaveUseFlags)
			}
		})
	}
}

func NewGenerateGithubBinaryTemplateDataFromString(s string) *GenerateGithubBinaryTemplateData {
	ics, err := ParseInputConfigReader(bytes.NewReader([]byte(s)))
	if err != nil {
		log.Panicf("ParseInputConfigReader: %s", err)
	}
	if len(ics) == 0 {
		log.Panicf("no configs in ics")
	}
	ic := ics[0]
	base := &GenerateGithubWorkflowBase{
		Version:     "1",
		Now:         time.Now(),
		ConfigFile:  "test.config",
		InputConfig: ic,
	}
	switch ic.Type {
	case "Github Binary Release":
		return &GenerateGithubBinaryTemplateData{
			GenerateGithubWorkflowBase: base,
		}
	}
	log.Panicf("unkown type %s", ic.Type)
	return nil
}
