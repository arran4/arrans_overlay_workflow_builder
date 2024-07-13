# Generated via: https://github.com/arran4/arrans_overlay/blob/main/.github/workflows/net-im-caprine-appimage-update.yaml
EAPI=8
DESCRIPTION="Elegant Facebook Messenger desktop app"
HOMEPAGE="https://sindresorhus.com/caprine"
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
  amd64? ( https://github.com/sindresorhus/caprine/releases/download/v2.60.1/Caprine-2.60.1.AppImage -> ${P}-Caprine-2.60.1.AppImage )
"

src_unpack() {
  if use amd64; then
    cp "${DISTDIR}/${P}-Caprine-2.60.1.AppImage" "Caprine.AppImage"  || die "Can't copy downloaded file"
  fi
  chmod a+x "Caprine.AppImage"  || die "Can't chmod archive file"
  ./Caprine.AppImage --appimage-extract "caprine.desktop" || die "Failed to extract .desktop from appimage"
  ./Caprine.AppImage --appimage-extract "usr/share/icons" || die "Failed to extract icons from app image"
}

src_prepare() {
  sed -i 's:^Exec=.*:Exec=/opt/bin/Caprine.AppImage:' 'squashfs-root/caprine.desktop'
  find squashfs-root -type f \( -name index.theme -or -name icon-theme.cache \) -exec rm {} \; 
  find squashfs-root -type d -exec rmdir -p --ignore-fail-on-non-empty {} \; 
  eapply_user
}

src_install() {
  exeinto /opt/bin
  doexe "Caprine.AppImage" || die "Failed to install AppImage"
  insinto /usr/share/applications
  doins "squashfs-root/caprine.desktop" || die "Failed to install desktop file"
  insinto /usr/share/icons
  doins -r squashfs-root/usr/share/icons/hicolor || die "Failed to install icons"
}

pkg_postinst() {
  xdg_desktop_database_update
}

