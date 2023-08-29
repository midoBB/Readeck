# SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
#
# SPDX-License-Identifier: AGPL-3.0-only

# This is the Container build file for Readeck
# If you intend to run it yourself, please set the following option to the build command:
# --ulimit=nofile=4000

# First build stage
FROM debian:bookworm as build

# Base dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    xz-utils \
    ca-certificates \
    make

# Install nodejs
ARG NODEVERSION=18
RUN echo "deb https://deb.nodesource.com/node_${NODEVERSION}.x bookworm main" > /etc/apt/sources.list.d/node.list
RUN curl -L -o /etc/apt/trusted.gpg.d/nodesource.gpg.asc "https://deb.nodesource.com/gpgkey/nodesource.gpg.key"

RUN apt-get update && apt-get install -y --no-install-recommends \
    nodejs

# Install Go
ARG GOVERSION=1.21.0
RUN curl -s -L https://dl.google.com/go/go${GOVERSION}.linux-amd64.tar.gz \
    | tar xvz -C /usr/local

# Install Zig
ARG ZIGVERSION=0.11.0
RUN curl -s -L https://ziglang.org/download/${ZIGVERSION}/zig-linux-x86_64-${ZIGVERSION}.tar.xz \
    | tar xvJ -C /usr/local

# Install UPX
ARG UPXVERSION=4.1.0
RUN curl -s -L https://github.com/upx/upx/releases/download/v${UPXVERSION}/upx-${UPXVERSION}-amd64_linux.tar.xz \
    | tar xvJ -C /usr/local/bin --strip-components=1 upx-${UPXVERSION}-amd64_linux/upx

ENV PATH=/usr/local/go/bin:/usr/local/zig-linux-x86_64-${ZIGVERSION}:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

# Let's build!
COPY go.work go.work.sum Makefile /build/
COPY src /build/src/
COPY tools /build/tools/
RUN ls -l /build/

WORKDIR /build

# Build assets
RUN make -C src/web setup
RUN make docs-build web-build

# Build readeck
# This creates a statically linked binary using zig cc.
# The VERSION and DATE args should be passed by the docker build
# command but can be left empty if you're building an image
# for yourself.
ARG VERSION=container-unknown
ARG DATE=
ENV CGO_ENABLED=1
ENV CC="zig cc -target x86_64-linux-musl"
ENV CXX="zig cc -target x86_64-linux-musl"
ENV LDFLAGS="-s -w -linkmode 'external' -extldflags '-static'"
RUN make VERSION=${VERSION} DATE=${DATE} build

# Final compression of our binary
RUN upx -v --best --lzma dist/readeck


# Second stage
# The binary is statically linked so we can just copy it in a small image
# and call it a day.
# You're welcome for the tiny image :)
FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /build/dist/readeck /bin/readeck

ARG VERSION
LABEL org.opencontainers.image.authors="olivier@readeck.com" \
      version="${VERSION}"

ENV READECK_SERVER_HOST=0.0.0.0
ENV READECK_SERVER_PORT=5000
EXPOSE 5000/tcp
VOLUME /readeck
WORKDIR /readeck
CMD ["/bin/readeck", "-c", "config.toml", "serve"]
