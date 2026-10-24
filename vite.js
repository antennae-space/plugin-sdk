// The Vite preset of a plugin's page module: one ES module, `web/index.js`, as the IDE loads it
// in a sandboxed frame (inline only: no other file, no network). Solid is compiled in, the CSS
// is injected by the module itself, the assets are inlined as data URLs.
//
//   // vite.config.js
//   import { defineConfig } from 'vite'
//   import { idePlugin } from '@antennae-space/plugin-sdk/vite'
//   export default defineConfig({ plugins: [idePlugin()] })
import solid from 'vite-plugin-solid'

/** The largest page module the IDE loads (plugins.bundle). */
export const maxBundle = 5 << 20

/**
 * @param {{ entry?: string, outFile?: string }} [o] the entry (`web/src/index.tsx`) and the
 *   output (`web/index.js`), relative to the root of the plugin.
 * @returns {import('vite').PluginOption[]}
 */
export function idePlugin(o = {}) {
  const entry = o.entry ?? 'web/src/index.tsx'
  const outFile = o.outFile ?? 'web/index.js'
  const slash = outFile.lastIndexOf('/')
  const outDir = slash < 0 ? '.' : outFile.slice(0, slash)
  const fileName = outFile.slice(slash + 1)
  /** @type {import('vite').Plugin} */
  const frame = {
    name: 'ide-plugin-frame',
    // After the CSS plugin of Vite, which emits the stylesheet in its own generateBundle.
    enforce: 'post',
    config() {
      return {
        // The frame has no base URL to load from: everything is inside the module.
        base: './',
        publicDir: false,
        build: {
          outDir,
          emptyOutDir: false,
          target: 'es2022',
          assetsInlineLimit: () => true,
          cssCodeSplit: false,
          modulePreload: false,
          lib: { entry, formats: ['es'], fileName: () => fileName },
          rolldownOptions: { output: { codeSplitting: false } },
        },
      }
    },
    generateBundle(_, bundle) {
      // The CSS emitted apart goes into the module, as a <style> it adds first.
      let css = ''
      for (const [name, f] of Object.entries(bundle)) {
        if (f.type === 'asset' && name.endsWith('.css')) {
          css += String(f.source)
          delete bundle[name]
        }
      }
      for (const f of Object.values(bundle)) {
        if (f.type === 'asset') this.error(`the plugin module must be one file: ${f.fileName} would be apart`)
        if (f.type !== 'chunk') continue
        if (f.fileName !== fileName) this.error(`the plugin module must be one file: ${f.fileName} would be apart`)
        if (css) f.code = `{const s=document.createElement('style');s.dataset.ide='plugin';s.textContent=${JSON.stringify(css)};document.head.append(s)}\n` + f.code
        if (f.code.length > maxBundle) this.error(`${fileName} is ${(f.code.length / 1048576).toFixed(1)} MB: the IDE loads 5 MB at most`)
      }
    },
  }
  return [solid(), frame]
}
