import type { Problem } from "./problem.ts";
import type { RenameNode404Code } from "./renameNode404Code.ts";

export type RenameNode404 = Problem & {
  code?: RenameNode404Code;
};
