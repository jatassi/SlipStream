import type { Problem } from "./problem.ts";
import type { RegenerateAPIKey500Code } from "./regenerateAPIKey500Code.ts";

export type RegenerateAPIKey500 = Problem & {
  code?: RegenerateAPIKey500Code;
};
