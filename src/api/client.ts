import { Model } from './models.ts';
import { withRetry } from '../utils/retry.ts';

export interface Message {
  role: 'user' | 'assistant' | 'system';
  content: string;
  timestamp: number;
}

export interface ChatRequest {
  model: Model;
  messages: Message[];
  maxTokens?: number;
  temperature?: number;
  stream?: boolean;
}

export interface ChatResponse {
  content: string;
  usage?: {
    inputTokens: number;
    outputTokens: number;
  };
}

export class ApiClient {
  private apiKey: string;
  private baseUrl: string = '';
  private headers: Record<string, string> = {};
  private currentProvider: 'anthropic' | 'openai' | 'openrouter' = 'anthropic';

  constructor(apiKey: string, provider: 'anthropic' | 'openai' | 'openrouter') {
    this.apiKey = apiKey;
    this.setupProvider(provider);
  }

  private setupProvider(provider: 'anthropic' | 'openai' | 'openrouter') {
    this.currentProvider = provider;
    switch (provider) {
      case 'anthropic':
        this.baseUrl = 'https://api.anthropic.com/v1';
        this.headers = {
          'Content-Type': 'application/json',
          'x-api-key': this.apiKey,
          'anthropic-version': '2023-06-01',
        };
        break;
      case 'openai':
        this.baseUrl = 'https://api.openai.com/v1';
        this.headers = {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${this.apiKey}`,
        };
        break;
      case 'openrouter':
        this.baseUrl = 'https://openrouter.ai/api/v1';
        this.headers = {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${this.apiKey}`,
          'HTTP-Referer': 'https://github.com/your-username/klip',
          'X-Title': 'Klip Chat',
        };
        break;
    }
  }

  async chat(request: ChatRequest): Promise<ChatResponse> {
    return withRetry(async () => {
      const payload = this.buildPayload(request);
      
      const endpoint = this.currentProvider === 'anthropic' ? '/messages' : '/chat/completions';
      const response = await fetch(`${this.baseUrl}${endpoint}`, {
        method: 'POST',
        headers: this.headers,
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        const error = await response.text();
        throw new Error(`API Error (${response.status}): ${error}`);
      }

      const data = await response.json();
      return this.parseResponse(data, request.model.provider);
    });
  }

  async *chatStream(request: ChatRequest): AsyncGenerator<string, void, unknown> {
    const response = await withRetry(async () => {
      const payload = this.buildPayload({ ...request, stream: true });
      
      const endpoint = this.currentProvider === 'anthropic' ? '/messages' : '/chat/completions';
      const response = await fetch(`${this.baseUrl}${endpoint}`, {
        method: 'POST',
        headers: this.headers,
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        const error = await response.text();
        throw new Error(`API Error (${response.status}): ${error}`);
      }

      return response;
    });

    const reader = response.body?.getReader();
    if (!reader) throw new Error('No response stream available');

    const decoder = new TextDecoder();
    let buffer = '';

    try {
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() || '';

        for (const line of lines) {
          const trimmed = line.trim();
          if (trimmed === '' || trimmed === 'data: [DONE]') continue;
          
          if (trimmed.startsWith('data: ')) {
            try {
              const data = JSON.parse(trimmed.slice(6));
              const content = this.extractStreamContent(data, request.model.provider);
              if (content) yield content;
            } catch (e) {
              // Skip invalid JSON
            }
          }
        }
      }
    } finally {
      reader.releaseLock();
    }
  }

  private buildPayload(request: ChatRequest): any {
    if (request.model.provider === 'anthropic') {
      const systemMessage = request.messages.find(m => m.role === 'system');
      const otherMessages = request.messages.filter(m => m.role !== 'system');
      
      const payload: any = {
        model: request.model.id,
        max_tokens: request.maxTokens || request.model.maxTokens,
        messages: otherMessages.map(m => ({
          role: m.role,
          content: m.content,
        })),
      };
      
      if (systemMessage?.content) {
        payload.system = systemMessage.content;
      }
      
      if (request.stream) {
        payload.stream = true;
      }
      
      return payload;
    } else {
      // OpenAI/OpenRouter format
      return {
        model: request.model.id,
        max_tokens: request.maxTokens || request.model.maxTokens,
        temperature: request.temperature || 0.7,
        stream: request.stream || false,
        messages: request.messages.map(m => ({
          role: m.role,
          content: m.content,
        })),
      };
    }
  }

  private parseResponse(data: any, provider: string): ChatResponse {
    if (provider === 'anthropic') {
      return {
        content: data.content[0].text,
        usage: data.usage ? {
          inputTokens: data.usage.input_tokens,
          outputTokens: data.usage.output_tokens,
        } : undefined,
      };
    } else {
      // OpenAI/OpenRouter format
      return {
        content: data.choices[0].message.content,
        usage: data.usage ? {
          inputTokens: data.usage.prompt_tokens,
          outputTokens: data.usage.completion_tokens,
        } : undefined,
      };
    }
  }

  private extractStreamContent(data: any, provider: string): string | null {
    if (provider === 'anthropic') {
      return data.delta?.text || null;
    } else {
      // OpenAI/OpenRouter format
      return data.choices?.[0]?.delta?.content || null;
    }
  }
}