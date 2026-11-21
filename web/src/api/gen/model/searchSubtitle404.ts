import type { Problem } from "./problem.ts";
import type { SearchSubtitle404Code } from "./searchSubtitle404Code.ts";

export type SearchSubtitle404 = Problem & {
  code?: SearchSubtitle404Code;
};
