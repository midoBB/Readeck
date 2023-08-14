#!/usr/bin/make

# SPDX-FileCopyrightText: © 2020 Olivier Meunier <olivier@neokraft.net>
#
# SPDX-License-Identifier: AGPL-3.0-only

-include .makerc

VERSION := $(shell git describe --tags)
DATE := $(shell git log -1 --format=%cI)

TAGS := netgo osusergo sqlite_omit_load_extension sqlite_foreign_keys sqlite_json1 sqlite_fts5 sqlite_secure_delete
BUILD_TAGS := $(TAGS)
VERSION_FLAGS := \
	-X 'github.com/readeck/readeck/configs.version=$(VERSION)' \
	-X 'github.com/readeck/readeck/configs.buildTimeStr=$(DATE)'

LDFLAGS ?= -s -w
CGO_ENABLED ?= 1
CGO_CFLAGS ?= -D_LARGEFILE64_SOURCE

SITECONFIG_SRC=./ftr-site-config
SITECONFIG_DEST=src/pkg/extract/fftr/site-config/standard

# Build the app
.PHONY: all
all: web-build docs-build build

# Build the server
.PHONY: build
build:
	@echo "CC: $(CC)"
	@echo "CXX: $(CXX)"
	CGO_ENABLED=$(CGO_ENABLED) CGO_CFLAGS=$(CGO_CFLAGS) \
	go build \
		-v \
		-tags "$(BUILD_TAGS)" \
		-ldflags="$(VERSION_FLAGS) $(LDFLAGS)" -trimpath \
		-o dist/readeck \
		./src

# Clean the build
.PHONY: clean
clean:
	rm -rf dist
	rm -rf src/assets/www/*
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
	@echo "CXX: $(CXX)"
	CGO_ENABLED=$(CGO_ENABLED) CGO_CFLAGS=$(CGO_CFLAGS) \
	go test \
		-tags "$(TAGS)" \
		-ldflags="$(VERSION_FLAGS) $(LDFLAGS)" -trimpath \
		-cover -count=1 ./src/...

# Start the HTTP server
.PHONY: serve
serve:
	modd -f modd.conf

# Update site-config folder with definitions from
# graby git repository
.PHONY: update-site-config
update-site-config:
	rm -rf $(SITECONFIG_DEST)
	go run ./tools/ftr $(SITECONFIG_SRC) $(SITECONFIG_DEST)

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
