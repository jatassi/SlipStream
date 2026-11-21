import type { Problem } from "./problem.ts";
import type { SearchSubtitle401Code } from "./searchSubtitle401Code.ts";

export type SearchSubtitle401 = Problem & {
  code?: SearchSubtitle401Code;
};
