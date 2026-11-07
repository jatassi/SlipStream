import type { Problem } from "./problem.ts";
import type { RenameTracker403Code } from "./renameTracker403Code.ts";

export type RenameTracker403 = Problem & {
  code?: RenameTracker403Code;
};
