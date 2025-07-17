import { colors } from '@cliffy/ansi/colors';

export interface RetryOptions {
  maxRetries?: number;
  baseDelay?: number;
  maxDelay?: number;
  backoffFactor?: number;
  retryCondition?: (error: Error) => boolean;
}

export class RetryError extends Error {
  constructor(message: string, public readonly attempts: number, public readonly lastError: Error) {
    super(message);
    this.name = 'RetryError';
  }
}

export async function withRetry<T>(
  operation: () => Promise<T>,
  options: RetryOptions = {}
): Promise<T> {
  const {
    maxRetries = 3,
    baseDelay = 1000,
    maxDelay = 30000,
    backoffFactor = 2,
    retryCondition = (error) => isRetryableError(error)
  } = options;

  let lastError: Error;
  let delay = baseDelay;

  for (let attempt = 0; attempt <= maxRetries; attempt++) {
    try {
      return await operation();
    } catch (error) {
      lastError = error as Error;
      
      if (attempt === maxRetries) {
        throw new RetryError(
          `Operation failed after ${attempt + 1} attempts`,
          attempt + 1,
          lastError
        );
      }

      if (!retryCondition(lastError)) {
        throw lastError;
      }

      console.log(colors.yellow(`\nRetrying in ${delay}ms... (attempt ${attempt + 1}/${maxRetries + 1})`));
      console.log(colors.dim(`Error: ${lastError.message}`));
      
      await sleep(delay);
      delay = Math.min(delay * backoffFactor, maxDelay);
    }
  }

  throw lastError!;
}

export function isRetryableError(error: Error): boolean {
  const message = error.message.toLowerCase();
  
  // Network errors
  if (message.includes('network') || 
      message.includes('connection') || 
      message.includes('timeout') ||
      message.includes('econnreset') ||
      message.includes('enotfound')) {
    return true;
  }
  
  // HTTP status codes that should be retried
  if (message.includes('500') || // Internal Server Error
      message.includes('502') || // Bad Gateway
      message.includes('503') || // Service Unavailable
      message.includes('504') || // Gateway Timeout
      message.includes('429')) { // Rate Limited
    return true;
  }
  
  // API-specific errors
  if (message.includes('rate limit') ||
      message.includes('overloaded') ||
      message.includes('temporarily unavailable')) {
    return true;
  }
  
  return false;
}

function sleep(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms));
}

export class InterruptibleOperation<T> {
  private abortController: AbortController;
  private interrupted = false;

  constructor() {
    this.abortController = new AbortController();
    this.setupSignalHandlers();
  }

  private setupSignalHandlers(): void {
    const handleSignal = () => {
      this.interrupt();
    };

    // Handle Ctrl+C
    Deno.addSignalListener('SIGINT', handleSignal);
    
    // Handle termination
    Deno.addSignalListener('SIGTERM', handleSignal);
  }

  interrupt(): void {
    this.interrupted = true;
    this.abortController.abort();
  }

  isInterrupted(): boolean {
    return this.interrupted;
  }

  getSignal(): AbortSignal {
    return this.abortController.signal;
  }

  async execute(operation: (signal: AbortSignal) => Promise<T>): Promise<T | null> {
    try {
      return await operation(this.abortController.signal);
    } catch (error) {
      if (error instanceof Error && error.name === 'AbortError') {
        return null;
      }
      throw error;
    }
  }
}