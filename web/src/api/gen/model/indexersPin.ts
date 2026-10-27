import type { IndexersPinValues } from "./indexersPinValues.ts";

export interface IndexersPin {
  baseAddress: string;
  id: string;
  /** The settings saved in plain, by name. Sealed settings are never returned */
  values: IndexersPinValues;
  /** The pinned version's git blob sha */
  version: string;
}
