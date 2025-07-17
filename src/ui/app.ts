import { colors } from '@cliffy/ansi/colors';
import { KeyStore } from '../storage/keystore.ts';
import { getDefaultModel, getAllModels, getAllModelsWithOpenRouter, getModel, type Model } from '../api/models.ts';
import { ApiClient, type Message, type ChatRequest } from '../api/client.ts';
import { InputHandler } from './input.ts';
import { displayHelp } from './banner.ts';
import { ChatLogger } from '../storage/logger.ts';
import { InterruptibleOperation } from '../utils/retry.ts';
import { AutocompleteInput } from './autocomplete.ts';
import { Spinner, withSpinner } from './spinner.ts';
import { promptUser } from './simple-input.ts';

export class App {
  private keyStore: KeyStore;
  private currentModel: Model;
  private client: ApiClient | null = null;
  private messages: Message[] = [];
  private inputHandler: InputHandler;
  private logger: ChatLogger;
  private interruptibleOp: InterruptibleOperation<string> | null = null;
  private autocomplete: AutocompleteInput;

  constructor() {
    this.keyStore = new KeyStore();
    this.currentModel = getDefaultModel();
    this.inputHandler = new InputHandler();
    this.logger = new ChatLogger();
    this.autocomplete = new AutocompleteInput();
  }

  async run(): Promise<void> {
    this.setupSignalHandlers();
    
    await withSpinner('Initializing keystore...', () => this.keyStore.init(), {
      successMessage: 'Keystore initialized'
    });
    
    await withSpinner('Setting up chat logger...', () => this.logger.init(), {
      successMessage: 'Chat logger ready'
    });
    
    // Check for API key
    if (!(await this.keyStore.hasKey(this.currentModel.provider))) {
      await this.setupApiKey(this.currentModel.provider);
    }
    
    await withSpinner('Initializing API client...', () => this.initializeClient(), {
      successMessage: 'API client ready'
    });
    
    console.log(colors.green(`\n✓ Using model: ${this.currentModel.name}`));
    console.log(colors.dim('Type /help for commands or start chatting!\n'));
    
    await this.chatLoop();
  }

  private setupSignalHandlers(): void {
    const handleSignal = () => {
      if (this.interruptibleOp) {
        this.interruptibleOp.interrupt();
        console.log(colors.yellow('\n\nOperation interrupted...'));
      } else {
        console.log(colors.yellow('\n\nGoodbye!'));
        Deno.exit(0);
      }
    };

    Deno.addSignalListener('SIGINT', handleSignal);
    Deno.addSignalListener('SIGTERM', handleSignal);
  }

  private async chatLoop(): Promise<void> {
    while (true) {
      const input = await promptUser(colors.brightBlue('You: '));
      
      if (input === null) {
        console.log(colors.yellow('\nGoodbye!'));
        break;
      }
      
      const trimmed = input.trim();
      if (!trimmed) continue;
      
      // Handle commands
      if (trimmed.startsWith('/')) {
        const handled = await this.handleCommand(trimmed);
        if (!handled) continue;
        if (trimmed === '/quit') break;
        continue;
      }
      
      // Regular chat message
      await this.handleUserMessage(trimmed);
    }
  }

  private async handleCommand(command: string): Promise<boolean> {
    const [cmd, ...args] = command.split(' ');
    
    switch (cmd) {
      case '/help':
        displayHelp();
        return false;
        
      case '/models':
        await this.showModels();
        return false;
        
      case '/model':
        await this.switchModel(args.join(' '));
        return false;
        
      case '/clear':
        await this.clearChat();
        return false;
        
      case '/keys':
        await this.manageKeys();
        return false;
        
      case '/edit':
        await this.editLastMessage();
        return false;
        
      case '/quit':
        return true;
        
      default:
        console.log(colors.red(`Unknown command: ${cmd}`));
        console.log(colors.dim('Type /help for available commands'));
        return false;
    }
  }

