import type { PluginOption } from 'vite'

/** The largest page module the IDE loads. */
export declare const maxBundle: number

/** The Vite preset of a plugin's page module: one ES module, `web/index.js` by default. */
export declare function idePlugin(o?: { entry?: string; outFile?: string }): PluginOption[]
