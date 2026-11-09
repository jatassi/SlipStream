import type { NamingFieldPreview } from "./namingFieldPreview.ts";

export interface NamingPreview {
  /** Every setting, in the settings' order */
  fields: NamingFieldPreview[];
}
