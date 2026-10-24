// Solid helpers for the views of a plugin: `view(Component)` makes a `views` entry of
// `ide.register`, and the hooks read the frame's API, theme, language and tab state.
import { createContext, createSignal, onCleanup, useContext, type Accessor, type Component } from 'solid-js'
import { render } from 'solid-js/web'
import type { SandboxAPI, SandboxTheme } from './sandbox-api'
import { injectStyle } from './ui/style'
import './global'

const Ctx = createContext<SandboxAPI>()

/** A `views` entry rendering the component in the frame, with the UI kit's styles. */
export function view(C: Component): (el: HTMLElement, ide: SandboxAPI) => void {
  return (el, ide) => {
    injectStyle()
    render(() => <Ctx.Provider value={ide}><C /></Ctx.Provider>, el)
  }
}

/** The sandbox API of the frame. */
export function useIde(): SandboxAPI {
  return useContext(Ctx) ?? globalThis.ide
}

/** The theme of the IDE, following its changes. */
export function useTheme(): Accessor<SandboxTheme> {
  const ide = useIde()
  const [theme, setTheme] = createSignal<SandboxTheme>(ide.theme ?? {})
  ide.onTheme(setTheme)
  return theme
}

/**
 * The state of this frame's tab, kept with it across reloads, desks and splits (a tab view):
 * the setter saves it.
 */
export function useViewState<T>(initial: T): [Accessor<T>, (v: T) => void] {
  const ide = useIde()
  const [state, setState] = createSignal<T>((ide.view?.state as T | undefined) ?? initial)
  return [state, (v: T) => { setState(() => v); ide.setViewState(v) }]
}

/** An event of the plugin (`<id>.*`) while the component lives. */
export function useEvent<T = unknown>(event: string, f: (data: T) => void): void {
  let live = true
  useIde().on(event, (d) => live && f(d as T))
  onCleanup(() => (live = false))
}

/** A call of the plugin's backend as signals: its value (kept while it reloads), its error, a reload. */
export function useRpc<T>(method: string, params?: () => unknown) {
  const ide = useIde()
  const [value, setValue] = createSignal<T>()
  const [error, setError] = createSignal<string>()
  const [loading, setLoading] = createSignal(false)
  const reload = async () => {
    setLoading(true)
    try {
      setError()
      const v = await ide.rpc<T>(method, params?.())
      setValue(() => v)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
    }
  }
  void reload()
  return { value, error, loading, reload }
}
