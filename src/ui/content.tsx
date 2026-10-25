// Content of the IDE in a frame: Markdown rendered by the IDE (ide.markdown), the "?" of a help
// topic, the fuzzy match of the IDE's lists.
import { createEffect, createSignal, Show } from 'solid-js'
import './../global'

/**
 * Markdown as the IDE renders it, sanitized by the IDE; its links open in a tab of the browser
 * (ide.openLink: a frame has no popups).
 */
export function Markdown(props: { text: string; class?: string }) {
  const [html, setHtml] = createSignal('')
  createEffect(() => {
    const text = props.text
    void ide.markdown(text).then((h) => text === props.text && setHtml(h), () => setHtml(''))
  })
  return (
    <div
      class={`md ${props.class ?? ''}`}
      innerHTML={html()}
      onClick={(e) => {
        const a = (e.target as HTMLElement).closest('a')
        if (!a) return
        e.preventDefault()
        if (/^https?:\/\//i.test(a.href)) ide.openLink(a.href)
      }}
    />
  )
}

/** The "?" opening a page of the IDE's documentation by its help topic. */
export function HelpLink(props: { topic: string; title?: string }) {
  return (
    <Show when={typeof ide !== 'undefined'}>
      <button class="ide-help-link" title={props.title ?? 'Help'} aria-label="Help" onClick={() => ide.help(props.topic)}>
        ?
      </button>
    </Show>
  )
}

/** The fuzzy score of a query in a text, as the IDE's lists rank (0: no match). */
export function fuzzy(q: string, text: string): number {
  if (!q) return 1
  const t = text.toLowerCase()
  const query = q.toLowerCase()
  let score = 0
  let ti = 0
  let prev = -2
  for (const ch of query) {
    const i = t.indexOf(ch, ti)
    if (i < 0) return 0
    score += i === prev + 1 ? 5 : 1
    if (i === 0 || /[/_\-. A-Z]/.test(text[i - 1] ?? '') || text[i] !== t[i]) score += 3
    prev = i
    ti = i + 1
  }
  return score - t.length * 0.01
}
