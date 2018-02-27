#!/bin/env bash
set -e

rm -rf build
mkdir -p build/dist

# clone client
git clone -b gh-pages https://github.com/viasite/planfix-toggl-client.git build/planfix-toggl-client

oses="windows linux darwin"

# go binaries
gox -output "build/bin/{{.OS}}/planfix-toggl-server" -arch="amd64" -os="$oses" github.com/viasite/planfix-toggl-server/app

rm -rf "build/archives"

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

    # archives at build/dist
    pushd "$dir"
    zip -5 -r -q "../../dist/planfix-toggl-$os.zip" .
    popd
done
