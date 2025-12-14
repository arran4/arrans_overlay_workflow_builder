module github.com/arran4/arrans_overlay_workflow_builder

go 1.25

toolchain go1.25.3

require (
	github.com/Masterminds/semver v1.5.0
	github.com/google/go-cmp v0.7.0
	github.com/google/go-github/v62 v62.0.0
	github.com/klauspost/compress v1.18.2
	github.com/probonopd/go-appimage v0.0.0-20251213150937-1cbba14e14d5
	github.com/stoewer/go-strcase v1.3.1
	github.com/ulikunitz/xz v0.5.15
	golang.org/x/net v0.48.0
)

require (
	github.com/CalebQ42/squashfs v1.0.4 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/adrg/xdg v0.5.3 // indirect
	github.com/alokmenghrajani/gpgeez v0.0.0-20161206084504-1a06f1c582f9 // indirect
	github.com/eclipse/paho.mqtt.golang v1.5.1 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/gliderlabs/ssh v0.3.8 // indirect
	github.com/google/go-github v17.0.0+incompatible // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/hashicorp/go-version v1.8.0 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/kevinburke/ssh_config v1.4.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.22 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rasky/go-lzo v0.0.0-20200203143853-96a758eda86e // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/sergi/go-diff v1.4.0 // indirect
	github.com/src-d/gcfg v1.4.0 // indirect
	github.com/stretchr/testify v1.10.0 // indirect
	github.com/therootcompany/xz v1.0.1 // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	golang.org/x/crypto v0.46.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/src-d/go-billy.v4 v4.3.2 // indirect
	// Known issue: gopkg.in/src-d/go-git.v4 has mulitple CVEs, however the upstream
	// probonopd/go-appimage has been archived and no longer updated. Due to this
	// the risk is being accepted as is.
	gopkg.in/src-d/go-git.v4 v4.13.1 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
)
