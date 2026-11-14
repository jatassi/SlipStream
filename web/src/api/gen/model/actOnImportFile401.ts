import type { ActOnImportFile401Code } from "./actOnImportFile401Code.ts";
import type { Problem } from "./problem.ts";

export type ActOnImportFile401 = Problem & {
  code?: ActOnImportFile401Code;
};
