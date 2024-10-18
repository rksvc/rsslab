import react from '@vitejs/plugin-react-swc'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [react()],
  css: {
    preprocessorOptions: { scss: { api: 'modern-compiler' } },
    modules: { localsConvention: 'camelCaseOnly' },
  },
  server: {
    proxy: {
      '^/(?!(@|node_modules|web)).+': 'http://localhost:1234',
    },
  },
})
