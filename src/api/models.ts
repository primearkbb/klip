export interface Model {
  id: string;
  name: string;
  provider: 'anthropic' | 'openai' | 'openrouter';
  maxTokens: number;
  contextWindow: number;
}

export const MODELS: Record<string, Model> = {
  // Anthropic Models
  'claude-3-5-sonnet-20241022': {
    id: 'claude-3-5-sonnet-20241022',
    name: 'Claude 3.5 Sonnet',
    provider: 'anthropic',
    maxTokens: 8192,
    contextWindow: 200000,
  },
  'claude-3-5-haiku-20241022': {
    id: 'claude-3-5-haiku-20241022',
    name: 'Claude 3.5 Haiku',
    provider: 'anthropic',
    maxTokens: 8192,
    contextWindow: 200000,
  },
  'claude-3-opus-20240229': {
    id: 'claude-3-opus-20240229',
    name: 'Claude 3 Opus',
    provider: 'anthropic',
    maxTokens: 4096,
    contextWindow: 200000,
  },

  // OpenAI Models
  'gpt-4-turbo-2024-04-09': {
    id: 'gpt-4-turbo-2024-04-09',
    name: 'GPT-4 Turbo',
    provider: 'openai',
    maxTokens: 4096,
    contextWindow: 128000,
  },
  'gpt-4o': {
    id: 'gpt-4o',
    name: 'GPT-4o',
    provider: 'openai',
    maxTokens: 16384,
    contextWindow: 128000,
  },
  'gpt-4o-mini': {
    id: 'gpt-4o-mini',
    name: 'GPT-4o Mini',
    provider: 'openai',
    maxTokens: 16384,
    contextWindow: 128000,
  },
  'gpt-3.5-turbo': {
    id: 'gpt-3.5-turbo',
    name: 'GPT-3.5 Turbo',
    provider: 'openai',
    maxTokens: 4096,
    contextWindow: 16385,
  },

  // OpenRouter Models (using OpenAI-compatible format)
  'openrouter/anthropic/claude-3.5-sonnet': {
    id: 'anthropic/claude-3.5-sonnet',
    name: 'Claude 3.5 Sonnet (OpenRouter)',
    provider: 'openrouter',
    maxTokens: 8192,
    contextWindow: 200000,
  },
  'openrouter/openai/gpt-4o': {
    id: 'openai/gpt-4o',
    name: 'GPT-4o (OpenRouter)',
    provider: 'openrouter',
    maxTokens: 16384,
    contextWindow: 128000,
  },
  'openrouter/meta-llama/llama-3.1-405b-instruct': {
    id: 'meta-llama/llama-3.1-405b-instruct',
    name: 'Llama 3.1 405B (OpenRouter)',
    provider: 'openrouter',
    maxTokens: 4096,
    contextWindow: 131072,
  },
};

export function getModelsByProvider(provider: 'anthropic' | 'openai' | 'openrouter'): Model[] {
  return Object.values(MODELS).filter(model => model.provider === provider);
}

export function getModel(id: string): Model | undefined {
  return MODELS[id];
}

export function getAllModels(): Model[] {
  return Object.values(MODELS);
}

export function getDefaultModel(): Model {
  return MODELS['claude-3-5-sonnet-20241022'];
}