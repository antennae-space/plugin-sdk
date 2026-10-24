// Builds the example with the preset, then runs its module as the IDE does: one script in a
// document, `ide` given before it, the view mounted on #root.
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync, rmSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { build } from 'vite'
import { JSDOM } from 'jsdom'
import { idePlugin, maxBundle } from '../vite.js'

const root = fileURLToPath(new URL('../example/', import.meta.url))
const src = fileURLToPath(new URL('../src/', import.meta.url))
// The example imports the SDK by its name, as a plugin does.
const alias = { '@antennae-space/plugin-sdk/solid': src + 'solid.tsx', '@antennae-space/plugin-sdk/ui': src + 'ui/index.ts' }

async function bundle() {
  rmSync(root + 'web/index.js', { force: true })
  const out = await build({ root, configFile: false, logLevel: 'silent', resolve: { alias }, plugins: [idePlugin()] })
  const outputs = (Array.isArray(out) ? out : [out]).flatMap((o) => o.output)
  return { outputs, js: readFileSync(root + 'web/index.js', 'utf8') }
}

/** A frame with a fake `ide`: what it registers, what it asks. */
function frame(js, files) {
  const dom = new JSDOM('<!doctype html><html><head></head><body><div id="root"></div></body></html>', { runScripts: 'outside-only', pretendToBeVisual: true })
  const w = dom.window
  w.ResizeObserver = class { observe() {} disconnect() {} }
  w.HTMLElement.prototype.scrollIntoView = function () {}
  const calls = []
  let registered
  w.ide = {
    view: { id: 'main', state: undefined },
    lang: 'en',
    theme: { '--bg': '#000', 'color-scheme': 'dark' },
    register: (c) => (registered = c),
    rpc: async (method, params) => { calls.push(['rpc', method, params]); return files },
    on: () => {},
    openFile: (path, line) => calls.push(['openFile', path, line]),
    openView: () => {},
    setViewState: (s) => calls.push(['setViewState', s]),
    toast: (text, kind) => calls.push(['toast', text, kind]),
    t: (k) => k,
    tn: (n, one, other) => (n === 1 ? one : other).replace('{n}', String(n)),
    onTheme: () => {},
  }
  w.eval(js)
  return { w, calls, registered }
}

const tick = () => new Promise((r) => setTimeout(r, 10))

test('the preset builds one self-contained ES module', async () => {
  const { outputs, js } = await bundle()
  assert.deepEqual(outputs.map((o) => o.fileName), ['index.js'])
  assert.ok(js.length < maxBundle)
  assert.doesNotMatch(js, /^\s*import\s/m, 'no import left')
  assert.doesNotMatch(js, /\bimport\(/, 'no dynamic import')
  assert.match(js, /example-count/, 'the CSS of the plugin is in the module')
  assert.match(js, /ide-btn/, 'the CSS of the kit is in the module')
})

test('the view renders with the kit and talks to the IDE', async () => {
  const { js } = await bundle()
  const { w, calls, registered } = frame(js, [{ name: 'a.go', size: 10 }, { name: 'b.go', size: null }])
  assert.deepEqual(Object.keys(registered.views), ['main'])
  const el = w.document.getElementById('root')
  registered.views.main(el, w.ide)
  await tick()
  const d = w.document
  assert.equal(d.querySelectorAll('style[data-ide]').length, 2, 'kit and plugin styles injected')
  assert.equal(d.querySelector('.example-count').textContent, '2 files')
  assert.deepEqual(calls[0], ['rpc', 'example.files', undefined])

  // The list: arrows move the selection, Enter opens the file.
  const list = d.querySelector('[role=listbox]')
  list.dispatchEvent(new w.KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true }))
  list.dispatchEvent(new w.KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true }))
  await tick()
  assert.equal(d.querySelector('[role=option][aria-selected=true]').textContent, 'b.go')
  list.dispatchEvent(new w.KeyboardEvent('keydown', { key: 'Enter', bubbles: true }))
  assert.deepEqual(calls.at(-1), ['openFile', 'b.go', undefined])

  // The tabs: the choice is kept with the tab; the table shows NULL.
  d.querySelectorAll('[role=tab]')[1].click()
  await tick()
  assert.deepEqual(calls.at(-1), ['setViewState', 'table'])
  const cells = [...d.querySelectorAll('.ide-table td')].map((td) => td.textContent)
  assert.deepEqual(cells, ['a.go', '10', 'b.go', 'NULL'])

  // A command.
  registered.commands.hello(w.ide)
  assert.deepEqual(calls.at(-1), ['toast', 'Hello!', 'ok'])
})
