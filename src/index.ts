// The plugin SDK of Web IDE: the types of the sandbox API (`sandbox-api.ts`, a copy of web-ide's
// web/src/plugins/sandbox-api.ts, checked by `npm run check-api`), the global `ide` of a frame,
// and the tokens of the theme. The UI kit is `@antennae-space/plugin-sdk/ui`, the Solid helpers
// `/solid`, the Vite preset `/vite`, the backend module `antennae.space/plugin-sdk/backend`.
import './global'

export type * from './sandbox-api'

/** The plugin API version these types describe. */
export const apiVersion = 1

/**
 * The CSS variables of the theme a frame gets (`ide.theme`, kept on its root element), and
 * `color-scheme`. A theme may add some; a plugin falls back when one is missing.
 */
export const themeTokens = [
  'bg', 'bg-2', 'bg-3', 'bg-hover', 'bg-active',
  'fg', 'fg-muted', 'fg-faint', 'line',
  'accent', 'accent-fg', 'sel-bg', 'cur-line',
  'danger', 'warn', 'ok', 'info',
  'match-bg', 'match-cur-bg', 'gutter', 'conflict-bg',
  'font-code', 'font-size-code', 'line-height-code', 'font-size-ui', 'ligatures',
  'focus-outline-width', 'focus-dim',
] as const

export type ThemeToken = (typeof themeTokens)[number]

/** Whether the theme of the IDE is dark. */
export function isDark(theme = globalThis.ide?.theme): boolean {
  return (theme?.['color-scheme'] ?? 'dark').trim() !== 'light'
}
