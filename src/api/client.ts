import type { Model } from './models.ts';
import { withRetry } from '../utils/retry.ts';
import type {
  AnalyticsLogger,
  RequestMetrics,
  ResponseMetrics,
} from '../storage/analytics.ts';

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
  enableWebSearch?: boolean;
}

export interface ChatResponse {
  content: string;
  usage?: {
    inputTokens: number;
    outputTokens: number;
  };
  metrics?: {
    latency_ms: number;
    tokens_input: number;
    tokens_output: number;
    response_length: number;
  };
}

export class ApiClient {
  private apiKey: string;
  private baseUrl: string = '';
  private headers: Record<string, string> = {};
  private currentProvider: 'anthropic' | 'openai' | 'openrouter' = 'anthropic';
  private analytics: AnalyticsLogger | null = null;

  constructor(
    apiKey: string,
    provider: 'anthropic' | 'openai' | 'openrouter',
    analytics?: AnalyticsLogger,
  ) {
    this.apiKey = apiKey;
    this.analytics = analytics || null;
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

  async chat(
    request: ChatRequest,
    signal?: AbortSignal,
  ): Promise<ChatResponse> {
    const startTime = Date.now();
    const requestMetrics = this.buildRequestMetrics(request, startTime);

    if (this.analytics) {
      await this.analytics.logRequest(requestMetrics);
    }

    let retryCount = 0;

    return withRetry(async () => {
      const payload = this.buildPayload(request);

      const endpoint = this.currentProvider === 'anthropic'
        ? '/messages'
        : '/chat/completions';

      try {
        const response = await fetch(`${this.baseUrl}${endpoint}`, {
          method: 'POST',
          headers: this.headers,
          body: JSON.stringify(payload),
          signal: signal || undefined,
        });

        if (!response.ok) {
          const error = await response.text();
          throw new Error(`API Error (${response.status}): ${error}`);
        }

        const data = await response.json();
        const endTime = Date.now();
        const chatResponse = this.parseResponse(data, request.model.provider);

        // Add metrics to response
        const latency = endTime - startTime;
        chatResponse.metrics = {
          latency_ms: latency,
          tokens_input: chatResponse.usage?.inputTokens || 0,
          tokens_output: chatResponse.usage?.outputTokens || 0,
          response_length: chatResponse.content.length,
        };

        // Log successful response
        if (this.analytics) {
          const responseMetrics: ResponseMetrics = {
            end_time: endTime,
            response_length: chatResponse.content.length,
            tokens_input: chatResponse.usage?.inputTokens,
            tokens_output: chatResponse.usage?.outputTokens,
            interrupted: false,
            success: true,
            retry_count: retryCount,
          };
          await this.analytics.logResponse(requestMetrics, responseMetrics);
        }

        return chatResponse;
      } catch (error) {
        retryCount++;
        const endTime = Date.now();

        // Log error response
        if (this.analytics) {
          const responseMetrics: ResponseMetrics = {
            end_time: endTime,
            response_length: 0,
            interrupted: false,
            success: false,
            error_type: error instanceof Error
              ? error.constructor.name
              : 'unknown',
            error_message: error instanceof Error
              ? error.message
              : String(error),
            status_code:
              (error instanceof Error && error.message.includes('API Error'))
                ? parseInt(error.message.match(/\((\d+)\)/)?.[1] || '0')
                : undefined,
            retry_count: retryCount,
          };
          await this.analytics.logResponse(requestMetrics, responseMetrics);
        }

        throw error;
      }
    });
  }

  async *chatStream(
    request: ChatRequest,
    signal?: AbortSignal,
  ): AsyncGenerator<string, ResponseMetrics, unknown> {
    const startTime = Date.now();
    const requestMetrics = this.buildRequestMetrics(request, startTime);

    if (this.analytics) {
      await this.analytics.logRequest(requestMetrics);
    }

    const combinedSignal = signal
      ? AbortSignal.any([signal, AbortSignal.timeout(120000)])
      : AbortSignal.timeout(120000);

    let retryCount = 0;
    let totalContent = '';
    let interrupted = false;

    try {
      const response = await withRetry(async () => {
        const payload = this.buildPayload({ ...request, stream: true });

        const endpoint = this.currentProvider === 'anthropic'
          ? '/messages'
          : '/chat/completions';

        try {
          const response = await fetch(`${this.baseUrl}${endpoint}`, {
            method: 'POST',
            headers: this.headers,
            body: JSON.stringify(payload),
            signal: combinedSignal,
          });

          if (!response.ok) {
            const error = await response.text();
            throw new Error(`API Error (${response.status}): ${error}`);
          }

          return response;
        } catch (error) {
          retryCount++;
          throw error;
        }
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
                const content = this.extractStreamContent(
                  data,
                  request.model.provider,
                );
                if (content) {
                  totalContent += content;
                  yield content;
                }
              } catch (_e) {
                // Skip invalid JSON
              }
            }
          }
        }
      } finally {
        reader.releaseLock();
      }

