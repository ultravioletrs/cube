package embedding

import (
	"context"
	"fmt"

	"github.com/ultravioletrs/cube/internal/embedder/domain"
	"github.com/ultravioletrs/cube/internal/embedder/embedding/ollama"
	"github.com/ultravioletrs/cube/internal/embedder/embedding/openai"
)

// Embedder converts a batch of text strings into dense float vectors.
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	Dimensions() int
}

// ProfileConfig defines one concrete embedding model configuration.
type ProfileConfig struct {
	Provider   string `json:"provider"`
	BaseURL    string `json:"base_url"`
	Model      string `json:"model"`
	Dimensions int    `json:"dimensions"`
	APIKey     string `json:"api_key,omitempty"`
}

// SelectionConfig maps records to embedding profiles.
// Record-format mappings take precedence over source-type mappings.
type SelectionConfig struct {
	DefaultProfile string                         `json:"default"`
	BySourceType   map[domain.SourceType]string   `json:"by_source_type,omitempty"`
	ByRecordFormat map[domain.RecordFormat]string `json:"by_record_format,omitempty"`
}

// Config is the full embedding registry configuration.
type Config struct {
	Profiles  map[string]ProfileConfig `json:"profiles"`
	Selection SelectionConfig          `json:"selection"`
}

// Registry resolves a record to the embedding client configured for it.
type Registry struct {
	clients   map[string]Embedder
	selection SelectionConfig
}

// NewRegistry constructs all configured provider clients up front.
func NewRegistry(cfg Config) (*Registry, error) {
	if len(cfg.Profiles) == 0 {
		return nil, fmt.Errorf("embedding profiles are required")
	}
	if cfg.Selection.DefaultProfile == "" {
		return nil, fmt.Errorf("embedding selection.default is required")
	}

	clients := make(map[string]Embedder, len(cfg.Profiles))
	for name, profile := range cfg.Profiles {
		client, err := newClient(profile)
		if err != nil {
			return nil, fmt.Errorf("configure embedding profile %q: %w", name, err)
		}
		clients[name] = client
	}

	if _, ok := clients[cfg.Selection.DefaultProfile]; !ok {
		return nil, fmt.Errorf("default embedding profile %q is not defined", cfg.Selection.DefaultProfile)
	}
	for sourceType, name := range cfg.Selection.BySourceType {
		if _, ok := clients[name]; !ok {
			return nil, fmt.Errorf("source type %q references unknown embedding profile %q", sourceType, name)
		}
	}
	for recordFormat, name := range cfg.Selection.ByRecordFormat {
		if _, ok := clients[name]; !ok {
			return nil, fmt.Errorf("record format %q references unknown embedding profile %q", recordFormat, name)
		}
	}

	return &Registry{
		clients:   clients,
		selection: cfg.Selection,
	}, nil
}

// ForRecord returns the configured embedding client for a record.
func (r *Registry) ForRecord(rec domain.Record) (Embedder, error) {
	profileName := r.selection.DefaultProfile
	if name, ok := r.selection.ByRecordFormat[rec.Format]; ok {
		profileName = name
	} else if rec.Source != nil {
		if name, ok := r.selection.BySourceType[rec.Source.Type]; ok {
			profileName = name
		}
	}

	client, ok := r.clients[profileName]
	if !ok {
		return nil, fmt.Errorf("embedding profile %q is not configured", profileName)
	}
	return client, nil
}

func newClient(cfg ProfileConfig) (Embedder, error) {
	switch cfg.Provider {
	case "ollama":
		return ollama.New(cfg.BaseURL, cfg.Model, cfg.Dimensions), nil
	case "openai":
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("api_key is required for openai profile")
		}
		return openai.New(cfg.BaseURL, cfg.Model, cfg.APIKey, cfg.Dimensions), nil
	default:
		return nil, fmt.Errorf("unknown embedding provider %q", cfg.Provider)
	}
}
