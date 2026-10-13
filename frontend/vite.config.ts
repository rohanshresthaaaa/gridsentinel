/// <reference types="vitest/config" />
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      // During development, forward /api requests to the Go backend.
      '/api': 'http://localhost:8080',
    },
  },
  test: {
    environment: 'jsdom', // a simulated browser for component tests
    setupFiles: ['./src/test/setup.ts'],
  },
})