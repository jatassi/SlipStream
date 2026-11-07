import type { IndexersPortPinValues } from "./indexersPortPinValues.ts";

export interface IndexersPortPin {
  baseAddress: string;
  id: string;
  /** The site's name */
  name: string;
  /**
   * When the Prowlarr commit the port was synced to was made; null once SlipStream no longer includes the site
   * @nullable
   */
  synced: string | null;
  /** The settings saved in plain, by name. Sealed settings are never returned */
  values: IndexersPortPinValues;
}
