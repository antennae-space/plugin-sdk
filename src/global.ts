// The global `ide` of a frame, typed for every module that imports the SDK.
import type { SandboxAPI } from './sandbox-api'

declare global {
  /** The sandbox API, set by the IDE in the frame before the plugin's module runs. */
  // eslint-disable-next-line no-var
  var ide: SandboxAPI
}

export {}
