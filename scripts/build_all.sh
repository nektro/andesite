#!/usr/bin/env bash

init() {
    go get -v -u github.com/rakyll/statik
    $GOPATH/bin/statik -src="./www/"
}
build_template() {
    CGO_ENABLED=0
    GOOS=$1
    GOARCH=$2
    EXT=$3
    TAG=$(date +'%Y.%m.%d')-$(git log --format=%h -1)
    echo $TAG-$GOOS-$GOARCH
    go build -ldflags="-s -w" -o ./bin/andesite-v$TAG-$GOOS-$GOARCH$EXT
}

init
build_template linux amd64
