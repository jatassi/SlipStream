import type { Problem } from "./problem.ts";
import type { SuggestFolders401Code } from "./suggestFolders401Code.ts";

export type SuggestFolders401 = Problem & {
  code?: SuggestFolders401Code;
};
