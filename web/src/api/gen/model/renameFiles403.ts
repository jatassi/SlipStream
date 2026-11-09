import type { Problem } from "./problem.ts";
import type { RenameFiles403Code } from "./renameFiles403Code.ts";

export type RenameFiles403 = Problem & {
  code?: RenameFiles403Code;
};
