import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

export function feedbackWorkspaceVite() {
  const sourceRoot = fileURLToPath(new URL('../src', import.meta.url))
  return defineConfig({
    plugins: [vue()],
    resolve: { alias: { '@': sourceRoot } },
    server: {
      port: 5173,
      proxy: {
        '/api': { target: 'http://localhost:8080', changeOrigin: false },
        '/healthz': { target: 'http://localhost:8080', changeOrigin: false }
      }
    },
    build: { sourcemap: true, reportCompressedSize: true }
  })
}
