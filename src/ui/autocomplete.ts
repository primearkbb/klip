import { colors } from '@cliffy/ansi/colors';
import { getAllModels } from '../api/models.ts';

export class AutocompleteInput {
  private suggestions: string[] = [];
  private currentInput = '';
  private selectedSuggestion = -1;
  private showingSuggestions = false;

  async readModelInput(prompt: string): Promise<string | null> {
    const models = getAllModels();
    const modelIds = models.map(m => m.id);
    
    return new Promise((resolve) => {
      Deno.stdout.write(new TextEncoder().encode(prompt));
      
      const stdin = Deno.stdin;
      const decoder = new TextDecoder();
      
      const handleKeypress = async () => {
        const buffer = new Uint8Array(1024);
        const bytesRead = await stdin.read(buffer);
        
        if (bytesRead === null) {
          resolve(null);
          return;
        }
        
        const input = decoder.decode(buffer.subarray(0, bytesRead));
        
        // Handle Ctrl+C
        if (input === '\u0003') {
          resolve(null);
          return;
        }
        
        // Handle Enter
        if (input === '\r' || input === '\n') {
          if (this.selectedSuggestion >= 0 && this.suggestions.length > 0) {
            const result = this.suggestions[this.selectedSuggestion];
            this.clearSuggestions();
            console.log();
            resolve(result);
            return;
          } else if (this.currentInput.trim()) {
            this.clearSuggestions();
            console.log();
            resolve(this.currentInput);
            return;
          }
        }
        
        // Handle Tab - select first suggestion
        if (input === '\t') {
          if (this.suggestions.length > 0) {
            this.selectedSuggestion = 0;
            this.currentInput = this.suggestions[0];
            this.redrawInput(prompt);
            this.showSuggestions();
          }
          handleKeypress();
          return;
        }
        
        // Handle arrow keys
        if (input === '\u001b[A') { // Up arrow
          this.navigateSuggestions(-1);
          this.showSuggestions();
          handleKeypress();
          return;
        }
        
        if (input === '\u001b[B') { // Down arrow
          this.navigateSuggestions(1);
          this.showSuggestions();
          handleKeypress();
          return;
        }
        
        // Handle backspace
        if (input === '\u007f' || input === '\b') {
          if (this.currentInput.length > 0) {
            this.currentInput = this.currentInput.slice(0, -1);
            this.updateSuggestions(modelIds);
            this.redrawInput(prompt);
            this.showSuggestions();
          }
          handleKeypress();
          return;
        }
        
        // Handle regular characters
        if (input.length === 1 && input >= ' ') {
          this.currentInput += input;
          this.updateSuggestions(modelIds);
          this.redrawInput(prompt);
          this.showSuggestions();
          handleKeypress();
          return;
        }
        
        // Continue reading
        handleKeypress();
      };
      
      handleKeypress();
    });
  }

  private updateSuggestions(modelIds: string[]): void {
    if (this.currentInput.trim() === '') {
      this.suggestions = [];
      this.selectedSuggestion = -1;
      return;
    }
    
    const input = this.currentInput.toLowerCase();
    this.suggestions = modelIds
      .filter(id => id.toLowerCase().includes(input))
      .sort((a, b) => {
        // Prioritize exact matches at the start
        const aStartsWith = a.toLowerCase().startsWith(input);
        const bStartsWith = b.toLowerCase().startsWith(input);
        
        if (aStartsWith && !bStartsWith) return -1;
        if (!aStartsWith && bStartsWith) return 1;
        
        // Then sort by length (shorter first)
        return a.length - b.length;
      })
      .slice(0, 8); // Show max 8 suggestions
    
    this.selectedSuggestion = this.suggestions.length > 0 ? 0 : -1;
  }

  private navigateSuggestions(direction: number): void {
    if (this.suggestions.length === 0) return;
    
    this.selectedSuggestion += direction;
    
    if (this.selectedSuggestion < 0) {
      this.selectedSuggestion = this.suggestions.length - 1;
    } else if (this.selectedSuggestion >= this.suggestions.length) {
      this.selectedSuggestion = 0;
    }
    
    this.currentInput = this.suggestions[this.selectedSuggestion];
    this.redrawInput(this.getLastPrompt());
  }

  private lastPrompt = '';
  private getLastPrompt(): string {
    return this.lastPrompt;
  }

  private redrawInput(prompt: string): void {
    this.lastPrompt = prompt;
    
    // Clear the current line
    Deno.stdout.write(new TextEncoder().encode('\r\x1b[K'));
    
    // Write the prompt and current input
    Deno.stdout.write(new TextEncoder().encode(prompt + this.currentInput));
  }

  private showSuggestions(): void {
    if (this.suggestions.length === 0) {
      this.clearSuggestions();
      return;
    }
    
    if (this.showingSuggestions) {
      // Move cursor up to clear previous suggestions
      Deno.stdout.write(new TextEncoder().encode(`\x1b[${this.suggestions.length + 1}A`));
    }
    
    // Move to next line and show suggestions
    Deno.stdout.write(new TextEncoder().encode('\n'));
    
    for (let i = 0; i < this.suggestions.length; i++) {
      const suggestion = this.suggestions[i];
      const isSelected = i === this.selectedSuggestion;
      
      if (isSelected) {
        Deno.stdout.write(new TextEncoder().encode(colors.bgBlue(colors.white(`  ${suggestion}  `)) + '\n'));
      } else {
        Deno.stdout.write(new TextEncoder().encode(colors.dim(`  ${suggestion}`) + '\n'));
      }
    }
    
    // Move cursor back up to input line
    Deno.stdout.write(new TextEncoder().encode(`\x1b[${this.suggestions.length}A`));
    
    this.showingSuggestions = true;
  }

  private clearSuggestions(): void {
    if (this.showingSuggestions) {
      // Move down to clear suggestions
      Deno.stdout.write(new TextEncoder().encode(`\x1b[${this.suggestions.length}B`));
      
      // Clear each suggestion line
      for (let i = 0; i < this.suggestions.length; i++) {
        Deno.stdout.write(new TextEncoder().encode('\x1b[K\n'));
      }
      
      // Move back up
      Deno.stdout.write(new TextEncoder().encode(`\x1b[${this.suggestions.length}A`));
      
      this.showingSuggestions = false;
    }
  }
}