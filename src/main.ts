#!/usr/bin/env -S deno run --allow-net --allow-read --allow-write --allow-env

import { App } from './ui/app.ts';
import { displayBanner } from './ui/banner.ts';

async function main() {
  try {
    displayBanner();
    
    const app = new App();
    await app.run();
  } catch (error) {
    console.error('Fatal error:', error);
    Deno.exit(1);
  }
}

if (import.meta.main) {
  main();
}