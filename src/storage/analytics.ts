import type { Model } from '../api/models.ts';

export interface AnalyticsEvent {
  timestamp: number;
  event_type:
    | 'request'
    | 'response'
    | 'error'
    | 'session_start'
    | 'session_end'
    | 'model_switch'
    | 'command_usage';
  session_id: string;
  model_id?: string;
  model_name?: string;
  provider?: 'anthropic' | 'openai' | 'openrouter';
  request_data?: {
    message_count: number;
    user_message_length: number;
    total_conversation_length: number;
    has_system_message: boolean;
    temperature?: number;
    max_tokens?: number;
  };
  response_data?: {
    response_length: number;
    tokens_input?: number;
    tokens_output?: number;
    total_tokens?: number;
    latency_ms: number;
    is_stream: boolean;
    interrupted: boolean;
  };
  error_data?: {
    error_type: string;
    error_message: string;
    status_code?: number;
    retry_count: number;
  };
  cost_data?: {
    estimated_cost_input?: number;
    estimated_cost_output?: number;
    estimated_cost_total?: number;
    currency: string;
  };
  command_data?: {
    command: string;
    success: boolean;
    execution_time_ms: number;
  };
  metadata?: {
    user_agent?: string;
    platform?: string;
    app_version?: string;
    [key: string]: unknown;
  };
}

export interface AnalyticsConfig {
  enabled: boolean;
  retain_days: number;
  max_file_size_mb: number;
  enable_cost_tracking: boolean;
  anonymize_content: boolean;
}

export interface RequestMetrics {
  start_time: number;
  model: Model;
  message_count: number;
  user_message_length: number;
  total_conversation_length: number;
  has_system_message: boolean;
  temperature?: number;
  max_tokens?: number;
  is_stream: boolean;
}

export interface ResponseMetrics {
  end_time: number;
  response_length: number;
  tokens_input?: number;
  tokens_output?: number;
  interrupted: boolean;
  success: boolean;
  error_type?: string;
  error_message?: string;
  status_code?: number;
  retry_count: number;
}

// Cost estimates per 1M tokens (approximate values, should be updated regularly)
const COST_ESTIMATES: Record<
  string,
  { input: number; output: number; currency: string }
> = {
  // Anthropic Claude models (USD per 1M tokens)
  'claude-opus-4-20250514': { input: 15.0, output: 75.0, currency: 'USD' },
  'claude-sonnet-4-20250514': { input: 3.0, output: 15.0, currency: 'USD' },
  'claude-3-7-sonnet-20250219': { input: 3.0, output: 15.0, currency: 'USD' },
  'claude-3-5-sonnet-20241022': { input: 3.0, output: 15.0, currency: 'USD' },
  'claude-3-5-sonnet-20240620': { input: 3.0, output: 15.0, currency: 'USD' },
  'claude-3-5-haiku-20241022': { input: 1.0, output: 5.0, currency: 'USD' },
  'claude-3-opus-20240229': { input: 15.0, output: 75.0, currency: 'USD' },
  'claude-3-haiku-20240307': { input: 0.25, output: 1.25, currency: 'USD' },

  // OpenAI models (USD per 1M tokens)
  'gpt-4.1': { input: 10.0, output: 30.0, currency: 'USD' },
  'gpt-4.1-mini': { input: 0.15, output: 0.6, currency: 'USD' },
  'gpt-4.1-nano': { input: 0.075, output: 0.3, currency: 'USD' },
  'o3': { input: 60.0, output: 240.0, currency: 'USD' },
  'o3-pro': { input: 200.0, output: 800.0, currency: 'USD' },
  'o4-mini': { input: 0.15, output: 0.6, currency: 'USD' },
  'gpt-4o': { input: 2.5, output: 10.0, currency: 'USD' },
  'gpt-4o-mini': { input: 0.15, output: 0.6, currency: 'USD' },

  // OpenRouter models (varies, using approximate values)
  'anthropic/claude-3.5-sonnet': { input: 3.0, output: 15.0, currency: 'USD' },
  'openai/gpt-4o': { input: 2.5, output: 10.0, currency: 'USD' },
  'meta-llama/llama-3.1-405b-instruct': {
    input: 2.7,
    output: 2.7,
    currency: 'USD',
  },
};

