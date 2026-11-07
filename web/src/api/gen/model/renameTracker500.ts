import type { Problem } from "./problem.ts";
import type { RenameTracker500Code } from "./renameTracker500Code.ts";

export type RenameTracker500 = Problem & {
  code?: RenameTracker500Code;
};
