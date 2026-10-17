import type { GetAPIKey500Code } from "./getAPIKey500Code.ts";
import type { Problem } from "./problem.ts";

export type GetAPIKey500 = Problem & {
  code?: GetAPIKey500Code;
};
