// Checks that src/sandbox-api.ts is the IDE's: web/src/plugins/sandbox-api.ts of web-ide, on
// main or on the ref given (WEB_IDE_REF), or a local checkout (WEB_IDE_DIR). The IDE's file is
// the source of truth: the SDK's copy follows it.
import { readFileSync } from 'node:fs'

const ours = readFileSync(new URL('../src/sandbox-api.ts', import.meta.url), 'utf8')
const path = 'web/src/plugins/sandbox-api.ts'
let theirs
if (process.env.WEB_IDE_DIR) theirs = readFileSync(`${process.env.WEB_IDE_DIR}/${path}`, 'utf8')
else {
  const ref = process.env.WEB_IDE_REF ?? 'main'
  const r = await fetch(`https://raw.githubusercontent.com/antennae-space/web-ide/${ref}/${path}`)
  if (!r.ok) throw new Error(`${path} of web-ide ${ref}: ${r.status}`)
  theirs = await r.text()
}
if (ours !== theirs) {
  console.error(`src/sandbox-api.ts differs from ${path} of web-ide: copy it again (and add what it adds to the kit).`)
  process.exit(1)
}
console.log('sandbox-api.ts: the same as the IDE')
