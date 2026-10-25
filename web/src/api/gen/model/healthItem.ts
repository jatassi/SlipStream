import type { HealthAction } from "./healthAction.ts";
import type { HealthFact } from "./healthFact.ts";
import type { HealthItemLevel } from "./healthItemLevel.ts";

export interface HealthItem {
  /** Its one action, null when it has none */
  action: HealthAction | null;
  /** When its level last changed */
  changedAt: string;
  detail: HealthFact[];
  /** Stable within its category and safe as one path segment */
  key: string;
  level: HealthItemLevel;
  /** When it first appeared at any level */
  since: string;
  subject: string;
  /** What to do first */
  text: string;
  /** SlipStream couldn't check it, so it shows the last known state */
  unsure: boolean;
}
