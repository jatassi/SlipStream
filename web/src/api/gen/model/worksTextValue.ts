import type { WorksTextValueKind } from "./worksTextValueKind.ts";

export interface WorksTextValue {
  key: string;
  kind: WorksTextValueKind;
  text: string;
}
