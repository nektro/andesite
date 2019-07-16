#!/usr/bin/env bash

build_template() {
    CGO_ENABLED=0
    GOOS=$1
    GOARCH=$2
    TAG=$(date +'%Y.%m.%d')-$(git log --format=%h -1)
    echo $TRAVIS_TAG
    go build -ldflags="-s -w" -o ./dist/andesite-v$TAG-$GOOS-$GOARCH
}

build_template linux amd64
