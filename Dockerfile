From golang:1.14-alpine as basis
LABEL maintainer="OpenSlides Team <info@openslides.com>"
WORKDIR /root/

RUN apk add git

COPY go.mod go.sum ./
RUN go mod download

COPY . .


# Build wsproxy in seperate stage
FROM basis as builder
RUN go build ./cmd/wsproxy


# Test build.
From basis as testing

RUN apk add build-base

CMD go vet ./... && go test ./...


# Development build.
FROM basis as development

RUN ["go", "get", "github.com/githubnemo/CompileDaemon"]

EXPOSE 9015
CMD CompileDaemon -log-prefix=false -build="go build ./cmd/wsproxy" -command="./wsproxy"


# Productive build.
FROM alpine:latest
WORKDIR /root/

COPY --from=builder /root/wsproxy .

EXPOSE 9015
CMD ./wsproxy