      // Stream completed successfully
      const endTime = Date.now();
      const responseMetrics: ResponseMetrics = {
        end_time: endTime,
        response_length: totalContent.length,
        interrupted: false,
        success: true,
        retry_count: retryCount,
      };

      if (this.analytics) {
        await this.analytics.logResponse(requestMetrics, responseMetrics);
      }

      return responseMetrics;
    } catch (error) {
      interrupted = true;
      const endTime = Date.now();

      const responseMetrics: ResponseMetrics = {
        end_time: endTime,
        response_length: totalContent.length,
        interrupted: true,
        success: false,
        error_type: error instanceof Error ? error.constructor.name : 'unknown',
        error_message: error instanceof Error ? error.message : String(error),
        status_code:
          (error instanceof Error && error.message.includes('API Error'))
            ? parseInt(error.message.match(/\((\d+)\)/)?.[1] || '0')
            : undefined,
        retry_count: retryCount,
      };

      if (this.analytics) {
        await this.analytics.logResponse(requestMetrics, responseMetrics);
      }

      throw error;
    }
  }

  private buildPayload(request: ChatRequest): Record<string, unknown> {
    if (request.model.provider === 'anthropic') {
      const systemMessage = request.messages.find((m) => m.role === 'system');
      const otherMessages = request.messages.filter((m) => m.role !== 'system');

      const payload: Record<string, unknown> = {
        model: request.model.id,
        max_tokens: request.maxTokens || request.model.maxTokens,
        messages: otherMessages.map((m) => ({
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

      // Add web search tool if enabled
      if (request.enableWebSearch) {
        payload.tools = [{
          type: 'web_search_20250305',
          name: 'web_search',
          max_uses: 5
        }];
      }

      return payload;
    } else {
      // OpenAI/OpenRouter format
      return {
        model: request.model.id,
        max_tokens: request.maxTokens || request.model.maxTokens,
        temperature: request.temperature || 0.7,
        stream: request.stream || false,
        messages: request.messages.map((m) => ({
          role: m.role,
          content: m.content,
        })),
      };
    }
  }

  private parseResponse(
    data: Record<string, unknown>,
    provider: string,
  ): ChatResponse {
    if (provider === 'anthropic') {
      const content = (data.content as Array<{ text: string }>)?.[0]?.text ||
        '';
      const usage = data.usage as {
        input_tokens: number;
        output_tokens: number;
      } | undefined;
      return {
        content,
        usage: usage
          ? {
            inputTokens: usage.input_tokens,
            outputTokens: usage.output_tokens,
          }
          : undefined,
      };
    } else {
      // OpenAI/OpenRouter format
      const choices = (data.choices as Array<{ message: { content: string } }>)
        ?.[0];
      const usage = data.usage as {
        prompt_tokens: number;
        completion_tokens: number;
      } | undefined;
      return {
        content: choices?.message?.content || '',
        usage: usage
          ? {
            inputTokens: usage.prompt_tokens,
            outputTokens: usage.completion_tokens,
          }
          : undefined,
      };
    }
  }

  private extractStreamContent(
    data: Record<string, unknown>,
    provider: string,
  ): string | null {
    if (provider === 'anthropic') {
      const delta = data.delta as { text?: string } | undefined;
      return delta?.text || null;
    } else {
      // OpenAI/OpenRouter format
      const choices = data.choices as
        | Array<{ delta?: { content?: string } }>
        | undefined;
      return choices?.[0]?.delta?.content || null;
    }
  }

  private buildRequestMetrics(
    request: ChatRequest,
    startTime: number,
  ): RequestMetrics {
    const systemMessage = request.messages.find((m) => m.role === 'system');
    const userMessages = request.messages.filter((m) => m.role === 'user');
    const lastUserMessage = userMessages[userMessages.length - 1];

    const totalLength = request.messages.reduce(
      (sum, msg) => sum + msg.content.length,
      0,
    );

    return {
      start_time: startTime,
      model: request.model,
      message_count: request.messages.length,
      user_message_length: lastUserMessage?.content.length || 0,
      total_conversation_length: totalLength,
      has_system_message: !!systemMessage,
      temperature: request.temperature,
      max_tokens: request.maxTokens,
      is_stream: request.stream || false,
    };
  }
}
