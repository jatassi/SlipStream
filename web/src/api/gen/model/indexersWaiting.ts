import type { IndexersWaitingStanding } from "./indexersWaitingStanding.ts";

export interface IndexersWaiting {
  /** The home Node's last Sync frame, or the Controller's start when it sent none since */
  since: string;
  /** How the home Node stands, out of step within the offline grace: until the grace passes, its searches finish indexer_unavailable */
  standing: IndexersWaitingStanding;
}
