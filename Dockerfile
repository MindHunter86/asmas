# -*- coding: utf-8 -*-
# vim: ft=Dockerfile

### container - builder
FROM golang:1.22.10-bullseye AS build
LABEL maintainer="mindhunter86 <mindhunter86@vkom.cc>"

ARG GOAPP_MAIN_VERSION="devel"
ARG GOAPP_MAIN_BUILDTIME="N/A"

ENV MAIN_VERSION=$GOAPP_MAIN_VERSION
ENV MAIN_BUILDTIME=$GOAPP_MAIN_BUILDTIME

ENV DEBIAN_FRONTEND=noninteractive

# hadolint/hadolint - DL4006
SHELL ["/bin/bash", "-o", "pipefail", "-c"]

WORKDIR /usr/sources/asmas
COPY . .

# skipcq: DOK-DL3008 pinning version for upx is not required
RUN echo "ready" \
  && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X 'main.version=$MAIN_VERSION' -X 'main.buildtime=$MAIN_BUILDTIME'" -o asmas cmd/asmas/main.go cmd/asmas/flags.go \
  && apt-get update && apt-get install --no-install-recommends -y upx-ucl \
  && upx -9 -k asmas


### container - runner
###   for image debuging use tag :debug
FROM gcr.io/distroless/static-debian11:latest-amd64
LABEL maintainer="mindhunter86 <mindhunter86@vkom.cc>"

WORKDIR /usr/local/bin/
COPY --from=build --chmod=0555 /usr/sources/asmas/asmas asmas

USER nobody
ENTRYPOINT ["/usr/local/bin/asmas"]
CMD ["--help"]
