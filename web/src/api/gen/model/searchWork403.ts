import type { Problem } from "./problem.ts";
import type { SearchWork403Code } from "./searchWork403Code.ts";

export type SearchWork403 = Problem & {
  code?: SearchWork403Code;
};
