// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	embedapi "github.com/ultravioletrs/cube/internal/embedder/api"
	"github.com/ultravioletrs/cube/internal/embedder/auth"
	"github.com/ultravioletrs/cube/internal/embedder/domain"
	"github.com/ultravioletrs/cube/internal/embedder/embedding"
	"github.com/ultravioletrs/cube/internal/embedder/ingest"
	"github.com/ultravioletrs/cube/internal/embedder/ingest/sources/google"
	"github.com/ultravioletrs/cube/internal/embedder/ingest/sources/microsoft"
	"github.com/ultravioletrs/cube/internal/embedder/ingest/sources/rclone"
	s3source "github.com/ultravioletrs/cube/internal/embedder/ingest/sources/s3"
	"github.com/ultravioletrs/cube/internal/embedder/llm"
	"github.com/ultravioletrs/cube/internal/embedder/llm/guardrails"
	"github.com/ultravioletrs/cube/internal/embedder/llm/ollama"
	llmopenai "github.com/ultravioletrs/cube/internal/embedder/llm/openai"
	"github.com/ultravioletrs/cube/internal/embedder/postgres"
	"github.com/ultravioletrs/cube/internal/embedder/service"
	objstore "github.com/ultravioletrs/cube/internal/embedder/storage"
)

type config struct {
	httpAddr                string
	dbURL                   string
	authGRPCAddr            string
	objectKeyPrefix         string
	logLevel                string
	googleOAuthClientID     string
	googleOAuthClientSecret string
	rcloneBinary            string
	rcloneConfigDir         string
	rcloneTimeout           time.Duration
	rclonePreflight         bool
	embeddingConfig         embedding.Config
	llmConfig               llm.Config
	chatTopK                int
	guardrailsURL           string
	rerankerModel           string
	rerankerBaseURL         string
	storageConfig           objstore.Config
	chunkSize               int
	chunkOverlap            int
	ingestBatchSize         int
	ingestMaxConcurrency    int
	ingestPollInterval      time.Duration
	ingestEmbedBatchSize    int
}

func loadConfig() config {
	defaultProfile := embedding.ProfileConfig{
		Provider:   "ollama",
		BaseURL:    "http://ollama:11434",
		Model:      "nomic-embed-text",
		Dimensions: 768,
	}

	embeddingConfig := embedding.Config{
		Profiles: map[string]embedding.ProfileConfig{
			// All local default profiles use 768-dim vectors to match the current pgvector schema.
			"text":  defaultProfile,
			"code":  defaultProfile,
			"image": defaultProfile,
		},
		Selection: embedding.SelectionConfig{
			DefaultProfile: "text",
			ByRecordFormat: map[domain.RecordFormat]string{
				domain.RecordFormatText:  "text",
				domain.RecordFormatMD:    "text",
				domain.RecordFormatPDF:   "text",
				domain.RecordFormatDOCX:  "text",
				domain.RecordFormatLink:  "text",
				domain.RecordFormatCode:  "code",
				domain.RecordFormatImage: "image",
			},
		},
	}

	loadEmbeddingConfigFromEnv(&embeddingConfig)

	return config{
		httpAddr:                env("EMBEDDER_HTTP_ADDR", ":8080"),
		dbURL:                   mustEnv("EMBEDDER_DB_URL"),
		authGRPCAddr:            env("EMBEDDER_AUTH_GRPC_URL", "auth:8181"),
		objectKeyPrefix:         env("EMBEDDER_OBJECT_STORAGE_PREFIX", "uploads"),
		logLevel:                env("EMBEDDER_LOG_LEVEL", "info"),
		googleOAuthClientID:     env("EMBEDDER_GOOGLE_OAUTH_CLIENT_ID", ""),
		googleOAuthClientSecret: env("EMBEDDER_GOOGLE_OAUTH_CLIENT_SECRET", ""),
		rcloneBinary:            env("EMBEDDER_RCLONE_BINARY", "rclone"),
		rcloneConfigDir:         env("EMBEDDER_RCLONE_CONFIG_DIR", "/etc/cube/rclone"),
		rcloneTimeout:           envDuration("EMBEDDER_RCLONE_TIMEOUT", 2*time.Minute),
		rclonePreflight:         envBool("EMBEDDER_RCLONE_PREFLIGHT", true),
		embeddingConfig:         embeddingConfig,
		storageConfig: objstore.Config{
			Provider:          env("EMBEDDER_OBJECT_STORAGE_PROVIDER", objstore.ProviderLocal),
			LocalDir:          env("EMBEDDER_UPLOAD_DIR", "/tmp/embedder/uploads"),
			S3Endpoint:        env("EMBEDDER_S3_ENDPOINT", ""),
			S3Region:          env("EMBEDDER_S3_REGION", "us-east-1"),
			S3Bucket:          env("EMBEDDER_S3_BUCKET", ""),
			S3AccessKeyID:     env("EMBEDDER_S3_ACCESS_KEY_ID", ""),
			S3SecretAccessKey: env("EMBEDDER_S3_SECRET_ACCESS_KEY", ""),
			S3UseSSL:          envBool("EMBEDDER_S3_USE_SSL", false),
			S3PathStyle:       envBool("EMBEDDER_S3_PATH_STYLE", true),
			S3EnsureBucket:    envBool("EMBEDDER_S3_ENSURE_BUCKET", true),
		},
		llmConfig: llm.Config{
			Provider: env("EMBEDDER_LLM_PROVIDER", "ollama"),
			BaseURL:  env("EMBEDDER_LLM_BASE_URL", "http://ollama:11434"),
			Model:    env("EMBEDDER_LLM_MODEL", "llama3.2:3b"),
			APIKey:   env("EMBEDDER_LLM_API_KEY", ""),
		},
		chatTopK:             envInt("EMBEDDER_CHAT_TOP_K", 15),
		guardrailsURL:        env("EMBEDDER_GUARDRAILS_URL", ""),
		rerankerModel:        env("EMBEDDER_RERANKER_MODEL", ""),
		rerankerBaseURL:      env("EMBEDDER_RERANKER_BASE_URL", ""),
		chunkSize:            envInt("EMBEDDER_CHUNK_SIZE", 512),
		chunkOverlap:         envInt("EMBEDDER_CHUNK_OVERLAP", 64),
		ingestBatchSize:      envInt("EMBEDDER_INGEST_BATCH_SIZE", 20),
		ingestMaxConcurrency: envInt("EMBEDDER_INGEST_MAX_CONCURRENCY", 4),
		ingestPollInterval:   envDuration("EMBEDDER_INGEST_POLL_INTERVAL", 3*time.Second),
		ingestEmbedBatchSize: envInt("EMBEDDER_INGEST_EMBED_BATCH_SIZE", 16),
	}
}

