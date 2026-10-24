// Lists, trees and tables, driven by the keyboard as the IDE's are: one tab stop, the arrows
// move the selection, Enter opens, a tree opens with → and closes with ←.
import { For, Show, createMemo, createSignal, onCleanup, onMount, type JSX } from 'solid-js'
import { Icon, type IconName } from './icons'

export type Item = { id: string; label: string; detail?: string; icon?: IconName }

/** Moves the selection with the keys; answers whether the key was handled. */
function arrows<T extends { id: string }>(e: KeyboardEvent, items: T[], selected: string | undefined, select: (id: string) => void) {
  const i = items.findIndex((x) => x.id === selected)
  const to = { ArrowDown: i + 1, ArrowUp: i - 1, Home: 0, End: items.length - 1, PageDown: i + 10, PageUp: i - 10 }[e.key]
  if (to === undefined || !items.length) return false
  e.preventDefault()
  select(items[Math.max(0, Math.min(items.length - 1, to))].id)
  return true
}

function reveal(root: HTMLElement) {
  queueMicrotask(() => root.querySelector('[aria-selected=true]')?.scrollIntoView({ block: 'nearest' }))
}

export function List<T extends Item>(props: {
  items: T[]
  selected?: string
  onSelect: (id: string) => void
  onOpen?: (item: T) => void
  onMenu?: (item: T, e: MouseEvent) => void
  label?: string
  render?: (item: T) => JSX.Element
}) {
  let el!: HTMLUListElement
  const key = (e: KeyboardEvent) => {
    if (arrows(e, props.items, props.selected, props.onSelect)) return reveal(el)
    const cur = props.items.find((x) => x.id === props.selected)
    if (e.key === 'Enter' && cur) props.onOpen?.(cur)
  }
  return (
    <ul class="ide-list" role="listbox" tabIndex={0} aria-label={props.label} ref={el} onKeyDown={key}>
      <For each={props.items}>
        {(it) => (
          <li class="ide-row" role="option" aria-selected={it.id === props.selected} onClick={() => props.onSelect(it.id)} onDblClick={() => props.onOpen?.(it)}
            onContextMenu={(e) => { if (props.onMenu) { e.preventDefault(); props.onSelect(it.id); props.onMenu(it, e) } }}>
            {props.render ? props.render(it) : <ItemRow item={it} />}
          </li>
        )}
      </For>
    </ul>
  )
}

function ItemRow(props: { item: Item }) {
  return (
    <>
      <Show when={props.item.icon}>{(i) => <Icon name={i()} size={14} />}</Show>
      <span class="label">{props.item.label}</span>
      <Show when={props.item.detail}><span class="detail">{props.item.detail}</span></Show>
    </>
  )
}

/** A node of a tree; children undefined: a leaf, a function: loaded when it opens. */
export type Node = Item & { children?: Node[] | (() => Promise<Node[]>) }

type Row = { node: Node; depth: number; open: boolean; leaf: boolean }

export function Tree(props: {
  nodes: Node[]
  selected?: string
  onSelect: (id: string) => void
  onOpen?: (node: Node) => void
  onMenu?: (node: Node, e: MouseEvent) => void
  label?: string
}) {
  let el!: HTMLUListElement
  const [open, setOpen] = createSignal<Record<string, boolean>>({})
  const [loaded, setLoaded] = createSignal<Record<string, Node[]>>({})
  const kids = (n: Node) => (Array.isArray(n.children) ? n.children : loaded()[n.id])
  const rows = createMemo(() => {
    const out: Row[] = []
    const walk = (ns: Node[], depth: number) => {
      for (const n of ns) {
        const o = !!open()[n.id]
        out.push({ node: n, depth, open: o, leaf: n.children === undefined })
        const k = kids(n)
        if (o && k) walk(k, depth + 1)
      }
    }
    walk(props.nodes, 0)
    return out
  })
  const toggle = async (n: Node, to = !open()[n.id]) => {
    if (n.children === undefined) return
    if (to && typeof n.children === 'function' && !loaded()[n.id]) {
      const k = await n.children()
      setLoaded((l) => ({ ...l, [n.id]: k }))
    }
    setOpen((o) => ({ ...o, [n.id]: to }))
  }
  const key = (e: KeyboardEvent) => {
    const list = rows().map((r) => ({ id: r.node.id, r }))
    if (arrows(e, list, props.selected, props.onSelect)) return reveal(el)
    const cur = list.find((x) => x.id === props.selected)?.r
    if (!cur) return
    if (e.key === 'ArrowRight' && !cur.leaf) { e.preventDefault(); void toggle(cur.node, true) }
    else if (e.key === 'ArrowLeft' && cur.open) { e.preventDefault(); void toggle(cur.node, false) }
    else if (e.key === 'Enter') cur.leaf ? props.onOpen?.(cur.node) : void toggle(cur.node)
  }
  return (
    <ul class="ide-tree" role="tree" tabIndex={0} aria-label={props.label} ref={el} onKeyDown={key}>
      <For each={rows()}>
        {(r) => (
          <li class="ide-row" role="treeitem" aria-level={r.depth + 1} aria-expanded={r.leaf ? undefined : r.open} aria-selected={r.node.id === props.selected}
            style={{ 'padding-left': `${8 + r.depth * 14}px` }}
            onClick={() => { props.onSelect(r.node.id); if (!r.leaf) void toggle(r.node) }}
            onDblClick={() => r.leaf && props.onOpen?.(r.node)}
            onContextMenu={(e) => { if (props.onMenu) { e.preventDefault(); props.onSelect(r.node.id); props.onMenu(r.node, e) } }}>
            <span class={`ide-twist${r.leaf ? ' none' : r.open ? ' open' : ''}`}><Icon name="chevron" size={12} /></span>
            <ItemRow item={r.node} />
          </li>
        )}
      </For>
    </ul>
  )
}

