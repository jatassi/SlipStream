import type { ImportsRootLabel } from "./importsRootLabel.ts";

export interface ImportsLead {
  /**
   * The Works the root's last library scan added
   * @minimum 0
   */
  added: number;
  /** @pattern ^[a-z][a-z0-9]{1,15}$ */
  contentType: string;
  /**
   * The root's unfinished root Imports that aren't ignored
   * @minimum 0
   */
  needsYou: number;
  root: ImportsRootLabel;
}
