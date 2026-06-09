# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.26.0-alpine AS builder
ARG SVC=proxy
ARG GOOS=linux
ARG GOARCH=amd64
ARG VERSION
ARG COMMIT

WORKDIR /go/src/github.com/ultraviolet/cube
COPY . .

RUN CGO_ENABLED=0 GOOS=${GOOS} GOARCH=${GOARCH} \
    go build -ldflags "-s -w" \
    -o build/cube-${SVC} cmd/${SVC}/main.go

RUN mv build/cube-${SVC} /exe

FROM alpine:3.21 AS rclone
ARG GOARCH=amd64
RUN apk add --no-cache curl unzip
RUN set -eux; \
    case "${GOARCH}" in \
      amd64|arm64|386) rclone_arch="${GOARCH}" ;; \
      arm) rclone_arch="arm" ;; \
      *) echo "unsupported GOARCH for rclone: ${GOARCH}" >&2; exit 1 ;; \
    esac; \
    curl -fsSL "https://downloads.rclone.org/rclone-current-linux-${rclone_arch}.zip" -o /tmp/rclone.zip; \
    unzip -q /tmp/rclone.zip -d /tmp; \
    cp /tmp/rclone-*-linux-${rclone_arch}/rclone /rclone; \
    chmod 0755 /rclone

FROM scratch
COPY --from=builder /exe /
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=rclone /rclone /usr/bin/rclone
COPY --from=builder /go/src/github.com/ultraviolet/cube/docker/rclone/rclone.conf /etc/cube/rclone/rclone.conf
ENTRYPOINT ["/exe"]
