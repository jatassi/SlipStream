import type { Problem } from "./problem.ts";
import type { RenameFiles500Code } from "./renameFiles500Code.ts";

export type RenameFiles500 = Problem & {
  code?: RenameFiles500Code;
};
