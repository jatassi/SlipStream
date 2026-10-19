import type { Problem } from "./problem.ts";
import type { SuggestFolders503Code } from "./suggestFolders503Code.ts";

export type SuggestFolders503 = Problem & {
  code?: SuggestFolders503Code;
};
