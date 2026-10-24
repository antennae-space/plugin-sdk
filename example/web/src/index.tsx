// An example view on the kit, built by the tests with the preset: a list, a table, tabs, a call
// of the backend, the state of the tab.
import { createSignal } from 'solid-js'
import { useEditor, useIde, useRpc, useSettings, useViewState, view } from '@antennae-space/plugin-sdk/solid'
import { Button, EmptyState, List, Message, Table, Tabs, View } from '@antennae-space/plugin-sdk/ui'
import './example.css'

type Row = { name: string; size: number | null }

function Main() {
  const ide = useIde()
  const [tab, setTab] = useViewState<'files' | 'table'>('files')
  const [sel, setSel] = createSignal<string>()
  const files = useRpc<Row[]>('example.files')
  return (
    <View toolbar={<><Button icon="refresh" small onClick={() => files.reload()}>{ide.t('Reload')}</Button><span class="example-count">{ide.tn(files.value()?.length ?? 0, '{n} file', '{n} files')}</span></>}>
      <Tabs tabs={[{ id: 'files', label: ide.t('Files') }, { id: 'table', label: ide.t('Table') }]} active={tab()} onSelect={setTab} />
      {files.error() && <Message kind="error">{files.error()}</Message>}
      {tab() === 'files' ? (
        files.value()?.length ? (
          <List label="files" items={(files.value() ?? []).map((f) => ({ id: f.name, label: f.name, icon: 'file' as const }))} selected={sel()} onSelect={setSel} onOpen={(it) => ide.openFile(it.id)} />
        ) : (
          <EmptyState text={ide.t('No files')} />
        )
      ) : (
        <Table label="sizes" columns={[{ id: 'name', label: 'name', value: (r: Row) => r.name }, { id: 'size', label: 'size', value: (r: Row) => r.size, align: 'right' }]} rows={files.value() ?? []} />
      )}
    </View>
  )
}

// A tab view with the IDE's editor above: its text, and a key of the manifest.
function Console() {
  const ed = useEditor()!
  const settings = useSettings<{ rows?: number }>()
  const [ran, setRan] = createSignal('')
  ed.editor.onKey('run', () => setRan(ed.text()))
  return (
    <View>
      <p class="example-text">{ed.text()}</p>
      <p class="example-ran">{ran()}</p>
      <p class="example-rows">{settings().rows}</p>
    </View>
  )
}

ide.register({
  views: { main: view(Main), console: view(Console) },
  commands: { hello: (ide) => ide.toast(ide.t('Hello!'), 'ok') },
})
