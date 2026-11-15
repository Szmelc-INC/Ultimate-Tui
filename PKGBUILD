# Maintainer: SilverX <serainox@gmail.com>
pkgname=ultui-git
_repo=Ultimate-Tui
pkgver=r38.630406f
pkgrel=1
pkgdesc="Config-driven TUI framework in Go: one JSON file describes a whole interface"
arch=('x86_64' 'aarch64')
url="https://github.com/Szmelc-INC/Ultimate-Tui"
license=('MIT')
makedepends=('git' 'go')
provides=('ultui')
conflicts=('ultui')
options=('!debug')
source=("git+$url.git")
sha256sums=('SKIP')

pkgver() {
  cd "$_repo"
  ( set -o pipefail
    git describe --long --tags --abbrev=7 2>/dev/null | sed 's/^v//;s/\([^-]*-g\)/r\1/;s/-/./g' ||
    printf "r%s.%s" "$(git rev-list --count HEAD)" "$(git rev-parse --short=7 HEAD)" )
}

prepare() {
  cd "$_repo"
  export GOPATH="$srcdir/gopath" GOFLAGS="-modcacherw"
  go mod download
}

build() {
  cd "$_repo"
  export GOPATH="$srcdir/gopath" CGO_ENABLED=0 GOFLAGS="-trimpath -mod=readonly -modcacherw"
  go build -ldflags "-s -w" -o ultui .
}

check() {
  cd "$_repo"
  export GOPATH="$srcdir/gopath" GOFLAGS="-mod=readonly -modcacherw"
  HOME="$srcdir" go test ./...
}

package() {
  cd "$_repo"
  install -Dm755 ultui "$pkgdir/usr/bin/ultui"
  install -Dm644 README.md DESIGN.md LLM.md -t "$pkgdir/usr/share/doc/$pkgname/"
  install -Dm644 examples/*.json examples/README.md -t "$pkgdir/usr/share/doc/$pkgname/examples/"
  install -Dm644 LICENSE "$pkgdir/usr/share/licenses/$pkgname/LICENSE"
}
