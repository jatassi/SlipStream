import type { FormatsFormatRef } from "./formatsFormatRef.ts";
import type { FormatsPastedFormatOutcome } from "./formatsPastedFormatOutcome.ts";

export interface FormatsPastedFormat {
  /** Its name in the text */
  name: string;
  /** Added, replacing sameName, or added beside sameName */
  outcome: FormatsPastedFormatOutcome;
  /** The Custom Format that already has its name */
  sameName?: FormatsFormatRef;
  /** The name it is saved under: its own, or with a number after it when another Custom Format has its name */
  savedAs: string;
}
