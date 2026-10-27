import type { IndexersSearchMode } from "./indexersSearchMode.ts";

export interface IndexersReport {
  /**
   * The newest items it listed
   * @minimum 0
   */
  items: number;
  /** By mode */
  searchModes: IndexersSearchMode[];
  /** The different sources a Torznab feed's items named: more than one is an aggregate feed */
  sources: string[];
}
