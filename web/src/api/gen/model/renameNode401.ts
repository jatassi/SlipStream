import type { Problem } from "./problem.ts";
import type { RenameNode401Code } from "./renameNode401Code.ts";

export type RenameNode401 = Problem & {
  code?: RenameNode401Code;
};
