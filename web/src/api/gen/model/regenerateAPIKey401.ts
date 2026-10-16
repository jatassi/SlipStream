import type { Problem } from "./problem.ts";
import type { RegenerateAPIKey401Code } from "./regenerateAPIKey401Code.ts";

export type RegenerateAPIKey401 = Problem & {
  code?: RegenerateAPIKey401Code;
};
