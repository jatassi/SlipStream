import type { HealthActionKind } from "./healthActionKind.ts";

export interface HealthAction {
  href?: string;
  /** check: Check now on the item; run: Run now on taskId; link: a page in the web */
  kind: HealthActionKind;
  /** The link's words */
  label?: string;
  taskId?: string;
}
