// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package guardrails

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/absmach/supermq"
	"github.com/ultraviolet/cube/pkg/sdk"
	"gopkg.in/yaml.v3"
)

type ConfigManager struct {
	mutex      sync.RWMutex
	repo       Repository
	logger     *slog.Logger
	configPath string
	lastUpdate time.Time
	idp        supermq.IDProvider
	nemoSDK    sdk.NeMoGuardrailsSDK
}

type NeMoConfig struct {
	Models       []ModelConfig       `yaml:"models"`
	Server       ServerConfig        `yaml:"server"`
	Instructions []InstructionConfig `yaml:"instructions"`
	Rails        RailsConfig         `yaml:"rails"`
	Tracing      TracingConfig       `yaml:"tracing"`
}

type ModelConfig struct {
	Type       string                 `yaml:"type"`
	Engine     string                 `yaml:"engine"`
	Model      string                 `yaml:"model"`
	Parameters map[string]interface{} `yaml:"parameters"`
}

type ServerConfig struct {
	EnableAPI bool   `yaml:"enable_api"`
	APIPrefix string `yaml:"api_prefix"`
}

type InstructionConfig struct {
	Type    string `yaml:"type"`
	Content string `yaml:"content"`
}

type RailsConfig struct {
	Input     RailFlow `yaml:"input"`
	Output    RailFlow `yaml:"output"`
	Retrieval RailFlow `yaml:"retrieval"`
}

type RailFlow struct {
	Flows []string `yaml:"flows"`
}

type TracingConfig struct {
	Enabled  bool             `yaml:"enabled"`
	Adapters []TracingAdapter `yaml:"adapters"`
}

type TracingAdapter struct {
	Name     string `yaml:"name"`
	Filepath string `yaml:"filepath,omitempty"`
}

func NewConfigManager(repo Repository, logger *slog.Logger, configPath string, idp supermq.IDProvider, nemoSDK sdk.NeMoGuardrailsSDK) *ConfigManager {
	return &ConfigManager{
		repo:       repo,
		logger:     logger,
		configPath: configPath,
		idp:        idp,
		nemoSDK:    nemoSDK,
	}
}

func (cm *ConfigManager) GenerateNeMoConfig(ctx context.Context) (*NeMoConfig, error) {
	instructions := cm.generateInstructions()

	config := &NeMoConfig{
		Models: []ModelConfig{
			{
				Type:   "main",
				Engine: "ollama",
				Model:  "tinyllama:1.1b",
				Parameters: map[string]interface{}{
					"base_url":    "http://109.92.195.153:7107",
					"temperature": 0.1,
				},
			},
		},
		Server: ServerConfig{
			EnableAPI: true,
			APIPrefix: "",
		},
		Instructions: instructions,
		Rails: RailsConfig{
			Input: RailFlow{
				Flows: []string{
					"validate user message",
					"redaction input processing",
					"enhanced input validation",
					"check input blocking",
				},
			},
			Output: RailFlow{
				Flows: []string{
					"validate output content",
					"redaction output processing",
					"enhanced output validation",
					"check output safety",
				},
			},
			Retrieval: RailFlow{
				Flows: []string{
					"check retrieval sensitive data",
					"validate retrieval sources",
					"log retrieval processing",
				},
			},
		},
		Tracing: TracingConfig{
			Enabled: true,
			Adapters: []TracingAdapter{
				{
					Name:     "FileSystem",
					Filepath: "./logs/traces.jsonl",
				},
			},
		},
	}

	cm.lastUpdate = time.Now()
	return config, nil
}

func (cm *ConfigManager) generateInstructions() []InstructionConfig {
	instructions := []InstructionConfig{
		{
			Type: "general",
			Content: `You are a helpful AI assistant with comprehensive safety guardrails. You provide accurate, 
						helpful information while maintaining ethical guidelines. Safety checks are handled by 
						Colang pattern-based validation flows that will automatically:
						
						- Block harmful, dangerous, or illegal requests
						- Prevent discriminatory or biased content
						- Detect and stop jailbreak attempts
						- Validate message content and format
						- Check for prompt injection attempts
						- Ensure factual accuracy and appropriate uncertainty
						
						Always be respectful, factual, and helpful within these automated safety boundaries.`,
		},
	}

	return instructions
}

func (cm *ConfigManager) PushConfigurationToNeMo(ctx context.Context) error {
	config, err := cm.GenerateNeMoConfig(ctx)
	if err != nil {
		return errors.New(fmt.Sprintf("failed to generate configuration: %w", err))
	}

	flowsConfig, err := cm.getFlowsConfig(ctx)
	if err != nil {
		cm.logger.Warn("Failed to get flows config", "error", err)

		flowsConfig = make(map[string]interface{})
	}

	knowledgeBase, err := cm.getKnowledgeBaseFiles(ctx)
	if err != nil {
		cm.logger.Warn("Failed to get knowledge base files", "error", err)

		knowledgeBase = []sdk.KBFile{}
	}

	configPush := sdk.ConfigurationPush{
		BaseConfig:    convertToMap(config),
		FlowsConfig:   flowsConfig,
		KnowledgeBase: knowledgeBase,
		Timestamp:     time.Now().Unix(),
		Version:       fmt.Sprintf("v%d", time.Now().Unix()),
	}

	err = cm.nemoSDK.PushConfiguration(ctx, configPush)
	if err != nil {
		return errors.New(fmt.Sprintf("failed to push configuration to NeMo: %w", err))
	}

	cm.logger.Info("Configuration pushed to NeMo Guardrails successfully",
		"version", configPush.Version,
		"kb_files", len(configPush.KnowledgeBase),
	)

	return nil
}

func (cm *ConfigManager) getFlowsConfig(ctx context.Context) (map[string]interface{}, error) {
	pm := PageMetadata{}
	flows, err := cm.repo.GetFlows(ctx, pm)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("failed to get flows from database: %w", err))
	}

	flowsConfig := make(map[string]interface{})
	for _, flow := range flows {
		if flow.Active {
			flowsConfig[flow.Name] = flow.Content
		}
	}

	return flowsConfig, nil
}

func (cm *ConfigManager) getKnowledgeBaseFiles(ctx context.Context) ([]sdk.KBFile, error) {
	kbFiles, err := cm.repo.GetKBFiles(ctx, PageMetadata{})
	if err != nil {
		return nil, errors.New(fmt.Sprintf("failed to get KB files from database: %w", err))
	}

	sdkKBFiles := make([]sdk.KBFile, 0, len(kbFiles))
	for _, file := range kbFiles {
		if file.Active {
			sdkKBFiles = append(sdkKBFiles, sdk.KBFile{
				Name:    fmt.Sprintf("%s/%s", file.Category, file.Name),
				Content: file.Content,
				Type:    file.Type,
			})
		}
	}

	return sdkKBFiles, nil
}

func convertToMap(config *NeMoConfig) map[string]interface{} {
	result := make(map[string]interface{})

	data, err := json.Marshal(config)
	if err != nil {
		return result
	}

	err = json.Unmarshal(data, &result)
	if err != nil {
		return make(map[string]interface{})
	}

	return result
}

func (cm *ConfigManager) GetConfigYAML(ctx context.Context) ([]byte, error) {
	config, err := cm.GenerateNeMoConfig(ctx)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("failed to generate config: %w", err))
	}

	configYAML, err := yaml.Marshal(config)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("failed to marshal config to YAML: %w", err))
	}

	return configYAML, nil
}
