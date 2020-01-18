#!/usr/bin/env bash

init() {
    go get -v -u github.com/rakyll/statik
    $GOPATH/bin/statik -src="./www/"
}
build_template() {
    export CGO_ENABLED=1
    export GOOS=$1
    export GOARCH=$2
<<<<<<< HEAD
    if [ $GOOS = 'windows' ]
    then
        if [ $GOARCH = 'amd64' ]
        then
            export CC="x86_64-w64-mingw32-gcc"
            export CXX="x86_64-w64-mingw32-g++"
        else
            export CC="i686-w64-mingw32-gcc"
            export CXX="i686-w64-mingw32-g++"
        fi
    fi
    EXT=$3
    TAG=$(date +'%Y.%m.%d').$(git log --format=%h -1)
    if [ $GOARCH = 'arm' ]
    then
        export GOARM=6
        echo $TAG-$GOOS-armv$GOARM
        go build -ldflags="-s -w" -o ./bin/andesite-v$TAG-$GOOS-armv$GOARM
    elif [ $GOARCH = 'arm64' ]
    then
        export GOARM=7
        echo $TAG-$GOOS-arm64v$GOARM
        go build -ldflags="-s -w" -o ./bin/andesite-v$TAG-$GOOS-arm64v$GOARM
    else
        echo $TAG-$GOOS-$GOARCH
        go build -ldflags="-s -w" -o ./bin/andesite-v$TAG-$GOOS-$GOARCH$EXT
    fi
=======
    export GOARM=7
    ext=$3
    date=$(date +'%Y.%m.%d')
    version=${CIRCLE_BUILD_NUM-$date}
    tag=v$version-$(git log --format=%h -1)
    echo $tag-$GOOS-$GOARCH
    go build -ldflags="-s -w" -o ./bin/andesite-$tag-$GOOS-$GOARCH$ext
>>>>>>> upstream/master
}

init
build_template darwin 386
build_template darwin amd64
build_template dragonfly amd64
build_template freebsd 386
build_template freebsd amd64
build_template freebsd arm
build_template linux 386
build_template linux amd64
build_template linux arm
build_template linux arm64
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
build_template netbsd 386
build_template netbsd amd64
build_template netbsd arm
build_template openbsd 386
build_template openbsd amd64
build_template openbsd arm
# build_template plan9 386
# build_template plan9 amd64
# build_template plan9 arm
build_template solaris amd64
build_template windows 386 .exe
build_template windows amd64 .exe