  private async handleUserMessage(content: string): Promise<void> {
    const userMessage: Message = {
      role: 'user',
      content,
      timestamp: Date.now(),
    };
    
    this.messages.push(userMessage);
    
    this.interruptibleOp = new InterruptibleOperation<string>();
    
    try {
      console.log(colors.brightGreen('\nAssistant: '));
      
      const request: ChatRequest = {
        model: this.currentModel,
        messages: this.messages,
      };
      
      // Show a brief connecting message
      const connectingSpinner = new Spinner('Connecting to API...');
      connectingSpinner.start();
      
      const result = await this.interruptibleOp.execute(async (signal) => {
        let assistantContent = '';
        let firstChunk = true;
        
        if (this.client) {
          for await (const chunk of this.client.chatStream(request)) {
            if (signal.aborted) {
              throw new Error('Operation was interrupted');
            }
            
            // Stop spinner on first chunk
            if (firstChunk) {
              connectingSpinner.stop();
              firstChunk = false;
            }
            
            Deno.stdout.write(new TextEncoder().encode(chunk));
            assistantContent += chunk;
          }
        }
        
        return assistantContent;
      });
      
      connectingSpinner.stop();
      
      if (result === null) {
        console.log(colors.yellow('\n\nResponse interrupted by user'));
        this.messages.pop(); // Remove the user message since it was interrupted
        return;
      }
      
      console.log('\n');
      
      const assistantMessage: Message = {
        role: 'assistant',
        content: result,
        timestamp: Date.now(),
      };
      
      this.messages.push(assistantMessage);
      
      // Log messages with a subtle spinner
      const logSpinner = new Spinner('Saving to log...');
      logSpinner.start();
      
      try {
        await this.logger.logMessage(userMessage);
        await this.logger.logMessage(assistantMessage);
        logSpinner.stop();
      } catch (error) {
        logSpinner.fail('Failed to save to log');
        console.log(colors.dim(`Log error: ${error instanceof Error ? error.message : String(error)}`));
      }
      
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      console.log(colors.red(`\nError: ${errorMessage}`));
      console.log(colors.dim('The message was not saved due to the error.'));
      // Remove the user message since it failed
      this.messages.pop();
    } finally {
      this.interruptibleOp = null;
    }
  }

  private async showModels(): Promise<void> {
    console.log(colors.brightBlue('\nAvailable Models:'));
    
    const spinner = new Spinner('Fetching models...');
    spinner.start();
    
    try {
      const models = await getAllModelsWithOpenRouter();
      spinner.stop();
      
      const groupedModels = models.reduce((acc, model) => {
        if (!acc[model.provider]) acc[model.provider] = [];
        acc[model.provider].push(model);
        return acc;
      }, {} as Record<string, Model[]>);
      
      for (const [provider, providerModels] of Object.entries(groupedModels)) {
        console.log(colors.yellow(`\n${provider.toUpperCase()}:`));
        
        for (const model of providerModels) {
          const current = model.id === this.currentModel.id ? colors.green(' (current)') : '';
          console.log(`  ${colors.cyan(model.id)} - ${model.name}${current}`);
        }
      }
      
      console.log(colors.dim('\nUse /model <model-id> to switch models'));
    } catch (error) {
      spinner.fail('Failed to fetch models');
      console.log(colors.red(`Error: ${error instanceof Error ? error.message : String(error)}`));
      console.log(colors.dim('Falling back to static model list...'));
      
      // Fallback to static models
      const models = getAllModels();
      const groupedModels = models.reduce((acc, model) => {
        if (!acc[model.provider]) acc[model.provider] = [];
        acc[model.provider].push(model);
        return acc;
      }, {} as Record<string, Model[]>);
      
      for (const [provider, providerModels] of Object.entries(groupedModels)) {
        console.log(colors.yellow(`\n${provider.toUpperCase()}:`));
        
        for (const model of providerModels) {
          const current = model.id === this.currentModel.id ? colors.green(' (current)') : '';
          console.log(`  ${colors.cyan(model.id)} - ${model.name}${current}`);
        }
      }
      
      console.log(colors.dim('\nUse /model <model-id> to switch models'));
    }
  }

