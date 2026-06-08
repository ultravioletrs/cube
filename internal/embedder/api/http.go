// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/ultravioletrs/cube/internal/embedder/api/transport"
	"github.com/ultravioletrs/cube/internal/embedder/auth"
	"github.com/ultravioletrs/cube/internal/embedder/domain"
	objstore "github.com/ultravioletrs/cube/internal/embedder/storage"
)

// NewRouter builds and returns the HTTP router for the embedder service.
func NewRouter(
	authenticator *auth.Authenticator,
	sourcesSvc domain.SourceService,
	sourceSyncSvc domain.SourceSyncService,
	recordsSvc domain.RecordService,
	retrieveSvc domain.VectorRetrieveService,
	chatSvc domain.ChatService,
	conversationsRepo domain.ConversationRepository,
	store objstore.Store,
	objectKeyPrefix string,
	trigger func(),
	googleOAuthClientID string,
	googleOAuthClientSecret string,
	ollamaBaseURL string,
	guardrailsCtrl transport.GuardrailsController,
) http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.Recoverer)
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)

	r.Get("/health", healthHandler)
	r.Handle("/metrics", promhttp.Handler())

	transport.MountModels(r, ollamaBaseURL)

	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(authenticator))
		transport.MountSources(r, sourcesSvc, sourceSyncSvc, trigger, googleOAuthClientID, googleOAuthClientSecret)
		transport.MountRecords(r, recordsSvc, sourcesSvc, store, objectKeyPrefix, trigger)
		transport.MountRetrieve(r, retrieveSvc)
		transport.MountChat(r, chatSvc, conversationsRepo)
		transport.MountConversations(r, conversationsRepo)
		transport.MountGuardrails(r, guardrailsCtrl)
		transport.MountModelConnection(r, ollamaBaseURL)
	})

	return r
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
