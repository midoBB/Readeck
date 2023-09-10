#!/usr/bin/make

# SPDX-FileCopyrightText: © 2020 Olivier Meunier <olivier@neokraft.net>
#
# SPDX-License-Identifier: AGPL-3.0-only

-include .makerc

ifeq (, $(shell which git && git status))
VERSION ?= dev
DATE ?= $(shell date --rfc-3339=seconds)
else
VERSION := $(shell git describe --tags)
DATE := $(shell git log -1 --format=%cI)
endif

BUILD_TAGS := netgo osusergo sqlite_omit_load_extension sqlite_foreign_keys sqlite_json1 sqlite_fts5 sqlite_secure_delete
VERSION_FLAGS := \
	-X 'codeberg.org/readeck/readeck/configs.version=$(VERSION)' \
	-X 'codeberg.org/readeck/readeck/configs.buildTimeStr=$(DATE)'

OUTFILE_NAME ?= readeck
LDFLAGS ?= -s -w
DIST ?= dist
export CGO_ENABLED ?= 0
export GOOS?=
export GOARCH?=
export CGO_CFLAGS ?= -D_LARGEFILE64_SOURCE
export CC?=
export XGO_VERSION ?= go-1.21.x
export XGO_PACKAGE ?= src.techknowlogick.com/xgo@latest
export XGO_FLAGS ?= ""

SITECONFIG_SRC=./ftr-site-config
SITECONFIG_DEST=src/pkg/extract/fftr/site-config/standard

# Build the app
.PHONY: all
all: generate build
	touch $(DIST)/.all

.PHONY: generate
generate: web-build docs-build
	mkdir -p $(DIST)
	touch $(DIST)/.generate

# Build the server
.PHONY: build
build:
	@echo "CC: $(CC)"
	@echo "CGO_ENABLED": $$CGO_ENABLED
	@echo "CGO_CFLAGS": $$CGO_CFLAGS
	@echo "GOOS": $$GOOS
	@echo "GOARCH": $$GOARCH
	go build \
		-v \
		-tags "$(BUILD_TAGS)" \
		-ldflags="$(VERSION_FLAGS) $(LDFLAGS)" -trimpath \
		-o $(DIST)/$(OUTFILE_NAME) \
		./src

# Clean the build
.PHONY: clean
clean:
	rm -rf $(DIST)
	rm -rf src/assets/www/*
	rm -rf src/docs/assets/*
	make -C src/web clean

list:
	CGO_ENABLED=$(CGO_ENABLED) CGO_CFLAGS=$(CGO_CFLAGS) \
	go list \
		-tags "$(BUILD_TAGS)" \
		-ldflags="$(VERSION_FLAGS) $(LDFLAGS)" \
		-f "{{ .GoFiles }}" \
		./src

# Linting
.PHONY: lint
lint:
	cd src && golangci-lint run

# SLOC
.PHONY: sloc
sloc:
	scc -i go,js,sass,html,md

# Launch tests
.PHONY: test
test: docs-build web-build
	@echo "CC: $(CC)"
	@echo "CGO_ENABLED": $$CGO_ENABLED
	@echo "CGO_CFLAGS": $$CGO_CFLAGS
	go test \
		-tags "$(BUILD_TAGS)" \
		-ldflags="$(VERSION_FLAGS) $(LDFLAGS)" -trimpath \
		-cover -count=1 ./src/...

# Start the HTTP server
.PHONY: serve
serve:
	modd -f modd.conf

.PHONY: dev
dev:
	${MAKE} -j2 web-watch serve

.PHONY: help-build
docs-build:
	${MAKE} -C src/docs all

.PHONY: web-build
web-build:
	@$(MAKE) -C src/web build

.PHONY: web-watch
web-watch:
	@$(MAKE) -C src/web watch


# Setup the development env
.PHONY: setup
setup:
	${MAKE} -C src/web setup
	go install github.com/cortesi/modd/cmd/modd@latest
	go install github.com/boyter/scc/v3@latest


# Update site-config folder with definitions from
# graby git repository
.PHONY: update-site-config
update-site-config:
	rm -rf $(SITECONFIG_DEST)
	go run ./tools/ftr $(SITECONFIG_SRC) $(SITECONFIG_DEST)


#
# Release targets
#
.PHONY: release-all
release-all:
	${MAKE} release-linux
	${MAKE} release-darwin
	${MAKE} release-windows
	${MAKE} release-checksums


.PHONY: xgo-build
xgo-build: | $(DIST)/.generate
	@echo "CC: $(CC)"
	@echo "CGO_ENABLED": $$CGO_ENABLED
	@echo "CGO_CFLAGS": $$CGO_CFLAGS
	@echo "XGO_VERSION": $(XGO_VERSION)
	@echo "XGO_PACKAGE": $(XGO_PACKAGE)
	@echo "XGO_FLAGS": $(XGO_FLAGS)
	@echo "LDFLAGS": $(LDFLAGS)

	test -d $(DIST) || mkdir $(DIST)
	go run $(XGO_PACKAGE) \
		-v \
		-go $(XGO_VERSION) \
		-dest $(DIST) \
		-tags "$(BUILD_TAGS)" \
		-targets "$(XGO_TARGET)" \
		-ldflags "$(VERSION_FLAGS) $(LDFLAGS)" \
		$(XGO_FLAGS) \
		-out readeck-$(VERSION) \
		./src


.PHONY: release-linux
release-linux: CC=
release-linux: CGO_ENABLED=1
release-linux: LDFLAGS=-linkmode external -extldflags "-static" -s -w
release-linux: XGO_FLAGS=
release-linux: XGO_TARGET="linux/amd64,linux/386,linux/arm-5,linux/arm-6,linux/arm64"
release-linux: xgo-build

.PHONY: release-windows
release-windows: CC=
release-windows: CGO_ENABLED=1
release-windows: LDFLAGS=-linkmode external -extldflags "-static" -s -w
release-windows: XGO_FLAGS=-buildmode exe
release-windows: XGO_TARGET="windows/amd64,windows/386"
release-windows: xgo-build

.PHONY: release-darwin
release-darwin: CC=
release-darwin: CGO_ENABLED=1
release-darwin: LDFLAGS=-s -w
release-darwin: XGO_FLAGS=
release-darwin: XGO_TARGET="darwin-10.12/amd64,darwin-10.12/arm64"
release-darwin: xgo-build

.PHONY: release-checksums
release-checksums:
	rm -rf $(DIST)/*.sha256
	cd $(DIST)/; for file in `find . -type f -name "*"`; do echo "checksumming $${file}" && sha256sum -b `echo $${file} | sed 's/^..//'` > $${file}.sha256; done;
