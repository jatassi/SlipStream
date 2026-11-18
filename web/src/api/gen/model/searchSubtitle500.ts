import type { Problem } from "./problem.ts";
import type { SearchSubtitle500Code } from "./searchSubtitle500Code.ts";

export type SearchSubtitle500 = Problem & {
  code?: SearchSubtitle500Code;
};
