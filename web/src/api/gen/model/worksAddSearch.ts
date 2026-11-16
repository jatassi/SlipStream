import type { WorksAddSearchKind } from "./worksAddSearchKind.ts";
import type { WorksSearchChoice } from "./worksSearchChoice.ts";

export interface WorksAddSearch {
  /** In the order the add sheet lists them */
  choices: WorksSearchChoice[];
  /** switch for a type of one Item, whose first choice is on and second off */
  kind: WorksAddSearchKind;
}
