import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test-setup.ts'],
    coverage: {
      provider: 'v8',
      include: ['src/components/**/*.tsx'],
      thresholds: { lines: 80, functions: 80 },
      reporter: ['text', 'lcov'],
    },
  },
});
