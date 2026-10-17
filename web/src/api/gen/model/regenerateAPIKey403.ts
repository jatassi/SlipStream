import type { Problem } from "./problem.ts";
import type { RegenerateAPIKey403Code } from "./regenerateAPIKey403Code.ts";

export type RegenerateAPIKey403 = Problem & {
  code?: RegenerateAPIKey403Code;
};
