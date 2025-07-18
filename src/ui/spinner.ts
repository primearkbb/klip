import { colors } from '@cliffy/ansi/colors';

export class Spinner {
  private frames = ['⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'];
  private interval: number | null = null;
  private currentFrame = 0;
  private message: string;
  private isRunning = false;

  constructor(message: string) {
    this.message = message;
  }

  start(): void {
    if (this.isRunning) return;

    this.isRunning = true;
    this.render();

    this.interval = setInterval(() => {
      this.currentFrame = (this.currentFrame + 1) % this.frames.length;
      this.render();
    }, 100);
  }

  stop(): void {
    if (!this.isRunning) return;

    this.isRunning = false;

    if (this.interval) {
      clearInterval(this.interval);
      this.interval = null;
    }

    // Clear the spinner line completely
    Deno.stdout.write(new TextEncoder().encode('\r\x1b[2K'));
  }

  succeed(message?: string): void {
    this.stop();
    const msg = message || this.message;
    console.log(colors.green(`✓ ${msg}`));
  }

  fail(message?: string): void {
    this.stop();
    const msg = message || this.message;
    console.log(colors.red(`✗ ${msg}`));
  }

  info(message?: string): void {
    this.stop();
    const msg = message || this.message;
    console.log(colors.blue(`ℹ ${msg}`));
  }

  warn(message?: string): void {
    this.stop();
    const msg = message || this.message;
    console.log(colors.yellow(`⚠ ${msg}`));
  }

  updateMessage(message: string): void {
    this.message = message;
    if (this.isRunning) {
      this.render();
    }
  }

  private render(): void {
    const frame = this.frames[this.currentFrame];
    const output = `\r${colors.cyan(frame)} ${colors.dim(this.message)}`;
    Deno.stdout.write(new TextEncoder().encode(output));
  }
}

export class ProgressBar {
  private width: number;
  private current: number;
  private total: number;
  private message: string;

  constructor(total: number, message: string, width = 30) {
    this.total = total;
    this.current = 0;
    this.message = message;
    this.width = width;
  }

  update(current: number, message?: string): void {
    this.current = Math.min(current, this.total);
    if (message) this.message = message;
    this.render();
  }

  increment(message?: string): void {
    this.update(this.current + 1, message);
  }

  complete(message?: string): void {
    this.update(this.total, message);
    console.log(); // New line after completion
  }

  private render(): void {
    const percentage = Math.round((this.current / this.total) * 100);
    const filled = Math.round((this.current / this.total) * this.width);
    const empty = this.width - filled;

    const bar = colors.green('█'.repeat(filled)) +
      colors.dim('░'.repeat(empty));
    const output = `\r${bar} ${percentage}% ${colors.dim(this.message)}`;

    Deno.stdout.write(new TextEncoder().encode(output));
  }
}

export async function withSpinner<T>(
  message: string,
  operation: () => Promise<T>,
  options: {
    successMessage?: string;
    failureMessage?: string;
    timeout?: number;
  } = {},
): Promise<T> {
  const spinner = new Spinner(message);
  spinner.start();

  try {
    let result: T;

    if (options.timeout) {
      const timeoutPromise = new Promise<never>((_, reject) => {
        setTimeout(
          () => reject(new Error('Operation timed out')),
          options.timeout,
        );
      });

      result = await Promise.race([operation(), timeoutPromise]);
    } else {
      result = await operation();
    }

    spinner.succeed(options.successMessage);
    return result;
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    spinner.fail(options.failureMessage || `Failed: ${errorMessage}`);
    throw error;
  }
}

export function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
