package api

import (
	"log/slog"
	"net/http"

	"github.com/absmach/supermq"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/ultraviolet/cube/proxy"
)

const ContentType = "application/json"

func MakeHandler(svc proxy.Service, logger *slog.Logger, instanceID string) http.Handler {
	proxy := svc.Proxy()

	mux := chi.NewRouter()

	mux.Handle("/", proxy)

	mux.Get("/health", supermq.Health("cube-proxy", instanceID))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}
