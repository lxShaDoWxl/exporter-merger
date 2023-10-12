PACKAGE=prometheus/exporter-merger

NAME=$(notdir $(PACKAGE))

BUILD_VERSION=$(shell git describe --always --dirty --tags | tr '-' '.' )
BUILD_DATE=$(shell date)
BUILD_HASH=$(shell git rev-parse HEAD)
BUILD_MACHINE=$(shell echo $$HOSTNAME)
BUILD_USER=$(shell whoami)

BUILD_FLAGS=-trimpath -a -ldflags "\
    -s -w \
	-X '$(PACKAGE)/cmd.BuildVersion=$(BUILD_VERSION)' \
	-X '$(PACKAGE)/cmd.BuildDate=$(BUILD_DATE)' \
	-X '$(PACKAGE)/cmd.BuildHash=$(BUILD_HASH)' \
	-X '$(PACKAGE)/cmd.BuildEnvironment=$(BUILD_USER)@$(BUILD_MACHINE)' \
"

GOFILES=$(shell find . -type f -name '*.go' -not -path "./vendor/*")
GOPKGS=$(shell go list ./...)

default: build

vendor:
	go mod download -x

format:
	gofmt -s -w $(GOFILES)


.PHONY: lint
lint: ## Run linter
	docker run --pull always --rm -v $(shell pwd):/app -w /app -v $(shell go env GOCACHE):/cache/go -e GOCACHE=/cache/go -e GOLANGCI_LINT_CACHE=/cache/go -v $(shell go env GOPATH)/pkg:/go/pkg golangci/golangci-lint:latest golangci-lint --color always run


test_gopath:
	test $$(go list) = "$(PACKAGE)"

test_packages: vendor
	go test $(GOPKGS)

test_format:
	gofmt -l $(GOFILES)

test: test_gopath test_format lint test_packages

cov:
	gocov test -v $(GOPKGS) \
		| gocov-html > coverage.html

build: vendor
	go build \
		$(BUILD_FLAGS) \
		-o $(NAME)-$(BUILD_VERSION)-$(shell go env GOOS)-$(shell go env GOARCH)
	ln -sf $(NAME)-$(BUILD_VERSION)-$(shell go env GOOS)-$(shell go env GOARCH) $(NAME)

xcbuild: vendor
	go build \
		$(BUILD_FLAGS) \
		-o /go/bin/$(NAME)

xc:
	GOOS=linux GOARCH=amd64 make build
	GOOS=darwin GOARCH=amd64 make build

install: test
	go install \
		$(BUILD_FLAGS)

clean:
	rm -f $(NAME)*

.PHONY: build install test

