#!/bin/sh
set -eu

# Temporary upstream snapshot containing the maintainer's CVE-2026-85091 fix.
# Replace this package with Alpine's zlib once a patched release is available.
commit=df84af25dc1942490e1d1c899a07619152a46148
version=1.3.2.1_git20260917-r0
checksum=2db756b67c4c54663afc21b7a7f1726a97ad58fd7a0b9d87b79e445f92784e56dd33a821b6ec80a0854ec6d767b08dae13eb2c1fe4a320b51ab6dc96a8a5b7d1

apk add --no-cache build-base curl
mkdir -p /build /out
cd /build
curl --fail --location --retry 3 "https://codeload.github.com/madler/zlib/tar.gz/$commit" -o zlib.tar.gz
echo "$checksum  zlib.tar.gz" | sha512sum -c -
tar -xzf zlib.tar.gz
cd "zlib-$commit"
./configure --prefix=/usr --shared --disable-crcvx
make -j"$(getconf _NPROCESSORS_ONLN)"
make check
make install DESTDIR=/build/package

# Ship only the runtime library, retaining normal apk ownership and dependency
# tracking rather than overwriting an installed package's files or version.
rm -rf /build/package/usr/include /build/package/usr/lib/pkgconfig \
    /build/package/usr/lib/libz.a /build/package/usr/lib/libz.so \
    /build/package/usr/share/man
mkdir -p /build/package/usr/share/licenses/zlib
cp LICENSE /build/package/usr/share/licenses/zlib/LICENSE
apk mkpkg --files /build/package --output /out/zlib.apk \
    --info "name:zlib" \
    --info "version:$version" \
    --info "arch:$(apk --print-arch)" \
    --info "description:zlib upstream snapshot with CVE-2026-85091 fixed" \
    --info "license:Zlib" \
    --info "origin:zlib" \
    --info "url:https://github.com/madler/zlib/commit/$commit" \
    --info "repo-commit:$commit" \
    --info "depends:so:libc.musl-$(apk --print-arch).so.1" \
    --info "provides:so:libz.so.1=1.3.2.1"
