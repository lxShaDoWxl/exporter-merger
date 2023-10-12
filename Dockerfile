# syntax=docker/dockerfile:1.4.1

FROM --platform=$BUILDPLATFORM golang:1.21.3-alpine AS build-env

RUN apk add --no-cache git make

# Configure Go
ENV GOPATH /go
ENV PATH /go/bin:$PATH
ENV GO111MODULE=on


WORKDIR /app/exporter-merger
COPY --link . /app/exporter-merger
RUN --mount=type=cache,target=/go/pkg/mod/ \
     make vendor
ARG TARGETOS TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod/ \
     GO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH make xcbuild

# final stage
FROM scratch
WORKDIR /app
COPY --from=build-env --link /app/exporter-merger/merger.yaml /etc/exporter-merger/config.yaml
COPY --from=build-env --link /go/bin/exporter-merger /app/
ENTRYPOINT [ "/app/exporter-merger" ]