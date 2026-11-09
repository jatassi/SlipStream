import type { Problem } from "./problem.ts";
import type { RenameFiles401Code } from "./renameFiles401Code.ts";

export type RenameFiles401 = Problem & {
  code?: RenameFiles401Code;
};
