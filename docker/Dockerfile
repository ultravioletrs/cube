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

FROM scratch
COPY --from=builder /exe /
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
ENTRYPOINT ["/exe"]
