# Make


extension=


all: build

build:
	go build

travis:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 extension=.exe make build-template
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make build-template

build-template:
	go build -o ./dist/andesite-$(TRAVIS_COMMIT)-$(GOOS)-$(GOARCH)$(extension)
