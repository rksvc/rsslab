import react from '@vitejs/plugin-react-swc'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [react()],
  build: {
    reportCompressedSize: false,
    rollupOptions: {
      external: [
        './paths-loaders/allPathsLoader',
        './paths-loaders/splitPathsBySizeLoader',
      ],
    },
  },
  server: {
    proxy: {
      '^/(?!(@|node_modules|web)).+': 'http://localhost:1234',
    },
  },
})
