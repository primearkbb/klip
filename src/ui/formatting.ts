import { colors } from '@cliffy/ansi/colors';

export interface FormatOptions {
  indentSize?: number;
  maxWidth?: number;
  wrapCode?: boolean;
  showLineNumbers?: boolean;
  leftMargin?: number;
  messageType?: 'user' | 'assistant' | 'thinking';
}

export class ResponseFormatter {
  private indentSize: number;
  private maxWidth: number;
  private wrapCode: boolean;
  private showLineNumbers: boolean;
  private leftMargin: number;
  private messageType: 'user' | 'assistant' | 'thinking';

  constructor(options: FormatOptions = {}) {
    this.indentSize = options.indentSize ?? 4;
    this.leftMargin = options.leftMargin ?? 2;
    this.maxWidth = options.maxWidth ??
      (this.getTerminalWidth() - this.leftMargin - 8);
    this.wrapCode = options.wrapCode ?? false;
    this.showLineNumbers = options.showLineNumbers ?? false;
    this.messageType = options.messageType ?? 'assistant';
  }

  private getTerminalWidth(): number {
    if (Deno.stdout.isTerminal()) {
      return Deno.consoleSize().columns;
    }
    return 80; // fallback
  }

  private getLeftMargin(): string {
    return ' '.repeat(this.leftMargin);
  }

  private getGutter(): string {
    switch (this.messageType) {
      case 'user':
        return colors.brightBlue('│');
      case 'thinking':
        return colors.dim('┊');
      case 'assistant':
      default:
        return colors.dim('│');
    }
  }

  formatResponse(content: string): string {
    const lines = content.split('\n');
    const formatted: string[] = [];
    let inCodeBlock = false;
    let codeBlockLanguage = '';
    let codeBlockLines: string[] = [];
    const _listDepth = 0;

    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];
      const trimmed = line.trim();

      // Handle code blocks
      if (trimmed.startsWith('```')) {
        if (inCodeBlock) {
          // End of code block
          formatted.push(
            this.formatCodeBlock(codeBlockLines, codeBlockLanguage),
          );
          inCodeBlock = false;
          codeBlockLines = [];
          codeBlockLanguage = '';
        } else {
          // Start of code block
          inCodeBlock = true;
          codeBlockLanguage = trimmed.slice(3);
        }
        continue;
      }

      if (inCodeBlock) {
        codeBlockLines.push(line);
        continue;
      }

