import type { Problem } from "./problem.ts";
import type { SearchWork422Code } from "./searchWork422Code.ts";

export type SearchWork422 = Problem & {
  code?: SearchWork422Code;
};
