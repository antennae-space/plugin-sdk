// The layout of a view: a column with a toolbar and a scrolled body, tabs, a splitter, the
// empty state, a message, a spinner.
import { For, Show, createSignal, type JSX, type ParentProps } from 'solid-js'
import { Icon, type IconName } from './icons'

/** A whole view: the frame's height, a toolbar on top, the rest scrolled. */
export function View(props: ParentProps<{ toolbar?: JSX.Element }>) {
  return (
    <div class="ide-view">
      <Show when={props.toolbar}>
        <Toolbar>{props.toolbar}</Toolbar>
      </Show>
      <div class="ide-scroll">{props.children}</div>
    </div>
  )
}

export function Toolbar(props: ParentProps) {
  return <div class="ide-toolbar" role="toolbar">{props.children}</div>
}

export function Separator() {
  return <span class="ide-sep" role="separator" />
}

export function EmptyState(props: ParentProps<{ icon?: IconName; text: string }>) {
  return (
    <div class="ide-empty">
      <Icon name={props.icon ?? 'info'} size={28} />
      <p>{props.text}</p>
      {props.children}
    </div>
  )
}

export function Spinner(props: { title?: string }) {
  return <span class="ide-spinner" role="status" aria-label={props.title} />
}

/** A line of information, an error, a warning or a success. */
export function Message(props: ParentProps<{ kind?: 'info' | 'error' | 'warn' | 'ok' }>) {
  return <div class={`ide-message ${props.kind ?? 'info'}`} role={props.kind === 'error' ? 'alert' : 'status'}>{props.children}</div>
}

export type Tab<T extends string> = { id: T; label: string }

/** Tabs, the arrow keys moving between them. */
export function Tabs<T extends string>(props: { tabs: Tab<T>[]; active: T; onSelect: (id: T) => void }) {
  let el!: HTMLDivElement
  const move = (e: KeyboardEvent) => {
    const i = props.tabs.findIndex((t) => t.id === props.active)
    const d = e.key === 'ArrowRight' ? 1 : e.key === 'ArrowLeft' ? -1 : 0
    if (!d || i < 0) return
    e.preventDefault()
    const next = props.tabs[(i + d + props.tabs.length) % props.tabs.length]
    props.onSelect(next.id)
    queueMicrotask(() => el.querySelector<HTMLElement>('[aria-selected=true]')?.focus())
  }
  return (
    <div class="ide-tabs" role="tablist" ref={el} onKeyDown={move}>
      <For each={props.tabs}>
        {(t) => (
          <button type="button" role="tab" class="ide-tab" aria-selected={t.id === props.active} tabIndex={t.id === props.active ? 0 : -1} onClick={() => props.onSelect(t.id)}>
            {t.label}
          </button>
        )}
      </For>
    </div>
  )
}

/** Two panes and a gutter to drag (or move with the arrow keys); size is the first pane's, in %. */
export function Splitter(props: { first: JSX.Element; second: JSX.Element; vertical?: boolean; size?: number; onResize?: (size: number) => void }) {
  const [size, setSize] = createSignal(props.size ?? 50)
  let el!: HTMLDivElement
  const set = (v: number) => {
    const s = Math.min(90, Math.max(10, v))
    setSize(s)
    props.onResize?.(s)
  }
  const drag = (e: PointerEvent) => {
    e.preventDefault()
    const r = el.getBoundingClientRect()
    const moveTo = (ev: PointerEvent) => set(props.vertical ? ((ev.clientY - r.top) / r.height) * 100 : ((ev.clientX - r.left) / r.width) * 100)
    const up = () => {
      window.removeEventListener('pointermove', moveTo)
      window.removeEventListener('pointerup', up)
    }
    window.addEventListener('pointermove', moveTo)
    window.addEventListener('pointerup', up)
  }
  const key = (e: KeyboardEvent) => {
    const d = { ArrowLeft: -5, ArrowUp: -5, ArrowRight: 5, ArrowDown: 5 }[e.key]
    if (d === undefined) return
    e.preventDefault()
    set(size() + d)
  }
  return (
    <div class={`ide-split${props.vertical ? ' vertical' : ''}`} ref={el}>
      <div class="pane" style={{ [props.vertical ? 'height' : 'width']: `${size()}%` }}>{props.first}</div>
      <div class="gutter" role="separator" tabIndex={0} aria-orientation={props.vertical ? 'horizontal' : 'vertical'} aria-valuenow={Math.round(size())} onPointerDown={drag} onKeyDown={key} />
      <div class="pane ide-grow">{props.second}</div>
    </div>
  )
}
