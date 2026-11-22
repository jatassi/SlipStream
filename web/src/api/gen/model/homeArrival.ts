import type { HomeWork } from "./homeWork.ts";

export interface HomeArrival {
  /** When its last file landed */
  at: string;
  importId: string;
  /** What it did, such as Imported 10 episodes · 2 upgrades */
  words: string;
  work: HomeWork;
}
