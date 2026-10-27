import type { Problem } from "./problem.ts";
import type { RenameTracker401Code } from "./renameTracker401Code.ts";

export type RenameTracker401 = Problem & {
  code?: RenameTracker401Code;
};
