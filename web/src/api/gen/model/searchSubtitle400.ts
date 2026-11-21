import type { Problem } from "./problem.ts";
import type { SearchSubtitle400Code } from "./searchSubtitle400Code.ts";

export type SearchSubtitle400 = Problem & {
  code?: SearchSubtitle400Code;
};
