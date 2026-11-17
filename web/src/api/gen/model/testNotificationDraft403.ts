import type { Problem } from "./problem.ts";
import type { TestNotificationDraft403Code } from "./testNotificationDraft403Code.ts";

export type TestNotificationDraft403 = Problem & {
  code?: TestNotificationDraft403Code;
};
