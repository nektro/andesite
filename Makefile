# Make


all: build

build:
	go build

packages:
	go get -u -v .

travis:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make build-template

build-template:
	go build -ldflags="-s -w" -o ./dist/andesite-$(TRAVIS_TAG)-$(GOOS)-$(GOARCH)
