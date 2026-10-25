// The UI kit of the plugin SDK: Solid components with the look and the keyboard of the IDE's
// own tools. Their styles come with `view()` of `@antennae-space/plugin-sdk/solid`, or
// `injectStyle()` for a view rendered by hand.
import '../global'

export { Icon, icons, type IconName } from './icons'
export { Button, IconButton, Field, TextInput, TextArea, Select, Checkbox, Switch, type Option } from './controls'
export { View, Toolbar, Separator, EmptyState, Spinner, Message, Tabs, Splitter, type Tab } from './layout'
export { List, Tree, Table, type Item, type Node, type Column } from './collections'
export { openMenu, Modal, confirm, type MenuItem } from './overlays'
export { Markdown, HelpLink, fuzzy } from './content'
export { injectStyle } from './style'
