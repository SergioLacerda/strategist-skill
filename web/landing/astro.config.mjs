import { defineConfig } from 'astro/config';
import react from '@astrojs/react';

export default defineConfig({
  site: 'https://sergiolacerda.github.io',
  base: '/strategist-skill',
  output: 'static',
  integrations: [react()],
  build: {
    assets: '_assets',
  },
});
