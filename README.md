# Plugin SDK of Web IDE

What an external plugin of [Web IDE](https://github.com/antennae-space/web-ide) is built with:

- **The types of the sandbox API**: the global `ide` its views get in their frame, and the messages behind it (`src/sandbox-api.ts`, a copy of the IDE's own file, checked by the CI).
- **A UI kit in Solid**, with the look, the theme and the keyboard of the IDE's own tools.
- **A Vite preset** building the page module as the IDE loads it: one ES module, nothing apart.
- **A Go module** for the confined backend: `antennae.space/plugin-sdk/backend`, JSON-RPC over stdio.
- **The JSON Schema of `plugin.json`** (`plugin.schema.json`), language packs included (`contributes.languages`, `contributes.debuggers`, grants `program:`, `path:` and `debugger:`, the `install` recipe of a server or an adapter the IDE installs when missing; [lang-go](https://github.com/antennae-space/lang-go) is one).

Start from [plugin-template](https://github.com/antennae-space/plugin-template); the IDE's how-to [Write a plugin](https://github.com/antennae-space/web-ide/blob/main/docs/howto/write-a-plugin.md) explains the rest (manifest, grants, releases). [plugin-greet](https://github.com/antennae-space/plugin-greet) is a small example.

## Versions

The major version is the plugin API's: `v1.x` is `apiVersion: 1`. A minor version adds, never removes.

## The page module

```sh
npm i -D git+https://github.com/antennae-space/plugin-sdk.git#v1.17.0 solid-js vite vite-plugin-solid
```

- `git+https`, not `github:` (npm tries ssh first) nor the archive of the tag (npm 10 may hang on it).
- npm 10 fails on the optional peers of Vite 8: put `legacy-peer-deps=true` in the `.npmrc` of the plugin (the peers are listed above).

```js
// vite.config.js
import { defineConfig } from 'vite'
import { idePlugin } from '@antennae-space/plugin-sdk/vite'

export default defineConfig({ plugins: [idePlugin()] }) // web/src/index.tsx → web/index.js
```

```tsx
// web/src/index.tsx
import { useIde, useRpc, view } from '@antennae-space/plugin-sdk/solid'
import { Button, List, View } from '@antennae-space/plugin-sdk/ui'

function Files() {
  const ide = useIde()
  const files = useRpc<string[]>('myplugin.files')
  return (
    <View toolbar={<Button icon="refresh" small onClick={() => files.reload()}>{ide.t('Reload')}</Button>}>
      <List items={(files.value() ?? []).map((f) => ({ id: f, label: f, icon: 'file' }))} onSelect={() => {}} onOpen={(f) => ide.openFile(f.id)} />
    </View>
  )
}

ide.register({ views: { files: view(Files) } })
```

The frame has no network and no storage (its CSP: inline scripts and styles, images as data): the preset inlines the CSS and the assets, and refuses a second file. Ask the backend for what is outside.

- `@antennae-space/plugin-sdk`: the types (`SandboxAPI`, `FrameCall`, `PageMessage`…), `themeTokens`, `isDark()`, and the global `ide`.
- `@antennae-space/plugin-sdk/solid`: `view(Component)` (a `views` entry, with the kit's styles), `useIde()`, `useTheme()`, `useViewState(initial)` (the state kept with the tab), `useDesk(initial)` (its state in the desk shown, each desk its own: `ide.desk`, `ide.setDesk`, `ide.onDesk`; IDE 1.25), `useEvent(event, f)`, `useRpc(method, params)`, `useEditor()` (the IDE's editor of a tab view with `editor` in the manifest: `{editor, text, selection}`), `useSettings()` (the plugin's settings), `useModal()` (in a modal view: `close(result)`), `useStatus(items, onChoice)` (the status bar items of a tab view and the choices of their menus), `useToolMenu(items, onChoice)` (the items of the ⋮ menu of its tool or tab, after the IDE's; IDE 1.24), `useService(capability)` (a capability another external plugin provides, one the manifest requires: `call(method, ...args)`, `value(key)` an accessor, `on(event, f)`; IDE 1.24). The dialogs of the IDE, centered in its window: `ide.confirm`, `ide.prompt`, `ide.pick`, and `ide.openModal(view, state, {title, size})` for a view of `contributes.modals` (IDE 1.24); the kit's `confirm` uses `ide.confirm` when the IDE has it.
- `@antennae-space/plugin-sdk/ui`: `Button`, `IconButton`, `Field`, `TextInput`, `TextArea`, `Select`, `Checkbox`, `Switch`, `View`, `Toolbar`, `Separator`, `Tabs`, `Splitter`, `EmptyState`, `Message`, `Spinner`, `List`, `Tree` (children loaded on opening), `Table` (virtual rows), `openMenu`, `Modal`, `confirm`, `Icon` (a part of the IDE's icons), `Markdown` (rendered and sanitized by the IDE, its links in a tab of the browser), `HelpLink` (a page of the IDE's documentation), `fuzzy` (IDE 1.24).

The kit follows the rules of the IDE: text never under 12px, the focus always visible, one tab stop per list, tree or table and the arrows inside it. Its colors are the theme's CSS variables (`themeTokens`: `--bg`, `--fg`, `--accent`…), set on the frame by the IDE and changed with the theme; use them in the plugin's own CSS too. `--ide-head-inset` is the room the IDE's own buttons take over the top right corner of the frame: the kit's toolbar leaves it; a header of the plugin's own takes the class `ide-head`.

## The backend

```go
package main

import (
	"context"
	"encoding/json"

	"antennae.space/plugin-sdk/backend"
)

func main() {
	backend.Serve(backend.Plugin{
		Methods: map[string]backend.Method{
			"myplugin.files": func(ctx context.Context, _ json.RawMessage) (any, error) {
				var entries []struct{ Name string }
				// A method of the core, with the grant core:fs.list.
				if err := backend.Core(ctx, "fs.list", map[string]string{"path": ""}, &entries); err != nil {
					return nil, err
				}
				names := []string{}
				for _, e := range entries {
					names = append(names, e.Name)
				}
				return names, nil
			},
		},
		Tools: map[string]backend.Tool{
			"myplugin_count": func(ctx context.Context, args json.RawMessage) (backend.Result, error) {
				return backend.Result{Content: "…", Summary: "…"}, nil
			},
		},
	})
}
```

- `Init` runs at `initialize` with the `Info` (grants, project, version); calls wait for it.
- `backend.Core(ctx, method, params, &out)`: a method of the core, refused without the grant `core:<method>`.
- `backend.Emit(ctx, "myplugin.changed", data)`: an event to the windows of the project (`ide.on`); `backend.EmitFamily` to those of its main project and of its worktrees too (IDE 1.24).
- `backend.Dial(ctx, host, port, ssh)`: a connection the IDE opens for the plugin (grant `dial`; through an SSH tunnel of the user's agent and keys with `&backend.SSH{…}` and the grant `ssh`), a `net.Conn`. The user allows each new destination once: the call waits for the answer. A confined backend has no network of its own unless granted `net`.
- `backend.ServiceCall(ctx, capability, method, params, &out)` and `backend.ServiceOn(ctx, capability)`: the backend providing a capability the manifest requires, its method `<provider>.*` and its events (to `Plugin.Events`); a provider's method knows who called it with `backend.FromOf(ctx)` (IDE 1.24). A plugin providing a capability registers its methods for the page in `ide.register({services: {<cap>: {...}}})` and publishes with `ide.publish` / `ide.emit`.
- `Plugin.MCP` (IDE 1.24): its MCP tools (`contributes.mcpTools`, described in `toolSchemas`), each a `func(ctx, Scope, args) (string, error)`; `Scope` is the project of the client's folder, its main project, the ticket of its worktree, the address of the IDE.
- `Plugin.Questions` (`contributes.questions: true`): `List` its questions waiting for the user (`Question`: a permission with `Options`, or a form with a `Schema`), `Answer` the user's answer; `backend.QuestionsChanged(ctx, asked)` when they change.
- `backend.Notify(ctx, Note{Kind, Query, Texts})`: a push notification to the user's devices (grant `notify`), opening the project at `?<Query>`.
- `Plugin.Notify["project.open"]` with `contributes.hooks: ["project.open"]`: a window opened the project.
- The bridges of the IDE (grants `use:terminal`, `use:assistant`, `use:agents`, `use:commands`): `useService('terminal')` in a view (`open`, `show`), `assistant` (`show` too), `agents` (`show(session)`), `commands` (the saved commands: `load`, `edit`, the value `list`), and `backend.ServiceCall(ctx, "agents", "agents.develop", …)` in the backend: a reviewed part of a built-in capability (IDE 1.24). `ide.showWorktree(project)` shows a worktree of the project, answering once it shows. In `plugin.json`, `uses` lists the capabilities a plugin calls when they are there, without needing them.
- `backend.Git(ctx, dir, args...)`, `backend.GitAny` (the output and the exit code, whatever it is), `backend.GH(ctx, args...)` (grants `git`, `gh`): git and `gh pr` run by the IDE with the user's keys, in the project or one of its `.ide/worktrees/*`; `backend.RegisterWorktree`, `UnregisterWorktree`, `SetupWorktree`, `KeepIdeOut` (grant `worktrees`): the worktree projects (IDE 1.24).
- `backend.Exec(ctx, argv, dir)`: runs a program on the host of an SSH project (grant `exec:<name>`, `exec:sqlite3`; nothing confines it there) and returns its output; refused on a local project, where the backend runs its programs itself.
- `backend.Socket(ctx, "docker")`: the Docker socket, opened by the IDE (grant `socket:docker`, as much as root on the host: the IDE says so), the host's on an SSH project; a `net.Conn` speaking the Docker Engine API.
- `backend.OpenTerminal(ctx, title)`: a console of the IDE the plugin feeds (grant `terminal`): write its output, read what is typed, `OnResize`; show it with `ide.showConsole(term.ID)` in a view. `Plugin.Notify` gets the other notifications of the IDE.
- `backend.Secret(ctx, key)`, `SetSecret`, `DeleteSecret`: the plugin's entries of the vault (grant `vault`), synced with the user's other pods; passwords and tokens go there, never in the plugin's files.
- `backend.Settings(ctx, &out)`: the plugin's settings (`contributes.settings`); `backend.DeclareTargets(ctx, targets)`: what its actions act on, for the user's AI access in Settings.
- `backend.InfoOf(ctx)`, `backend.LangOf(ctx)` (the language of the window or the conversation).
- A coded error, which the views tell apart (`err.code`, `err.data` in the frame): `backend.Coded("db_password", msg, data)`; `backend.CodeOf(err)` reads the code of one the IDE's core answered (`auth_required`: ask the SSH password). A tool's error goes to the model as its result.
- `Info.Project.Main`, `MainRoot`, `MainName`: the main project of a worktree, its folder and its name; key per-project data by it.
- A tab view's frame: `ide.project` (the project of the window, pushed again when it shows another worktree), `ide.setTitle`, `ide.closeView`, `ide.showTool`; `"alone": true` on a tab view keeps it a tab of its own in the Full mode (IDE 1.24).
- Standard error goes to `backend.log` of its data folder; standard output is the protocol, never print to it.

## Develop

```sh
npm ci && npm run check && npm test   # the types, the example built and run in jsdom
go test -race ./...                   # the backend against a fake IDE
npm run check-api                     # the sandbox API, against web-ide main (WEB_IDE_DIR for a checkout)
```

## License

MIT
