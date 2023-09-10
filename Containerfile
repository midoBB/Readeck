# SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
#
# SPDX-License-Identifier: AGPL-3.0-only

# This is the Container build file for Readeck
# It uses the release files in the dist folder

# First stage, only to get the ca-certificates
FROM alpine:edge as build

RUN apk add --no-cache ca-certificates

# Second stage
# The binary is statically linked so we can just copy it in a small image
# and call it a day.
# You're welcome for the tiny image :)

FROM scratch
ARG VERSION
ARG DIST

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY ${DIST}/readeck-${VERSION}-linux-amd64 /bin/readeck

LABEL org.opencontainers.image.authors="olivier@readeck.com" \
      version="${VERSION}"

ENV READECK_SERVER_HOST=0.0.0.0
ENV READECK_SERVER_PORT=5000
EXPOSE 5000/tcp
VOLUME /readeck
WORKDIR /readeck
CMD ["/bin/readeck", "serve", "-config", "config.toml"]
