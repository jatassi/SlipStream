import type { ImportsEntry } from "./importsEntry.ts";
import type { ImportsLead } from "./importsLead.ts";

export interface ImportsList {
  /** In root and folder order */
  entries: ImportsEntry[];
  /** One per root a library scan compared, in root order */
  leads: ImportsLead[];
}