export class AnalyticsLogger {
  private analyticsDir: string;
  private sessionId: string;
  private config: AnalyticsConfig;
  private currentDate: string;
  private pendingEvents: AnalyticsEvent[] = [];
  private writeQueue: Promise<void> = Promise.resolve();

  constructor(config: Partial<AnalyticsConfig> = {}) {
    const home = Deno.env.get('HOME') || Deno.env.get('USERPROFILE') || '/tmp';
    this.analyticsDir = `${home}/.klip/analytics`;
    this.sessionId = this.generateSessionId();
    this.currentDate = this.getCurrentDate();
    this.config = {
      enabled: true,
      retain_days: 30,
      max_file_size_mb: 10,
      enable_cost_tracking: true,
      anonymize_content: false,
      ...config,
    };
  }

  async init(): Promise<void> {
    if (!this.config.enabled) return;

    await this.ensureAnalyticsDir();
    await this.logEvent({
      timestamp: Date.now(),
      event_type: 'session_start',
      session_id: this.sessionId,
      metadata: {
        platform: Deno.build.os,
        app_version: '1.0.0', // Should be dynamic
      },
    });

    // Start cleanup routine
    this.startCleanupRoutine();
  }

  async logRequest(metrics: RequestMetrics): Promise<void> {
    if (!this.config.enabled) return;

    const event: AnalyticsEvent = {
      timestamp: metrics.start_time,
      event_type: 'request',
      session_id: this.sessionId,
      model_id: metrics.model.id,
      model_name: metrics.model.name,
      provider: metrics.model.provider,
      request_data: {
        message_count: metrics.message_count,
        user_message_length: metrics.user_message_length,
        total_conversation_length: metrics.total_conversation_length,
        has_system_message: metrics.has_system_message,
        temperature: metrics.temperature,
        max_tokens: metrics.max_tokens,
      },
      response_data: {
        response_length: 0,
        latency_ms: 0,
        is_stream: metrics.is_stream,
        interrupted: false,
      },
    };

    await this.logEvent(event);
  }

  async logResponse(
    requestMetrics: RequestMetrics,
    responseMetrics: ResponseMetrics,
  ): Promise<void> {
    if (!this.config.enabled) return;

    const latency = responseMetrics.end_time - requestMetrics.start_time;
    const costData = this.calculateCost(
      requestMetrics.model.id,
      responseMetrics.tokens_input,
      responseMetrics.tokens_output,
    );

    const event: AnalyticsEvent = {
      timestamp: responseMetrics.end_time,
      event_type: responseMetrics.success ? 'response' : 'error',
      session_id: this.sessionId,
      model_id: requestMetrics.model.id,
      model_name: requestMetrics.model.name,
      provider: requestMetrics.model.provider,
      request_data: {
        message_count: requestMetrics.message_count,
        user_message_length: requestMetrics.user_message_length,
        total_conversation_length: requestMetrics.total_conversation_length,
        has_system_message: requestMetrics.has_system_message,
        temperature: requestMetrics.temperature,
        max_tokens: requestMetrics.max_tokens,
      },
      response_data: {
        response_length: responseMetrics.response_length,
        tokens_input: responseMetrics.tokens_input,
        tokens_output: responseMetrics.tokens_output,
        total_tokens: (responseMetrics.tokens_input || 0) +
          (responseMetrics.tokens_output || 0),
        latency_ms: latency,
        is_stream: requestMetrics.is_stream,
        interrupted: responseMetrics.interrupted,
      },
      error_data: responseMetrics.success ? undefined : {
        error_type: responseMetrics.error_type || 'unknown',
        error_message: responseMetrics.error_message || 'Unknown error',
        status_code: responseMetrics.status_code,
        retry_count: responseMetrics.retry_count,
      },
      cost_data: costData,
    };

    await this.logEvent(event);
  }

  async logCommand(
    command: string,
    success: boolean,
    executionTimeMs: number,
  ): Promise<void> {
    if (!this.config.enabled) return;

    const event: AnalyticsEvent = {
      timestamp: Date.now(),
      event_type: 'command_usage',
      session_id: this.sessionId,
      command_data: {
        command,
        success,
        execution_time_ms: executionTimeMs,
      },
    };

    await this.logEvent(event);
  }

