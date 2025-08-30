# Maintainer: Doridian <git at doridian dot net>

# This should ideally be inside a pkgver() subroutine, but that is not possible
# as part of the version comes from the commit count since the latest tag
# so if you commit your current changes the PKGBUILD that would push it one tag further
# than it just calculated, so it would cause a perma-diff in git which is very suboptimal
latest_tag="$(git describe --tags --abbrev=0)"
commits_since_tag="$(git rev-list --count ${latest_tag}..HEAD)"
tag_suffix=""
if [ -n "$(git status --porcelain)" ]; then
  tag_suffix="dev"
fi

pkgname=tapemgr
pkgver="${latest_tag}.${commits_since_tag}${tag_suffix}"
pkgrel="1"
pkgdesc='Tape backup manager'
arch=('x86_64' 'arm64')
url='https://github.com/FoxDenHome/tapemgr.git'
license=('GPL-3.0-or-later')
makedepends=('git' 'go')
source=()
sha256sums=()

goldflags='' # Hidden tweak for source-ing this file

build() {
  cd "${startdir}"
  go build -trimpath -ldflags "${goldflags} -X github.com/FoxDenHome/tapemgr/util.version=${pkgver} -X github.com/FoxDenHome/tapemgr/util.gitrev=$(git rev-parse HEAD)" -o "${srcdir}/tapemgr" ./cmd/tapemgr
}

package() {
  cd "${srcdir}"
  install -Dm755 ./tapemgr "${pkgdir}/usr/bin/tapemgr"
  mkdir -p "${pkgdir}/etc/tapemgr" "${pkgdir}/var/lib/tapemgr/tapes"
}
