import type { WorksNumberValueKind } from "./worksNumberValueKind.ts";

export interface WorksNumberValue {
  key: string;
  kind: WorksNumberValueKind;
  number: number;
}
