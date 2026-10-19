import type { Problem } from "./problem.ts";
import type { SuggestFolders500Code } from "./suggestFolders500Code.ts";

export type SuggestFolders500 = Problem & {
  code?: SuggestFolders500Code;
};
