import type { BackupsDestination } from "./backupsDestination.ts";
import type { BackupsRoom } from "./backupsRoom.ts";

export interface BackupsAdded {
  destination: BackupsDestination;
  /** The room the folder's volume lacks to hold every listed Backup; null when it has enough */
  room: BackupsRoom | null;
}
