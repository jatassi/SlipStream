import type { Problem } from "./problem.ts";
import type { SuggestFolders403Code } from "./suggestFolders403Code.ts";

export type SuggestFolders403 = Problem & {
  code?: SuggestFolders403Code;
};
