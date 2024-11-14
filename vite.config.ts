import preact from '@preact/preset-vite'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [preact()],
  build: {
    outDir: 'server/dist',
    rollupOptions: {
      external: ['./paths-loaders/allPathsLoader', './paths-loaders/splitPathsBySizeLoader'],
    },
  },
  server: {
    host: '127.0.0.1',
    proxy: {
      '/api': 'http://localhost:1234',
    },
  },
})
