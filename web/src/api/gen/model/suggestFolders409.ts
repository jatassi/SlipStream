import type { Problem } from "./problem.ts";
import type { SuggestFolders409Code } from "./suggestFolders409Code.ts";

export type SuggestFolders409 = Problem & {
  code?: SuggestFolders409Code;
};
