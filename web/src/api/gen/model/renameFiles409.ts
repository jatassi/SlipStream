import type { Problem } from "./problem.ts";
import type { RenameFiles409Code } from "./renameFiles409Code.ts";

export type RenameFiles409 = Problem & {
  code?: RenameFiles409Code;
};
