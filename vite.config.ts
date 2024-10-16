import react from '@vitejs/plugin-react-swc'
import autoprefixer from 'autoprefixer'
import tailwindcss from 'tailwindcss'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [react()],
  css: {
    preprocessorOptions: { scss: { api: 'modern-compiler' } },
    modules: { localsConvention: 'camelCaseOnly' },
    postcss: {
      plugins: [
        tailwindcss({ content: ['./index.html', './web/**/*.{js,ts,jsx,tsx}'] }),
        autoprefixer,
      ],
    },
  },
  server: {
    proxy: {
      '^/(?!(@|node_modules|web)).+': 'http://localhost:1234',
    },
  },
})
