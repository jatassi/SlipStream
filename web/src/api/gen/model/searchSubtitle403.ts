import type { Problem } from "./problem.ts";
import type { SearchSubtitle403Code } from "./searchSubtitle403Code.ts";

export type SearchSubtitle403 = Problem & {
  code?: SearchSubtitle403Code;
};
