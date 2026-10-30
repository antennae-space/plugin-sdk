// Solid helpers for the views of a plugin: `view(Component)` makes a `views` entry of
// `ide.register`, and the hooks read the frame's API, theme, language and tab state.
import { createContext, createEffect, createSignal, onCleanup, useContext, type Accessor, type Component } from 'solid-js'
import { render } from 'solid-js/web'
import type { SandboxAPI, SandboxStatusItem, SandboxTheme, SandboxToolMenuItem } from './sandbox-api'
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

/**
 * The editor of the tab (a tab view whose manifest asks `editor`), with its text and selection
 * as signals; null in a view without one.
 */
export function useEditor() {
  const ide = useIde()
  const ed = ide.editor
  if (!ed) return null
  const [text, setText] = createSignal(ed.text())
  const [selection, setSelection] = createSignal(ed.selection())
  ed.onChange(setText)
  ed.onSelection(() => setSelection(ed.selection()))
  return { editor: ed, text, selection }
}

/** The settings of the plugin, following the user's changes. */
export function useSettings<T extends Record<string, unknown> = Record<string, unknown>>(): Accessor<T> {
  const ide = useIde()
  const [values, setValues] = createSignal<T>(ide.settings as T)
  ide.onSettings((v) => setValues(() => v as T))
  return values
}

/**
 * The plugin's state in the desk shown (IDE 1.25): each desk keeps its own, a reload or another
 * machine finds it. The accessor follows another desk shown; set replaces it.
 */
export function useDesk<T>(initial: T): [Accessor<T>, (v: T) => void] {
  const ide = useIde()
  const [value, setValue] = createSignal<T>((ide.desk as T | null) ?? initial)
  ide.onDesk((v) => setValue(() => (v as T | null) ?? initial))
  const set = (v: T) => {
    setValue(() => v)
    ide.setDesk(v)
  }
  return [value, set]
}

/** A modal view (contributes.modals, opened by ide.openModal): close gives its opener a result. */
export function useModal(): { isModal: boolean; close: (result?: unknown) => void } {
  const ide = useIde()
  return { isModal: ide.modal, close: (result) => ide.closeModal(result) }
}

/**
 * A capability another external plugin provides, one the manifest requires (`ide.use`): its
 * calls, and a value it publishes as an accessor.
 */
export function useService(capability: string) {
  const svc = useIde().use(capability)
  return {
    call: svc.call,
    on: svc.on,
    value<T = unknown>(key: string): Accessor<T | undefined> {
      const [v, setV] = createSignal<T | undefined>(svc.value<T>(key))
      svc.onValue(key, (x) => setV(() => x as T))
      return v
    },
  }
}

/**
 * The items of the status bar of this tab (a tab view), kept in step with items(); onChoice gets
 * the choices of their menus.
 */
export function useStatus(items: Accessor<SandboxStatusItem[]>, onChoice: (item: string, choice: string) => void): void {
  const ide = useIde()
  ide.onStatus((e) => onChoice(e.item, e.choice))
  createEffect(() => ide.setStatus(items()))
}

/**
 * The items this frame adds to the options menu (the ⋮) of its tool or of its tab, kept in step
 * with items(); onChoice gets the id of the item chosen. IDE 1.24.
 */
export function useToolMenu(items: Accessor<SandboxToolMenuItem[]>, onChoice: (item: string) => void): void {
  const ide = useIde()
  ide.onToolMenu((e) => onChoice(e.item))
  createEffect(() => ide.setToolMenu(items()))
}
