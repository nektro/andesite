#!/usr/bin/env bash

init() {
    go get -v -u github.com/rakyll/statik
    $GOPATH/bin/statik -src="./www/"
}
build_template() {
    export CGO_ENABLED=1
    export GOOS=$1
    export GOARCH=$2
    export GOARM=7
    EXT=$3
    TAG=$(date +'%Y.%m.%d')-$(git log --format=%h -1)
    echo $TAG-$GOOS-$GOARCH
    go build -ldflags="-s -w" -o ./bin/andesite-v$TAG-$GOOS-$GOARCH$EXT
}

init
build_template linux amd64
