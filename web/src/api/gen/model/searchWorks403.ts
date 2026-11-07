import type { Problem } from "./problem.ts";
import type { SearchWorks403Code } from "./searchWorks403Code.ts";

export type SearchWorks403 = Problem & {
  code?: SearchWorks403Code;
};
