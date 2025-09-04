// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
package middleware

import (
	"net/http/httputil"

	"github.com/ultraviolet/cube/proxy"
	"go.opentelemetry.io/otel/trace"
)

var _ proxy.Service = (*tracingMiddleware)(nil)

type tracingMiddleware struct {
	tracer trace.Tracer
	svc    proxy.Service
}

func NewTracingMiddleware(tracer trace.Tracer, svc proxy.Service) proxy.Service {
	return &tracingMiddleware{
		tracer: tracer,
		svc:    svc,
	}
}

// Proxy implements proxy.Service.
func (t *tracingMiddleware) Proxy() *httputil.ReverseProxy {
	proxy := t.svc.Proxy()
	// todo : add tracing to the proxy transport
	/*proxy.Transport = &tracingTransport{
		tracer: t.tracer,
		next:   proxy.Transport,
	}*/

	return proxy
}
