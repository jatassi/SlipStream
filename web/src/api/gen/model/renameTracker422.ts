import type { Problem } from "./problem.ts";
import type { RenameTracker422Code } from "./renameTracker422Code.ts";

export type RenameTracker422 = Problem & {
  code?: RenameTracker422Code;
};
