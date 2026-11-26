import type { Problem } from "./problem.ts";
import type { UseAnotherPlexAccount401Code } from "./useAnotherPlexAccount401Code.ts";

export type UseAnotherPlexAccount401 = Problem & {
  code?: UseAnotherPlexAccount401Code;
};
