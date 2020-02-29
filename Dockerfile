FROM golang:1.14-alpine AS build-env
MAINTAINER Aaron France "afrance@6river.com"

WORKDIR /app

RUN apk add --no-cache git make
COPY . .
RUN go mod download
RUN make build

# final stage
FROM alpine
WORKDIR /app
COPY --from=build-env /app/exporter-merger /app/
ENTRYPOINT ./exporter-merger
