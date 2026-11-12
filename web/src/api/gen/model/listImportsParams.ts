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
   * List the ignored Imports instead of the ones that need the Admin
   */
  ignored?: boolean;
};
