#!/usr/bin/make

# SPDX-FileCopyrightText: © 2020 Olivier Meunier <olivier@neokraft.net>
#
# SPDX-License-Identifier: AGPL-3.0-only

VERSION := $(shell git describe --tags)
DATE := $(shell git log -1 --format=%cI)

TAGS := omit_load_extension foreign_keys json1 fts5 secure_delete
BUILD_TAGS := $(TAGS)
VERSION_FLAGS := \
	-X 'github.com/readeck/readeck/configs.version=$(VERSION)' \
	-X 'github.com/readeck/readeck/configs.buildTimeStr=$(DATE)'

SITECONFIG_SRC=./ftr-site-config
SITECONFIG_DEST=src/pkg/extract/fftr/site-config/standard

# Build the app
.PHONY: all
all: web-build docs-build build build-pg

# Build the server
.PHONY: build
build:
	go build \
		-tags "$(BUILD_TAGS)" \
		-ldflags="$(VERSION_FLAGS) -s -w" \
		-o dist/readeck \
		./src

# Build the server with only PG support (full static)
.PHONY: build-pg
build-pg:
	go build \
		-tags "$(BUILD_TAGS) without_sqlite" \
		-ldflags="$(VERSION_FLAGS) -s -w" \
		-o dist/readeck_pg \
		./src

# Clean the build
.PHONY: clean
clean:
	rm -rf dist
	rm -rf src/assets/www/*
	make -C src/web clean

list:
	go list \
		-tags "$(BUILD_TAGS)" \
		-ldflags="$(VERSION_FLAGS) -s -w" \
		-f "{{ .GoFiles }}" \
		./src

# Linting
.PHONY: lint
lint:
	cd src && golangci-lint run

# SLOC
.PHONY: sloc
sloc:
	scc -i go,js,sass

# Launch tests
.PHONY: test
test: docs-build web-build
	go test -tags "$(TAGS)" -cover -count=1 ./src/...

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
