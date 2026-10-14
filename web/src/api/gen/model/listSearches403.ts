import type { ListSearches403Code } from "./listSearches403Code.ts";
import type { Problem } from "./problem.ts";

export type ListSearches403 = Problem & {
  code?: ListSearches403Code;
};
