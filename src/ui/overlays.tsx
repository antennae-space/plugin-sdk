// Menus and modals, drawn inside the frame (a plugin draws nothing outside it). Escape closes;
// the focus goes back where it was.
import { For, Show, onCleanup, onMount, type JSX, type ParentProps } from 'solid-js'
import { render } from 'solid-js/web'
import { Button, IconButton } from './controls'

export type MenuItem = { label: string; run: () => void; keys?: string; danger?: boolean; disabled?: boolean } | 'separator'

/** Opens a menu at a point (a right click, a button); it closes on a choice, Escape or outside. */
export function openMenu(items: MenuItem[], at: { x: number; y: number } | MouseEvent): void {
  const back = document.activeElement as HTMLElement | null
  const host = document.createElement('div')
  document.body.append(host)
  const close = () => { dispose(); host.remove(); back?.focus?.() }
  const dispose = render(() => <Menu items={items} x={at instanceof MouseEvent ? at.clientX : at.x} y={at instanceof MouseEvent ? at.clientY : at.y} close={close} />, host)
}

function Menu(props: { items: MenuItem[]; x: number; y: number; close: () => void }) {
  let el!: HTMLDivElement
  const buttons = () => [...el.querySelectorAll<HTMLButtonElement>('button:not(:disabled)')]
  const outside = (e: PointerEvent) => !el.contains(e.target as Node) && props.close()
  onMount(() => {
    // Kept inside the frame.
    const r = el.getBoundingClientRect()
    el.style.left = `${Math.max(0, Math.min(props.x, innerWidth - r.width))}px`
    el.style.top = `${Math.max(0, Math.min(props.y, innerHeight - r.height))}px`
    buttons()[0]?.focus()
    setTimeout(() => document.addEventListener('pointerdown', outside))
  })
  onCleanup(() => document.removeEventListener('pointerdown', outside))
  const key = (e: KeyboardEvent) => {
    const bs = buttons()
    const i = bs.indexOf(document.activeElement as HTMLButtonElement)
    if (e.key === 'Escape') { e.preventDefault(); e.stopPropagation(); props.close() }
    else if (e.key === 'ArrowDown' || e.key === 'ArrowUp') { e.preventDefault(); bs[(i + (e.key === 'ArrowDown' ? 1 : -1) + bs.length) % bs.length]?.focus() }
  }
  return (
    <div class="ide-menu" role="menu" ref={el} onKeyDown={key} style={{ left: `${props.x}px`, top: `${props.y}px` }}>
      <For each={props.items}>
        {(it) =>
          it === 'separator' ? <div class="ide-menu-sep" role="separator" /> : (
            <button type="button" role="menuitem" class={`ide-menu-item${it.danger ? ' danger' : ''}`} disabled={it.disabled} onClick={() => { props.close(); it.run() }}>
              <span>{it.label}</span>
              <Show when={it.keys}><kbd>{it.keys}</kbd></Show>
            </button>
          )
        }
      </For>
    </div>
  )
}

/** A modal of the frame; the Tab key stays inside it. */
export function Modal(props: ParentProps<{ title: string; onClose: () => void; footer?: JSX.Element; closeLabel?: string }>) {
  let el!: HTMLDivElement
  const back = document.activeElement as HTMLElement | null
  onMount(() => (el.querySelector<HTMLElement>('[autofocus], input, select, textarea, button.primary') ?? el).focus())
  onCleanup(() => back?.focus?.())
  const key = (e: KeyboardEvent) => {
    if (e.key === 'Escape') { e.preventDefault(); e.stopPropagation(); props.onClose() }
    if (e.key !== 'Tab') return
    const f = [...el.querySelectorAll<HTMLElement>('button, input, select, textarea, [tabindex]:not([tabindex="-1"])')].filter((x) => !x.hasAttribute('disabled'))
    if (!f.length) return
    const [a, z] = [f[0], f[f.length - 1]]
    if (e.shiftKey && document.activeElement === a) { e.preventDefault(); z.focus() }
    else if (!e.shiftKey && document.activeElement === z) { e.preventDefault(); a.focus() }
  }
  return (
    <div class="ide-backdrop" onPointerDown={(e) => e.target === e.currentTarget && props.onClose()}>
      <div class="ide-modal" role="dialog" aria-modal="true" aria-label={props.title} tabIndex={-1} ref={el} onKeyDown={key}>
        <div class="ide-modal-head">
          <h2>{props.title}</h2>
          <IconButton icon="close" title={props.closeLabel ?? 'Close'} onClick={() => props.onClose()} />
        </div>
        <div class="ide-modal-body">{props.children}</div>
        <Show when={props.footer}><div class="ide-modal-foot">{props.footer}</div></Show>
      </div>
    </div>
  )
}

/** Asks a yes or no in a modal; the labels are the plugin's (through `ide.t`). */
export function confirm(o: { title: string; text: string; ok: string; cancel: string; danger?: boolean }): Promise<boolean> {
  // The IDE's own dialog, centered in its window (IDE 1.24): the frame's only before it.
  if (typeof ide !== 'undefined' && typeof ide.confirm === 'function') return ide.confirm({ title: o.title, text: o.text, ok: o.ok, danger: o.danger })
  return new Promise((resolve) => {
    const host = document.createElement('div')
    document.body.append(host)
    const done = (v: boolean) => { dispose(); host.remove(); resolve(v) }
    const dispose = render(() => (
      <Modal title={o.title} closeLabel={o.cancel} onClose={() => done(false)}
        footer={<><Button onClick={() => done(false)}>{o.cancel}</Button><Button kind={o.danger ? 'danger' : 'primary'} onClick={() => done(true)}>{o.ok}</Button></>}>
        <p style={{ margin: 0 }}>{o.text}</p>
      </Modal>
    ), host)
  })
}
