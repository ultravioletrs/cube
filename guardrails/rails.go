package guardrails

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/absmach/supermq/pkg/errors"
	"net/http"
	"net/http/httputil"
)

var (
	ErrViewEntity   = errors.New("failed to view entity")
	ErrNotFound     = errors.New("entity not found")
	ErrUpdateEntity = errors.New("failed to update entity")
	ErrRemoveEntity = errors.New("failed to remove entity")
	ErrCreateEntity = errors.New("failed to create entity")
)

type TLSConfig struct {
	Enabled            bool
	InsecureSkipVerify bool
	CertFile           string
	KeyFile            string
	CAFile             string
	MinVersion         uint16
	MaxVersion         uint16
}

type ServiceConfig struct {
	GuardrailsURL    string `env:"GUARDRAILS_URL"       envDefault:"http://nemo-guardrails:8001"`
	TargetURL        string `env:"TARGET_URL"           envDefault:"http://cube-agent:8901"`
	PolicyConfigPath string `env:"POLICY_CONFIG_PATH"   envDefault:"/config/guardrails_config.yaml"`
	Timeout          int    `env:"TIMEOUT"              envDefault:"30"`
	TLS              TLSConfig
}

type Flow struct {
	ID          string `json:"id" db:"id"`
	Name        string `json:"name" db:"name"`
	Description string `json:"description" db:"description"`
	Content     string `json:"content" db:"content"`
	Type        string `json:"type" db:"type"`
	Active      bool   `json:"active" db:"active"`
	Version     int    `json:"version" db:"version"`
	CreatedAt   string `json:"created_at" db:"created_at"`
	UpdatedAt   string `json:"updated_at" db:"updated_at"`
}

type KBFile struct {
	ID        string                 `json:"id" db:"id"`
	Name      string                 `json:"name" db:"name"`
	Content   string                 `json:"content" db:"content"`
	Type      string                 `json:"type" db:"type"`
	Category  string                 `json:"category" db:"category"`
	Tags      []string               `json:"tags" db:"tags"`
	Metadata  map[string]interface{} `json:"metadata" db:"metadata"`
	Active    bool                   `json:"active" db:"active"`
	Version   int                    `json:"version" db:"version"`
	CreatedAt string                 `json:"created_at" db:"created_at"`
	UpdatedAt string                 `json:"updated_at" db:"updated_at"`
}

type ChatCompletionRequest struct {
	Model       string        `json:"model,omitempty"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
	UserID      string        `json:"-"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionResponse struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Created int64        `json:"created"`
	Model   string       `json:"model"`
	Choices []ChatChoice `json:"choices"`
	Usage   Usage        `json:"usage"`
}

type ChatChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func DefaultTLSConfig() TLSConfig {
	return TLSConfig{
		Enabled:            true,
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS12,
		MaxVersion:         tls.VersionTLS13,
	}
}

func InsecureTLSConfig() TLSConfig {
	return TLSConfig{
		Enabled:            false,
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS12,
		MaxVersion:         tls.VersionTLS13,
	}
}

func setTLSConfig(config *ServiceConfig) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: config.TLS.InsecureSkipVerify,
	}

	if config.TLS.MinVersion != 0 {
		tlsConfig.MinVersion = config.TLS.MinVersion
	}

	if config.TLS.MaxVersion != 0 {
		tlsConfig.MaxVersion = config.TLS.MaxVersion
	}

	if config.TLS.CertFile != "" && config.TLS.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(config.TLS.CertFile, config.TLS.KeyFile)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("failed to load client certificate: %w", err))
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

type Service interface {
	CreateFlow(ctx context.Context, flow Flow) error
	GetFlow(ctx context.Context, id string) (Flow, error)
	GetFlows(ctx context.Context, pm PageMetadata) ([]Flow, error)
	UpdateFlow(ctx context.Context, flow Flow) error
	DeleteFlow(ctx context.Context, id string) error

	CreateKBFile(ctx context.Context, file KBFile) error
	GetKBFile(ctx context.Context, id string) (KBFile, error)
	GetKBFiles(ctx context.Context, pm PageMetadata) ([]KBFile, error)
	UpdateKBFile(ctx context.Context, file KBFile) error
	DeleteKBFile(ctx context.Context, id string) error
	SearchKBFiles(ctx context.Context, query string, categories []string, tags []string, limit int) ([]KBFile, error)

	ProcessRequest(ctx context.Context, body []byte, headers http.Header) ([]byte, http.Header, error)
	ProcessResponse(ctx context.Context, body []byte, headers http.Header) ([]byte, http.Header, error)
	ValidateRequest(ctx context.Context, request interface{}) error
	ValidateResponse(ctx context.Context, response interface{}) error

	Proxy() *httputil.ReverseProxy

	ProcessChatCompletion(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error)
	GetNeMoConfig(ctx context.Context) ([]byte, error)
	GetNeMoConfigYAML(ctx context.Context) ([]byte, error)
}

type Repository interface {
	CreateFlow(ctx context.Context, flow Flow) error
	GetFlow(ctx context.Context, id string) (Flow, error)
	GetFlows(ctx context.Context, pm PageMetadata) ([]Flow, error)
	UpdateFlow(ctx context.Context, flow Flow) error
	DeleteFlow(ctx context.Context, id string) error

	CreateKBFile(ctx context.Context, file KBFile) error
	GetKBFile(ctx context.Context, id string) (KBFile, error)
	GetKBFiles(ctx context.Context, pm PageMetadata) ([]KBFile, error)
	UpdateKBFile(ctx context.Context, file KBFile) error
	DeleteKBFile(ctx context.Context, id string) error
	SearchKBFiles(ctx context.Context, query string, categories []string, tags []string, limit int) ([]KBFile, error)

	ExportConfig(ctx context.Context) ([]byte, error)
	ImportConfig(ctx context.Context, data []byte) error
}
