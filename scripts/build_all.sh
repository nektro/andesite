#!/usr/bin/env bash

init() {
    go get -v -u github.com/rakyll/statik
    $GOPATH/bin/statik -src="./www/" -f
}
build_template() {
    export CGO_ENABLED=1
    export GOOS=$1
    export GOARCH=$2
    export GOARM=7
    ext=$3
    date=$(date +'%Y.%m.%d')
    version=${CIRCLE_BUILD_NUM-$date}
    tag=v$version-$(git log --format=%h -1)
    echo $tag-$GOOS-$GOARCH
    go build -ldflags="-s -w -X main.Version=$tag" -o ./bin/andesite-$tag-$GOOS-$GOARCH$ext
}

init
build_template linux amd64
