// The styles of the UI kit, added once to the frame (the CSP allows inline styles only).
import css from './ui.css?inline'

let done = false

export function injectStyle(): void {
  if (done || typeof document === 'undefined') return
  done = true
  const el = document.createElement('style')
  el.dataset.ide = 'ui'
  el.textContent = css
  document.head.prepend(el)
}
