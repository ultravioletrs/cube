// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

const STORAGE_KEY = 'cube_model_config'

export type LLMProvider = 'openai' | 'anthropic' | 'local'

export interface ModelConfig {
  // UI-level provider selection
  provider: LLMProvider
  model: string
  apiKey: string
  temperature: number
  maxTokens: number
  streamResponses: boolean
  systemPrompt: string
}

export const DEFAULT_MODEL_CONFIG: ModelConfig = {
  provider: 'local',
  model: '',
  apiKey: '',
  temperature: 0.2,
  maxTokens: 1024,
  streamResponses: true,
  systemPrompt:
    'You are a helpful, knowledgeable assistant. When document excerpts are provided, use them as your primary source and cite them, but also use your general knowledge to give complete, thorough answers.',
}

export function loadModelConfig(): ModelConfig {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return { ...DEFAULT_MODEL_CONFIG }
    return { ...DEFAULT_MODEL_CONFIG, ...(JSON.parse(raw) as Partial<ModelConfig>) }
  } catch {
    return { ...DEFAULT_MODEL_CONFIG }
  }
}

export function saveModelConfig(cfg: ModelConfig): void {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(cfg))
}

// Maps the UI provider to the backend provider string and base URL.
// "local" uses the server-configured Ollama endpoint (empty base_url = server default).
export function toBackendModelConfig(cfg: ModelConfig): {
  provider: string
  base_url: string
  model: string
  api_key: string
  temperature: number
  max_tokens: number
} | null {
  if (!cfg.model) return null
  const providerMap: Record<LLMProvider, string> = {
    openai: 'openai',
    anthropic: 'openai', // Anthropic is OpenAI-compatible
    local: 'ollama',
  }
  const baseURLMap: Record<LLMProvider, string> = {
    openai: 'https://api.openai.com',
    anthropic: 'https://api.anthropic.com',
    local: '', // use server default Ollama URL
  }
  return {
    provider: providerMap[cfg.provider],
    base_url: baseURLMap[cfg.provider],
    model: cfg.model,
    api_key: cfg.apiKey,
    temperature: cfg.temperature,
    max_tokens: cfg.maxTokens,
  }
}
