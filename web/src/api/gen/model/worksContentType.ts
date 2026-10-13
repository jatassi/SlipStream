import type { WorksFieldSpec } from "./worksFieldSpec.ts";
import type { WorksNoun } from "./worksNoun.ts";

export interface WorksContentType {
  enabled: boolean;
  fields: WorksFieldSpec[];
  /** @pattern ^[a-z][a-z0-9]{1,15}$ */
  id: string;
  item: WorksNoun;
  /** Outermost first; empty when Items hang off the Work */
  levels: WorksNoun[];
  name: string;
  work: WorksNoun;
}
