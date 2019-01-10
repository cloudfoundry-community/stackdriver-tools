.EXPORT_ALL_VARIABLES:
COMMIT_HASH := $(shell git rev-parse HEAD)
TIMESTAMP := $(shell date +%s)
VERSION ?= $(shell git describe --tags --exact-match `git rev-parse HEAD` 2>/dev/null || echo 0.0.$(TIMESTAMP)-custom.$(COMMIT_HASH))
TILE_NAME ?= $(shell if [ `echo $(VERSION) | grep -o custom` ]; then echo stackdriver-nozzle-custom; else echo stackdriver-nozzle; fi)
TILE_LABEL ?= $(shell if [ `echo $(VERSION) | grep -o custom` ]; then echo "Stackdriver Nozzle (custom build)"; else echo Stackdriver Nozzle; fi)
TILE_FILENAME := $(TILE_NAME)-$(VERSION).pivotal
TILE_SHA256 := $(TILE_FILENAME).sha256
RELEASE_TARBALL := stackdriver-tools-release-$(VERSION).tar.gz
RELEASE_SHA256 := $(RELEASE_TARBALL).sha256
RELEASE_BUILD_DIR := build
TILE_BUILD_DIR := product
RELEASE_PATH := $(RELEASE_BUILD_DIR)/$(RELEASE_TARBALL)

build:
	go build -v ./...

test:
	go test -v ./...

lint:
	# Tests for output
	# Disabling gosec for https://github.com/securego/gosec/issues/267
	# Disabling vetshadow for https://github.com/golang/go/issues/19490
	# Disabling maligned because it also affect the config struct. TODO(mattysweeps) re-enable maligned
	# Ignoring missing comments for now TODO(mattysweeps) fix godoc
	[ -z "$$($(GOPATH)/bin/gometalinter --deadline=300s --disable gosec --disable vetshadow --disable maligned --vendor ./... | grep -v exported)" ]

get-deps:
	# For gometalinter linting
	cd $(GOPATH) && wget -q -O - https://git.io/vp6lP | sh

	# Simplify cross-compiling
	go get github.com/mitchellh/gox

	# Ginkgo and omega test tools
	go get github.com/onsi/ginkgo/ginkgo
	go get github.com/onsi/gomega

bosh-release:
	echo $(VERSION) > src/stackdriver-nozzle/release
	bosh sync-blobs
	bosh create-release --name=stackdriver-tools --version=$(VERSION) --tarball=$(RELEASE_TARBALL) --force --sha2
	mkdir -p $(RELEASE_BUILD_DIR)
	mv $(RELEASE_TARBALL) $(RELEASE_BUILD_DIR)/$(RELEASE_TARBALL)
	sha256sum $(RELEASE_BUILD_DIR)/$(RELEASE_TARBALL) | cut -d' ' -f 1 > $(RELEASE_BUILD_DIR)/$(RELEASE_SHA256)

tile: bosh-release
	erb tile.yml.erb > tile.yml
	mkdir -p $(TILE_BUILD_DIR)
	echo $(RELEASE_PATH)
	tile build $(VERSION)
	sha256sum $(TILE_BUILD_DIR)/$(TILE_FILENAME) | cut -d' ' -f 1 > $(TILE_BUILD_DIR)/$(TILE_SHA256)

clean:
	go clean ./...
	rm -f tile.yml tile-history.yml
	rm -fr blobs dev_releases product build release .blobs .dev_builds

.PHONY: build test integration-test lint get-deps clean tile bosh-release
