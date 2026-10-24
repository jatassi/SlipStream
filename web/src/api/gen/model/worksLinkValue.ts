import type { WorksLinkValueKind } from "./worksLinkValueKind.ts";

export interface WorksLinkValue {
  key: string;
  kind: WorksLinkValueKind;
  label: string;
  url: string;
}
