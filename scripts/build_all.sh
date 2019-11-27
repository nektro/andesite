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
# build_template darwin 386
# build_template darwin amd64
# build_template dragonfly amd64
# build_template freebsd 386
# build_template freebsd amd64
# build_template freebsd arm
# build_template linux 386
build_template linux amd64
# build_template linux arm
# build_template linux arm64
# build_template linux ppc64
# build_template linux ppc64le
# build_template linux mips
# build_template linux mipsle
# build_template linux mips64
# build_template linux mips64le
# build_template linux s390x
# build_template nacl 386
# build_template nacl amd64p32
# build_template nacl arm
# build_template netbsd 386
# build_template netbsd amd64
# build_template netbsd arm
# build_template openbsd 386
# build_template openbsd amd64
# build_template openbsd arm
# build_template plan9 386
# build_template plan9 amd64
# build_template plan9 arm
# build_template solaris amd64
# build_template windows 386
# build_template windows amd64
