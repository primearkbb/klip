import type { Message } from '../api/client.ts';

export interface ChatLog {
  timestamp: number;
  sessionId: string;
  messages: Message[];
}

export class ChatLogger {
  private logDir: string;
  private sessionId: string;
  private currentLog: ChatLog;

  constructor() {
    const home = Deno.env.get('HOME') || Deno.env.get('USERPROFILE') || '/tmp';
    this.logDir = `${home}/.klip/logs`;
    this.sessionId = this.generateSessionId();
    this.currentLog = {
      timestamp: Date.now(),
      sessionId: this.sessionId,
      messages: [],
    };
  }

  async init(): Promise<void> {
    await this.ensureLogDir();
  }

  private async ensureLogDir(): Promise<void> {
    try {
      await Deno.stat(this.logDir);
    } catch {
      await Deno.mkdir(this.logDir, { recursive: true });
    }
  }

  private generateSessionId(): string {
    const timestamp = Date.now().toString(36);
    const random = Math.random().toString(36).substring(2, 8);
    return `${timestamp}-${random}`;
  }

  async logMessage(message: Message): Promise<void> {
    this.currentLog.messages.push(message);
    await this.saveLog();
  }

  async clearLog(): Promise<void> {
    this.currentLog.messages = [];
    await this.saveLog();
  }

  private async saveLog(): Promise<void> {
    const date = new Date().toISOString().split('T')[0];
    const filename = `${date}-${this.sessionId}.json`;
    const filepath = `${this.logDir}/${filename}`;

    const logData = {
      ...this.currentLog,
      lastUpdated: Date.now(),
    };

    await Deno.writeTextFile(filepath, JSON.stringify(logData, null, 2));
  }

  async getRecentLogs(limit = 10): Promise<ChatLog[]> {
    const logs: ChatLog[] = [];

    try {
      for await (const dirEntry of Deno.readDir(this.logDir)) {
        if (dirEntry.isFile && dirEntry.name.endsWith('.json')) {
          const filepath = `${this.logDir}/${dirEntry.name}`;
          const content = await Deno.readTextFile(filepath);
          const log = JSON.parse(content) as ChatLog;
          logs.push(log);
        }
      }
    } catch {
      // Directory doesn't exist or other error
      return [];
    }

    logs.sort((a, b) => b.timestamp - a.timestamp);
    return logs.slice(0, limit);
  }

  async exportLog(format: 'json' | 'txt' = 'json'): Promise<string> {
    const date = new Date().toISOString().split('T')[0];
    const filename = `klip-export-${date}-${this.sessionId}.${format}`;
    const filepath = `${this.logDir}/${filename}`;

    if (format === 'json') {
      await Deno.writeTextFile(
        filepath,
        JSON.stringify(this.currentLog, null, 2),
      );
    } else {
      const txtContent = this.formatAsText(this.currentLog);
      await Deno.writeTextFile(filepath, txtContent);
    }

    return filepath;
  }

  private formatAsText(log: ChatLog): string {
    const header = `Chat Log - ${new Date(log.timestamp).toLocaleString()}\n`;
    const separator = '='.repeat(50) + '\n';

    let content = header + separator + '\n';

    for (const message of log.messages) {
      const timestamp = new Date(message.timestamp).toLocaleString();
      const role = message.role.toUpperCase();

      content += `[${timestamp}] ${role}:\n`;
      content += message.content + '\n\n';
      content += '-'.repeat(30) + '\n\n';
    }

    return content;
  }
}
