FROM golang:alpine as golang
WORKDIR /app
COPY . .
RUN apk add --no-cache git gcc musl-dev \
    && export VCS_REF=$(git tag --points-at HEAD) \
    && echo $VCS_REF \
    && go install -v github.com/rakyll/statik \
    && $GOPATH/bin/statik -src="./www/" \
    && go get -u . \
    && CGO_ENABLED=1 go build -ldflags "-s -w -X main.Version=$VCS_REF" .

FROM alpine
COPY --from=golang /app/andesite /app/andesite

VOLUME /data
ENTRYPOINT ["/app/andesite", "--port", "8000", "--config", "/data/config.json"]
