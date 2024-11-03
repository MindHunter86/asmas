# -*- coding: utf-8 -*-
# vim: ft=Dockerfile

### container - builder
FROM golang:1.19.10-bullseye AS build
LABEL maintainer="mindhunter86 <mindhunter86@vkom.cc>"

ARG CGO_ENABLED=0

ARG GOAPP_MAIN_VERSION="devel"
ARG GOAPP_MAIN_BUILDTIME="N/A"

ENV MAIN_VERSION=$GOAPP_MAIN_VERSION
ENV MAIN_BUILDTIME=$GOAPP_MAIN_BUILDTIME

ARG APT_GET_OPTIONS=" -o APT::Install-Recommends=0 -o APT::Install-Suggests=0"
ARG DEBCONF_NOWARNINGS="yes"
ARG DEBCONF_NONINTERACTIVE_SEEN=true
ARG DEBIAN_FRONTEND=noninteractive
ARG DEBIAN_PRIORITY=critical
ARG TERM=linux

# hadolint/hadolint - DL4006
SHELL ["/bin/bash", "-e", "-c", "-o", "pipefail"]

WORKDIR /usr/sources/asmas
COPY . .

# skipcq: DOK-DL3008 pinning version for upx is not required
RUN echo "ready" \
  && go build -trimpath -ldflags="-s -w -X 'main.version=$MAIN_VERSION' -X 'main.buildtime=$MAIN_BUILDTIME'" -o asmas cmd/asmas/main.go cmd/asmas/flags.go \
  && apt-get -qq update && apt-get -yqq install ca-certificates upx-ucl \
  && upx -9 -k asmas 2>&1

### container - runner
###   for image debuging use tag :debug
FROM gcr.io/distroless/static-debian11:latest-amd64
LABEL maintainer="mindhunter86 <mindhunter86@vkom.cc>"

WORKDIR /usr/local/bin/
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build --chmod=0555 /usr/sources/asmas/asmas asmas

USER nobody
ENTRYPOINT ["/usr/local/bin/asmas"]
CMD ["--help"]
