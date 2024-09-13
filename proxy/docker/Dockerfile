FROM golang:1.23.1-alpine AS builder
ARG SVC
ARG GOOS
ARG GOARCH
ARG VERSION
ARG COMMIT

WORKDIR /go/src/github.com/ultraviolet/vault-proxy
COPY . .
RUN apk update \
    && apk add make upx\
    && make build \
    && upx build/vault-proxy \
    && mv build/vault-proxy /exe

FROM scratch
COPY --from=builder /exe /
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
ENTRYPOINT ["/exe"]
