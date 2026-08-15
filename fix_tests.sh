#!/bin/bash
cat << 'INNER_EOF' > testdata/txtar/github-binary/hugo_alternative.txtar
Test correct deduction and non-duplication of USE flags
-- input.config --
Type Github Binary Release
GithubProjectUrl https://github.com/gohugoio/hugo
EbuildName hugo-bin
Category app-text
Description Fast and Flexible Static Site Generator
Homepage https://gohugo.io/
License Apache-2.0
Workaround Programs as Alternatives => amd64:extended arm64:extended
ProgramName hugo
Binary amd64=>hugo_${VERSION}_linux-amd64.tar.gz > hugo > hugo
Binary arm=>hugo_${VERSION}_linux-arm.tar.gz > hugo > hugo
Binary arm64=>hugo_${VERSION}_linux-arm64.tar.gz > hugo > hugo
ProgramName extended
Binary amd64=>hugo_extended_${VERSION}_linux-amd64.tar.gz > hugo > hugo
Binary arm64=>hugo_extended_${VERSION}_linux-arm64.tar.gz > hugo > hugo
-- expected.yaml --
INNER_EOF

cat << 'INNER_EOF' > testdata/txtar/github-binary/chezmoi_style.txtar
Test correct deduction and non-duplication of USE flags
-- input.config --
Type Github Binary Release
GithubProjectUrl https://github.com/twpayne/chezmoi
EbuildName chezmoi-bin
Category app-admin
Description Manage your dotfiles across multiple diverse machines, securely.
Homepage https://www.chezmoi.io/
License MIT License
Workaround Programs as Alternatives => amd64:glibc amd64:loong64 arm64:android ppc64:le
ProgramName android
Binary arm64=>chezmoi_${VERSION}_android_arm64.tar.gz > chezmoi > chezmoi
ProgramName chezmoi
Dependencies sys-libs/glibc
Binary amd64=>chezmoi_${VERSION}_linux-musl_amd64.tar.gz > chezmoi > chezmoi
Binary arm=>chezmoi_${VERSION}_linux_arm.tar.gz > chezmoi > chezmoi
Binary arm64=>chezmoi_${VERSION}_linux_arm64.tar.gz > chezmoi > chezmoi
Binary ppc64=>chezmoi_${VERSION}_linux_ppc64.tar.gz > chezmoi > chezmoi
ProgramName glibc
Binary amd64=>chezmoi_${VERSION}_linux-glibc_amd64.tar.gz > chezmoi > chezmoi
ProgramName le
Binary ppc64=>chezmoi_${VERSION}_linux_ppc64le.tar.gz > chezmoi > chezmoi
ProgramName loong64
Binary amd64=>chezmoi_${VERSION}_linux_loong64.tar.gz > chezmoi > chezmoi
-- expected.yaml --
INNER_EOF

cat << 'INNER_EOF' > testdata/txtar/github-binary/hugo_overlap.txtar
Test correct deduction and non-duplication of USE flags
-- input.config --
Type Github Binary Release
GithubProjectUrl https://github.com/gohugoio/hugo
EbuildName hugo-bin
Category app-text
Description Fast and Flexible Static Site Generator
Homepage https://gohugo.io/
License Apache-2.0
Workaround Programs as Alternatives => amd64:extended arm64:extended
IUse extended
ProgramName hugo
Binary amd64=>hugo_${VERSION}_linux-amd64.tar.gz > hugo > hugo
Binary arm=>hugo_${VERSION}_linux-arm.tar.gz > hugo > hugo
Binary arm64=>hugo_${VERSION}_linux-arm64.tar.gz > hugo > hugo
ProgramName extended
Binary amd64=>hugo_extended_${VERSION}_linux-amd64.tar.gz > hugo > hugo
Binary arm64=>hugo_extended_${VERSION}_linux-arm64.tar.gz > hugo > hugo
-- expected.yaml --
INNER_EOF
