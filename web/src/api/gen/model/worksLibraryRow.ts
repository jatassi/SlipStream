import type { FilesState } from "./filesState.ts";
import type { FilesStateCounts } from "./filesStateCounts.ts";
import type { WorksLibraryRowMonitored } from "./worksLibraryRowMonitored.ts";
import type { WorksLibraryRowValues } from "./worksLibraryRowValues.ts";
import type { WorksRowRating } from "./worksRowRating.ts";

export interface WorksLibraryRow {
  addedAt: string;
  /**
   * Its one Item's availability date, or for many Items the earliest not yet out, else the latest out; null while no Item has a date
   * @nullable
   * @pattern ^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$
   */
  availableOn: string | null;
  /**
   * The size of its library files, Kept files included
   * @minimum 0
   */
  bytes: number;
  /** Its cells by state */
  cells: FilesStateCounts;
  /** @pattern ^[a-z][a-z0-9]{1,15}$ */
  contentType: string;
  /**
   * Its wanted cells whose download failed
   * @minimum 0
   */
  failed: number;
  /**
   * Its library files, Kept files included
   * @minimum 0
   */
  files: number;
  /**
   * When the metadata source last reported that it no longer has this Work; null while it has it
   * @nullable
   */
  goneUpstreamAt: string | null;
  id: string;
  /** Its Items by status; an Item with no cell isn't counted */
  itemStatus: FilesStateCounts;
  /** @minimum 0 */
  items: number;
  /**
   * Its Kept files
   * @minimum 0
   */
  kept: number;
  key: string;
  label: string;
  /** How many of its Items are monitored */
  monitored: WorksLibraryRowMonitored;
  /**
   * The entries of its Needs you
   * @minimum 0
   */
  needsYou: number;
  /**
   * The artwork route for its poster, without a size; null when it has none
   * @nullable
   */
  posterUrl: string | null;
  profileId: string;
  /** The first of its Content Type's ratings sources that it holds; null when it holds none */
  rating: WorksRowRating | null;
  refreshedAt: string;
  /**
   * Null while the Work has no root
   * @nullable
   */
  rootId: string | null;
  sortLabel: string;
  /** The first status its Items hold, in the glossary's order; null for a Work with no cell */
  status: FilesState | null;
  /** Its listed fields by key: a string for text and date, a number for number and duration in ms, a list of strings for texts; a field it lacks is left out */
  values: WorksLibraryRowValues;
}