  async logModelSwitch(oldModel: Model, newModel: Model): Promise<void> {
    if (!this.config.enabled) return;

    const event: AnalyticsEvent = {
      timestamp: Date.now(),
      event_type: 'model_switch',
      session_id: this.sessionId,
      model_id: newModel.id,
      model_name: newModel.name,
      provider: newModel.provider,
      metadata: {
        previous_model_id: oldModel.id,
        previous_model_name: oldModel.name,
        previous_provider: oldModel.provider,
      },
    };

    await this.logEvent(event);
  }

  async logSessionEnd(): Promise<void> {
    if (!this.config.enabled) return;

    const event: AnalyticsEvent = {
      timestamp: Date.now(),
      event_type: 'session_end',
      session_id: this.sessionId,
      metadata: {
        session_duration_ms: Date.now() -
          parseInt(this.sessionId.split('-')[0], 36),
      },
    };

    await this.logEvent(event);

    // Wait for all pending writes to complete
    await this.writeQueue;
  }

  private async logEvent(event: AnalyticsEvent): Promise<void> {
    if (!this.config.enabled) return;

    // Queue the event for writing
    this.pendingEvents.push(event);

    // Chain the write operation
    this.writeQueue = this.writeQueue.then(() => this.flushEvents());
  }

  private async flushEvents(): Promise<void> {
    if (this.pendingEvents.length === 0) return;

    const eventsToWrite = [...this.pendingEvents];
    this.pendingEvents = [];

    const currentDate = this.getCurrentDate();
    if (currentDate !== this.currentDate) {
      this.currentDate = currentDate;
    }

    const filename = `analytics-${this.currentDate}.jsonl`;
    const filepath = `${this.analyticsDir}/${filename}`;

    try {
      // Check if we need to rotate the file (size limit)
      let shouldRotate = false;
      try {
        const stat = await Deno.stat(filepath);
        const sizeMB = stat.size / (1024 * 1024);
        if (sizeMB > this.config.max_file_size_mb) {
          shouldRotate = true;
        }
      } catch {
        // File doesn't exist, no rotation needed
      }

      if (shouldRotate) {
        const rotatedFilename =
          `analytics-${this.currentDate}-${Date.now()}.jsonl`;
        const rotatedFilepath = `${this.analyticsDir}/${rotatedFilename}`;
        try {
          await Deno.rename(filepath, rotatedFilepath);
        } catch {
          // Rotation failed, continue with current file
        }
      }

      // Write events as JSON Lines
      const jsonLines = eventsToWrite.map((event) =>
        JSON.stringify(event)
      ).join('\n') + '\n';

      await Deno.writeTextFile(filepath, jsonLines, { append: true });
    } catch (error) {
      console.error('Failed to write analytics events:', error);
    }
  }

  private calculateCost(
    modelId: string,
    inputTokens?: number,
    outputTokens?: number,
  ): AnalyticsEvent['cost_data'] {
    if (!this.config.enable_cost_tracking || !inputTokens || !outputTokens) {
      return undefined;
    }

    const costInfo = COST_ESTIMATES[modelId];
    if (!costInfo) {
      return undefined;
    }

    const inputCost = (inputTokens / 1_000_000) * costInfo.input;
    const outputCost = (outputTokens / 1_000_000) * costInfo.output;
    const totalCost = inputCost + outputCost;

    return {
      estimated_cost_input: inputCost,
      estimated_cost_output: outputCost,
      estimated_cost_total: totalCost,
      currency: costInfo.currency,
    };
  }

  private async ensureAnalyticsDir(): Promise<void> {
    try {
      await Deno.stat(this.analyticsDir);
    } catch {
      await Deno.mkdir(this.analyticsDir, { recursive: true });
    }
  }

  private generateSessionId(): string {
    const timestamp = Date.now().toString(36);
    const random = Math.random().toString(36).substring(2, 8);
    return `${timestamp}-${random}`;
  }

  private getCurrentDate(): string {
    return new Date().toISOString().split('T')[0];
  }

  private startCleanupRoutine(): void {
    // Run cleanup every 24 hours
    setInterval(() => {
      this.cleanupOldFiles().catch((error) => {
        console.error('Analytics cleanup failed:', error);
      });
    }, 24 * 60 * 60 * 1000);

    // Also run cleanup on startup
    this.cleanupOldFiles().catch((error) => {
      console.error('Initial analytics cleanup failed:', error);
    });
  }

