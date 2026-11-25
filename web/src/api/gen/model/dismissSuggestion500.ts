import type { DismissSuggestion500Code } from "./dismissSuggestion500Code.ts";
import type { Problem } from "./problem.ts";

export type DismissSuggestion500 = Problem & {
  code?: DismissSuggestion500Code;
};
