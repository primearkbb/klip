export class SimpleInput {
  async readLine(prompt: string): Promise<string | null> {
    // Use a more reliable input method
    Deno.stdout.write(new TextEncoder().encode(prompt));

    const buf = new Uint8Array(1024);
    const n = await Deno.stdin.read(buf);

    if (n === null) return null;

    const input = new TextDecoder().decode(buf.subarray(0, n));
    return input.trim();
  }

  readPassword(prompt: string): Promise<string | null> {
    // For password input, we'll use the same method but could enhance with masking
    return this.readLine(prompt);
  }

  readWithHistory(
    prompt: string,
    _history: string[] = [],
  ): Promise<string | null> {
    // Simplified version - just read a line for now
    return this.readLine(prompt);
  }
}

// Read a line from stdin with cursor positioning
export async function promptUser(message: string): Promise<string | null> {
  try {
    // Write the prompt
    await Deno.stdout.write(new TextEncoder().encode(message));

    // Read from stdin
    const buffer = new Uint8Array(1024);
    const bytesRead = await Deno.stdin.read(buffer);

    if (bytesRead === null) {
      return null;
    }

    const input = new TextDecoder().decode(buffer.subarray(0, bytesRead));
    
    // Clear the input line by moving cursor up and clearing
    await Deno.stdout.write(new TextEncoder().encode('\x1b[1A\x1b[2K'));
    
    return input.trim();
  } catch (error) {
    console.error('Input error:', error);
    return null;
  }
}
