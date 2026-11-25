import type { DismissSuggestion422Code } from "./dismissSuggestion422Code.ts";
import type { Problem } from "./problem.ts";

export type DismissSuggestion422 = Problem & {
  code?: DismissSuggestion422Code;
};
