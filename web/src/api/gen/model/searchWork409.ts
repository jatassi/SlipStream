import type { Problem } from "./problem.ts";
import type { SearchWork409Code } from "./searchWork409Code.ts";

export type SearchWork409 = Problem & {
  code?: SearchWork409Code;
};
