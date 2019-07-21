#!/usr/bin/env bash

build_template() {
    CGO_ENABLED=0
    GOOS=$1
    GOARCH=$2
    TAG=$(date +'%Y.%m.%d')-$(git log --format=%h -1)
    echo $TAG-$GOOS-$GOARCH
    go build -ldflags="-s -w" -o ./dist/andesite-v$TAG-$GOOS-$GOARCH
}

go get github.com/rakyll/statik
~/go/bin/statik -src="./www/"

build_template linux amd64
