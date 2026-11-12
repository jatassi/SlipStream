import type { ActOnImportFile403Code } from "./actOnImportFile403Code.ts";
import type { Problem } from "./problem.ts";

export type ActOnImportFile403 = Problem & {
  code?: ActOnImportFile403Code;
};
