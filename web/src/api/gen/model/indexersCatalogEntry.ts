import type { IndexersCatalogEntryOrigin } from "./indexersCatalogEntryOrigin.ts";
import type { IndexersCatalogEntryPrivacy } from "./indexersCatalogEntryPrivacy.ts";

export interface IndexersCatalogEntry {
  /** The Newznab category ids it maps to */
  categories: number[];
  /** The definition's own words */
  description: string;
  id: string;
  /** A language tag such as en-US */
  language: string;
  name: string;
  origin: IndexersCatalogEntryOrigin;
  privacy: IndexersCatalogEntryPrivacy;
  /**
   * Why SlipStream can't run it; null when it can
   * @nullable
   */
  reason: string | null;
  /** A custom definition has this catalog entry's id and runs in its place */
  replacedByCustom: boolean;
  /** When SlipStream downloaded this version */
  version: string;
}
