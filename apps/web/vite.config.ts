/// <reference types="vitest/config" />
import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// ADR-0005: the browser only talks to Vite; Vite forwards /api to the Go API.
// ADR-0006: in Compose the API is reached by service name, on the host by localhost.
export default defineConfig(({ mode }) => {
  // '' = load all variables, not just VITE_ ones. These stay in Node and never reach the browser.
  const env = loadEnv(mode, '.', '')

  return {
    plugins: [react(), tailwindcss()],
    server: {
      host: env.DEV_HOST || '127.0.0.1', // Compose sets 0.0.0.0; on the host, localhost only
      port: 5173,
      strictPort: true, // fail loudly instead of drifting to 5174 (bucket CORS only allows 5173)
      proxy: {
        '/api': { target: env.API_PROXY_TARGET || 'http://localhost:8080' },
      },
    },
    test: {
      environment: 'jsdom', // a fake browser DOM inside Node
      setupFiles: './src/test/setup.ts', // runs before every test file
    },
  }
})