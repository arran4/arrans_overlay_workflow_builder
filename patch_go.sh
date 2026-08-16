cat << 'INNER_EOF' > modify.go.sh
sed -i -e '/func (ggbtd \*GenerateGithubBinaryTemplateData) ExtractedUseFlags() \[\]string {/i \
func (ggbtd *GenerateGithubBinaryTemplateData) IUseFlags() []g2.Flag {\
	seen := make(map[string]bool)\
\
	var finalUses []g2.Flag\
\
	addFlag := func(name, text string) {\
		cleaned := strcase.SnakeCase(name)\
		if cleaned != "" && !seen[cleaned] {\
			seen[cleaned] = true\
			finalUses = append(finalUses, g2.Flag{Name: cleaned, Text: text})\
		}\
	}\
\
	for use := range ggbtd.ReverseProgramsAsAlternatives() {\
		addFlag(use, fmt.Sprintf("Install %s binary", use))\
	}\
	if ggbtd.HasManualPages() {\
		addFlag("man", "Install manual pages")\
	}\
	if ggbtd.HasDocuments() {\
		addFlag("doc", "Install documentation")\
	}\
	for _, shell := range ggbtd.ShellCompletionShells() {\
		addFlag(shell, fmt.Sprintf("Install %s completion", shell))\
	}\
	for _, use := range ggbtd.IUse {\
		addFlag(use, fmt.Sprintf("Enable %s", use))\
	}\
	for _, use := range ggbtd.ExtractedUseFlags() {\
		addFlag(use, fmt.Sprintf("Enable %s", use))\
	}\
\
	sort.Slice(finalUses, func(i, j int) bool {\
		return finalUses[i].Name < finalUses[j].Name\
	})\
\
	return finalUses\
}\
' generateGithubBinaryWorkflow.go

sed -i -e '/func (ggbtd \*GenerateGithubBinaryTemplateData) Metadata() (string, error) {/,$d' generateGithubBinaryWorkflow.go
cat << 'APP_EOF' >> generateGithubBinaryWorkflow.go
func (ggbtd *GenerateGithubBinaryTemplateData) Metadata() (string, error) {
	pkgMd := &g2.PkgMetadata{
		XMLName: xml.Name{
			Local: "pkgmetadata",
		},
		Upstream: &g2.Upstream{
			RemoteID: []g2.RemoteID{},
		},
	}
	if ggbtd.MaintainerEmail != "" {
		pkgMd.Maintainers = append(pkgMd.Maintainers, g2.Maintainer{
			Email: ggbtd.MaintainerEmail,
			Name:  ggbtd.MaintainerName,
			Type:  "person",
		})
	}
	if ggbtd.GithubOwner != "" && ggbtd.GithubRepo != "" {
		pkgMd.Upstream.RemoteID = append(pkgMd.Upstream.RemoteID, g2.RemoteID{
			Type: "github",
			Text: fmt.Sprintf("%s/%s", ggbtd.GithubOwner, ggbtd.GithubRepo),
		})
	}
	if len(pkgMd.Upstream.RemoteID) == 0 {
		pkgMd.Upstream = nil
	}

	if len(pkgMd.Use) == 0 {
		pkgMd.Use = []g2.Use{{}}
	}

	pkgMd.Use[0].Flags = append(pkgMd.Use[0].Flags, ggbtd.IUseFlags()...)

	for i := range pkgMd.Use {
		var newFlags []g2.Flag
		for _, f := range pkgMd.Use[i].Flags {
			if len(f.Name) > 0 {
				newFlags = append(newFlags, f)
			}
		}
		pkgMd.Use[i].Flags = newFlags
		if len(pkgMd.Use[i].Flags) == 0 {
			pkgMd.Use[i].Flags = nil
		}
	}

	var newUses []g2.Use
	for _, u := range pkgMd.Use {
		if len(u.Flags) > 0 {
			newUses = append(newUses, u)
		}
	}
	pkgMd.Use = newUses
	if len(pkgMd.Use) == 0 {
		pkgMd.Use = nil
	}

	o, err := xml.MarshalIndent(pkgMd, "", "	")
	if err != nil {
		return "", fmt.Errorf("marshalling metadata: %w", err)
	}
	return fmt.Sprintf("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<!DOCTYPE pkgmetadata SYSTEM \"http://www.gentoo.org/dtd/metadata.dtd\">\n%s", string(o)), nil
}

func (ggbtd *GenerateGithubBinaryTemplateData) G2MetadataArgs() string {
	if ggbtd == nil || ggbtd.GenerateGithubWorkflowBase == nil {
		return ""
	}
	args := ggbtd.GenerateGithubWorkflowBase.G2MetadataArgs() + " "

	for _, f := range ggbtd.IUseFlags() {
		args += fmt.Sprintf("--use-add \"%s:%s\" ", f.Name, f.Text)
	}
	return strings.TrimSpace(args)
}
APP_EOF
INNER_EOF
chmod +x modify.go.sh
./modify.go.sh

cat << 'INNER_EOF2' > modify.tmpl.sh
for tmpl in templates/github-binary.tmpl templates/web-binary.tmpl; do
  sed -i -e '/echo -n '"'"'IUSE="'"'"'/,/echo '"'"'"'"'"'/c\
                echo -n '"'"'IUSE="'"'"'\
[[- if $.IUseFlags ]]\
                echo -n '"'"'[[- range $i, $use := $.IUseFlags]] [[$use.Name | UseFlagSafe ]][[end]]'"'"'\
[[- end ]]\
                echo '"'"'"'"'"'' $tmpl
done
INNER_EOF2
chmod +x modify.tmpl.sh
./modify.tmpl.sh
