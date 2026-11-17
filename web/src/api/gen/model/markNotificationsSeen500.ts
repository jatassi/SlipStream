import type { MarkNotificationsSeen500Code } from "./markNotificationsSeen500Code.ts";
import type { Problem } from "./problem.ts";

export type MarkNotificationsSeen500 = Problem & {
  code?: MarkNotificationsSeen500Code;
};
