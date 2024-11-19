import { readFile, rm, writeFile } from 'node:fs/promises'
import { join } from 'node:path'
import { promisify } from 'node:util'
import zlib from 'node:zlib'
import { type RsbuildPlugin, defineConfig } from '@rsbuild/core'
import { pluginPreact } from '@rsbuild/plugin-preact'

export default defineConfig({
  html: {
    title: 'RSSLab',
    favicon: './web/assets/wind.svg',
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
    legalComments: 'inline',
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
    pluginPreact(),
    {
      name: 'plugin-compression',
      setup(api) {
        api.onAfterBuild(async ({ stats }) => {
          if (!stats) return
          const { assets, outputPath } = stats.toJson(null)
          if (!assets || !outputPath) return
          await Promise.all(
            assets.map(async ({ name }) => {
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
