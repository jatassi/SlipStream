import type { IndexersPortPinValues } from "./indexersPortPinValues.ts";

export interface IndexersPortPin {
  baseAddress: string;
  id: string;
  /** The site's name */
  name: string;
  /** The names of the secret settings that have a value, in order */
  sealed: string[];
  /**
   * The date of the Prowlarr commit that SlipStream's support for this site is based on; null once the site is no longer built in
   * @nullable
   */
  synced: string | null;
  /** The settings that aren't secret, by name. Secret settings, such as passwords, are never returned */
  values: IndexersPortPinValues;
}
