import type { IndexersQuery } from "./indexersQuery.ts";

export interface IndexersQueryList {
  /**
   * The offset of the next page; null on the last
   * @nullable
   */
  nextOffset: number | null;
  /** Newest first */
  queries: IndexersQuery[];
}