func profileEnvPrefix(name string) string {
	return "EMBEDDER_EMBEDDING_" + strings.ToUpper(name)
}

func loadProfileFromEnv(name string, fallback embedding.ProfileConfig) embedding.ProfileConfig {
	prefix := profileEnvPrefix(name)
	profile := fallback
	profile.Provider = env(prefix+"_PROVIDER", profile.Provider)
	profile.BaseURL = env(prefix+"_BASE_URL", profile.BaseURL)
	profile.Model = env(prefix+"_MODEL", profile.Model)
	profile.Dimensions = envInt(prefix+"_DIMENSIONS", profile.Dimensions)
	profile.APIKey = env(prefix+"_API_KEY", profile.APIKey)
	return profile
}

func main() {
	cfg := loadConfig()

	level := slog.LevelInfo
	if cfg.logLevel == "debug" {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := runRclonePreflight(ctx, cfg); err != nil {
		slog.Error("rclone preflight failed", "err", err)
		os.Exit(1)
	}

	// ── Database ──────────────────────────────────────────────────────────────

	pool, err := pgxpool.New(ctx, cfg.dbURL)
	if err != nil {
		slog.Error("connect to database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("ping database", "err", err)
		os.Exit(1)
	}

	if err := postgres.Migrate(ctx, pool); err != nil {
		slog.Error("run migrations", "err", err)
		os.Exit(1)
	}
	slog.Info("migrations applied")

	// ── Auth (gRPC) ───────────────────────────────────────────────────────────

	authenticator, authConn, err := auth.NewAuthenticator(cfg.authGRPCAddr)
	if err != nil {
		slog.Error("connect to auth gRPC", "addr", cfg.authGRPCAddr, "err", err)
		os.Exit(1)
	}
	defer authConn.Close()
	slog.Info("auth gRPC connected", "addr", cfg.authGRPCAddr)

	// ── Repositories & services ───────────────────────────────────────────────

	sourcesRepo := postgres.NewSourcesRepository(pool)
	recordsRepo := postgres.NewRecordsRepository(pool)
	chunksRepo := postgres.NewChunksRepository(pool)
	conversationsRepo := postgres.NewConversationsRepository(pool)
	rcloneClient := ingest.NewCommandRcloneClient(cfg.rcloneBinary, cfg.rcloneConfigDir, cfg.rcloneTimeout)
	sourceProviders := ingest.NewSourceProviderRegistry(
		google.NewSourceProvider(),
		s3source.NewSourceProvider(),
		microsoft.NewSourceProvider(),
		rclone.NewSourceProvider(rcloneClient),
	)
	for alias, target := range domain.SourceProviderAliases() {
		sourceProviders.RegisterAlias(alias, target)
	}

	sourcesSvc := service.NewSourcesService(sourcesRepo)
	sourceSyncSvc := service.NewSourceSyncService(sourcesRepo, recordsRepo, sourceProviders)
	recordsSvc := service.NewRecordsService(recordsRepo)
	embeddingRegistry, err := embedding.NewRegistry(cfg.embeddingConfig)
	if err != nil {
		slog.Error("configure embeddings", "err", err)
		os.Exit(1)
	}
	uploadStore, err := objstore.NewStore(cfg.storageConfig)
	if err != nil {
		slog.Error("configure object storage", "provider", cfg.storageConfig.Provider, "err", err)
		os.Exit(1)
	}

	worker := ingest.NewWorker(
		recordsRepo,
		sourcesRepo,
		chunksRepo,
		embeddingRegistry,
		uploadStore,
		sourceProviders,
		cfg.chunkSize,
		cfg.chunkOverlap,
	)
	worker.SetBatchSize(cfg.ingestBatchSize)
	worker.SetMaxConcurrent(cfg.ingestMaxConcurrency)
	worker.SetPollInterval(cfg.ingestPollInterval)
	worker.SetEmbedBatchSize(cfg.ingestEmbedBatchSize)
	go worker.Run(ctx)

	retrieveSvc := service.NewVectorRetrieveService(chunksRepo, embeddingRegistry)

	// ── LLM client & chat service ─────────────────────────────────────────────

	var llmClient llm.Client
	switch cfg.llmConfig.Provider {
	case "openai":
		llmClient = llmopenai.New(cfg.llmConfig.BaseURL, cfg.llmConfig.Model, cfg.llmConfig.APIKey)
	default:
		llmClient = ollama.New(cfg.llmConfig.BaseURL, cfg.llmConfig.Model)
	}

	var guardrailsCtrl *guardrails.Controller
	if cfg.guardrailsURL != "" {
		guardrailsCtrl = guardrails.NewController(guardrails.New(cfg.guardrailsURL))
		llmClient = guardrailsCtrl.Wrap(llmClient)
		slog.Info("guardrails enabled", "url", cfg.guardrailsURL)
	}

	// clientFactory builds a per-request LLM client when the caller overrides the model.
	clientFactory := llm.ClientFactory(func(cfg llm.Config) llm.Client {
		var client llm.Client
		if cfg.Provider == "openai" {
			client = llmopenai.NewFromConfig(cfg)
		} else {
			client = ollama.NewFromConfig(cfg)
		}
		if guardrailsCtrl != nil {
			return guardrailsCtrl.Wrap(client)
		}
		return client
	})

	var reranker llm.Reranker
	if cfg.rerankerModel != "" {
		rerankerBase := cfg.rerankerBaseURL
		if rerankerBase == "" {
			rerankerBase = cfg.llmConfig.BaseURL
		}
		reranker = ollama.NewReranker(rerankerBase, cfg.rerankerModel)
		slog.Info("reranker enabled", "model", cfg.rerankerModel, "base_url", rerankerBase)
	}

	chatSvc := service.NewChatService(retrieveSvc, llmClient, reranker, cfg.chatTopK, cfg.llmConfig, clientFactory)

	// ── HTTP server ───────────────────────────────────────────────────────────

	router := embedapi.NewRouter(
		authenticator,
		sourcesSvc,
		sourceSyncSvc,
		recordsSvc,
		retrieveSvc,
		chatSvc,
		conversationsRepo,
		uploadStore,
		cfg.objectKeyPrefix,
		worker.Trigger,
		cfg.googleOAuthClientID,
		cfg.googleOAuthClientSecret,
		cfg.llmConfig.BaseURL,
		guardrailsCtrl,
	)
	srv := &http.Server{
		Addr:        cfg.httpAddr,
		Handler:     router,
		ReadTimeout: 15 * time.Second,
		// WriteTimeout must be 0 for SSE — a non-zero value will cut streaming
		// responses when the deadline expires.
		WriteTimeout: 0,
		IdleTimeout:  60 * time.Second,
	}

	slog.Info("embedder starting", "addr", cfg.httpAddr)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http server", "err", err)
			cancel()
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		slog.Error("graceful shutdown", "err", err)
	}
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid integer for %s: %v\n", key, err)
			os.Exit(1)
		}
		return n
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid duration for %s: %v\n", key, err)
			os.Exit(1)
		}
		return d
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "t", "yes", "y", "on":
			return true
		case "0", "false", "f", "no", "n", "off":
			return false
		default:
			fmt.Fprintf(os.Stderr, "invalid boolean for %s: %q\n", key, v)
			os.Exit(1)
		}
	}
	return fallback
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		fmt.Fprintf(os.Stderr, "required env var %s is not set\n", key)
		os.Exit(1)
	}
	return v
}

