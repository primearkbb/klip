import { colors } from '@cliffy/ansi/colors';

export function displayBanner(): void {
  const banner = `
${colors.magenta('  ╭────────────────────────────────────────────────────────────╮')}
${colors.magenta('  │')}                                                            ${colors.magenta('│')}
${colors.magenta('  │')}      ${colors.brightCyan('██╗  ██╗██╗     ██╗██████╗')}                        ${colors.magenta('│')}
${colors.magenta('  │')}      ${colors.brightCyan('██║ ██╔╝██║     ██║██╔══██╗')}                       ${colors.magenta('│')}
${colors.magenta('  │')}      ${colors.brightCyan('█████╔╝ ██║     ██║██████╔╝')}                       ${colors.magenta('│')}
${colors.magenta('  │')}      ${colors.brightCyan('██╔═██╗ ██║     ██║██╔═══╝')}                        ${colors.magenta('│')}
${colors.magenta('  │')}      ${colors.brightCyan('██║  ██╗███████╗██║██║')}                            ${colors.magenta('│')}
${colors.magenta('  │')}      ${colors.brightCyan('╚═╝  ╚═╝╚══════╝╚═╝╚═╝')}                            ${colors.magenta('│')}
${colors.magenta('  │')}                                                            ${colors.magenta('│')}
${colors.magenta('  │')}          ${colors.brightGreen('⚡ Terminal AI Chat Interface ⚡')}              ${colors.magenta('│')}
${colors.magenta('  │')}                                                            ${colors.magenta('│')}
${colors.magenta('  ╰────────────────────────────────────────────────────────────╯')}

${colors.dim('  Type /help for commands • Ctrl+C to exit • Tab for autocomplete')}
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