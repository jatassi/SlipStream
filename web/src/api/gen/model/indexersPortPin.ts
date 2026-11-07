import type { IndexersPortPinValues } from "./indexersPortPinValues.ts";

export interface IndexersPortPin {
  baseAddress: string;
  id: string;
  /** When the Prowlarr commit the port was synced to was made */
  synced: string;
  /** The settings saved in plain, by name. Sealed settings are never returned */
  values: IndexersPortPinValues;
}
