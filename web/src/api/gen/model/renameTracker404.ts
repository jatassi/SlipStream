import type { Problem } from "./problem.ts";
import type { RenameTracker404Code } from "./renameTracker404Code.ts";

export type RenameTracker404 = Problem & {
  code?: RenameTracker404Code;
};
