import type { Problem } from "./problem.ts";
import type { SearchWork401Code } from "./searchWork401Code.ts";

export type SearchWork401 = Problem & {
  code?: SearchWork401Code;
};
