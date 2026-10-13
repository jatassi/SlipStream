import type { WorksDateValueKind } from "./worksDateValueKind.ts";

export interface WorksDateValue {
  /**
   * YYYY, YYYY-MM or YYYY-MM-DD
   * @pattern ^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$
   */
  date: string;
  key: string;
  kind: WorksDateValueKind;
}
