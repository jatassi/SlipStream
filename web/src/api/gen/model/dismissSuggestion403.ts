import type { DismissSuggestion403Code } from "./dismissSuggestion403Code.ts";
import type { Problem } from "./problem.ts";

export type DismissSuggestion403 = Problem & {
  code?: DismissSuggestion403Code;
};
