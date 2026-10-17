import type { GetAPIKey401Code } from "./getAPIKey401Code.ts";
import type { Problem } from "./problem.ts";

export type GetAPIKey401 = Problem & {
  code?: GetAPIKey401Code;
};
