import type { FilesState } from "./filesState.ts";
import type { FilesStateCounts } from "./filesStateCounts.ts";
import type { WorksLibraryRowAddedBy } from "./worksLibraryRowAddedBy.ts";
import type { WorksLibraryRowMonitored } from "./worksLibraryRowMonitored.ts";
import type { WorksLibraryRowSubtitles } from "./worksLibraryRowSubtitles.ts";
import type { WorksLibraryRowValues } from "./worksLibraryRowValues.ts";
import type { WorksLibraryRowWanted } from "./worksLibraryRowWanted.ts";
import type { WorksRowRating } from "./worksRowRating.ts";

export interface WorksLibraryRow {
  addedAt: string;
  /** Who added it: an Admin, or a library scan that matched a folder on disk */
  addedBy: WorksLibraryRowAddedBy;
  /**
   * Its one Item's availability date, or for many Items the earliest not yet out, else the latest out, read over its monitored Items, or over every Item while none is monitored; null while none of those has a date
   * @nullable
   * @pattern ^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$
   */
  availableOn: string | null;
  /**
   * The size of its library files, Kept files included
   * @minimum 0
   */
  bytes: number;
  /** Its monitored Items' Editions at each Target, counted by state */
  cells: FilesStateCounts;
  /** @pattern ^[a-z][a-z0-9]{1,15}$ */
  contentType: string;
  /**
   * Its library files, Kept files included
   * @minimum 0
   */
  files: number;
  /**
   * Its files a library scan found missing from its root, which the other counts leave out
   * @minimum 0
   */
  goneFiles: number;
  /**
   * When the metadata source last reported that it no longer has this Work; null while it has it
   * @nullable
   */
  goneUpstreamAt: string | null;
  id: string;
  /** Its monitored Items by status; an Item with no Target isn't counted */
  itemStatus: FilesStateCounts;
  /**
   * Its monitored Items that its source still lists, the only Items the status and counts read
   * @minimum 0
   */
  items: number;
  /**
   * Its Kept files
   * @minimum 0
   */
  kept: number;
  key: string;
  label: string;
  /** How many of the Items its source still lists are monitored */
  monitored: WorksLibraryRowMonitored;
  /**
   * The entries of its Needs you, leaving out not_found ones, which show on the Work page only
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
  /** The most urgent status among its monitored Items, in the order failed, missing, downloading, upgradable, unreleased, met; null while none of them has a Target */
  status: FilesState | null;
  /**
   * missing while a wanted subtitle of one of its files has nothing that meets it, none_passed when the last search for one found subtitles and none matched the timing; null while every wanted subtitle is met
   * @nullable
   */
  subtitles: WorksLibraryRowSubtitles;
  /** Its listed fields by key: a string for text and date, a number for number and duration in ms, a list of strings for texts; a field it lacks is left out */
  values: WorksLibraryRowValues;
  /**
   * The first reason Wanted lists it for, in this order; null while Wanted doesn't list it
   * @nullable
   */
  wanted: WorksLibraryRowWanted;
}
