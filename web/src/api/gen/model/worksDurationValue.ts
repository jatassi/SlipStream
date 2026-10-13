import type { WorksDurationValueKind } from "./worksDurationValueKind.ts";

export interface WorksDurationValue {
  key: string;
  kind: WorksDurationValueKind;
  /** @minimum 0 */
  ms: number;
}
