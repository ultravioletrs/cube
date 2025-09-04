// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
package middleware

import (
	"net/http/httputil"

	"github.com/go-kit/kit/metrics"
	"github.com/ultraviolet/cube/proxy"
)

var _ proxy.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     proxy.Service
}

func NewMetricsMiddleware(counter metrics.Counter, latency metrics.Histogram, svc proxy.Service) proxy.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

// Proxy implements proxy.Service.
func (m *metricsMiddleware) Proxy() *httputil.ReverseProxy {
	proxy := m.svc.Proxy()
	//todo : add metrics to the proxy transport
	/*proxy.Transport = &metricsTransport{
		counter: m.counter,
		latency: m.latency,
		next:    proxy.Transport,
	}*/
	m.counter.With("method", "proxy").Add(1)
	m.latency.With("method", "proxy").Observe(0) // Placeholder for actual latency measurement

	return proxy
}
