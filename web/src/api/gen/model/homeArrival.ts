import type { HomeWork } from "./homeWork.ts";

export interface HomeArrival {
  /** When SlipStream imported its last file */
  at: string;
  importId: string;
  /** What it did, such as Imported 10 episodes · 2 upgrades */
  words: string;
  work: HomeWork;
}
