import type { BackupsDestination } from "./backupsDestination.ts";

export interface BackupsDestinationList {
  /** In the order they were added */
  destinations: BackupsDestination[];
}
