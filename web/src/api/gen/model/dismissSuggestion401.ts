import type { DismissSuggestion401Code } from "./dismissSuggestion401Code.ts";
import type { Problem } from "./problem.ts";

export type DismissSuggestion401 = Problem & {
  code?: DismissSuggestion401Code;
};