  private async switchModel(modelId: string): Promise<void> {
    if (!modelId) {
      console.log(colors.yellow('Available models:'));
      
      const spinner = new Spinner('Fetching models...');
      spinner.start();
      
      try {
        const models = await getAllModelsWithOpenRouter();
        spinner.stop();
        
        models.forEach((model, i) => {
          const current = model.id === this.currentModel.id ? colors.green(' (current)') : '';
          console.log(`  ${i + 1}. ${colors.cyan(model.id)} - ${model.name}${current}`);
        });
      } catch (error) {
        spinner.fail('Failed to fetch models');
        console.log(colors.red(`Error: ${error instanceof Error ? error.message : String(error)}`));
        console.log(colors.dim('Falling back to static model list...'));
        
        const models = getAllModels();
        models.forEach((model, i) => {
          const current = model.id === this.currentModel.id ? colors.green(' (current)') : '';
          console.log(`  ${i + 1}. ${colors.cyan(model.id)} - ${model.name}${current}`);
        });
      }
      
      const selectedModel = await promptUser(colors.brightBlue('Enter model ID: '));
      
      if (!selectedModel) {
        console.log(colors.red('Model selection cancelled'));
        return;
      }
      
      modelId = selectedModel;
    }
    
    // First check static models
    let model = getModel(modelId);
    
    // If not found in static models, try to find in dynamic OpenRouter models
    if (!model) {
      try {
        const allModels = await getAllModelsWithOpenRouter();
        model = allModels.find(m => m.id === modelId);
      } catch (error) {
        console.log(colors.red(`Error fetching dynamic models: ${error instanceof Error ? error.message : String(error)}`));
      }
    }
    
    if (!model) {
      console.log(colors.red(`Model not found: ${modelId}`));
      console.log(colors.dim('Use /models to see available models'));
      return;
    }
    
    // Check if we have API key for this provider
    if (!(await this.keyStore.hasKey(model.provider))) {
      await this.setupApiKey(model.provider);
    }
    
    this.currentModel = model;
    
    await withSpinner('Switching model...', () => this.initializeClient(), {
      successMessage: `Switched to model: ${model.name}`,
      timeout: 10000
    });
  }

  private async clearChat(): Promise<void> {
    await withSpinner('Clearing chat history...', async () => {
      this.messages = [];
      await this.logger.clearLog();
    }, {
      successMessage: 'Chat history cleared!',
      timeout: 5000
    });
  }

  private async manageKeys(): Promise<void> {
    console.log(colors.brightBlue('\nAPI Key Management:'));
    
    const providers = ['anthropic', 'openai', 'openrouter'] as const;
    
    for (const provider of providers) {
      const hasKey = await this.keyStore.hasKey(provider);
      const status = hasKey ? colors.green('✓ Set') : colors.red('✗ Not set');
      console.log(`  ${provider}: ${status}`);
    }
    
    console.log(colors.dim('\nTo set a key, use: /keys <provider> <key>'));
  }

  private async editLastMessage(): Promise<void> {
    if (this.messages.length === 0) {
      console.log(colors.red('No messages to edit'));
      return;
    }
    
    const lastUserMessage = [...this.messages].reverse().find(m => m.role === 'user');
    if (!lastUserMessage) {
      console.log(colors.red('No user message to edit'));
      return;
    }
    
    console.log(colors.dim(`Current message: ${lastUserMessage.content}`));
    
    const newContent = await promptUser(colors.brightBlue('Edit: '));
    if (!newContent || !newContent.trim()) {
      console.log(colors.red('Edit cancelled'));
      return;
    }
    
    // Remove messages after the last user message
    const lastUserIndex = this.messages.lastIndexOf(lastUserMessage);
    this.messages = this.messages.slice(0, lastUserIndex);
    
    // Send the edited message
    await this.handleUserMessage(newContent.trim());
  }

  private async setupApiKey(provider: 'anthropic' | 'openai' | 'openrouter'): Promise<void> {
    console.log(colors.yellow(`\nAPI key required for ${provider}`));
    
    const key = await promptUser(`Enter ${provider} API key: `);
    if (!key || !key.trim()) {
      console.log(colors.red('API key is required to continue'));
      Deno.exit(1);
    }
    
    await withSpinner(`Saving ${provider} API key...`, () => this.keyStore.setKey(provider, key.trim()), {
      successMessage: `API key saved for ${provider}`,
      timeout: 10000
    });
    
    // Validate the key works
    try {
      await withSpinner(`Validating ${provider} API key...`, async () => {
        const testClient = new ApiClient(key.trim(), provider);
        // Try a minimal request to validate the key
        const testModel = this.currentModel.provider === provider ? this.currentModel : {
          ...this.currentModel,
          provider: provider as 'anthropic' | 'openai' | 'openrouter'
        };
        
        await testClient.chat({
          model: testModel,
          messages: [{ role: 'user', content: 'test', timestamp: Date.now() }],
          maxTokens: 1
        });
      }, {
        successMessage: `${provider} API key validated`,
        timeout: 15000
      });
    } catch (error) {
      console.log(colors.red(`\n⚠️  Warning: Could not validate ${provider} API key`));
      console.log(colors.dim(`Error: ${error instanceof Error ? error.message : String(error)}`));
      console.log(colors.dim('The key was saved but may not work properly.'));
    }
  }

  private async initializeClient(): Promise<void> {
    const apiKey = await this.keyStore.getKey(this.currentModel.provider);
    if (!apiKey) {
      throw new Error(`No API key found for ${this.currentModel.provider}`);
    }
    
    this.client = new ApiClient(apiKey, this.currentModel.provider);
  }
}