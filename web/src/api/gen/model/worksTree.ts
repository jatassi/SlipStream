import type { WorksFieldValue } from "./worksFieldValue.ts";
import type { WorksItem } from "./worksItem.ts";
import type { WorksLevel } from "./worksLevel.ts";

export interface WorksTree {
  fields: WorksFieldValue[];
  /** By their level's order, live before withdrawn, then position */
  items: WorksItem[];
  /** By depth, then their parent's order, live before withdrawn, then position */
  levels: WorksLevel[];
}
