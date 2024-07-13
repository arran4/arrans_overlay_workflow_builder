# Generated via: https://github.com/arran4/arrans_overlay/blob/main/.github/workflows/app-misc-StabilityMatrix-appimage-update.yaml
EAPI=8
DESCRIPTION="Multi-Platform Package Manager for Stable Diffusion"
HOMEPAGE="https://lykos.ai"
LICENSE="MIT"
SLOT="0"
KEYWORDS="~amd64"
IUSE=""
DEPEND=""
RDEPEND=""
S="${WORKDIR}"
RESTRICT="strip"

inherit xdg-utils

SRC_URI="https://github.com/LykosAI/StabilityMatrix/releases/download/v${PV}/StabilityMatrix-linux-x64.zip -> ${P}-StabilityMatrix.amd64"

src_unpack() {
  if use amd64; then
    unpack "${DISTDIR}/${P}-StabilityMatrix.${ARCH}" || die "Can't unpack archive file"
    mv "${DESTDIR}/" "StabilityMatrix.AppImage"  || die "Can't move archived file"
  fi
  chmod a+x "StabilityMatrix.AppImage"  || die "Can't chmod archive file"
  ./StabilityMatrix.AppImage --appimage-extract "zone.lykos.stabilitymatrix.desktop" || die "Failed to extract .desktop from appimage"
  ./StabilityMatrix.AppImage --appimage-extract "usr/share/icons" || die "Failed to extract icons from app image"
}

src_prepare() {
  sed -i 's:^Exec=.*:Exec=/opt/bin/StabilityMatrix.AppImage:' 'squashfs-root/zone.lykos.stabilitymatrix.desktop'
  find squashfs-root -type f \( -name index.theme -or -name icon-theme.cache \) -exec rm {} \; 
  find squashfs-root -type d -exec rmdir -p {} \; 
  eapply_user
}

src_install() {
  exeinto /opt/bin
  doexe "StabilityMatrix.AppImage" || die "Failed to install AppImage"
  insinto /usr/share/applications
  doins "squashfs-root/zone.lykos.stabilitymatrix.desktop" || die "Failed to install desktop file"
  insinto /usr/share/icons
  doins -r squashfs-root/usr/share/icons/hicolor || die "Failed to install icons"
}

pkg_postinst() {
  xdg_desktop_database_update
}

