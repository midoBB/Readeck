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

BUILD_TAGS := netgo osusergo
VERSION_FLAGS := \
	-X 'codeberg.org/readeck/readeck/configs.version=$(VERSION)' \
	-X 'codeberg.org/readeck/readeck/configs.buildTimeStr=$(DATE)'

OUTFILE_NAME ?= readeck
LDFLAGS ?= -s -w
DIST ?= dist
GO ?= go
export CGO_ENABLED ?= 0
export GOOS?=
export GOARCH?=

SITECONFIG_SRC=./ftr-site-config
SITECONFIG_DEST=pkg/extract/contentscripts/assets/site-config


FILE_COMPOSE_PKG ?= codeberg.org/readeck/file-compose@latest
GOLANGCI_PKG ?= github.com/golangci/golangci-lint/cmd/golangci-lint@v1.61.0
AIR_PKG ?= github.com/air-verse/air@v1.61.1
SLOC_PKG ?= github.com/boyter/scc/v3@v3.4.0

# -------------------------------------------------------------------
# Base targets
# -------------------------------------------------------------------

# These targets provide all what's needed to build Readeck.
# On a fresh copy, the workflow would be:
#   make setup && make all
#
# If you plan to write code, have a look at the development
# targets below.


# Build the app
.PHONY: all
all: generate build
	touch $(DIST)/.all

# Generate files (web and documentation artifacts)
.PHONY: generate
generate: web-build docs-build
	mkdir -p $(DIST)
	touch $(DIST)/.generate

# Setup prepares the environment
.PHONY: setup
setup:
	${MAKE} -C web setup

# Build the server
.PHONY: build
build:
	@echo "GOOS": $$GOOS
	@echo "GOARCH": $$GOARCH
	@echo "LDFLAGS": $(LDFLAGS)
	$(GO) build \
		-v \
		-tags "$(BUILD_TAGS)" \
		-ldflags="$(VERSION_FLAGS) $(LDFLAGS)" -trimpath \
		-o $(DIST)/$(OUTFILE_NAME)

# Build the documentation
.PHONY: docs-build
docs-build:
	$(GO) run $(FILE_COMPOSE_PKG) -format json docs/api/api.yaml docs/assets/api.json
	$(GO) run ./tools/docs docs/src docs/assets

# Build the frontend assets
.PHONY: web-build
web-build:
	@$(MAKE) -C web build

# Launch tests
.PHONY: test
test: docs-build
	test -f assets/www/manifest.json || echo "{}" > assets/www/manifest.json
	$(GO) test \
		-tags "$(BUILD_TAGS)" \
		-ldflags="$(VERSION_FLAGS) $(LDFLAGS)" \
		-cover -count=1 ./...

# Clean the build
.PHONY: clean
clean:
	rm -rf $(DIST)
	rm -rf assets/www/*
	rm -rf docs/assets/*
	make -C web clean

# List all the modules included by the build
list:
	$(GO) list \
		-tags "$(BUILD_TAGS)" \
		-ldflags="$(VERSION_FLAGS) $(LDFLAGS)" \
		-m all

# Run the linter and print a report
.PHONY: lint
lint: docs-build
	test -f assets/www/manifest.json || echo "{}" > assets/www/manifest.json
	$(GO) run $(GOLANGCI_PKG) run

# Single lines of code
.PHONY: sloc
sloc:
	$(GO) run $(SLOC_PKG) -i go,js,sass,html,md


# Update site-config folder with definitions from
# graby git repository
.PHONY: update-site-config
update-site-config:
	rm -rf $(SITECONFIG_DEST)
	$(GO) run ./tools/ftr $(SITECONFIG_SRC) $(SITECONFIG_DEST)


# -------------------------------------------------------------------
# Development targets
# -------------------------------------------------------------------

# These targets provide helper for autoreload during development.
# `make dev` starts all the needed watch/autoreload and is only
# needed when working on web/* or docs/src/*.
# When working only on go files, `make serve` is enough.

# Starts 3 watchers/reloaders for a full autoreload
# dev server.
# The initial errors during startup are normal
.PHONY: dev
dev:
	${MAKE} -j3 docs-watch web-watch serve

# Starts the HTTP server
# It runs air watching the source files and the assets. It builds and reloads
# the server on any change.
.PHONY: serve
serve:
	$(GO) run $(AIR_PKG) \
		--tmp_dir "dist" \
		--build.log "" \
		--build.cmd "${MAKE} DATE= build" \
		--build.bin "dist/readeck" \
		--build.args_bin "serve" \
		--build.exclude_dir "" \
		--build.include_dir "assets,configs,docs,locales,internal,pkg" \
		--build.include_ext "go,html,json,js,po,tmpl,toml" \
		--build.delay 2000

# Watch the docs/src folder and rebuild the documentation
# on changes.
.PHONY: docs-watch
docs-watch:
	$(GO) run $(AIR_PKG) \
		--tmp_dir "dist" \
		--build.log "" \
		--build.cmd "${MAKE} docs-build" \
		--build.bin "" \
		--build.exclude_dir "" \
		--build.include_file "CHANGELOG.md" \
		--build.include_dir "docs/src,docs/api" \
		--build.include_ext "md,png,svg,json,yaml" \
		--build.delay 200

# Starts the watcher on the web folder.
.PHONY: web-watch
web-watch:
	@$(MAKE) -C web watch


# -------------------------------------------------------------------
# Release targets
# -------------------------------------------------------------------

# Unless you plan to build a full release, you don't need to use
# these targets.

.PHONY: stamp-version
stamp-version:
	echo $(VERSION) > $(DIST)/VERSION

.PHONY: release-all
release-all:
	${MAKE} release-linux
	${MAKE} release-darwin
	${MAKE} release-freebsd
	${MAKE} release-windows
	${MAKE} release-checksums
	echo $(VERSION) > $(DIST)/VERSION
	cp CHANGELOG.md $(DIST)
	touch $(DIST)/.release


.PHONY: xbuild
xbuild: CGO_ENABLED=0
xbuild: LDFLAGS=-s -w
xbuild: OUTFILE_NAME=readeck-$(VERSION)-$(GOOS)-$(GOARCH)
xbuild: build


xbuild/%:
	$(eval GOOS=$(shell echo $* | cut -d"/" -f1))
	$(eval GOARCH=$(shell echo $* | cut -d"/" -f2))
	${MAKE} xbuild

.PHONY: release-linux
release-linux:
	${MAKE} xbuild/linux/amd64
	${MAKE} xbuild/linux/arm64
	touch $(DIST)/.release-linux

.PHONY: release-windows
release-windows:
	${MAKE} xbuild/windows/amd64
	touch $(DIST)/.release-windows

.PHONY: release-darwin
release-darwin:
	${MAKE} xbuild/darwin/amd64
	${MAKE} xbuild/darwin/arm64
	touch $(DIST)/.release-darwin

.PHONY: release-freebsd
release-freebsd:
	${MAKE} xbuild/freebsd/amd64
	${MAKE} xbuild/freebsd/arm64
	touch $(DIST)/.release-freebsd


.PHONY: release-checksums
release-checksums:
	rm -rf $(DIST)/*.sha256
	cd $(DIST)/; for file in `find . -type f -name "*"`; do echo "checksumming $${file}" && sha256sum -b `echo $${file} | sed 's/^..//'` > $${file}.sha256; done;


.PHONY: release-container
release-container: TAG?=readeck-release:$(VERSION)
release-container: | $(DIST)/.release-linux
	./tools/build-container $(VERSION) $(DIST)/container-$(VERSION).tar --rm