      // Handle different line types
      if (trimmed === '') {
        formatted.push('');
      } else if (trimmed.startsWith('# ')) {
        formatted.push(this.formatHeading(trimmed, 1));
      } else if (trimmed.startsWith('## ')) {
        formatted.push(this.formatHeading(trimmed, 2));
      } else if (trimmed.startsWith('### ')) {
        formatted.push(this.formatHeading(trimmed, 3));
      } else if (trimmed.startsWith('- ') || trimmed.startsWith('* ')) {
        formatted.push(this.formatListItem(trimmed, 0));
      } else if (trimmed.match(/^\d+\. /)) {
        formatted.push(this.formatOrderedListItem(trimmed, 0));
      } else if (trimmed.startsWith('> ')) {
        formatted.push(this.formatBlockquote(trimmed));
      } else if (
        trimmed.startsWith('`') && trimmed.endsWith('`') &&
        !trimmed.includes('\n')
      ) {
        formatted.push(this.formatInlineCode(line));
      } else {
        formatted.push(this.formatParagraph(line));
      }
    }

    // Handle any remaining code block
    if (inCodeBlock && codeBlockLines.length > 0) {
      formatted.push(this.formatCodeBlock(codeBlockLines, codeBlockLanguage));
    }

    return formatted.join('\n');
  }

  private formatHeading(text: string, level: number): string {
    const content = text.replace(/^#+\s*/, '');
    const leftMargin = this.getLeftMargin();
    const gutter = this.getGutter();
    const indent = ' '.repeat(this.indentSize);

    switch (level) {
      case 1:
        return `${leftMargin}${gutter}${indent}${
          colors.bold(colors.brightBlue(content))
        }`;
      case 2:
        return `${leftMargin}${gutter}${indent}${
          colors.bold(colors.blue(content))
        }`;
      case 3:
        return `${leftMargin}${gutter}${indent}${
          colors.bold(colors.cyan(content))
        }`;
      default:
        return `${leftMargin}${gutter}${indent}${colors.bold(content)}`;
    }
  }

  private formatListItem(text: string, depth: number): string {
    const content = text.replace(/^[-*]\s*/, '');
    const leftMargin = this.getLeftMargin();
    const gutter = this.getGutter();
    const indent = ' '.repeat(this.indentSize + depth * 2);
    const bullet = colors.cyan('•');
    return `${leftMargin}${gutter}${indent}${bullet} ${content}`;
  }

  private formatOrderedListItem(text: string, depth: number): string {
    const match = text.match(/^(\d+)\.\s*(.*)/);
    if (!match) return text;

    const [, number, content] = match;
    const leftMargin = this.getLeftMargin();
    const gutter = this.getGutter();
    const indent = ' '.repeat(this.indentSize + depth * 2);
    return `${leftMargin}${gutter}${indent}${
      colors.cyan(number + '.')
    } ${content}`;
  }

  private formatBlockquote(text: string): string {
    const content = text.replace(/^>\s*/, '');
    const leftMargin = this.getLeftMargin();
    const indent = ' '.repeat(this.indentSize);
    return `${leftMargin}${indent}${colors.dim('│')} ${colors.dim(content)}`;
  }

  private formatInlineCode(text: string): string {
    const leftMargin = this.getLeftMargin();
    const gutter = this.getGutter();
    const indent = ' '.repeat(this.indentSize);
    return `${leftMargin}${gutter}${indent}${
      text.replace(/`([^`]+)`/g, colors.bgBlack(colors.brightWhite(' $1 ')))
    }`;
  }

  private formatCodeBlock(lines: string[], language: string): string {
    const leftMargin = this.getLeftMargin();
    const indent = ' '.repeat(this.indentSize);
    const availableWidth = this.maxWidth - this.leftMargin - this.indentSize;

    const header = language
      ? `${leftMargin}${indent}${colors.dim('┌─')} ${colors.cyan(language)} ${
        colors.dim(
          '─'.repeat(Math.max(0, availableWidth - language.length - 10)),
        )
      }`
      : `${leftMargin}${indent}${
        colors.dim('┌' + '─'.repeat(Math.max(0, availableWidth - 6)))
      }`;

    const footer = `${leftMargin}${indent}${
      colors.dim('└' + '─'.repeat(Math.max(0, availableWidth - 6)))
    }`;

    const formattedLines = lines.map((line, i) => {
      const lineNumber = this.showLineNumbers
        ? colors.dim((i + 1).toString().padStart(3) + ' │ ')
        : '';
      return `${leftMargin}${indent}${colors.dim('│')} ${lineNumber}${
        colors.brightWhite(line)
      }`;
    });

    return [header, ...formattedLines, footer].join('\n');
  }

  private formatParagraph(text: string): string {
    const leftMargin = this.getLeftMargin();
    const gutter = this.getGutter();
    const indent = ' '.repeat(this.indentSize);

    // Handle inline formatting
    const formatted = text
      .replace(/\*\*(.*?)\*\*/g, colors.bold('$1'))
      .replace(/\*(.*?)\*/g, colors.italic('$1'))
      .replace(/`([^`]+)`/g, colors.bgBlack(colors.brightWhite(' $1 ')));

    // Calculate available width for text
    const availableWidth = this.maxWidth - this.leftMargin - this.indentSize -
      2; // 2 for gutter

    // Word wrap if needed
    if (this.getTextLength(formatted) > availableWidth) {
      const wrapped = this.wrapWords(formatted, availableWidth);
      return wrapped.map((line) => `${leftMargin}${gutter}${indent}${line}`)
        .join('\n');
    }

    return `${leftMargin}${gutter}${indent}${formatted}`;
  }

  private getTextLength(text: string): number {
    // Remove ANSI escape codes to get actual text length
    return text.replace(/\x1b\[[0-9;]*m/g, '').length;
  }

  private wrapWords(text: string, maxWidth: number): string[] {
    const words = text.split(' ');
    const wrapped: string[] = [];
    let currentLine = '';

    for (const word of words) {
      const testLine = currentLine ? `${currentLine} ${word}` : word;

      if (this.getTextLength(testLine) > maxWidth) {
        if (currentLine) {
          wrapped.push(currentLine);
          currentLine = word;
        } else {
          // Word is too long, break it up
          const chunks = this.breakLongWord(word, maxWidth);
          wrapped.push(...chunks.slice(0, -1));
          currentLine = chunks[chunks.length - 1];
        }
      } else {
        currentLine = testLine;
      }
    }

    if (currentLine) {
      wrapped.push(currentLine);
    }

    return wrapped;
  }

  private breakLongWord(word: string, maxWidth: number): string[] {
    const chunks: string[] = [];
    let remaining = word;

    while (this.getTextLength(remaining) > maxWidth) {
      let chunk = '';
      let i = 0;

      while (
        i < remaining.length &&
        this.getTextLength(chunk + remaining[i]) <= maxWidth
      ) {
        chunk += remaining[i];
        i++;
      }

      if (chunk) {
        chunks.push(chunk);
        remaining = remaining.slice(i);
      } else {
        // Single character is too wide, just take it
        chunks.push(remaining[0]);
        remaining = remaining.slice(1);
      }
    }

    if (remaining) {
      chunks.push(remaining);
    }

    return chunks;
  }

  updateMaxWidth(width: number): void {
    this.maxWidth = width;
  }

  setMessageType(type: 'user' | 'assistant' | 'thinking'): void {
    this.messageType = type;
  }
}

// Helper functions for message formatting
export function formatUserMessage(content: string): string {
  const formatter = new ResponseFormatter({ messageType: 'user' });
  return formatter.formatResponse(content);
}

export function formatAssistantMessage(content: string): string {
  const formatter = new ResponseFormatter({ messageType: 'assistant' });
  return formatter.formatResponse(content);
}

export function formatThinkingMessage(content: string): string {
  const formatter = new ResponseFormatter({ messageType: 'thinking' });
  const leftMargin = '  ';
  const lines = content.split('\n');
  const formattedLines = lines.map((line) => {
    if (line.trim()) {
      return `${leftMargin}${colors.dim('┊')} ${colors.dim(line)}`;
    }
    return '';
  });
  return formattedLines.join('\n');
}

export class StreamingFormatter {
  private buffer: string = '';
  private formatter: ResponseFormatter;
  private currentLine: string = '';
  private options: FormatOptions;

  constructor(options: FormatOptions = {}) {
    this.options = options;
    this.formatter = new ResponseFormatter(options);
  }

  addChunk(chunk: string): string {
    this.buffer += chunk;

    // For streaming, we want to format each chunk as it comes
    // but only return complete formatted lines
    const lines = this.buffer.split('\n');

    if (lines.length > 1) {
      // Keep the last incomplete line in buffer
      this.buffer = lines.pop() || '';

      // Format and return complete lines
      const completedContent = lines.join('\n');
      if (completedContent.trim()) {
        return this.formatter.formatResponse(completedContent);
      }
    }

    return '';
  }

  finalize(): string {
    if (this.buffer.trim()) {
      const final = this.formatter.formatResponse(this.buffer);
      this.buffer = '';
      return final;
    }
    return '';
  }

  setMessageType(type: 'user' | 'assistant' | 'thinking'): void {
    this.formatter.setMessageType(type);
  }

  // Check if current buffer contains thinking tags
  private detectThinkingContent(): boolean {
    const content = this.buffer.toLowerCase();
    return content.includes('<thinking>') || content.includes('<think>') ||
      content.includes('<thinking>');
  }

  // Handle thinking message formatting
  processThinkingContent(content: string): string {
    // Simple thinking content detection and formatting
    if (
      content.includes('<thinking>') || content.includes('<think>') ||
      content.includes('<thinking>')
    ) {
      return formatThinkingMessage(content);
    }
    return this.formatter.formatResponse(content);
  }

  reset(): void {
    this.buffer = '';
    this.currentLine = '';
  }
}
