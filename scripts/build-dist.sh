#!/bin/env bash
set -e

# icon generate
icon_src="assets/icon.ico"
icon_dest="app/resource.syso"
[ -e "$icon_dest" ] && rm "$icon_dest" | true
rsrc -ico "$icon_src" -o "$icon_dest"

rm -rf build
mkdir -p build/dist

# clone client
git clone -b gh-pages https://github.com/viasite/planfix-toggl-client.git build/planfix-toggl-client

# set version
version="$(git tag | grep '^[0-9]\.' | tail -n1)"
from='var version string'
to="$from = \"$version\""
sed -i "s/$from\$/$to/g" app/main.go

# go binaries windows
gox -output "build/bin/{{.OS}}/planfix-toggl-server" -arch="amd64" -os="windows" github.com/viasite/planfix-toggl-server/app
[ -e "$icon_dest" ] && rm "$icon_dest" | true
# go binaries other
gox -output "build/bin/{{.OS}}/planfix-toggl-server" -arch="amd64" -os="linux darwin" github.com/viasite/planfix-toggl-server/app

# unset version
sed -i "s/$to/$from/g" app/main.go

# remove old builds
rm -rf "build/archives"

oses="windows linux darwin"
for os in $oses; do
    dir="build/archives/$os"
    mkdir -p "$dir"

    # bin
    cp -r build/bin/$os/* "$dir"

    # client
    mkdir -p "$dir/docroot"
    cp -r build/planfix-toggl-client/* "$dir/docroot"

    # config
    cp config.default.yml "$dir"
    cp config.dist.yml "$dir/config.yml"

    # archives at build/dist zip
    pushd "$dir"
    zip -5 -r -q "../../dist/planfix-toggl-$os.zip" .
    popd
done
