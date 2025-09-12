# Maintainer: Doridian <git at doridian dot net>

# This should ideally be inside a pkgver() subroutine, but that is not possible
# as part of the version comes from the commit count since the latest tag
# so if you commit your current changes the PKGBUILD that would push it one tag further
# than it just calculated, so it would cause a perma-diff in git which is very suboptimal
latest_tag="$(git describe --tags --abbrev=0)"
commits_since_tag="$(git rev-list --count "${latest_tag}..HEAD")"
pre_tag='pre'
if [ -n "$(git status --porcelain)" ]; then
  pre_tag='dev'
fi
if [ "${commits_since_tag}" != "0" ]; then
  # Increment last element on semver
  IFS='.' read -r major minor patch <<< "${latest_tag}"
  latest_tag="${major}.${minor}.$((patch + 1))-${pre_tag}.${commits_since_tag}"
fi

pkgname=tapemgr
pkgver="${latest_tag}"
pkgrel="1"
pkgdesc='Tape backup manager'
arch=('x86_64' 'arm64')
url='https://github.com/FoxDenHome/tapemgr.git'
license=('GPL-3.0-or-later')
makedepends=('git' 'go')
depends=('ltfs')
source=(
  'config.json'
)
sha256sums=(
  'SKIP'
)

goldflags='' # Hidden tweak for source-ing this file

build() {
  cd "${startdir}"
  go build -trimpath -ldflags "${goldflags} -X github.com/FoxDenHome/tapemgr/util.version=${pkgver} -X github.com/FoxDenHome/tapemgr/util.gitrev=$(git rev-parse HEAD)" -o "${srcdir}/tapemgr" ./cmd/tapemgr
}

package() {
  backup=('etc/tapemgr/config.json')
  cd "${srcdir}"
  mkdir -p "${pkgdir}/etc/tapemgr" "${pkgdir}/var/lib/tapemgr/tapes"
  install -Dm755 ./tapemgr "${pkgdir}/usr/bin/tapemgr"
  install -Dm600 ./config.json "${pkgdir}/etc/tapemgr/config.json"
}
