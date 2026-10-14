import type { Problem } from "./problem.ts";
import type { SearchWork404Code } from "./searchWork404Code.ts";

export type SearchWork404 = Problem & {
  code?: SearchWork404Code;
};
