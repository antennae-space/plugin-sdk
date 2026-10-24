# Plugin SDK of Web IDE

What an external plugin of [Web IDE](https://github.com/antennae-space/web-ide) is built with:

- **The types of the sandbox API**: the global `ide` its views get in their frame, and the messages behind it (`src/sandbox-api.ts`, a copy of the IDE's own file, checked by the CI).
- **A UI kit in Solid**, with the look, the theme and the keyboard of the IDE's own tools.
- **A Vite preset** building the page module as the IDE loads it: one ES module, nothing apart.
- **A Go module** for the confined backend: `antennae.space/plugin-sdk/backend`, JSON-RPC over stdio.
- **The JSON Schema of `plugin.json`** (`plugin.schema.json`).

Start from [plugin-template](https://github.com/antennae-space/plugin-template); the IDE's how-to [Write a plugin](https://github.com/antennae-space/web-ide/blob/main/docs/howto/write-a-plugin.md) explains the rest (manifest, grants, releases). [plugin-greet](https://github.com/antennae-space/plugin-greet) is a small example.

## Versions

The major version is the plugin API's: `v1.x` is `apiVersion: 1`. A minor version adds, never removes.

## The page module

```sh
npm i -D github:antennae-space/plugin-sdk#v1.0.0 solid-js vite vite-plugin-solid
```

npm 10 fails on the optional peers of Vite 8: put `legacy-peer-deps=true` in the `.npmrc` of the plugin (the peers are listed above).

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
- `@antennae-space/plugin-sdk/solid`: `view(Component)` (a `views` entry, with the kit's styles), `useIde()`, `useTheme()`, `useViewState(initial)` (the state kept with the tab), `useEvent(event, f)`, `useRpc(method, params)`.
- `@antennae-space/plugin-sdk/ui`: `Button`, `IconButton`, `Field`, `TextInput`, `TextArea`, `Select`, `Checkbox`, `Switch`, `View`, `Toolbar`, `Separator`, `Tabs`, `Splitter`, `EmptyState`, `Message`, `Spinner`, `List`, `Tree` (children loaded on opening), `Table` (virtual rows), `openMenu`, `Modal`, `confirm`, `Icon` (a part of the IDE's icons).

The kit follows the rules of the IDE: text never under 12px, the focus always visible, one tab stop per list, tree or table and the arrows inside it. Its colors are the theme's CSS variables (`themeTokens`: `--bg`, `--fg`, `--accent`…), set on the frame by the IDE and changed with the theme; use them in the plugin's own CSS too.

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
- `backend.Emit(ctx, "myplugin.changed", data)`: an event to the windows of the project (`ide.on`).
- `backend.InfoOf(ctx)`, `backend.LangOf(ctx)` (the language of the window or the conversation).
- An error with a code: `&backend.Error{Code, Message}`. A tool's error goes to the model as its result.
- Standard error goes to `backend.log` of its data folder; standard output is the protocol, never print to it.

## Develop

```sh
npm ci && npm run check && npm test   # the types, the example built and run in jsdom
go test -race ./...                   # the backend against a fake IDE
npm run check-api                     # the sandbox API, against web-ide main (WEB_IDE_DIR for a checkout)
```

## License

MIT
