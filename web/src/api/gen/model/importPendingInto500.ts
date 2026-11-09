import type { ImportPendingInto500Code } from "./importPendingInto500Code.ts";
import type { Problem } from "./problem.ts";

export type ImportPendingInto500 = Problem & {
  code?: ImportPendingInto500Code;
};
