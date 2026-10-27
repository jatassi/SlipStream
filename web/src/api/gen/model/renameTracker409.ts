import type { Problem } from "./problem.ts";
import type { RenameTracker409Code } from "./renameTracker409Code.ts";

export type RenameTracker409 = Problem & {
  code?: RenameTracker409Code;
};
