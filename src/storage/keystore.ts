import { decodeHex, encodeHex } from '@std/encoding/hex';

export interface ApiKeys {
  anthropic?: string;
  openai?: string;
  openrouter?: string;
}

export class KeyStore {
  private configDir: string;
  private keyFile: string;
  private key: CryptoKey | null = null;

  constructor() {
    const home = Deno.env.get('HOME') || Deno.env.get('USERPROFILE') || '/tmp';
    this.configDir = `${home}/.klip`;
    this.keyFile = `${this.configDir}/keys.enc`;
  }

  async init(): Promise<void> {
    await this.ensureConfigDir();
    await this.initKey();
  }

  private async ensureConfigDir(): Promise<void> {
    try {
      await Deno.stat(this.configDir);
    } catch {
      await Deno.mkdir(this.configDir, { recursive: true });
    }
  }

  private async initKey(): Promise<void> {
    const keyFile = `${this.configDir}/.key`;

    try {
      const keyData = await Deno.readTextFile(keyFile);
      const keyMaterial = decodeHex(keyData);
      this.key = await crypto.subtle.importKey(
        'raw',
        keyMaterial,
        { name: 'AES-GCM' },
        false,
        ['encrypt', 'decrypt'],
      );
    } catch {
      // Generate new key
      this.key = await crypto.subtle.generateKey(
        { name: 'AES-GCM', length: 256 },
        true,
        ['encrypt', 'decrypt'],
      );

      const keyMaterial = await crypto.subtle.exportKey('raw', this.key);
      await Deno.writeTextFile(keyFile, encodeHex(new Uint8Array(keyMaterial)));
      await Deno.chmod(keyFile, 0o600);
    }
  }

  async getKeys(): Promise<ApiKeys> {
    if (!this.key) await this.init();

    try {
      const encryptedData = await Deno.readTextFile(this.keyFile);
      const [ivHex, dataHex] = encryptedData.split(':');

      const iv = decodeHex(ivHex);
      const data = decodeHex(dataHex);

      const decrypted = await crypto.subtle.decrypt(
        { name: 'AES-GCM', iv },
        this.key!,
        data,
      );

      const json = new TextDecoder().decode(decrypted);
      return JSON.parse(json);
    } catch {
      return {};
    }
  }

  async setKeys(keys: ApiKeys): Promise<void> {
    if (!this.key) await this.init();

    const json = JSON.stringify(keys);
    const data = new TextEncoder().encode(json);

    const iv = crypto.getRandomValues(new Uint8Array(12));
    const encrypted = await crypto.subtle.encrypt(
      { name: 'AES-GCM', iv },
      this.key!,
      data,
    );

    const encryptedData = `${encodeHex(iv)}:${
      encodeHex(new Uint8Array(encrypted))
    }`;
    await Deno.writeTextFile(this.keyFile, encryptedData);
    await Deno.chmod(this.keyFile, 0o600);
  }

  async setKey(provider: keyof ApiKeys, key: string): Promise<void> {
    const keys = await this.getKeys();
    keys[provider] = key;
    await this.setKeys(keys);
  }

  async getKey(provider: keyof ApiKeys): Promise<string | undefined> {
    const keys = await this.getKeys();
    return keys[provider];
  }

  async hasKey(provider: keyof ApiKeys): Promise<boolean> {
    const key = await this.getKey(provider);
    return !!key;
  }
}
