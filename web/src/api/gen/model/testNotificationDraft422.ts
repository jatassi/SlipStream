import type { Problem } from "./problem.ts";
import type { TestNotificationDraft422Code } from "./testNotificationDraft422Code.ts";

export type TestNotificationDraft422 = Problem & {
  code?: TestNotificationDraft422Code;
  field?: string;
};
