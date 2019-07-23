#!/usr/bin/env bash

init() {
    go get -v -u github.com/rakyll/statik
    $GOPATH/bin/statik -src="./www/"
}
build_template() {
    export GOOS=$1
    export GOARCH=$2
    export GOARM=7
    EXT=$3
    TAG=$(date +'%Y.%m.%d')-$(git log --format=%h -1)
    echo $TAG-$GOOS-$GOARCH
    go build -ldflags="-s -w" -o ./bin/andesite-v$TAG-$GOOS-$GOARCH$EXT
}

init
build_template darwin 386
build_template darwin amd64
build_template dragonfly amd64
build_template freebsd 386
build_template freebsd amd64
build_template linux 386
build_template linux amd64
build_template netbsd 386
build_template netbsd amd64
build_template openbsd 386
build_template openbsd amd64
build_template solaris amd64
build_template windows 386 .exe
build_template windows amd64 .exe
