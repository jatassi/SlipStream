import type { NamingFieldPreview } from "./namingFieldPreview.ts";
import type { NamingTestedName } from "./namingTestedName.ts";
import type { NamingWorkFile } from "./namingWorkFile.ts";

export interface NamingPreview {
  /** Every setting, in the settings' order */
  fields: NamingFieldPreview[];
  /** Each of names, in order; empty while a setting has a problem */
  tested: NamingTestedName[];
  /** Each live file of workId's Work, in the order it was added; empty without workId and while a setting has a problem */
  work: NamingWorkFile[];
}