func runRclonePreflight(ctx context.Context, cfg config) error {
	if !cfg.rclonePreflight {
		slog.Info("rclone preflight skipped")
		return nil
	}

	binary := strings.TrimSpace(cfg.rcloneBinary)
	if binary == "" {
		return fmt.Errorf("EMBEDDER_RCLONE_BINARY is empty")
	}
	configDir := strings.TrimSpace(cfg.rcloneConfigDir)
	if configDir == "" {
		return fmt.Errorf("EMBEDDER_RCLONE_CONFIG_DIR is empty")
	}

	stat, err := os.Stat(configDir)
	if err != nil {
		return fmt.Errorf("check EMBEDDER_RCLONE_CONFIG_DIR %q: %w", configDir, err)
	}
	if !stat.IsDir() {
		return fmt.Errorf("EMBEDDER_RCLONE_CONFIG_DIR %q is not a directory", configDir)
	}
	if _, err := os.ReadDir(configDir); err != nil {
		return fmt.Errorf("read EMBEDDER_RCLONE_CONFIG_DIR %q: %w", configDir, err)
	}

	timeout := cfg.rcloneTimeout
	if timeout <= 0 || timeout > 15*time.Second {
		timeout = 15 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	out, err := exec.CommandContext(runCtx, binary, "version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("exec %q version: %w (%s)", binary, err, strings.TrimSpace(string(out)))
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	versionLine := ""
	if len(lines) > 0 {
		versionLine = strings.TrimSpace(lines[0])
	}
	slog.Info("rclone preflight passed", "binary", binary, "config_dir", configDir, "version", versionLine)
	return nil
}

func loadEmbeddingConfigFromEnv(cfg *embedding.Config) {
	for name, profile := range cfg.Profiles {
		cfg.Profiles[name] = loadProfileFromEnv(name, profile)
	}

	cfg.Selection.DefaultProfile = env("EMBEDDER_EMBEDDING_DEFAULT_PROFILE", cfg.Selection.DefaultProfile)

	if raw := os.Getenv("EMBEDDER_EMBEDDING_PROFILE_BY_SOURCE_TYPE"); raw != "" {
		var bySourceType map[domain.SourceType]string
		if err := json.Unmarshal([]byte(raw), &bySourceType); err != nil {
			fmt.Fprintf(os.Stderr, "invalid EMBEDDER_EMBEDDING_PROFILE_BY_SOURCE_TYPE: %v\n", err)
			os.Exit(1)
		}
		cfg.Selection.BySourceType = bySourceType
	}

	if raw := os.Getenv("EMBEDDER_EMBEDDING_PROFILE_BY_RECORD_FORMAT"); raw != "" {
		var byRecordFormat map[domain.RecordFormat]string
		if err := json.Unmarshal([]byte(raw), &byRecordFormat); err != nil {
			fmt.Fprintf(os.Stderr, "invalid EMBEDDER_EMBEDDING_PROFILE_BY_RECORD_FORMAT: %v\n", err)
			os.Exit(1)
		}
		cfg.Selection.ByRecordFormat = byRecordFormat
	}

	if raw := os.Getenv("EMBEDDER_EMBEDDING_PROFILES"); raw != "" {
		var profiles map[string]embedding.ProfileConfig
		if err := json.Unmarshal([]byte(raw), &profiles); err != nil {
			fmt.Fprintf(os.Stderr, "invalid EMBEDDER_EMBEDDING_PROFILES: %v\n", err)
			os.Exit(1)
		}
		cfg.Profiles = profiles
	}

	if raw := os.Getenv("EMBEDDER_EMBEDDING_SELECTION"); raw != "" {
		var selection embedding.SelectionConfig
		if err := json.Unmarshal([]byte(raw), &selection); err != nil {
			fmt.Fprintf(os.Stderr, "invalid EMBEDDER_EMBEDDING_SELECTION: %v\n", err)
			os.Exit(1)
		}
		cfg.Selection = selection
	}
}
