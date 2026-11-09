import type { NamingSamplePath } from "./namingSamplePath.ts";

export interface NamingFieldPreview {
  key: string;
  /** What the save check refuses in the setting, each a sentence; empty when it passes */
  problems: string[];
  /** The full paths a folder or file template names for the type's samples, with renaming on; empty for any other setting and for a template that doesn't parse */
  samples: NamingSamplePath[];
}
