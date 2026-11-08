import type { IndexersPinValues } from "./indexersPinValues.ts";

export interface IndexersPin {
  baseAddress: string;
  /** When SlipStream downloaded the pinned version */
  fetchedAt: string;
  id: string;
  /** The names of the sealed settings that hold a value, in order */
  sealed: string[];
  /** The settings saved in plain, by name. Sealed settings are never returned */
  values: IndexersPinValues;
  /** The pinned version's git blob sha */
  version: string;
}
