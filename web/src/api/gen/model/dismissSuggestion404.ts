import type { DismissSuggestion404Code } from "./dismissSuggestion404Code.ts";
import type { Problem } from "./problem.ts";

export type DismissSuggestion404 = Problem & {
  code?: DismissSuggestion404Code;
};
