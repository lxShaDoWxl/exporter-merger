# syntax=docker/dockerfile:1.4.1

FROM --platform=$BUILDPLATFORM golang:alpine AS build-env

RUN apk add --no-cache git make

# Configure Go
ENV GOPATH /go
ENV PATH /go/bin:$PATH
ENV GO111MODULE=off

# Install Go Tools
RUN go get -u golang.org/x/lint/golint
RUN go get -u github.com/golang/dep/cmd/dep

WORKDIR /go/src/github.com/rebuy-de/exporter-merger
COPY --link . /go/src/github.com/rebuy-de/exporter-merger/
RUN --mount=type=cache,id=go-build,target=/root/.cache/go-build \
    make vendor
ARG TARGETOS TARGETARCH
RUN --mount=type=cache,id=go-build,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH make xcbuild

# final stage
FROM alpine
WORKDIR /app
COPY --from=build-env --link /go/src/github.com/rebuy-de/exporter-merger/merger.yaml /app/
COPY --from=build-env --link /go/bin/exporter-merger /app/
CMD [ "./exporter-merger" ]