  private async cleanupOldFiles(): Promise<void> {
    if (!this.config.enabled) return;

    const cutoffDate = new Date();
    cutoffDate.setDate(cutoffDate.getDate() - this.config.retain_days);
    const cutoffDateStr = cutoffDate.toISOString().split('T')[0];

    try {
      for await (const dirEntry of Deno.readDir(this.analyticsDir)) {
        if (
          dirEntry.isFile && dirEntry.name.startsWith('analytics-') &&
          dirEntry.name.endsWith('.jsonl')
        ) {
          // Extract date from filename
          const dateMatch = dirEntry.name.match(
            /analytics-(\d{4}-\d{2}-\d{2})/,
          );
          if (dateMatch && dateMatch[1] < cutoffDateStr) {
            try {
              await Deno.remove(`${this.analyticsDir}/${dirEntry.name}`);
            } catch {
              // Ignore errors when deleting old files
            }
          }
        }
      }
    } catch {
      // Directory doesn't exist or other error, ignore
    }
  }

  // Utility methods for analytics queries
  async getAnalyticsData(
    startDate?: string,
    endDate?: string,
    eventType?: string,
  ): Promise<AnalyticsEvent[]> {
    const events: AnalyticsEvent[] = [];

    if (!this.config.enabled) return events;

    try {
      for await (const dirEntry of Deno.readDir(this.analyticsDir)) {
        if (dirEntry.isFile && dirEntry.name.endsWith('.jsonl')) {
          const filepath = `${this.analyticsDir}/${dirEntry.name}`;
          const content = await Deno.readTextFile(filepath);

          const lines = content.split('\n').filter((line) => line.trim());
          for (const line of lines) {
            try {
              const event = JSON.parse(line) as AnalyticsEvent;

              // Apply filters
              if (
                startDate &&
                new Date(event.timestamp).toISOString().split('T')[0] <
                  startDate
              ) continue;
              if (
                endDate &&
                new Date(event.timestamp).toISOString().split('T')[0] > endDate
              ) continue;
              if (eventType && event.event_type !== eventType) continue;

              events.push(event);
            } catch {
              // Skip malformed lines
            }
          }
        }
      }
    } catch {
      // Directory doesn't exist or other error
    }

    return events.sort((a, b) => a.timestamp - b.timestamp);
  }

  async getUsageStats(days: number = 7): Promise<{
    total_requests: number;
    total_tokens: number;
    total_cost: number;
    avg_latency: number;
    error_rate: number;
    models_used: Record<string, number>;
    daily_usage: Record<string, number>;
  }> {
    const startDate = new Date();
    startDate.setDate(startDate.getDate() - days);
    const startDateStr = startDate.toISOString().split('T')[0];

    const events = await this.getAnalyticsData(startDateStr);

    const stats = {
      total_requests: 0,
      total_tokens: 0,
      total_cost: 0,
      avg_latency: 0,
      error_rate: 0,
      models_used: {} as Record<string, number>,
      daily_usage: {} as Record<string, number>,
    };

    let totalLatency = 0;
    let requestCount = 0;
    let errorCount = 0;

    for (const event of events) {
      const date = new Date(event.timestamp).toISOString().split('T')[0];

      if (event.event_type === 'request') {
        stats.total_requests++;
        requestCount++;

        if (event.model_id) {
          stats.models_used[event.model_id] =
            (stats.models_used[event.model_id] || 0) + 1;
        }

        stats.daily_usage[date] = (stats.daily_usage[date] || 0) + 1;
      } else if (event.event_type === 'response') {
        if (event.response_data?.latency_ms) {
          totalLatency += event.response_data.latency_ms;
        }
        if (event.response_data?.total_tokens) {
          stats.total_tokens += event.response_data.total_tokens;
        }
        if (event.cost_data?.estimated_cost_total) {
          stats.total_cost += event.cost_data.estimated_cost_total;
        }
      } else if (event.event_type === 'error') {
        errorCount++;
      }
    }

    stats.avg_latency = requestCount > 0 ? totalLatency / requestCount : 0;
    stats.error_rate = requestCount > 0 ? errorCount / requestCount : 0;

    return stats;
  }
}
