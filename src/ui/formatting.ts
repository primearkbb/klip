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
  const formatter = new StreamingFormatter({ messageType: 'user' });
  return formatter.formatContent(content);
}

export function formatAssistantMessage(content: string): string {
  const formatter = new StreamingFormatter({ messageType: 'assistant' });
  return formatter.formatContent(content);
}

export function formatThinkingMessage(content: string): string {
  const formatter = new StreamingFormatter({ messageType: 'thinking' });
  return formatter.formatContent(content);
}

export class StreamingFormatter {
  private buffer: string = '';
  private messageType: 'user' | 'assistant' | 'thinking' = 'assistant';
  private isFirstChunk: boolean = true;
  private maxWidth: number;
  private baseIndent: string = '  ';

  constructor(options: FormatOptions = {}) {
    this.messageType = options.messageType ?? 'assistant';
    this.maxWidth = Math.floor(this.getTerminalWidth() * 0.85);
  }

  private getTerminalWidth(): number {
    if (Deno.stdout.isTerminal()) {
      return Deno.consoleSize().columns;
    }
    return 80;
  }

  private getSpeakerHeader(): string {
    switch (this.messageType) {
      case 'user':
        return colors.brightBlue('*');
      case 'thinking':
        return colors.dim('~');
      case 'assistant':
      default:
        return colors.dim('~');
    }
  }

  private getSpeakerLabel(): string {
    switch (this.messageType) {
      case 'user':
        return colors.brightBlue(' You: ');
      case 'thinking':
        return colors.dim(' Thinking');
      case 'assistant':
      default:
        return colors.dim(' Klip');
    }
  }

  private wrapText(text: string, firstLinePrefix: string, continuationPrefix: string): string[] {
    const words = text.split(' ');
    const lines: string[] = [];
    let currentLine = '';
    let isFirstLine = true;
    
    const getAvailableWidth = (isFirst: boolean) => {
      const prefix = isFirst ? firstLinePrefix : continuationPrefix;
      return this.maxWidth - prefix.length;
    };

    for (const word of words) {
      const testLine = currentLine ? `${currentLine} ${word}` : word;
      const availableWidth = getAvailableWidth(isFirstLine);
      
      if (testLine.length > availableWidth && currentLine) {
        lines.push((isFirstLine ? firstLinePrefix : continuationPrefix) + currentLine);
        currentLine = word;
        isFirstLine = false;
      } else {
        currentLine = testLine;
      }
    }
    
    if (currentLine) {
      lines.push((isFirstLine ? firstLinePrefix : continuationPrefix) + currentLine);
    }
    
    return lines;
  }

  addChunk(chunk: string): string {
    this.buffer += chunk;
    const lines = this.buffer.split('\n');
    
    if (lines.length > 1) {
      // Keep the last incomplete line in buffer
      this.buffer = lines.pop() || '';
      
      // Format complete lines, preserving original line breaks
      const completedContent = lines.join('\n');
      if (completedContent.trim()) {
        const result = this.formatContent(completedContent);
        return result;
      }
    }

    return '';
  }

  formatContent(content: string): string {
    if (!content.trim()) return '';
    
    let result = '';
    
    // Add speaker header only for first chunk
    if (this.isFirstChunk) {
      const header = this.getSpeakerHeader();
      const label = this.getSpeakerLabel();
      
      if (this.messageType === 'user') {
        // Check if content is multiline or will wrap
        const lines = content.split('\n');
        const hasMultipleLines = lines.length > 1;
        const firstLineLength = (header + label + lines[0]).length;
        const willWrap = firstLineLength > this.maxWidth;
        
        if (hasMultipleLines || willWrap) {
          // Multiline format: header + label on own line, then content
          result = header + label + '\n';
          result += this.formatMultilineContent(content);
        } else {
          // Single line format: header + label + content
          result = header + label + content;
        }
      } else {
        // For assistant messages, header + label on its own line
        result = header + label + '\n';
        result += this.formatMultilineContent(content);
      }
      this.isFirstChunk = false;
    } else {
      // Continuation content without speaker header
      result = this.formatMultilineContent(content);
    }
    
    return result + '\n';
  }

  private formatMultilineContent(content: string): string {
    const lines = content.split('\n');
    const formattedLines: string[] = [];
    
    for (const line of lines) {
      if (line.trim() === '') {
        // Preserve empty lines for spacing
        formattedLines.push('');
      } else {
        // Apply indentation and wrapping to non-empty lines
        const wrappedLines = this.wrapText(line, this.baseIndent, this.baseIndent);
        formattedLines.push(...wrappedLines);
      }
    }
    
    return formattedLines.join('\n');
  }

  finalize(): string {
    if (this.buffer.trim()) {
      const result = this.formatContent(this.buffer);
      this.buffer = '';
      return result;
    }
    return '';
  }

  setMessageType(type: 'user' | 'assistant' | 'thinking'): void {
    this.messageType = type;
    this.isFirstChunk = true;
  }

  reset(): void {
    this.buffer = '';
    this.isFirstChunk = true;
  }
}
