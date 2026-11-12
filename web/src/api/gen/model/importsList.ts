import type { ImportsImportEntry } from "./importsImportEntry.ts";
import type { ImportsLead } from "./importsLead.ts";

export interface ImportsList {
  /** The unfinished Imports, in root and folder order */
  imports: ImportsImportEntry[];
  /** One per root a library scan compared, in root order; empty when source leaves out root */
  leads: ImportsLead[];
}