export type Column<R> = { id: string; label: string; value: (row: R) => unknown; align?: 'left' | 'right' }

const ROW = 22

/**
 * A table with a fixed header and virtual rows: only the visible ones are in the page, so it
 * holds a large result. A null or undefined cell shows NULL.
 */
export function Table<R>(props: { columns: Column<R>[]; rows: R[]; selected?: number; onSelect?: (i: number) => void; onOpen?: (row: R, i: number) => void; label?: string }) {
  let wrap!: HTMLDivElement
  const [top, setTop] = createSignal(0)
  const [height, setHeight] = createSignal(400)
  onMount(() => {
    const ro = new ResizeObserver(() => setHeight(wrap.clientHeight || 400))
    ro.observe(wrap)
    onCleanup(() => ro.disconnect())
  })
  const first = () => Math.max(0, Math.floor(top() / ROW) - 10)
  const last = () => Math.min(props.rows.length, Math.ceil((top() + height()) / ROW) + 10)
  const shown = createMemo(() => props.rows.slice(first(), last()).map((row, k) => ({ row, i: first() + k })))
  const key = (e: KeyboardEvent) => {
    if (!props.onSelect || !props.rows.length) return
    const i = props.selected ?? -1
    const page = Math.max(1, Math.floor(height() / ROW) - 1)
    const to = { ArrowDown: i + 1, ArrowUp: i - 1, Home: 0, End: props.rows.length - 1, PageDown: i + page, PageUp: i - page }[e.key]
    if (to !== undefined) {
      e.preventDefault()
      const n = Math.max(0, Math.min(props.rows.length - 1, to))
      props.onSelect(n)
      // The header is sticky: a row goes under it when scrolled to the top.
      if (n * ROW < wrap.scrollTop) wrap.scrollTop = n * ROW
      else if ((n + 2) * ROW > wrap.scrollTop + wrap.clientHeight) wrap.scrollTop = (n + 2) * ROW - wrap.clientHeight
    } else if (e.key === 'Enter' && i >= 0) props.onOpen?.(props.rows[i], i)
  }
  const cell = (v: unknown) => (v === null || v === undefined ? <span class="null">NULL</span> : typeof v === 'object' ? JSON.stringify(v) : String(v))
  return (
    <div class="ide-table-wrap" tabIndex={0} ref={wrap} onScroll={() => setTop(wrap.scrollTop)} onKeyDown={key}>
      <table class="ide-table" aria-label={props.label} aria-rowcount={props.rows.length + 1}>
        <thead>
          <tr><For each={props.columns}>{(c) => <th style={{ 'text-align': c.align ?? 'left' }}>{c.label}</th>}</For></tr>
        </thead>
        <tbody>
          <tr style={{ height: `${first() * ROW}px` }} aria-hidden="true" />
          <For each={shown()}>
            {(r) => (
              <tr style={{ height: `${ROW}px` }} aria-rowindex={r.i + 2} aria-selected={r.i === props.selected} onClick={() => props.onSelect?.(r.i)} onDblClick={() => props.onOpen?.(r.row, r.i)}>
                <For each={props.columns}>{(c) => <td style={{ 'text-align': c.align ?? 'left' }}>{cell(c.value(r.row))}</td>}</For>
              </tr>
            )}
          </For>
          <tr style={{ height: `${(props.rows.length - last()) * ROW}px` }} aria-hidden="true" />
        </tbody>
      </table>
    </div>
  )
}
