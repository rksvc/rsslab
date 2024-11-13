import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [react()],
  build: {
    outDir: 'server/dist',
    rollupOptions: {
      external: ['./paths-loaders/allPathsLoader', './paths-loaders/splitPathsBySizeLoader'],
    },
  },
  server: {
    host: '127.0.0.1',
    proxy: {
      '^/(?!(@|node_modules|web)).+': 'http://localhost:1234',
    },
  },
})
