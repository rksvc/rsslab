import { promisify } from 'node:util'
import zlib from 'node:zlib'
import preact from '@preact/preset-vite'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [preact()],
  build: {
    outDir: 'server/dist',
    rollupOptions: {
      external: ['./paths-loaders/allPathsLoader', './paths-loaders/splitPathsBySizeLoader'],
      plugins: [
        {
          name: 'rollup-plugin-compression',
          async generateBundle(_, bundles) {
            const encoder = new TextEncoder()
            await Promise.all(
              Object.entries(bundles).map(async ([fileName, bundle]) => {
                const source = bundle.type === 'asset' ? bundle.source : bundle.code
                const buf = typeof source === 'string' ? encoder.encode(source) : source
                const compressed = await promisify(zlib.gzip)(buf, {
                  level: zlib.constants.Z_BEST_COMPRESSION,
                })
                if (compressed.length >= buf.length) return
                delete bundles[fileName]
                this.emitFile({ type: 'asset', fileName: `${fileName}.gz`, source: compressed })
              }),
            )
          },
        },
      ],
    },
  },
  server: {
    host: '127.0.0.1',
    proxy: {
      '/api': 'http://localhost:1234',
    },
  },
})
