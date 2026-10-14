import type { WorksFieldValue } from "./worksFieldValue.ts";
import type { WorksItem } from "./worksItem.ts";
import type { WorksLevel } from "./worksLevel.ts";

export interface WorksTree {
  addedAt: string;
  /** @pattern ^[a-z][a-z0-9]{1,15}$ */
  contentType: string;
  fields: WorksFieldValue[];
  id: string;
  /** By their level's order, live before withdrawn, then position */
  items: WorksItem[];
  key: string;
  label: string;
  /** By depth, then their parent's order, live before withdrawn, then position */
  levels: WorksLevel[];
  refreshedAt: string;
  sortLabel: string;
}
