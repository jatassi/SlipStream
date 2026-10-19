import type { WorksFieldValue } from "./worksFieldValue.ts";
import type { WorksItem } from "./worksItem.ts";
import type { WorksLevel } from "./worksLevel.ts";
import type { WorksRating } from "./worksRating.ts";

export interface WorksTree {
  addedAt: string;
  /** @pattern ^[a-z][a-z0-9]{1,15}$ */
  contentType: string;
  fields: WorksFieldValue[];
  /**
   * When the metadata source last answered that it no longer has this Work; null while it has it
   * @nullable
   */
  goneUpstreamAt: string | null;
  id: string;
  /** By their level's order, live before withdrawn, then position */
  items: WorksItem[];
  key: string;
  label: string;
  /** By depth, then their parent's order, live before withdrawn, then position */
  levels: WorksLevel[];
  /** The Work's own, by source */
  ratings: WorksRating[];
  refreshedAt: string;
  sortLabel: string;
}
