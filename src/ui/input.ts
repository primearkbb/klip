import { colors } from '@cliffy/ansi/colors';

export class InputHandler {
  private history: string[] = [];
  private historyIndex = -1;
  private currentInput = '';
  private cursorPosition = 0;
  private interrupted = false;

  async readInput(prompt: string): Promise<string | null> {
    return new Promise((resolve) => {
      Deno.stdout.write(new TextEncoder().encode(prompt));
      
      const stdin = Deno.stdin;
      const decoder = new TextDecoder();
      
      // Add a timeout to prevent hanging
      const timeout = setTimeout(() => {
        console.log('\n⚠️  Input timeout - press Ctrl+C to exit');
      }, 30000); // 30 second timeout warning
      
      const handleKeypress = async () => {
        const buffer = new Uint8Array(1024);
        const bytesRead = await stdin.read(buffer);
        
        if (bytesRead === null) {
          clearTimeout(timeout);
          resolve(null);
          return;
        }
        
        const input = decoder.decode(buffer.subarray(0, bytesRead));
        
        // Handle special keys
        if (input === '\u0003') { // Ctrl+C
          clearTimeout(timeout);
          this.interrupted = true;
          resolve(null);
          return;
        }
        
        if (input === '\u0004') { // Ctrl+D
          if (this.currentInput.trim()) {
            clearTimeout(timeout);
            this.addToHistory(this.currentInput);
            const result = this.currentInput;
            this.currentInput = '';
            this.cursorPosition = 0;
            this.historyIndex = -1;
            console.log();
            resolve(result);
            return;
          }
        }
        
        if (input === '\r' || input === '\n') {
          if (this.currentInput.trim()) {
            clearTimeout(timeout);
            this.addToHistory(this.currentInput);
            const result = this.currentInput;
            this.currentInput = '';
            this.cursorPosition = 0;
            this.historyIndex = -1;
            console.log();
            resolve(result);
            return;
          }
        }
        
        // Handle arrow keys
        if (input === '\u001b[A') { // Up arrow
          this.navigateHistory(-1);
          this.redrawLine(prompt);
          handleKeypress();
          return;
        }
        
        if (input === '\u001b[B') { // Down arrow
          this.navigateHistory(1);
          this.redrawLine(prompt);
          handleKeypress();
          return;
        }
        
        if (input === '\u001b[C') { // Right arrow
          if (this.cursorPosition < this.currentInput.length) {
            this.cursorPosition++;
            this.redrawLine(prompt);
          }
          handleKeypress();
          return;
        }
        
        if (input === '\u001b[D') { // Left arrow
          if (this.cursorPosition > 0) {
            this.cursorPosition--;
            this.redrawLine(prompt);
          }
          handleKeypress();
          return;
        }
        
        // Handle backspace
        if (input === '\u007f' || input === '\b') {
          if (this.cursorPosition > 0) {
            this.currentInput = 
              this.currentInput.slice(0, this.cursorPosition - 1) + 
              this.currentInput.slice(this.cursorPosition);
            this.cursorPosition--;
            this.redrawLine(prompt);
          }
          handleKeypress();
          return;
        }
        
        // Handle regular characters
        if (input.length === 1 && input >= ' ') {
          this.currentInput = 
            this.currentInput.slice(0, this.cursorPosition) + 
            input + 
            this.currentInput.slice(this.cursorPosition);
          this.cursorPosition++;
          this.redrawLine(prompt);
          handleKeypress();
          return;
        }
        
        // Continue reading
        handleKeypress();
      };
      
      handleKeypress();
    });
  }

  private addToHistory(input: string): void {
    const trimmed = input.trim();
    if (trimmed && (!this.history.length || this.history[this.history.length - 1] !== trimmed)) {
      this.history.push(trimmed);
      if (this.history.length > 100) {
        this.history.shift();
      }
    }
  }

  private navigateHistory(direction: number): void {
    if (this.history.length === 0) return;
    
    if (this.historyIndex === -1) {
      this.historyIndex = this.history.length - 1;
    } else {
      this.historyIndex += direction;
    }
    
    if (this.historyIndex < 0) {
      this.historyIndex = 0;
    } else if (this.historyIndex >= this.history.length) {
      this.historyIndex = this.history.length - 1;
    }
    
    this.currentInput = this.history[this.historyIndex];
    this.cursorPosition = this.currentInput.length;
  }

  private redrawLine(prompt: string): void {
    // Clear the line
    Deno.stdout.write(new TextEncoder().encode('\r\x1b[K'));
    
    // Write the prompt and current input
    Deno.stdout.write(new TextEncoder().encode(prompt + this.currentInput));
    
    // Move cursor to the correct position
    if (this.cursorPosition < this.currentInput.length) {
      const moveCursor = this.currentInput.length - this.cursorPosition;
      Deno.stdout.write(new TextEncoder().encode(`\x1b[${moveCursor}D`));
    }
  }

  isInterrupted(): boolean {
    return this.interrupted;
  }

  resetInterrupted(): void {
    this.interrupted = false;
  }
}