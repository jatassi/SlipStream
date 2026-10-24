import type { WorksTextsValueKind } from "./worksTextsValueKind.ts";

export interface WorksTextsValue {
  key: string;
  kind: WorksTextsValueKind;
  /** In the source's order */
  texts: string[];
}
