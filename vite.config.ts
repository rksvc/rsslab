import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import tailwindcss from 'tailwindcss';
import autoprefixer from 'autoprefixer';

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  css: {
    modules: { localsConvention: 'camelCaseOnly' },
    postcss: {
      plugins: [
        tailwindcss({
          content: ['./index.html', './web/**/*.{js,ts,jsx,tsx}'],
          theme: {
            extend: {},
          },
          plugins: [],
        }),
        autoprefixer,
      ],
    },
  },
  server: {
    proxy: {
      '/api': 'http://localhost:1234',
      '/rsshub': 'http://localhost:1234',
    },
  },
});
