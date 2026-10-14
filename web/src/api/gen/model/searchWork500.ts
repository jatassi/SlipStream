import type { Problem } from "./problem.ts";
import type { SearchWork500Code } from "./searchWork500Code.ts";

export type SearchWork500 = Problem & {
  code?: SearchWork500Code;
};
