import react from '@vitejs/plugin-react-swc'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '^/(?!(@|node_modules|web)).+': 'http://localhost:1234',
    },
  },
})
