import { readFile, rm, writeFile } from 'node:fs/promises'
import { join } from 'node:path'
import { promisify } from 'node:util'
import zlib from 'node:zlib'
import { defineConfig, type RsbuildPlugin } from '@rsbuild/core'
import { pluginReact } from '@rsbuild/plugin-react'

export default defineConfig({
  html: {
    title: 'RSSLab',
    favicon: './web/assets/icon.svg',
    meta: {
      'dark-theme': process.env.NODE_ENV === 'production' ? '%DARK_THEME%' : '',
    },
  },
  source: {
    entry: {
      index: './web/main.tsx',
    },
  },
  output: {
    distPath: {
      root: 'server/dist',
      js: './',
      css: './',
    },
    cleanDistPath: process.env.NODE_ENV === 'production',
    externals: ({ request }, callback) =>
      request === './iconLoader' ? callback(undefined, '{}', 'var') : callback(),
    legalComments: 'none',
  },
  performance: {
    chunkSplit: {
      strategy: 'all-in-one',
    },
  },
  server: {
    host: '127.0.0.1',
    proxy: {
      '/api': 'http://localhost:1234',
    },
  },
  plugins: [
    pluginReact(),
    {
      name: 'plugin-compression',
      setup(api) {
        api.onAfterBuild(async ({ stats }) => {
          if (!stats) return
          const { assets, outputPath } = stats.toJson(null)
          if (!assets || !outputPath) return
          await Promise.all(
            assets.map(async ({ name }) => {
              if (name === 'index.html') return
              const path = join(outputPath, name)
              const content = await readFile(path)
              const compressed = await promisify(zlib.gzip)(content, {
                level: zlib.constants.Z_BEST_COMPRESSION,
              })
              if (compressed.length >= content.length) return
              await writeFile(`${path}.gz`, compressed)
              await rm(path)
            }),
          )
        })
      },
    } satisfies RsbuildPlugin,
  ],
})
