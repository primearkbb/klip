import { colors } from '@cliffy/ansi/colors';

export interface FormatOptions {
  indentSize?: number;
  maxWidth?: number;
  wrapCode?: boolean;
  showLineNumbers?: boolean;
}

export class ResponseFormatter {
  private indentSize: number;
  private maxWidth: number;
  private wrapCode: boolean;
  private showLineNumbers: boolean;

  constructor(options: FormatOptions = {}) {
    this.indentSize = options.indentSize ?? 4;
    this.maxWidth = options.maxWidth ?? (this.getTerminalWidth() - 8);
    this.wrapCode = options.wrapCode ?? false;
    this.showLineNumbers = options.showLineNumbers ?? false;
  }

  private getTerminalWidth(): number {
    if (Deno.stdout.isTerminal()) {
      return Deno.consoleSize().columns;
    }
    return 80; // fallback
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
    const gutter = colors.dim('│');
    const indent = ' '.repeat(this.indentSize);

    switch (level) {
      case 1:
        return `${gutter}${indent}${colors.bold(colors.brightBlue(content))}`;
      case 2:
        return `${gutter}${indent}${colors.bold(colors.blue(content))}`;
      case 3:
        return `${gutter}${indent}${colors.bold(colors.cyan(content))}`;
      default:
        return `${gutter}${indent}${colors.bold(content)}`;
    }
  }

  private formatListItem(text: string, depth: number): string {
    const content = text.replace(/^[-*]\s*/, '');
    const gutter = colors.dim('│');
    const indent = ' '.repeat(this.indentSize + depth * 2);
    const bullet = colors.cyan('•');
    return `${gutter}${indent}${bullet} ${content}`;
  }

  private formatOrderedListItem(text: string, depth: number): string {
    const match = text.match(/^(\d+)\.\s*(.*)/);
    if (!match) return text;

    const [, number, content] = match;
    const indent = ' '.repeat(this.indentSize + depth * 2);
    return `${indent}${colors.cyan(number + '.')} ${content}`;
  }

  private formatBlockquote(text: string): string {
    const content = text.replace(/^>\s*/, '');
    const indent = ' '.repeat(this.indentSize);
    return `${indent}${colors.dim('│')} ${colors.dim(content)}`;
  }

  private formatInlineCode(text: string): string {
    const indent = ' '.repeat(this.indentSize);
    return `${indent}${
      text.replace(/`([^`]+)`/g, colors.bgBlack(colors.brightWhite(' $1 ')))
    }`;
  }

  private formatCodeBlock(lines: string[], language: string): string {
    const indent = ' '.repeat(this.indentSize);
    const header = language
      ? `${indent}${colors.dim('┌─')} ${colors.cyan(language)} ${
        colors.dim(
          '─'.repeat(Math.max(0, this.maxWidth - language.length - 10)),
        )
      }`
      : `${indent}${
        colors.dim('┌' + '─'.repeat(Math.max(0, this.maxWidth - 6)))
      }`;

    const footer = `${indent}${
      colors.dim('└' + '─'.repeat(Math.max(0, this.maxWidth - 6)))
    }`;

    const formattedLines = lines.map((line, i) => {
      const lineNumber = this.showLineNumbers
        ? colors.dim((i + 1).toString().padStart(3) + ' │ ')
        : '';
      return `${indent}${colors.dim('│')} ${lineNumber}${
        colors.brightWhite(line)
      }`;
    });

    return [header, ...formattedLines, footer].join('\n');
  }

  private formatParagraph(text: string): string {
    const gutter = colors.dim('│');
    const indent = ' '.repeat(this.indentSize);

    // Handle inline formatting
    const formatted = text
      .replace(/\*\*(.*?)\*\*/g, colors.bold('$1'))
      .replace(/\*(.*?)\*/g, colors.italic('$1'))
      .replace(/`([^`]+)`/g, colors.bgBlack(colors.brightWhite(' $1 ')));

    // Word wrap if needed
    if (formatted.length > this.maxWidth - this.indentSize) {
      const words = formatted.split(' ');
      const wrapped: string[] = [];
      let currentLine = '';

      for (const word of words) {
        if ((currentLine + word).length > this.maxWidth - this.indentSize) {
          if (currentLine) {
            wrapped.push(currentLine.trim());
            currentLine = word + ' ';
          } else {
            wrapped.push(word);
          }
        } else {
          currentLine += word + ' ';
        }
      }

      if (currentLine.trim()) {
        wrapped.push(currentLine.trim());
      }

      return wrapped.map((line) => `${gutter}${indent}${line}`).join('\n');
    }

    return `${gutter}${indent}${formatted}`;
  }

  updateMaxWidth(width: number): void {
    this.maxWidth = width;
  }
}

export class StreamingFormatter {
  private buffer: string = '';
  private formatter: ResponseFormatter;
  private currentLine: string = '';

  constructor(options: FormatOptions = {}) {
    this.formatter = new ResponseFormatter(options);
  }

  addChunk(chunk: string): string {
    this.buffer += chunk;

    // Process complete lines
    const lines = this.buffer.split('\n');
    if (lines.length > 1) {
      // Keep the last incomplete line in buffer
      this.buffer = lines.pop() || '';

      // Process complete lines
      const completedContent = lines.join('\n');
      return this.formatter.formatResponse(completedContent);
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

  reset(): void {
    this.buffer = '';
    this.currentLine = '';
  }
}
