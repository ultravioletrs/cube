# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.24.2-alpine AS builder
ARG SVC
ARG GOOS
ARG GOARCH
ARG VERSION
ARG COMMIT

WORKDIR /go/src/github.com/ultraviolet/cube
COPY . .
RUN apk update \
    && apk add make upx\
    && make build \
    && upx build/cube-proxy \
    && mv build/cube-proxy /exe

FROM scratch
COPY --from=builder /exe /
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
ENTRYPOINT ["/exe"]
