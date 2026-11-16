import type { IndexersPinValues } from "./indexersPinValues.ts";

export interface IndexersPin {
  baseAddress: string;
  /** When SlipStream downloaded the pinned version */
  fetchedAt: string;
  id: string;
  /** The names of the secret settings that have a value, in order */
  sealed: string[];
  /** The settings that aren't secret, by name. Secret settings, such as passwords, are never returned */
  values: IndexersPinValues;
  /** The pinned version's git blob sha */
  version: string;
}
