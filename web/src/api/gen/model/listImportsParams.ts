import type { ListImportsSource } from "./listImportsSource.ts";

export type ListImportsParams = {
  /**
   * Only Imports from this source; every source's when left out
   */
  source?: ListImportsSource;
  /**
   * Only this root's Imports and lead; every root's when left out
   */
  rootId?: string;
  /**
   * Only the Imports into this Work; every Work's when left out
   */
  work?: string;
  /**
   * List the ignored Imports instead of the ones that need the Admin
   */
  ignored?: boolean;
  /**
   * Only the Imports with a file that waits for the Admin
   */
  waiting?: boolean;
  /**
   * The next value of the page before; the first page when left out
   */
  after?: string;
  /**
   * The most Imports a page holds; every one when left out
   * @minimum 1
   * @maximum 500
   */
  limit?: number;
};
