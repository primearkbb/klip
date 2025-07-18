import { colors } from '@cliffy/ansi/colors';

export function displayBanner(): void {
  const terminalSize = () => {
    if (Deno.stdout.isTerminal()) {
      return Deno.consoleSize();
    }
    return { columns: 80, rows: 24 }; // fallback
  };

  const terminalWidth = terminalSize().columns;

  // Simple, readable banner that works across terminal sizes
  const brandLine = 'KLIP';
  const tagLine = 'Terminal AI Chat Interface';
  const helpLine = 'Type /help for commands • Ctrl+C to exit';

  // Calculate padding based on terminal width
  const minWidth = Math.max(40, Math.min(terminalWidth - 4, 60));
  const padding = Math.floor((minWidth - brandLine.length) / 2);
  const tagPadding = Math.floor((minWidth - tagLine.length) / 2);
  const helpPadding = Math.floor((minWidth - helpLine.length) / 2);

  // Create responsive banner
  const topBorder = '╭' + '─'.repeat(minWidth) + '╮';
  const bottomBorder = '╰' + '─'.repeat(minWidth) + '╯';
  const emptyLine = '│' + ' '.repeat(minWidth) + '│';

  const banner = `
  ${colors.magenta(topBorder)}
  ${colors.magenta(emptyLine)}
  ${colors.magenta('│')}${' '.repeat(padding)}${colors.brightCyan(brandLine)}${
    ' '.repeat(minWidth - padding - brandLine.length)
  }${colors.magenta('│')}
  ${colors.magenta(emptyLine)}
  ${colors.magenta('│')}${' '.repeat(tagPadding)}${
    colors.brightGreen(tagLine)
  }${' '.repeat(minWidth - tagPadding - tagLine.length)}${colors.magenta('│')}
  ${colors.magenta(emptyLine)}
  ${colors.magenta(bottomBorder)}

  ${colors.dim(' '.repeat(helpPadding) + helpLine)}
`;

  console.log(banner);
}

export function displayHelp(): void {
  const help = `
  ${colors.brightBlue('Available Commands:')}

  ${colors.green('  /help')}      - Show this help message
  ${colors.green('  /models')}    - List available models
  ${colors.green('  /model')}     - Switch to a different model
  ${colors.green('  /clear')}     - Clear the chat history
  ${colors.green('  /keys')}      - Manage API keys
  ${colors.green('  /edit')}      - Edit and resend last message
  ${colors.green('  /quit')}      - Exit the application

  ${colors.brightBlue('Shortcuts:')}

  ${colors.yellow('  Ctrl+C')}    - Exit the application
  ${colors.yellow('  Ctrl+D')}    - Send message (when typing)
  ${colors.yellow('  Ctrl+L')}    - Clear screen
  ${colors.yellow('  ↑/↓')}       - Navigate message history

  ${colors.brightBlue('Usage:')}

    Just type your message and press Enter or Ctrl+D to send it.
    Use the commands above to control the chat behavior.
`;

  console.log(help);
}
