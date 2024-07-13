# Generated via: https://github.com/arran4/arrans_overlay/blob/main/.github/workflows/app-misc-flutter-google-datastore-appimage-update.yaml
EAPI=8
DESCRIPTION="Google Datastore and Datastore emulator client for easy modification of values"
HOMEPAGE=""
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
  amd64? ( https://github.com/arran4/flutter_google_datastore/releases/download/v0.1.5/flutter_google_datastore-linux-x86_64.AppImage -> ${P}-flutter_google_datastore-linux-x86_64.AppImage )
"

src_unpack() {
  if use amd64; then
    cp "${DISTDIR}/${P}-flutter_google_datastore-linux-x86_64.AppImage" "flutter_google_datastore.AppImage"  || die "Can't copy downloaded file"
  fi
  chmod a+x "flutter_google_datastore.AppImage"  || die "Can't chmod archive file"
  ./flutter_google_datastore.AppImage --appimage-extract "flutter_google_datastore.desktop" || die "Failed to extract .desktop from appimage"
  ./flutter_google_datastore.AppImage --appimage-extract "usr/share/icons" || die "Failed to extract icons from app image"
}

src_prepare() {
  sed -i 's:^Exec=.*:Exec=/opt/bin/flutter_google_datastore.AppImage:' 'squashfs-root/flutter_google_datastore.desktop'
  find squashfs-root -type f \( -name index.theme -or -name icon-theme.cache \) -exec rm {} \; 
  find squashfs-root -type d -exec rmdir -p --ignore-fail-on-non-empty {} \; 
  eapply_user
}

src_install() {
  exeinto /opt/bin
  doexe "flutter_google_datastore.AppImage" || die "Failed to install AppImage"
  insinto /usr/share/applications
  doins "squashfs-root/flutter_google_datastore.desktop" || die "Failed to install desktop file"
  insinto /usr/share/icons
  doins -r squashfs-root/usr/share/icons/hicolor || die "Failed to install icons"
}

pkg_postinst() {
  xdg_desktop_database_update
}

