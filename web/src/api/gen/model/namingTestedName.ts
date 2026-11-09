import type { NamingUnplaced } from "./namingUnplaced.ts";

export interface NamingTestedName {
  /** Why it would go nowhere, then what to do, for any other reason */
  detail?: string;
  name: string;
  /** Where a file of that name would go if a library scan found it now, relative to the root; empty when it would go nowhere */
  path: string;
  /** Why it would go nowhere, when no Item of the Work holds the name */
  unplaced?: NamingUnplaced;
}
