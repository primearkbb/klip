export interface Model {
  id: string;
  name: string;
  provider: 'anthropic' | 'openai' | 'openrouter';
  maxTokens: number;
  contextWindow: number;
}

export interface OpenRouterModel {
  id: string;
  name: string;
  description: string;
  context_length: number;
  architecture: {
    modality: string;
    tokenizer: string;
    instruct_type: string;
  };
  pricing: {
    prompt: string;
    completion: string;
  };
  top_provider: {
    max_completion_tokens: number;
  };
}

export const MODELS: Record<string, Model> = {
  // Anthropic Models - Claude 4 Series
  'claude-opus-4-20250514': {
    id: 'claude-opus-4-20250514',
    name: 'Claude Opus 4',
    provider: 'anthropic',
    maxTokens: 32000,
    contextWindow: 200000,
  },
  'claude-sonnet-4-20250514': {
    id: 'claude-sonnet-4-20250514',
    name: 'Claude Sonnet 4',
    provider: 'anthropic',
    maxTokens: 8192,
    contextWindow: 200000,
  },
  
  // Anthropic Models - Claude 3.7 Series
  'claude-3-7-sonnet-20250219': {
    id: 'claude-3-7-sonnet-20250219',
    name: 'Claude 3.7 Sonnet',
    provider: 'anthropic',
    maxTokens: 8192,
    contextWindow: 200000,
  },
  
  // Anthropic Models - Claude 3.5 Series
  'claude-3-5-sonnet-20241022': {
    id: 'claude-3-5-sonnet-20241022',
    name: 'Claude 3.5 Sonnet (v2)',
    provider: 'anthropic',
    maxTokens: 8192,
    contextWindow: 200000,
  },
  'claude-3-5-sonnet-20240620': {
    id: 'claude-3-5-sonnet-20240620',
    name: 'Claude 3.5 Sonnet (v1)',
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
  
  // Anthropic Models - Claude 3 Series
  'claude-3-opus-20240229': {
    id: 'claude-3-opus-20240229',
    name: 'Claude 3 Opus',
    provider: 'anthropic',
    maxTokens: 4096,
    contextWindow: 200000,
  },
  'claude-3-haiku-20240307': {
    id: 'claude-3-haiku-20240307',
    name: 'Claude 3 Haiku',
    provider: 'anthropic',
    maxTokens: 4096,
    contextWindow: 200000,
  },

  // OpenAI Models
  'gpt-4.1': {
    id: 'gpt-4.1',
    name: 'GPT-4.1',
    provider: 'openai',
    maxTokens: 16384,
    contextWindow: 1000000,
  },
  'gpt-4.1-mini': {
    id: 'gpt-4.1-mini',
    name: 'GPT-4.1 Mini',
    provider: 'openai',
    maxTokens: 16384,
    contextWindow: 1000000,
  },
  'gpt-4.1-nano': {
    id: 'gpt-4.1-nano',
    name: 'GPT-4.1 Nano',
    provider: 'openai',
    maxTokens: 16384,
    contextWindow: 1000000,
  },
  'o3': {
    id: 'o3',
    name: 'OpenAI o3',
    provider: 'openai',
    maxTokens: 16384,
    contextWindow: 200000,
  },
  'o3-pro': {
    id: 'o3-pro',
    name: 'OpenAI o3 Pro',
    provider: 'openai',
    maxTokens: 16384,
    contextWindow: 200000,
  },
  'o4-mini': {
    id: 'o4-mini',
    name: 'OpenAI o4 Mini',
    provider: 'openai',
    maxTokens: 16384,
    contextWindow: 200000,
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
  return MODELS['claude-sonnet-4-20250514'];
}

let openRouterModelsCache: Model[] | null = null;
let openRouterModelsCacheTime: number = 0;
const CACHE_DURATION = 5 * 60 * 1000; // 5 minutes in milliseconds

export async function fetchOpenRouterModels(): Promise<Model[]> {
  const now = Date.now();
  
  // Return cached models if cache is still valid
  if (openRouterModelsCache && (now - openRouterModelsCacheTime) < CACHE_DURATION) {
    return openRouterModelsCache;
  }

  try {
    const response = await fetch('https://openrouter.ai/api/v1/models', {
      headers: {
        'Authorization': `Bearer ${Deno.env.get('OPENROUTER_API_KEY') || ''}`,
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      throw new Error(`Failed to fetch OpenRouter models: ${response.status}`);
    }

    const data = await response.json();
    const openRouterModels: OpenRouterModel[] = data.data || [];

    // Convert OpenRouter models to our Model interface
    const convertedModels: Model[] = openRouterModels.map((model: OpenRouterModel) => ({
      id: model.id,
      name: model.name,
      provider: 'openrouter' as const,
      maxTokens: model.top_provider?.max_completion_tokens || 4096,
      contextWindow: model.context_length || 4096,
    }));

    // Cache the results
    openRouterModelsCache = convertedModels;
    openRouterModelsCacheTime = now;

    return convertedModels;
  } catch (error) {
    console.error('Error fetching OpenRouter models:', error);
    
    // Fallback to static OpenRouter models if dynamic fetch fails
    return Object.values(MODELS).filter(model => model.provider === 'openrouter');
  }
}

export async function getAllModelsWithOpenRouter(): Promise<Model[]> {
  const staticModels = Object.values(MODELS).filter(model => model.provider !== 'openrouter');
  const openRouterModels = await fetchOpenRouterModels();
  
  return [...staticModels, ...openRouterModels];
}