#
# Copyright (c) 2018, 2019 Intel
#
# SPDX-License-Identifier: Apache-2.0
#
ARG ALPINE=golang:1.11-alpine
FROM ${ALPINE} AS builder
ARG ALPINE_PKG_BASE="build-base git openssh-client"
ARG ALPINE_PKG_EXTRA=""

# Replicate the APK repository override.
# If it is no longer necessary to avoid the CDN mirros we should consider dropping this as it is brittle.
RUN sed -e 's/dl-cdn[.]alpinelinux.org/mirrors.ustc.edu.cn/g' -i~ /etc/apk/repositories

# Install our build time packages.
RUN apk add ${ALPINE_PKG_BASE} ${ALPINE_PKG_EXTRA}

WORKDIR $GOPATH/src/github.com/edgexfoundry/device-opcua-go

COPY . .

RUN make build

# Next image - Copy built Go binary into new workspace
FROM scratch

ENV APP_PORT=49997
#expose command data port
EXPOSE $APP_PORT

COPY  --from=builder /go/src/github.com/edgexfoundry/device-opcua-go/cmd /

CMD ["/device-opcua2","--registry","--profile=docker","--confdir=/res"]
