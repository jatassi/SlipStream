import type { Problem } from "./problem.ts";
import type { RegenerateAPIKey409Code } from "./regenerateAPIKey409Code.ts";

export type RegenerateAPIKey409 = Problem & {
  code?: RegenerateAPIKey409Code;
};
