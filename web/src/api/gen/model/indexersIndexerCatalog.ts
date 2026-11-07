import type { IndexersCatalogEntry } from "./indexersCatalogEntry.ts";

export interface IndexersIndexerCatalog {
  /** The catalog's base address, to which SlipStream adds the schema version */
  address: string;
  addressIsDefault: boolean;
  /** The catalog's definitions by name, then the custom definitions by id */
  entries: IndexersCatalogEntry[];
  /**
   * When the Indexer catalog refresh runs next; null while it runs or waits
   * @nullable
   */
  nextRefreshAt: string | null;
  /**
   * Why the catalog lists no site yet; null once it does
   * @nullable
   */
  notice: string | null;
  /**
   * Null until a refresh has downloaded the catalog
   * @nullable
   */
  refreshedAt: string | null;
}
