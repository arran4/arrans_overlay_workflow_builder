# Generated via: https://github.com/arran4/arrans_overlay/blob/main/.github/workflows/app-misc-jan-appimage-update.yaml
EAPI=8
DESCRIPTION="Jan is an open source alternative to ChatGPT that runs 100% offline on your computer. Multiple engine support (llama.cpp, TensorRT-LLM)"
HOMEPAGE="https://jan.ai/"
LICENSE="MIT"
SLOT="0"
KEYWORDS="~amd64"
IUSE=""
DEPEND=""
RDEPEND=""
S="${WORKDIR}"
RESTRICT="strip"

inherit xdg-utils

SRC_URI="
  amd64? ( https://github.com/janhq/jan/releases/download/v0.5.1/jan-linux-x86_64-${PV}.AppImage -> ${P}-jan-linux-x86_64-${PV}.AppImage )
"

src_unpack() {
  if use amd64; then
    cp "${DISTDIR}/${P}-jan-linux-x86_64-${PV}.AppImage" "jan.AppImage"  || die "Can't copy downloaded file"
  fi
  chmod a+x "jan.AppImage"  || die "Can't chmod archive file"
  ./jan.AppImage --appimage-extract "jan.desktop" || die "Failed to extract .desktop from appimage"
  ./jan.AppImage --appimage-extract "usr/share/icons" || die "Failed to extract icons from app image"
}

src_prepare() {
  sed -i 's:^Exec=.*:Exec=/opt/bin/jan.AppImage:' 'squashfs-root/jan.desktop'
  find squashfs-root -type f \( -name index.theme -or -name icon-theme.cache \) -exec rm {} \; 
  find squashfs-root -type d -exec rmdir -p --ignore-fail-on-non-empty {} \; 
  eapply_user
}

src_install() {
  exeinto /opt/bin
  doexe "jan.AppImage" || die "Failed to install AppImage"
  insinto /usr/share/applications
  doins "squashfs-root/jan.desktop" || die "Failed to install desktop file"
  insinto /usr/share/icons
  doins -r squashfs-root/usr/share/icons/hicolor || die "Failed to install icons"
}

pkg_postinst() {
  xdg_desktop_database_update
}

