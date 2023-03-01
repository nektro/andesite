FROM golang:alpine as golang
WORKDIR /app
COPY . .
ARG VERSION
RUN apk add --no-cache git gcc musl-dev
RUN go install -v github.com/rakyll/statik
RUN go get -v .
RUN $GOPATH/bin/statik -src="./www/"
RUN CGO_ENABLED=1 go build -ldflags "-s -w -X main.Version=$VERSION" .

FROM alpine
COPY --from=golang /app/andesite /app/andesite

VOLUME /data
ENTRYPOINT ["/app/andesite", "--port", "8000", "--config", "/data/config.json"]
