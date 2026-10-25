import type { Problem } from "./problem.ts";
import type { SuggestFolders404Code } from "./suggestFolders404Code.ts";

export type SuggestFolders404 = Problem & {
  code?: SuggestFolders404Code;
};
