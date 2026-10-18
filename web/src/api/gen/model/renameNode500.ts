import type { Problem } from "./problem.ts";
import type { RenameNode500Code } from "./renameNode500Code.ts";

export type RenameNode500 = Problem & {
  code?: RenameNode500Code;
};
