// The controls of the kit: buttons, text fields, select, checkbox, switch.
import { For, Show, splitProps, type JSX, type ParentProps } from 'solid-js'
import { Icon, type IconName } from './icons'

type ButtonProps = ParentProps<{ kind?: 'primary' | 'danger'; small?: boolean; icon?: IconName }> &
  JSX.ButtonHTMLAttributes<HTMLButtonElement>

export function Button(props: ButtonProps) {
  const [own, rest] = splitProps(props, ['kind', 'small', 'icon', 'children', 'class'])
  return (
    <button type="button" {...rest} class={['ide-btn', own.kind, own.small && 'small', own.class].filter(Boolean).join(' ')}>
      <Show when={own.icon}>{(i) => <Icon name={i()} size={14} />}</Show>
      {own.children}
    </button>
  )
}

/** A button with an icon only: its title is its label, for the screen readers too. */
export function IconButton(props: { icon: IconName; title: string; on?: boolean } & JSX.ButtonHTMLAttributes<HTMLButtonElement>) {
  const [own, rest] = splitProps(props, ['icon', 'title', 'on', 'class'])
  return (
    <button type="button" {...rest} class={['ide-icon-btn', own.on && 'on', own.class].filter(Boolean).join(' ')} title={own.title} aria-label={own.title} aria-pressed={own.on === undefined ? undefined : own.on}>
      <Icon name={own.icon} />
    </button>
  )
}

/** A label above its control. */
export function Field(props: ParentProps<{ label: string }>) {
  return (
    <label class="ide-field">
      <span>{props.label}</span>
      {props.children}
    </label>
  )
}

type InputProps = { value: string; onInput: (v: string) => void; small?: boolean } & Omit<JSX.InputHTMLAttributes<HTMLInputElement>, 'value' | 'onInput'>

export function TextInput(props: InputProps) {
  const [own, rest] = splitProps(props, ['value', 'onInput', 'small', 'class'])
  return <input {...rest} class={['ide-input', own.small && 'small', own.class].filter(Boolean).join(' ')} value={own.value} onInput={(e) => own.onInput(e.currentTarget.value)} />
}

export function TextArea(props: { value: string; onInput: (v: string) => void } & Omit<JSX.TextareaHTMLAttributes<HTMLTextAreaElement>, 'value' | 'onInput'>) {
  const [own, rest] = splitProps(props, ['value', 'onInput', 'class'])
  return <textarea {...rest} class={['ide-input', own.class].filter(Boolean).join(' ')} value={own.value} onInput={(e) => own.onInput(e.currentTarget.value)} />
}

export type Option<T extends string> = { value: T; label: string }

export function Select<T extends string>(props: { value: T; options: Option<T>[]; onChange: (v: T) => void; small?: boolean; title?: string; disabled?: boolean }) {
  return (
    <select class={['ide-input', props.small && 'small'].filter(Boolean).join(' ')} value={props.value} title={props.title} disabled={props.disabled} onChange={(e) => props.onChange(e.currentTarget.value as T)}>
      <For each={props.options}>{(o) => <option value={o.value}>{o.label}</option>}</For>
    </select>
  )
}

export function Checkbox(props: ParentProps<{ checked: boolean; onChange: (v: boolean) => void; disabled?: boolean }>) {
  return (
    <label class="ide-check">
      <input type="checkbox" checked={props.checked} disabled={props.disabled} onChange={(e) => props.onChange(e.currentTarget.checked)} />
      {props.children}
    </label>
  )
}

/** An on/off setting, applied at once (a checkbox is for a choice confirmed later). */
export function Switch(props: ParentProps<{ checked: boolean; onChange: (v: boolean) => void; disabled?: boolean }>) {
  return (
    <button type="button" role="switch" class="ide-switch" aria-checked={props.checked} disabled={props.disabled} onClick={() => props.onChange(!props.checked)}>
      <span class="track" />
      {props.children}
    </button>
  )
}
