import type { Problem } from "./problem.ts";
import type { RenameFiles422Code } from "./renameFiles422Code.ts";

export type RenameFiles422 = Problem & {
  code?: RenameFiles422Code;
};
